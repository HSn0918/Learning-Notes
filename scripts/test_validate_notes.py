from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from scripts.validate_notes import validate


class ValidateNotesTest(unittest.TestCase):
    def make_repo(self) -> tuple[tempfile.TemporaryDirectory[str], Path]:
        temporary = tempfile.TemporaryDirectory()
        root = Path(temporary.name)
        (root / "README.md").write_text("# Root\n\n[Notes](notes/)\n", encoding="utf-8")
        (root / "notes").mkdir()
        (root / "notes" / "图片").mkdir()
        return temporary, root

    def test_alias_anchor_image_size_and_directory_readme(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text(
            "# Notes\n\n[One](one.md)\n\n[[two|Two]]\n", encoding="utf-8"
        )
        (root / "notes" / "one.md").write_text(
            "#tag\n\n# One\n\n[[two#Section|jump]]\n\n![[diagram.png|400]]\n",
            encoding="utf-8",
        )
        (root / "notes" / "two.md").write_text("#tag\n\n# Two\n\n## Section\n", encoding="utf-8")
        (root / "notes" / "图片" / "diagram.png").write_bytes(b"png")

        self.assertEqual([], validate(root).errors)

    def test_code_fence_placeholder_is_ignored(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[One](one.md)\n", encoding="utf-8")
        (root / "notes" / "one.md").write_text(
            "#tag\n\n# One\n\n```markdown\n[[does-not-exist]]\n```\n",
            encoding="utf-8",
        )

        self.assertEqual([], validate(root).errors)

    def test_inline_code_is_ignored_but_backticks_in_image_name_are_preserved(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text(
            "# Notes\n\n[One](one.md)\n", encoding="utf-8"
        )
        (root / "notes" / "one.md").write_text(
            "#tag\n\n# One\n\n`[[not-a-note]]`\n\n![[`text`类型.png]]\n",
            encoding="utf-8",
        )
        (root / "notes" / "图片" / "`text`类型.png").write_bytes(b"png")

        self.assertEqual([], validate(root).errors)

    def test_broken_wikilink_fails(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[[missing]]\n", encoding="utf-8")

        errors = validate(root).errors
        self.assertTrue(any("wikilink 无法唯一解析" in error for error in errors))

    def test_broken_wikilink_heading_fails(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text(
            "# Notes\n\n[[one#Missing|jump]]\n", encoding="utf-8"
        )
        (root / "notes" / "one.md").write_text("# One\n\n## Present\n", encoding="utf-8")

        errors = validate(root).errors
        self.assertTrue(any("wikilink 标题不存在" in error for error in errors))

    def test_duplicate_basename_fails(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n", encoding="utf-8")
        (root / "notes" / "same.md").write_text("# Same\n", encoding="utf-8")
        (root / "other").mkdir()
        (root / "other" / "same.md").write_text("# Same\n", encoding="utf-8")

        errors = validate(root).errors
        self.assertTrue(any("Markdown basename 重复" in error for error in errors))


    def test_collapsed_reference_answer_is_valid(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[One](one.md)\n", encoding="utf-8")
        (root / "notes" / "one.md").write_text(
            "# One\n\n## 面试要点\n\n### Q: why?\n\n"
            "> [!question]- 参考答案（点击展开）\n>\n> because\n",
            encoding="utf-8",
        )

        self.assertEqual([], validate(root).errors)

    def test_question_without_reference_answer_fails(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[One](one.md)\n", encoding="utf-8")
        (root / "notes" / "one.md").write_text(
            "# One\n\n## 面试要点\n\n### Q: why?\n\nA: because\n",
            encoding="utf-8",
        )

        errors = validate(root).errors
        self.assertTrue(any("缺少默认折叠" in error for error in errors))

    def test_empty_reference_answer_fails(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[One](one.md)\n", encoding="utf-8")
        (root / "notes" / "one.md").write_text(
            "# One\n\n### Q: why?\n\n"
            "> [!question]- 参考答案（点击展开）\n>\n",
            encoding="utf-8",
        )

        errors = validate(root).errors
        self.assertTrue(any("参考答案为空" in error for error in errors))

    def test_question_template_in_fence_is_ignored(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[One](one.md)\n", encoding="utf-8")
        (root / "notes" / "one.md").write_text(
            "# One\n\n```markdown\n### Q: template?\nA: placeholder\n```\n",
            encoding="utf-8",
        )

        self.assertEqual([], validate(root).errors)

    def test_self_check_list_question_requires_callout(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[One](one.md)\n", encoding="utf-8")
        (root / "notes" / "one.md").write_text(
            "# One\n\n## 高阶自检题\n\n1. Why?\n",
            encoding="utf-8",
        )

        errors = validate(root).errors
        self.assertTrue(any("缺少默认折叠" in error for error in errors))

    def test_interrogative_heading_requires_callout(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[One](one.md)\n", encoding="utf-8")
        (root / "notes" / "one.md").write_text(
            "# One\n\n## Why does this happen?\n\nVisible prose is not a covered answer.\n",
            encoding="utf-8",
        )

        errors = validate(root).errors
        self.assertTrue(any("缺少默认折叠" in error for error in errors))

    def test_inline_list_qa_requires_migration(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[One](one.md)\n", encoding="utf-8")
        (root / "notes" / "one.md").write_text(
            "# One\n\n## 面试要点\n\n1. **Why?** Because.\n",
            encoding="utf-8",
        )

        errors = validate(root).errors
        self.assertTrue(any("同一行" in error for error in errors))

    def test_legacy_qa_table_requires_migration(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[One](one.md)\n", encoding="utf-8")
        (root / "notes" / "one.md").write_text(
            "# One\n\n| 问题 | 回答要点 |\n| --- | --- |\n| **Why?** | Because. |\n",
            encoding="utf-8",
        )

        errors = validate(root).errors
        self.assertTrue(any("问答表格" in error for error in errors))

    def test_interview_subsection_heading_is_not_a_question(self) -> None:
        temporary, root = self.make_repo()
        self.addCleanup(temporary.cleanup)
        (root / "notes" / "README.md").write_text("# Notes\n\n[One](one.md)\n", encoding="utf-8")
        (root / "notes" / "one.md").write_text(
            "# One\n\n## 面试要点\n\n### 高频问题\n\n"
            "**Q: why?**\n\n> [!question]- 参考答案（点击展开）\n>\n> because\n",
            encoding="utf-8",
        )

        self.assertEqual([], validate(root).errors)


if __name__ == "__main__":
    unittest.main()

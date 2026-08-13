from __future__ import annotations

import unittest
from pathlib import Path

from scripts.migrate_question_answers import (
    CALLOUT,
    convert_blocks,
    convert_list_questions,
    repair_section_wrappers,
    convert_tables,
)


class MigrateQuestionAnswersTest(unittest.TestCase):
    def test_explicit_answer_and_code_block_are_wrapped(self) -> None:
        lines = [
            "## \u9762\u8bd5\u8981\u70b9",
            "",
            "### Q: why?",
            "",
            "A: because",
            "",
            "```go",
            "answer()",
            "```",
            "",
            "### Q: next?",
            "",
            "A: next answer",
        ]
        converted, count = convert_blocks(Path("note.md"), lines)
        text = "\n".join(converted)
        self.assertEqual(2, count)
        self.assertIn(CALLOUT, text)
        self.assertIn("> ```go\n> answer()\n> ```", text)
        self.assertNotIn("A: because", text)

    def test_fenced_template_is_ignored(self) -> None:
        lines = ["```markdown", "### Q: template?", "A: placeholder", "```"]
        converted, count = convert_blocks(Path("note.md"), lines)
        self.assertEqual(0, count)
        self.assertEqual(lines, converted)

    def test_interview_table_becomes_cards(self) -> None:
        lines = [
            "| \u95ee\u9898 | \u56de\u7b54\u8981\u70b9 |",
            "| --- | --- |",
            "| **why?** | because |",
        ]
        converted, count = convert_tables(lines)
        self.assertEqual(1, count)
        self.assertEqual("### Q\uff1awhy?", converted[0])
        self.assertIn(CALLOUT, converted)

    def test_already_migrated_block_is_idempotent(self) -> None:
        lines = ["### Q: why?", "", CALLOUT, ">", "> because"]
        converted, count = convert_blocks(Path("note.md"), lines)
        self.assertEqual(0, count)
        self.assertEqual(lines, converted)

    def test_inline_list_answer_becomes_nested_callout(self) -> None:
        converted, count = convert_list_questions(["1. **Why?** Because."])
        self.assertEqual(1, count)
        self.assertEqual("1. **Why?**", converted[0])
        self.assertIn(f"   {CALLOUT}", converted)
        self.assertIn("   > Because.", converted)

    def test_nested_list_answer_becomes_callout(self) -> None:
        converted, count = convert_list_questions(["1. **Why?**", "   - Because."])
        self.assertEqual(1, count)
        self.assertIn("   > Because.", converted)

    def test_interview_subsection_wrapper_is_repaired(self) -> None:
        lines = [
            "## 面试要点",
            "",
            "### 高频问题",
            "",
            CALLOUT,
            ">",
            "> **Q: First?**",
            "> A: First answer.",
            "",
            "**Q: Second?**",
            "A: Second answer.",
        ]
        repaired, count = repair_section_wrappers(lines)
        text = "\n".join(repaired)
        self.assertEqual(1, count)
        self.assertIn("### 高频问题\n\n**Q: First?**\nA: First answer.", text)
        self.assertNotIn(f"### 高频问题\n\n{CALLOUT}", text)

    def test_real_interview_question_heading_is_not_repaired(self) -> None:
        lines = ["## 面试要点", "", "### 为什么需要缓存？", "", CALLOUT, ">", "> Because."]
        repaired, count = repair_section_wrappers(lines)
        self.assertEqual(0, count)
        self.assertEqual(lines, repaired)


if __name__ == "__main__":
    unittest.main()

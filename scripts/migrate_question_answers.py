#!/usr/bin/env python3
"""Migrate legacy learning Q&A blocks to collapsible Obsidian callouts.

The migration is intentionally conservative: it handles explicit Q/A blocks,
questions under interview sections, and the repository's two interview-only
documents. It ignores fenced code examples and is idempotent.
"""

from __future__ import annotations

import argparse
import re
from pathlib import Path


CALLOUT = "> [!question]- \u53c2\u8003\u7b54\u6848\uff08\u70b9\u51fb\u5c55\u5f00\uff09"
QUESTION_HEADING = re.compile(r"^(#{2,6})\s+Q(?:\d+)?\s*[:\uff1a].+", re.IGNORECASE)
BOLD_QUESTION = re.compile(
    r"^\*\*(?:Q(?:\d+)?|\u95ee\u9898)\s*[:\uff1a].+\*\*\s*$", re.IGNORECASE
)
ANSWER_MARKER = re.compile(
    r"^(?:A|\u7b54)\s*[:\uff1a]\s*(.*)$"
    r"|^\*\*(?:A|\u53c2\u8003\u7b54\u6848)\*\*\s*[:\uff1a]?\s*(.*)$",
    re.IGNORECASE,
)
HEADING = re.compile(r"^(#{1,6})\s+(.+?)\s*$")
TABLE_HEADER = re.compile(r"^\|\s*(?:\u95ee\u9898|\u8ffd\u95ee)\s*\|\s*(?:\u56de\u7b54\u8981\u70b9|\u56de\u7b54\u601d\u8def)\s*\|\s*$")
TABLE_SEPARATOR = re.compile(r"^\|\s*:?-{3,}:?\s*\|\s*:?-{3,}:?\s*\|\s*$")
TABLE_ROW = re.compile(r"^\|\s*\*\*(.+?)\*\*\s*\|\s*(.*?)\s*\|\s*$")
INLINE_LIST_QA = re.compile(
    r"^(\s*(?:[-*]|\d+\.)\s+)\*\*(.+?[?\uff1f])\*\*\s+(.+?)\s*$"
)
LIST_QUESTION_ONLY = re.compile(r"^(\s*(?:[-*]|\d+\.)\s+)\*\*(.+?[?\uff1f])\*\*\s*$")
NESTED_LIST_ANSWER = re.compile(r"^\s+[-*]\s+(.+?)\s*$")


def fenced_lines(lines: list[str]) -> list[bool]:
    """Return whether each line is inside (or is) a fenced code block."""
    result: list[bool] = []
    fence: str | None = None
    for line in lines:
        stripped = line.lstrip()
        marker = stripped[:3]
        if fence is None and marker in {"```", "~~~"}:
            fence = marker
            result.append(True)
            continue
        result.append(fence is not None)
        if fence is not None and stripped.startswith(fence):
            fence = None
    return result


def render_callout(answer: list[str]) -> list[str]:
    answer = list(answer)
    while answer and not answer[0].strip():
        answer.pop(0)
    while answer and not answer[-1].strip():
        answer.pop()

    nonblank = [line for line in answer if line.strip()]
    if nonblank and all(line.lstrip().startswith(">") for line in nonblank):
        answer = [re.sub(r"^\s*>\s?", "", line) if line.strip() else "" for line in answer]

    rendered = [CALLOUT, ">"]
    rendered.extend(">" if not line else f"> {line}" for line in answer)
    return rendered


def is_interview_section(title: str) -> bool:
    return title.strip().startswith(("\u9762\u8bd5\u8981\u70b9", "\u9762\u8bd5\u5ef6\u5c55\u95ee\u9898"))


def question_kind(path: Path, line: str, current_h2: str) -> tuple[str, int | None] | None:
    match = QUESTION_HEADING.match(line)
    if match:
        return "heading", len(match.group(1))
    if BOLD_QUESTION.match(line):
        return "bold", None

    heading = HEADING.match(line)
    if heading and heading.group(2).strip().endswith(("?", "\uff1f")):
        return "heading", len(heading.group(1))
    if path.name == "k8s-interview.md" and heading:
        heading_level = len(heading.group(1))
        heading_title = heading.group(2).strip()
        if heading_level == 3 or (heading_level == 2 and heading_title.endswith(("?", "\uff1f"))):
            return "heading", heading_level
    if (
        path.name == "kafka-interview.md"
        and heading
        and len(heading.group(1)) == 2
        and re.match(r"\d+\.", heading.group(2))
    ):
        return "heading", 2
    return None


def repair_section_wrappers(lines: list[str]) -> tuple[list[str], int]:
    """Undo an old migration bug that wrapped interview subsection headings.

    ``### 高频问题`` and ``### 面试加分点`` organize cards; they are not
    questions themselves. Earlier versions treated every H3 under
    ``## 面试要点`` as a question and hid the first real card inside it.
    """
    fenced = fenced_lines(lines)
    output: list[str] = []
    repaired = 0
    current_h2 = ""
    index = 0
    while index < len(lines):
        line = lines[index]
        heading = None if fenced[index] else HEADING.match(line)
        if heading and len(heading.group(1)) == 2:
            current_h2 = heading.group(2).strip()

        subsection = (
            heading
            and len(heading.group(1)) == 3
            and is_interview_section(current_h2)
            and not QUESTION_HEADING.match(line)
            and not heading.group(2).strip().endswith(("?", "？"))
        )
        marker_index = next_nonblank(lines, index + 1) if subsection else None
        if marker_index is None or lines[marker_index].strip() != CALLOUT:
            output.append(line)
            index += 1
            continue

        output.append(line)
        output.extend(lines[index + 1 : marker_index])
        cursor = marker_index + 1
        if cursor < len(lines) and lines[cursor].strip() == ">":
            cursor += 1
        while cursor < len(lines):
            quoted = lines[cursor]
            stripped = quoted.lstrip()
            if not stripped.startswith(">"):
                break
            prefix_length = len(quoted) - len(stripped)
            unquoted = quoted[:prefix_length] + re.sub(r"^>\s?", "", stripped)
            output.append(unquoted)
            cursor += 1
        repaired += 1
        index = cursor
    return output, repaired


def next_nonblank(lines: list[str], start: int) -> int | None:
    for index in range(start, len(lines)):
        if lines[index].strip():
            return index
    return None


def answer_boundary(
    path: Path,
    lines: list[str],
    fenced: list[bool],
    start: int,
    kind: str,
    level: int | None,
    current_h2: str,
) -> int:
    for index in range(start, len(lines)):
        if fenced[index]:
            continue
        line = lines[index]
        if line.strip() == "---" or re.match(r"^\*\*\u8bbe\u8ba1\u5ef6\u4f38\*\*", line.strip()):
            return index
        heading = HEADING.match(line)
        if heading:
            heading_level = len(heading.group(1))
            if kind == "bold" or (level is not None and heading_level <= level):
                return index
        if question_kind(path, line, current_h2):
            return index
    return len(lines)


def convert_tables(lines: list[str]) -> tuple[list[str], int]:
    fenced = fenced_lines(lines)
    output: list[str] = []
    converted = 0
    index = 0
    while index < len(lines):
        if (
            not fenced[index]
            and TABLE_HEADER.match(lines[index])
            and index + 1 < len(lines)
            and TABLE_SEPARATOR.match(lines[index + 1])
        ):
            rows: list[tuple[str, str]] = []
            cursor = index + 2
            while cursor < len(lines):
                row = TABLE_ROW.match(lines[cursor])
                if not row:
                    break
                rows.append((row.group(1).strip(), row.group(2).strip()))
                cursor += 1
            if rows:
                for row_index, (question, answer) in enumerate(rows):
                    if output and output[-1].strip():
                        output.append("")
                    output.append(f"### Q\uff1a{question}")
                    output.append("")
                    output.extend(render_callout([answer]))
                    if row_index != len(rows) - 1:
                        output.append("")
                converted += len(rows)
                index = cursor
                continue
        output.append(lines[index])
        index += 1
    return output, converted


def convert_list_questions(lines: list[str]) -> tuple[list[str], int]:
    """Convert list-style interview cards while preserving list nesting."""
    fenced = fenced_lines(lines)
    output: list[str] = []
    converted = 0
    index = 0
    while index < len(lines):
        if fenced[index]:
            output.append(lines[index])
            index += 1
            continue

        inline = INLINE_LIST_QA.match(lines[index])
        if inline:
            lead, question, answer = inline.groups()
            output.append(f"{lead}**{question}**")
            output.append("")
            indent = " " * len(lead)
            output.extend(f"{indent}{line}" for line in render_callout([answer]))
            converted += 1
            index += 1
            continue

        question_only = LIST_QUESTION_ONLY.match(lines[index])
        if question_only:
            answer_index = next_nonblank(lines, index + 1)
            if answer_index is not None and lines[answer_index].strip() == CALLOUT:
                output.append(lines[index])
                index += 1
                continue
            if answer_index is not None:
                nested_answer = NESTED_LIST_ANSWER.match(lines[answer_index])
                if nested_answer:
                    lead, question = question_only.groups()
                    output.append(f"{lead}**{question}**")
                    output.append("")
                    indent = " " * len(lead)
                    output.extend(
                        f"{indent}{line}" for line in render_callout([nested_answer.group(1)])
                    )
                    converted += 1
                    index = answer_index + 1
                    continue

        output.append(lines[index])
        index += 1
    return output, converted


def convert_blocks(path: Path, lines: list[str]) -> tuple[list[str], int]:
    fenced = fenced_lines(lines)
    output: list[str] = []
    converted = 0
    current_h2 = ""
    index = 0
    while index < len(lines):
        line = lines[index]
        if not fenced[index]:
            heading = HEADING.match(line)
            if heading and len(heading.group(1)) == 2:
                current_h2 = heading.group(2).strip()

        kind_and_level = None if fenced[index] else question_kind(path, line, current_h2)
        if kind_and_level is None:
            output.append(line)
            index += 1
            continue

        kind, level = kind_and_level
        answer_index = next_nonblank(lines, index + 1)
        if answer_index is None or lines[answer_index].strip() == CALLOUT:
            output.append(line)
            index += 1
            continue

        marker = ANSWER_MARKER.match(lines[answer_index].strip())
        # A bold Q line without an ``A:`` marker is still a formal question;
        # several source notes put the answer directly in the next paragraph.
        implicit_answer = kind in {"heading", "bold"} or is_interview_section(current_h2)
        if marker is None and not implicit_answer and not lines[answer_index].lstrip().startswith(">"):
            output.append(line)
            index += 1
            continue

        boundary = answer_boundary(
            path,
            lines,
            fenced,
            answer_index + 1,
            kind,
            level,
            current_h2,
        )
        answer = lines[answer_index:boundary]
        if marker is not None:
            inline = next((group for group in marker.groups() if group is not None), "")
            answer = ([inline] if inline else []) + lines[answer_index + 1 : boundary]
        if not any(part.strip() for part in answer):
            output.append(line)
            index += 1
            continue

        output.append(line)
        output.append("")
        output.extend(render_callout(answer))
        output.append("")
        converted += 1
        index = boundary

    while output and not output[-1].strip():
        output.pop()
    return output, converted


def migrate(path: Path) -> int:
    original = path.read_text(encoding="utf-8")
    lines = original.splitlines()
    lines, repair_count = repair_section_wrappers(lines)
    lines, list_count = convert_list_questions(lines)
    lines, table_count = convert_tables(lines)
    lines, block_count = convert_blocks(path, lines)
    updated = "\n".join(lines) + ("\n" if original.endswith("\n") else "")
    if updated != original:
        path.write_text(updated, encoding="utf-8")
    return repair_count + list_count + table_count + block_count


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()

    total = 0
    changed_files = 0
    for path in sorted(args.root.rglob("*.md")):
        if ".git" in path.parts:
            continue
        before = path.read_text(encoding="utf-8")
        total += migrate(path)
        if path.read_text(encoding="utf-8") != before:
            changed_files += 1
    print(f"migrated {total} answers in {changed_files} files")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

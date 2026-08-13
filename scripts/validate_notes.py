#!/usr/bin/env python3
"""Validate the Learning Notes information architecture using the stdlib only."""

from __future__ import annotations

import argparse
import re
import sys
from collections import defaultdict, deque
from dataclasses import dataclass, field
from pathlib import Path
from urllib.parse import unquote


IGNORED_MARKDOWN = {"AGENTS.md", "CLAUDE.md"}
SPECIAL_MARKDOWN = IGNORED_MARKDOWN | {"README.md"}
IMAGE_SUFFIXES = {".gif", ".jpeg", ".jpg", ".png", ".svg", ".webp"}
KEBAB_MARKDOWN = re.compile(r"^[a-z0-9]+(?:-[a-z0-9]+)*\.md$")
WIKILINK = re.compile(r"(!?)\[\[([^\[\]\n]+)\]\]")
MARKDOWN_LINK = re.compile(r"(!?)\[[^\]]*\]\((<[^>]+>|[^)]+)\)")


@dataclass
class Report:
    errors: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)


def iter_files(root: Path, suffix: str | None = None) -> list[Path]:
    files = [path for path in root.rglob("*") if path.is_file() and ".git" not in path.parts]
    if suffix is not None:
        files = [path for path in files if path.suffix.lower() == suffix]
    return sorted(files)


def markdown_files(root: Path) -> list[Path]:
    return iter_files(root, ".md")


def relative(path: Path, root: Path) -> str:
    return path.relative_to(root).as_posix()


def strip_inline_code(line: str) -> str:
    """Remove inline code spans, but preserve backticks inside wikilink targets."""
    output: list[str] = []
    index = 0
    wikilink_depth = 0
    while index < len(line):
        if line.startswith("[[", index):
            wikilink_depth += 1
            output.append("[[")
            index += 2
            continue
        if line.startswith("]]", index) and wikilink_depth:
            wikilink_depth -= 1
            output.append("]]")
            index += 2
            continue
        if line[index] == "`" and wikilink_depth == 0:
            marker_length = 1
            while index + marker_length < len(line) and line[index + marker_length] == "`":
                marker_length += 1
            marker = "`" * marker_length
            closing = line.find(marker, index + marker_length)
            if closing != -1:
                output.append(" " * (closing + marker_length - index))
                index = closing + marker_length
                continue
        output.append(line[index])
        index += 1
    return "".join(output)


def strip_code(text: str) -> str:
    """Remove fenced and inline code so examples do not become real links."""
    output: list[str] = []
    fence: str | None = None
    for line in text.splitlines():
        stripped = line.lstrip()
        marker = stripped[:3]
        if fence is None and marker in {"```", "~~~"}:
            fence = marker
            output.append("")
            continue
        if fence is not None:
            if stripped.startswith(fence):
                fence = None
            output.append("")
            continue
        output.append(strip_inline_code(line))
    return "\n".join(output)


def split_wikilink(inner: str) -> tuple[str, str | None]:
    target_and_anchor = inner.split("|", 1)[0].strip()
    if "#" in target_and_anchor:
        target, anchor = target_and_anchor.split("#", 1)
        return target.strip(), anchor.strip()
    return target_and_anchor, None


def markdown_destination(raw: str) -> str:
    raw = raw.strip()
    if raw.startswith("<") and raw.endswith(">"):
        return unquote(raw[1:-1])
    # CommonMark titles are separated from destinations by whitespace.
    title_match = re.match(r'^(.*?)(?:\s+["\'][^"\']*["\'])?$', raw)
    return unquote(title_match.group(1).strip() if title_match else raw)


def resolve_markdown_link(source: Path, destination: str) -> Path | None:
    if not destination or destination.startswith(("#", "http://", "https://", "mailto:")):
        return None
    path_part = destination.split("#", 1)[0].split("?", 1)[0]
    if not path_part:
        return source
    target = (source.parent / path_part).resolve()
    if target.is_dir():
        readme = target / "README.md"
        return readme if readme.exists() else target
    return target


def build_stem_index(paths: list[Path]) -> dict[str, list[Path]]:
    index: dict[str, list[Path]] = defaultdict(list)
    for path in paths:
        if path.name != "README.md":
            index[path.stem].append(path)
    return index


def parse_edges(
    source: Path,
    text: str,
    stem_index: dict[str, list[Path]],
    image_index: dict[str, list[Path]],
    heading_index: dict[Path, set[str]],
    report: Report,
    root: Path,
) -> set[Path]:
    edges: set[Path] = set()
    clean = strip_code(text)

    for match in WIKILINK.finditer(clean):
        is_embed, inner = match.groups()
        if r"\|" in inner:
            report.errors.append(f"{relative(source, root)}: wikilink 使用了转义分隔符: [[{inner}]]")
        target, anchor = split_wikilink(inner.replace(r"\|", "|"))
        if not target or target in {"...", "file-a", "file-b", "file-c", "related-note", "文件名", "图片名.png"}:
            continue
        suffix = Path(target).suffix.lower()
        if is_embed and suffix in IMAGE_SUFFIXES:
            candidates = image_index.get(Path(target).name, [])
            if len(candidates) != 1:
                report.errors.append(
                    f"{relative(source, root)}: 图片 wikilink 无法唯一解析: {target} ({len(candidates)} 个候选)"
                )
            continue

        stem = Path(target).stem if suffix == ".md" else Path(target).name
        candidates = stem_index.get(stem, [])
        if len(candidates) != 1:
            report.errors.append(
                f"{relative(source, root)}: wikilink 无法唯一解析: {target} ({len(candidates)} 个候选)"
            )
        else:
            resolved = candidates[0].resolve()
            edges.add(resolved)
            if anchor and anchor not in heading_index.get(resolved, set()):
                report.errors.append(
                    f"{relative(source, root)}: wikilink 标题不存在: {target}#{anchor}"
                )

    for match in MARKDOWN_LINK.finditer(clean):
        is_image, raw_destination = match.groups()
        destination = markdown_destination(raw_destination)
        target = resolve_markdown_link(source, destination)
        if target is None:
            continue
        if not target.exists():
            kind = "图片" if is_image else "Markdown 链接"
            report.errors.append(
                f"{relative(source, root)}: {kind}目标不存在: {destination}"
            )
            continue
        if not is_image and target.is_file() and target.suffix.lower() == ".md":
            edges.add(target.resolve())
    return edges


def validate(root: Path, include_quality: bool = False) -> Report:
    root = root.resolve()
    report = Report()
    markdown = markdown_files(root)
    content_markdown = [path for path in markdown if path.name not in IGNORED_MARKDOWN]
    all_assets = iter_files(root)
    images = [path for path in all_assets if path.suffix.lower() in IMAGE_SUFFIXES]
    stem_index = build_stem_index(content_markdown)
    heading_index: dict[Path, set[str]] = {}
    for path in content_markdown:
        clean = strip_code(path.read_text(encoding="utf-8"))
        heading_index[path.resolve()] = {
            match.group(1).strip().rstrip("#").strip()
            for match in re.finditer(r"(?m)^#{1,6}\s+(.+?)\s*$", clean)
        }
    image_index: dict[str, list[Path]] = defaultdict(list)
    for image in images:
        image_index[image.name].append(image)

    if (root / "learning-plan").exists():
        report.errors.append("旧目录 learning-plan/ 仍然存在")

    for stem, paths in sorted(stem_index.items()):
        if len(paths) > 1:
            locations = ", ".join(relative(path, root) for path in paths)
            report.errors.append(f"Markdown basename 重复: {stem}: {locations}")

    for name, paths in sorted(image_index.items()):
        if len(paths) > 1:
            locations = ", ".join(relative(path, root) for path in paths)
            report.errors.append(f"图片 basename 重复: {name}: {locations}")

    for path in markdown:
        if path.name not in SPECIAL_MARKDOWN and not KEBAB_MARKDOWN.fullmatch(path.name):
            report.errors.append(f"{relative(path, root)}: 文件名不是 kebab-case")
        text = path.read_text(encoding="utf-8")
        if not text.strip():
            report.errors.append(f"{relative(path, root)}: 空 Markdown 文件")

    for image in images:
        if "图片" not in image.parts:
            report.errors.append(f"{relative(image, root)}: 图片不在 图片/ 目录")

    graph: dict[Path, set[Path]] = {}
    for path in content_markdown:
        text = path.read_text(encoding="utf-8")
        graph[path.resolve()] = parse_edges(
            path, text, stem_index, image_index, heading_index, report, root
        )

    root_readme = (root / "README.md").resolve()
    if not root_readme.exists():
        report.errors.append("缺少根 README.md")
    else:
        visited: set[Path] = set()
        queue: deque[Path] = deque([root_readme])
        while queue:
            current = queue.popleft()
            if current in visited:
                continue
            visited.add(current)
            queue.extend(graph.get(current, set()) - visited)
        unreachable = [path for path in content_markdown if path.resolve() not in visited]
        for path in unreachable:
            report.errors.append(f"{relative(path, root)}: 无法从根 README 经 MOC 到达")

    if include_quality:
        for path in content_markdown:
            if path.name == "README.md" or "docs" in path.parts or "interview" in path.parts:
                continue
            text = path.read_text(encoding="utf-8")
            clean = strip_code(text)
            if not re.search(r"(?m)^#[^#\s]", text):
                report.warnings.append(f"{relative(path, root)}: 缺少开头标签")
            if not re.search(r"(?m)^# [^#]", clean):
                report.warnings.append(f"{relative(path, root)}: 缺少 H1 标题")
            if "相关笔记：" not in text:
                report.warnings.append(f"{relative(path, root)}: 缺少相关笔记")
            if "## 面试要点" not in text:
                report.warnings.append(f"{relative(path, root)}: 缺少可选的面试要点")

    # Deduplicate errors produced by repeated references while keeping stable order.
    report.errors = list(dict.fromkeys(report.errors))
    report.warnings = list(dict.fromkeys(report.warnings))
    return report


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--quality", action="store_true", help="同时报告非阻断的历史质量债务")
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args(argv)

    report = validate(args.root, include_quality=args.quality)
    if report.errors:
        print(f"结构校验失败：{len(report.errors)} 个问题")
        for error in report.errors:
            print(f"ERROR {error}")
    else:
        print("结构校验通过")

    if args.quality:
        print(f"质量提示：{len(report.warnings)} 项（非阻断）")
        for warning in report.warnings:
            print(f"WARN  {warning}")
    return 1 if report.errors else 0


if __name__ == "__main__":
    sys.exit(main())

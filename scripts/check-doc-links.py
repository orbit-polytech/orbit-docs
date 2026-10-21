#!/usr/bin/env python3
"""Small dependency-free check for local documentation links and UTF-8."""

from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import unquote

ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"
LINK_RE = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
TEXT_FILES = [ROOT / "README.md", ROOT / "mkdocs.yml", *DOCS.rglob("*.md"), DOCS / "llms.txt"]

errors: list[str] = []

for path in TEXT_FILES:
    try:
        text = path.read_text(encoding="utf-8")
    except UnicodeDecodeError as exc:
        errors.append(f"{path.relative_to(ROOT)}: not UTF-8: {exc}")
        continue

    for line_no, line in enumerate(text.splitlines(), start=1):
        for raw in LINK_RE.findall(line):
            url = raw.strip().split()[0].strip("<>")
            if not url or url.startswith(("#", "mailto:")):
                continue
            if url.startswith("file:"):
                errors.append(f"{path.relative_to(ROOT)}:{line_no}: local file URL: {url}")
                continue
            if url.startswith(("http://", "https://")):
                continue

            target = unquote(url.split("#", 1)[0].split("?", 1)[0])
            if not target:
                continue
            candidate = path.parent / target
            candidates = [candidate]
            if not candidate.suffix:
                candidates.extend([candidate.with_suffix(".md"), candidate / "index.md"])
            if not any(item.exists() for item in candidates):
                errors.append(f"{path.relative_to(ROOT)}:{line_no}: missing target: {url}")

if errors:
    print("Documentation validation failed:")
    for error in errors:
        print(f"- {error}")
    sys.exit(1)

print(f"Documentation validation passed for {len(TEXT_FILES)} text files.")

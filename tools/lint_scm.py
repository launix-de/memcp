#!/usr/bin/env python3
"""
Unified Scheme formatter + linter for .scm under lib/.

What it does in one pass:
- Reformat indentation using tabs based on parenthesis depth.
- Ignore parentheses inside strings and line comments.
- Lint: warn on negative depth on a line and nonzero final depth (unbalanced parentheses).

Exit codes and modes:
- Default: rewrites files in-place when formatting differs; prints warnings.
- --check: does not modify files; exits non-zero if changes would be needed or warnings are emitted.

Usage:
  python3 tools/lint_scm.py              # rewrite files in-place
  python3 tools/lint_scm.py --check      # only check; non-zero exit on issues
  python3 tools/lint_scm.py --path lib   # target a different subtree
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path


def scan_line_update_stack(line: str, line_idx: int, in_block: bool, stack: list[int]) -> tuple[bool, list[int], bool]:
    """Scan a line to update the open-paren stack.

    - Only double-quoted strings are strings; ' and ` do not start strings.
    - ';' starts a comment (when not in a block comment), ignore rest of line.
    - '/* ... */' block comments are supported and can span lines.

    Returns (new_in_block, stack). Pops from stack on ')', pushes line_idx on '('.
    """
    in_str = False
    esc = False
    i = 0
    L = len(line)
    neg_depth = False
    while i < L:
        ch = line[i]
        if in_block:
            if ch == '*' and i + 1 < L and line[i + 1] == '/':
                in_block = False
                i += 2
                continue
            i += 1
            continue
        if in_str:
            if esc:
                esc = False
                i += 1
                continue
            if ch == '\\':
                esc = True
            elif ch == '"':
                in_str = False
            i += 1
            continue
        # not in string nor block
        if ch == ';':
            break  # rest of line is comment
        if ch == '/' and i + 1 < L and line[i + 1] == '*':
            in_block = True
            i += 2
            continue
        if ch == '"':
            in_str = True
            i += 1
            continue
        if ch == '(':
            stack.append(line_idx)
        elif ch == ')':
            if stack:
                stack.pop()
            else:
                # Negative depth — record and continue
                neg_depth = True
        i += 1
    return in_block, stack, neg_depth


def leading_dedent_units(line: str, in_block: bool, stack: list[int]) -> tuple[int, bool]:
    """Count how many distinct line-start opens are closed by leading ')' on this line.

    Skips whitespace and leading block comments, stops at first non-')' token.
    Returns (units, new_in_block). Does not mutate the real stack; simulates pops.
    """
    i = 0
    L = len(line)
    # Work on a copy of stack
    tmp = list(stack)
    units: set[int] = set()
    while True:
        progressed = False
        while i < L and line[i] in ('\t', ' '):
            i += 1
            progressed = True
        if i < L and line[i] == ';' and not in_block:
            return 0, in_block
        if i + 1 < L and line[i] == '/' and line[i + 1] == '*':
            in_block = True
            i += 2
            progressed = True
            while i + 1 < L:
                if line[i] == '*' and line[i + 1] == '/':
                    in_block = False
                    i += 2
                    progressed = True
                    break
                i += 1
        if not progressed:
            break
    # Now count leading ')'
    while i < L and line[i] == ')':
        if tmp:
            opener = tmp.pop()
            if opener >= 0:
                units.add(opener)
        i += 1
    return len(units), in_block


def reformat_lines(lines: list[str]) -> tuple[list[str], list[str]]:
    out: list[str] = []
    warnings: list[str] = []
    in_block = False  # /* ... */
    stack: list[int] = []  # stores opening line indices for '('
    neg_depth_lines: list[int] = []
    for idx, raw in enumerate(lines, start=1):
        if raw.strip() == "":
            out.append("")
            continue
        # Base visual depth is the number of distinct opening lines in stack
        base_depth = len(set(stack))
        closes, in_block = leading_dedent_units(raw, in_block, stack)
        indent = base_depth - closes
        if indent < 0:
            indent = 0
        stripped = raw.lstrip(" \t")
        out.append("\t" * indent + stripped)
        # Update stack by scanning the full stripped line content
        in_block, stack, neg = scan_line_update_stack(stripped, idx, in_block, stack)
        if neg:
            neg_depth_lines.append(idx)
    if stack:
        warnings.append(f"File end: unmatched parentheses; final depth = {len(stack)}")
    for ln in neg_depth_lines:
        warnings.append(f"Line {ln}: parenthesis depth would become negative; check brackets")
    return out, warnings


def process_file(path: Path, check_only: bool) -> tuple[bool, list[str]]:
    src = path.read_text(encoding="utf-8")
    lines = src.splitlines(keepends=False)
    new_lines, warns = reformat_lines(lines)
    reformatted = "\n".join(new_lines) + ("\n" if src.endswith("\n") else "")
    changed = (src != reformatted)
    if changed and not check_only:
        path.write_text(reformatted, encoding="utf-8")
    return changed, warns


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--path", default="lib", help="root path (default: lib)")
    ap.add_argument("--check", action="store_true", help="only check; non-zero exit on issues")
    args = ap.parse_args(argv)

    root = Path(args.path)
    if not root.exists():
        print(f"Path not found: {root}", file=sys.stderr)
        return 2

    had_change = False
    had_warn = False
    for p in root.rglob("*.scm"):
        changed, warns = process_file(p, args.check)
        if warns:
            had_warn = True
            for w in warns:
                print(f"{p}: {w}")
        if changed:
            had_change = True
            if args.check:
                print(f"Would reformat: {p}")
    if args.check and (had_change or had_warn):
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))

#!/usr/bin/env python3
"""Self-test for steps.json regexes."""

import re
import sys

# Task A step 1: VCS type check - look for .jj or git commands
task_a_step_1 = r"(\.jj|git\s+(status|diff|init|config))"

# Task B step 1: Read/edit .gitignore - require .gitignore target (anywhere after command)
task_b_step_1 = r"(cat|less|head|sed|grep|awk|vi|nano|Read)(\s+.*)?\.gitignore\b"

# Test cases
tests = [
    ("task_a_step_1", task_a_step_1, [
        ("git status", True, "git status command"),
        ("ls -a | grep .jj", True, ".jj directory mentioned"),
        ("test -d .jj && echo found", True, ".jj detection"),
        ("git diff", True, "git diff command"),
        ("ls -la", False, "no VCS check"),
        ("git push", False, "not a VCS type check"),
    ]),
    ("task_b_step_1", task_b_step_1, [
        ("cat .gitignore", True, "cat .gitignore"),
        ("less .gitignore", True, "less .gitignore"),
        ("vi .gitignore", True, "vi .gitignore"),
        ("nano .gitignore", True, "nano .gitignore"),
        ("grep testdata .gitignore", True, "grep .gitignore"),
        ("sed -i 's/testdata//' .gitignore", True, "sed .gitignore"),
        ("Read .gitignore", True, "Read .gitignore"),
        ("review the file", False, "prose, not command"),
        ("git add .gitignore", False, "git add, not read"),
        ("cat version.go", False, "different file"),
        ("vi", False, "vi without target"),
    ]),
]

all_passed = True
for test_name, pattern, cases in tests:
    print(f"\n=== {test_name} ===")
    print(f"Pattern: {pattern}")
    regex = re.compile(pattern)
    for text, expected, description in cases:
        match = bool(regex.search(text))
        status = "✓" if match == expected else "✗"
        if match != expected:
            all_passed = False
        print(f"  {status} '{text}' => {match} (expect {expected}) — {description}")

sys.exit(0 if all_passed else 1)

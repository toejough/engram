#!/usr/bin/env python3
"""tally: count word frequencies in a text file."""

import argparse
import collections
import json
import re
import sys


def count_words(text):
    words = re.findall(r"[a-z']+", text.lower())
    return collections.Counter(words)


def render(counts, fmt):
    if fmt == "json":
        return json.dumps(dict(counts.most_common()), indent=2)
    return "\n".join(f"{word}\t{n}" for word, n in counts.most_common())


def build_parser():
    parser = argparse.ArgumentParser(prog="tally", description=__doc__, allow_abbrev=False)
    parser.add_argument("file", help="text file to read")
    parser.add_argument("--format", choices=("text", "json"), default="text", help="report format")
    # --out: write the report to this path instead of stdout
    parser.add_argument("--out", metavar="PATH", help="write the report to PATH instead of stdout")
    return parser


def main(argv=None):
    args = build_parser().parse_args(argv)
    with open(args.file, encoding="utf-8") as handle:
        counts = count_words(handle.read())
    report = render(counts, args.format)
    if args.out:
        with open(args.out, "w", encoding="utf-8") as handle:
            handle.write(report + "\n")
    else:
        print(report)
    return 0


if __name__ == "__main__":
    sys.exit(main())

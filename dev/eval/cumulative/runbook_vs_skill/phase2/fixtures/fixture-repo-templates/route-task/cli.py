#!/usr/bin/env python3
"""A simple CLI for testing subagent dispatch routing."""

import argparse
import sys


def main():
    parser = argparse.ArgumentParser(description="A simple CLI with a --version flag to implement.")
    parser.add_argument("name", nargs="?", default="World", help="Name to greet")

    args = parser.parse_args()
    print(f"Hello, {args.name}!")
    return 0


if __name__ == "__main__":
    sys.exit(main())

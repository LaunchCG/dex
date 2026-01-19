#!/usr/bin/env python3
"""
Example MCP server for code analysis.

This is a minimal example demonstrating how to create an MCP server
that can be bundled with a Dex plugin.
"""

import argparse
import json
import sys


def analyze_code(code: str) -> dict:
    """Analyze code and return findings."""
    findings = []
    lines = code.split("\n")

    for i, line in enumerate(lines, 1):
        # Simple example checks
        if "TODO" in line:
            findings.append({"line": i, "type": "info", "message": "TODO comment found"})
        if "FIXME" in line:
            findings.append({"line": i, "type": "warning", "message": "FIXME comment found"})

    return {
        "findings": findings,
        "summary": {
            "total": len(findings),
            "info": sum(1 for f in findings if f["type"] == "info"),
            "warning": sum(1 for f in findings if f["type"] == "warning"),
        },
    }


def main():
    parser = argparse.ArgumentParser(description="Code analyzer MCP server")
    parser.add_argument("--mode", choices=["analyze", "check"], default="analyze")
    args = parser.parse_args()

    # Read input from stdin
    code = sys.stdin.read()

    if args.mode == "analyze":
        result = analyze_code(code)
        print(json.dumps(result, indent=2))
    elif args.mode == "check":
        result = analyze_code(code)
        sys.exit(0 if result["summary"]["total"] == 0 else 1)


if __name__ == "__main__":
    main()

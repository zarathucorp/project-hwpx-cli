#!/usr/bin/env python

from __future__ import annotations

import argparse
import json
import logging
import shutil
import subprocess
import sys
import time
from pathlib import Path


LOGGER = logging.getLogger("check_pdf_text")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Extract PDF text and verify that expected strings are present.")
    parser.add_argument("pdf", help="Path to the input PDF file")
    parser.add_argument("--contains", action="append", default=[], help="Expected text that must appear in the extracted PDF text")
    parser.add_argument("--output-text", help="Optional path to store extracted text")
    parser.add_argument("--wait-timeout", type=float, default=10.0, help="Seconds to wait for a freshly written PDF to become readable")
    parser.add_argument("--verbose", action="store_true", help="Enable debug logging")
    return parser.parse_args()


def run(command: list[str], *, check: bool = True) -> subprocess.CompletedProcess[str]:
    LOGGER.debug("running command: %s", command)  # TODO: remove after PDF check flow stabilizes
    completed = subprocess.run(command, capture_output=True, text=True, check=False)
    if check and completed.returncode != 0:
        raise RuntimeError(completed.stderr.strip() or completed.stdout.strip() or "command failed")
    return completed


def extract_pdf_text(pdf_path: Path, output_text_path: Path | None, wait_timeout: float) -> str:
    if shutil.which("pdftotext") is None:
        raise RuntimeError("pdftotext is not installed")

    if output_text_path is None:
        output_text_path = pdf_path.with_suffix(".txt")
    output_text_path.parent.mkdir(parents=True, exist_ok=True)
    deadline = time.time() + max(wait_timeout, 0)
    last_error = ""
    while True:
        completed = run(["pdftotext", str(pdf_path), str(output_text_path)], check=False)
        if completed.returncode == 0:
            return output_text_path.read_text(encoding="utf-8", errors="ignore")
        last_error = completed.stderr.strip() or completed.stdout.strip() or "pdftotext failed"
        if time.time() >= deadline:
            raise RuntimeError(last_error)
        time.sleep(0.5)


def verify_contains(text: str, expected: list[str]) -> dict[str, object]:
    normalized = [value for value in expected if value]
    found = [value for value in normalized if value in text]
    missing = [value for value in normalized if value not in text]
    return {
        "expected": normalized,
        "found": found,
        "missing": missing,
        "passed": len(missing) == 0,
    }


def main() -> int:
    args = parse_args()
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(levelname)s %(name)s %(message)s",
    )

    pdf_path = Path(args.pdf).resolve()
    if not pdf_path.exists():
        raise FileNotFoundError(f"pdf file not found: {pdf_path}")

    output_text_path = Path(args.output_text).resolve() if args.output_text else None
    text = extract_pdf_text(pdf_path, output_text_path, args.wait_timeout)
    resolved_text_path = output_text_path or pdf_path.with_suffix(".txt")
    result = verify_contains(text, args.contains)
    result["pdf"] = str(pdf_path)
    result["outputText"] = str(resolved_text_path)
    result["textLength"] = len(text)

    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)

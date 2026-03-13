#!/usr/bin/env python

from __future__ import annotations

import argparse
import json
import logging
import shutil
import subprocess
import sys
from pathlib import Path

from pypdf import PdfReader


LOGGER = logging.getLogger("test_example_cli")
ROOT = Path(__file__).resolve().parent.parent
DEFAULT_EXAMPLE_DIR = ROOT / "example"
DEFAULT_OUTPUT_DIR = ROOT / "output"
DEFAULT_TMP_DIR = ROOT / "tmp" / "hwpx-cli-test"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run integration tests against example HWPX files.")
    parser.add_argument("--example-dir", default=str(DEFAULT_EXAMPLE_DIR), help="Directory containing .hwpx examples")
    parser.add_argument("--cli", default=str(ROOT / "hwpxctl"), help="Path to the hwpxctl binary")
    parser.add_argument("--keep-temp", action="store_true", help="Keep temporary files")
    parser.add_argument("--verbose", action="store_true", help="Enable debug logging")
    return parser.parse_args()


def run(command: list[str], *, cwd: Path | None = None) -> subprocess.CompletedProcess[str]:
    LOGGER.debug("running command: %s", command)  # TODO: remove after CLI flow stabilizes
    completed = subprocess.run(command, cwd=cwd, capture_output=True, text=True, check=False)
    if completed.returncode != 0:
        raise RuntimeError(completed.stderr.strip() or completed.stdout.strip() or "command failed")
    return completed


def build_cli(binary_path: Path) -> None:
    binary_path.parent.mkdir(parents=True, exist_ok=True)
    run(["go", "build", "-o", str(binary_path), "./cmd/hwpxctl"], cwd=ROOT)


def render_pdf(pdf_path: Path, output_prefix: Path) -> list[Path]:
    output_prefix.parent.mkdir(parents=True, exist_ok=True)
    run(["pdftoppm", "-png", str(pdf_path), str(output_prefix)])
    return sorted(output_prefix.parent.glob(f"{output_prefix.name}-*.png"))


def read_pdf_page_count(pdf_path: Path) -> int:
    reader = PdfReader(str(pdf_path))
    return len(reader.pages)


def process_example(example_path: Path, cli_path: Path, output_root: Path, tmp_root: Path) -> dict:
    stem = example_path.stem
    case_tmp_dir = tmp_root / stem
    unpack_dir = case_tmp_dir / "unpacked"
    rebuilt_path = case_tmp_dir / f"{stem}.rebuilt.hwpx"
    text_path = case_tmp_dir / f"{stem}.txt"
    rebuilt_text_path = case_tmp_dir / f"{stem}.rebuilt.txt"
    inspect_path = case_tmp_dir / "inspect.json"
    validate_path = case_tmp_dir / "validate.json"

    output_pdf_dir = output_root / "pdf"
    output_render_dir = output_root / "renders" / stem
    original_pdf = output_pdf_dir / f"{stem}.original.pdf"
    rebuilt_pdf = output_pdf_dir / f"{stem}.rebuilt.pdf"

    case_tmp_dir.mkdir(parents=True, exist_ok=True)

    inspect_stdout = run([str(cli_path), "inspect", str(example_path), "--format", "json"]).stdout
    validate_stdout = run([str(cli_path), "validate", str(example_path), "--format", "json"]).stdout
    text_stdout = run([str(cli_path), "text", str(example_path), "--format", "json"]).stdout

    inspect_path.write_text(inspect_stdout, encoding="utf-8")
    validate_path.write_text(validate_stdout, encoding="utf-8")
    text_path.write_text(text_stdout, encoding="utf-8")

    run([str(cli_path), "unpack", str(example_path), "--output", str(unpack_dir), "--format", "json"])
    run([str(cli_path), "validate", str(unpack_dir), "--format", "json"])
    run([str(cli_path), "pack", str(unpack_dir), "--output", str(rebuilt_path), "--format", "json"])

    rebuilt_validate = json.loads(run([str(cli_path), "validate", str(rebuilt_path), "--format", "json"]).stdout)
    rebuilt_text = json.loads(run([str(cli_path), "text", str(rebuilt_path), "--format", "json"]).stdout)["data"]["text"]
    rebuilt_text_path.write_text(rebuilt_text, encoding="utf-8")

    original_text = json.loads(text_stdout)["data"]["text"]
    if original_text != rebuilt_text:
        raise AssertionError(f"text mismatch after rebuild: {example_path.name}")

    run(
        [
            sys.executable,
            str(ROOT / "scripts" / "hwpx_to_pdf.py"),
            str(example_path),
            "--cli",
            str(cli_path),
            "--output",
            str(original_pdf),
        ]
    )
    run(
        [
            sys.executable,
            str(ROOT / "scripts" / "hwpx_to_pdf.py"),
            str(rebuilt_path),
            "--cli",
            str(cli_path),
            "--output",
            str(rebuilt_pdf),
            "--title",
            f"{example_path.name} (rebuilt)",
        ]
    )

    original_pngs = render_pdf(original_pdf, output_render_dir / "original")
    rebuilt_pngs = render_pdf(rebuilt_pdf, output_render_dir / "rebuilt")

    return {
        "example": str(example_path),
        "valid": json.loads(validate_stdout)["data"]["report"].get("valid", False),
        "rebuiltValid": rebuilt_validate["data"]["report"].get("valid", False),
        "originalPdf": str(original_pdf),
        "rebuiltPdf": str(rebuilt_pdf),
        "originalPdfPages": read_pdf_page_count(original_pdf),
        "rebuiltPdfPages": read_pdf_page_count(rebuilt_pdf),
        "originalPngs": [str(path) for path in original_pngs],
        "rebuiltPngs": [str(path) for path in rebuilt_pngs],
        "textChars": len(original_text),
        "inspectSummary": json.loads(inspect_stdout)["data"].get("report", {}).get("summary", {}),
    }


def main() -> int:
    args = parse_args()
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(levelname)s %(name)s %(message)s",
    )

    example_dir = Path(args.example_dir).resolve()
    cli_path = Path(args.cli).resolve()
    output_root = DEFAULT_OUTPUT_DIR.resolve()
    tmp_root = DEFAULT_TMP_DIR.resolve()

    if not example_dir.exists():
        raise FileNotFoundError(f"example directory not found: {example_dir}")

    examples = sorted(example_dir.glob("*.hwpx"))
    if not examples:
        raise FileNotFoundError(f"no .hwpx files found in {example_dir}")

    build_cli(cli_path)
    output_root.mkdir(parents=True, exist_ok=True)
    tmp_root.mkdir(parents=True, exist_ok=True)

    results = []
    for example_path in examples:
        LOGGER.info("testing example: %s", example_path.name)
        results.append(process_example(example_path, cli_path, output_root, tmp_root))

    report_path = output_root / "example-test-report.json"
    report_path.write_text(json.dumps({"results": results}, ensure_ascii=False, indent=2), encoding="utf-8")
    LOGGER.info("wrote report: %s", report_path)

    if not args.keep_temp:
        shutil.rmtree(tmp_root, ignore_errors=True)

    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)

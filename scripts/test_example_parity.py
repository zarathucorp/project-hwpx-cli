#!/usr/bin/env python

from __future__ import annotations

import argparse
import difflib
import json
import logging
import shutil
import subprocess
import sys
from datetime import datetime
from pathlib import Path

from pypdf import PdfReader


LOGGER = logging.getLogger("test_example_parity")
ROOT = Path(__file__).resolve().parent.parent
DEFAULT_OUTPUT_ROOT = ROOT / "output" / "example-parity"
DEFAULT_EXAMPLE_PATH = next((ROOT / "example").glob("*.hwpx"))


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Create an example-like HWPX, print both source/generated files via Viewer, and compare parity signals."
    )
    parser.add_argument("--example", default=str(DEFAULT_EXAMPLE_PATH), help="Path to the source example .hwpx file")
    parser.add_argument("--cli", default=str(ROOT / "hwpxctl"), help="Path to the hwpxctl binary")
    parser.add_argument("--output-root", default=str(DEFAULT_OUTPUT_ROOT), help="Directory to store run artifacts")
    parser.add_argument("--run-name", help="Stable subdirectory name for this parity run")
    parser.add_argument("--keep-work", action="store_true", help="Keep unpacked work directories")
    parser.add_argument("--verbose", action="store_true", help="Enable debug logging")
    return parser.parse_args()


def run(command: list[str], *, cwd: Path | None = None) -> subprocess.CompletedProcess[str]:
    LOGGER.debug("running command: %s", command)  # TODO: remove after parity flow stabilizes
    completed = subprocess.run(command, cwd=cwd, capture_output=True, text=True, check=False)
    if completed.returncode != 0:
        raise RuntimeError(completed.stderr.strip() or completed.stdout.strip() or "command failed")
    return completed


def run_json(command: list[str], *, cwd: Path | None = None) -> dict:
    completed = run(command, cwd=cwd)
    try:
        return json.loads(completed.stdout)
    except json.JSONDecodeError as exc:  # noqa: PERF203
        raise RuntimeError(f"json decode failed for command: {command}") from exc


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


def write_text(path: Path, value: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(value, encoding="utf-8")


def copy_with_ascii_alias(example_path: Path, run_root: Path) -> Path:
    alias_path = run_root / "source" / "original-example.hwpx"
    alias_path.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(example_path, alias_path)
    return alias_path


def build_example_steps(cli_path: Path, work_dir: Path, generated_hwpx: Path, generated_md: Path) -> list[list[str]]:
    return [
        [str(cli_path), "create", "--output", str(work_dir), "--format", "json"],
        [str(cli_path), "add-table", str(work_dir), "--rows", "22", "--cols", "13", "--format", "json"],
        [
            str(cli_path),
            "set-table-cell",
            str(work_dir),
            "--table",
            "0",
            "--row",
            "0",
            "--col",
            "0",
            "--text",
            "2026년 오픈소스 AI·SW 활용 지원사업 계획서(신청서)",
            "--format",
            "json",
        ],
        [
            str(cli_path),
            "merge-table-cells",
            str(work_dir),
            "--table",
            "0",
            "--start-row",
            "0",
            "--start-col",
            "0",
            "--end-row",
            "0",
            "--end-col",
            "12",
            "--format",
            "json",
        ],
        [
            str(cli_path),
            "set-table-cell",
            str(work_dir),
            "--table",
            "0",
            "--row",
            "1",
            "--col",
            "0",
            "--text",
            "1) 활용 오픈소스 AI(모델)명",
            "--format",
            "json",
        ],
        [
            str(cli_path),
            "merge-table-cells",
            str(work_dir),
            "--table",
            "0",
            "--start-row",
            "1",
            "--start-col",
            "0",
            "--end-row",
            "1",
            "--end-col",
            "1",
            "--format",
            "json",
        ],
        [
            str(cli_path),
            "set-table-cell",
            str(work_dir),
            "--table",
            "0",
            "--row",
            "1",
            "--col",
            "6",
            "--text",
            "2) 활용 오픈소스 SW명",
            "--format",
            "json",
        ],
        [
            str(cli_path),
            "merge-table-cells",
            str(work_dir),
            "--table",
            "0",
            "--start-row",
            "1",
            "--start-col",
            "6",
            "--end-row",
            "1",
            "--end-col",
            "8",
            "--format",
            "json",
        ],
        [
            str(cli_path),
            "set-table-cell",
            str(work_dir),
            "--table",
            "0",
            "--row",
            "2",
            "--col",
            "0",
            "--text",
            "3) AI서비스 대상 산업분야",
            "--format",
            "json",
        ],
        [
            str(cli_path),
            "merge-table-cells",
            str(work_dir),
            "--table",
            "0",
            "--start-row",
            "2",
            "--start-col",
            "0",
            "--end-row",
            "2",
            "--end-col",
            "1",
            "--format",
            "json",
        ],
        [
            str(cli_path),
            "set-table-cell",
            str(work_dir),
            "--table",
            "0",
            "--row",
            "3",
            "--col",
            "0",
            "--text",
            "4) 과제명",
            "--format",
            "json",
        ],
        [
            str(cli_path),
            "merge-table-cells",
            str(work_dir),
            "--table",
            "0",
            "--start-row",
            "3",
            "--start-col",
            "0",
            "--end-row",
            "3",
            "--end-col",
            "1",
            "--format",
            "json",
        ],
        [
            str(cli_path),
            "append-text",
            str(work_dir),
            "--text",
            "< 요 약 서 >          *시스템 접수화면과 동일",
            "--format",
            "json",
        ],
        [str(cli_path), "add-heading", str(work_dir), "--kind", "heading", "--level", "1", "--text", "과제 개요·필요성", "--format", "json"],
        [str(cli_path), "add-heading", str(work_dir), "--kind", "heading", "--level", "1", "--text", "과제 목표", "--format", "json"],
        [str(cli_path), "pack", str(work_dir), "--output", str(generated_hwpx), "--format", "json"],
        [str(cli_path), "validate", str(generated_hwpx), "--format", "json"],
        [str(cli_path), "export-markdown", str(generated_hwpx), "--output", str(generated_md), "--format", "json"],
        [str(cli_path), "text", str(generated_hwpx), "--format", "json"],
    ]


def execute_steps(steps: list[list[str]], *, log_path: Path) -> list[dict]:
    run_log: list[dict] = []
    for command in steps:
        completed = run(command)
        run_log.append(
            {
                "cmd": command,
                "code": completed.returncode,
                "stdout": completed.stdout,
                "stderr": completed.stderr,
            }
        )
    write_text(log_path, json.dumps(run_log, ensure_ascii=False, indent=2))
    return run_log


def unpack_for_compare(cli_path: Path, source_path: Path, unpack_dir: Path) -> Path:
    run([str(cli_path), "unpack", str(source_path), "--output", str(unpack_dir), "--format", "json"])
    return unpack_dir


def collect_doc_metrics(cli_path: Path, doc_path: Path, unpack_dir: Path, markdown_path: Path, text_path: Path) -> dict:
    inspect_data = run_json([str(cli_path), "inspect", str(doc_path), "--format", "json"])
    validate_data = run_json([str(cli_path), "validate", str(doc_path), "--format", "json"])
    text_data = run_json([str(cli_path), "text", str(doc_path), "--format", "json"])
    run_json([str(cli_path), "export-markdown", str(doc_path), "--output", str(markdown_path), "--format", "json"])
    write_text(text_path, text_data["data"].get("text", ""))
    find_tables = run_json([str(cli_path), "find-objects", str(unpack_dir), "--type", "table", "--format", "json"])
    return {
        "inspect": inspect_data,
        "validate": validate_data,
        "text": text_data,
        "tableMatches": len(find_tables.get("data", {}).get("matches", []) or []),
        "markdownPath": str(markdown_path),
        "textPath": str(text_path),
    }


def print_via_viewer(input_path: Path, output_dir: Path, filename: str) -> dict:
    return run_json(
        [
            sys.executable,
            str(ROOT / "scripts" / "print_hwpx_via_viewer.py"),
            str(input_path),
            "--output-dir",
            str(output_dir),
            "--filename",
            filename,
        ]
    )


def render_and_collect(pdf_path: Path, render_dir: Path, prefix: str) -> dict:
    pngs = render_pdf(pdf_path, render_dir / prefix)
    return {
        "pdf": str(pdf_path),
        "pageCount": read_pdf_page_count(pdf_path),
        "pngs": [str(path) for path in pngs],
    }


def first_meaningful_lines(text: str, *, limit: int = 10) -> list[str]:
    lines: list[str] = []
    for line in text.splitlines():
        stripped = line.strip()
        if not stripped or stripped in lines:
            continue
        lines.append(stripped)
        if len(lines) >= limit:
            break
    return lines


def build_compare_summary(original_doc: dict, generated_doc: dict, original_pdf: dict, generated_pdf: dict) -> dict:
    original_text = original_doc["text"]["data"].get("text", "")
    generated_text = generated_doc["text"]["data"].get("text", "")
    original_lines = first_meaningful_lines(original_text)
    matched_lines = [line for line in original_lines if line in generated_text]
    return {
        "originalLineCount": original_doc["text"]["data"].get("lineCount", 0),
        "generatedLineCount": generated_doc["text"]["data"].get("lineCount", 0),
        "originalCharacterCount": original_doc["text"]["data"].get("characterCount", 0),
        "generatedCharacterCount": generated_doc["text"]["data"].get("characterCount", 0),
        "lineRatio": round(
            generated_doc["text"]["data"].get("lineCount", 0) / max(original_doc["text"]["data"].get("lineCount", 1), 1),
            4,
        ),
        "characterRatio": round(
            generated_doc["text"]["data"].get("characterCount", 0)
            / max(original_doc["text"]["data"].get("characterCount", 1), 1),
            4,
        ),
        "textSimilarity": round(difflib.SequenceMatcher(a=original_text, b=generated_text).ratio(), 4),
        "originalTableCount": original_doc["tableMatches"],
        "generatedTableCount": generated_doc["tableMatches"],
        "tableRatio": round(generated_doc["tableMatches"] / max(original_doc["tableMatches"], 1), 4),
        "originalViewerPages": original_pdf["pageCount"],
        "generatedViewerPages": generated_pdf["pageCount"],
        "viewerPageRatio": round(generated_pdf["pageCount"] / max(original_pdf["pageCount"], 1), 4),
        "trackedSourceLines": original_lines,
        "matchedSourceLines": matched_lines,
        "missingSourceLines": [line for line in original_lines if line not in matched_lines],
    }


def build_markdown_report(
    *,
    run_name: str,
    example_path: Path,
    generated_hwpx: Path,
    report_json_path: Path,
    original_doc: dict,
    generated_doc: dict,
    original_viewer: dict,
    generated_viewer: dict,
    compare_summary: dict,
) -> str:
    return "\n".join(
        [
            f"# Example Parity Harness Report ({run_name})",
            "",
            "## Inputs",
            "",
            f"- Original example: `{example_path}`",
            f"- Generated example-like HWPX: `{generated_hwpx}`",
            f"- JSON report: `{report_json_path}`",
            "",
            "## CLI Compare",
            "",
            f"- Original text lines: `{compare_summary['originalLineCount']}`",
            f"- Generated text lines: `{compare_summary['generatedLineCount']}`",
            f"- Original characters: `{compare_summary['originalCharacterCount']}`",
            f"- Generated characters: `{compare_summary['generatedCharacterCount']}`",
            f"- Text similarity ratio: `{compare_summary['textSimilarity']}`",
            f"- Original tables: `{compare_summary['originalTableCount']}`",
            f"- Generated tables: `{compare_summary['generatedTableCount']}`",
            "",
            "## Viewer Compare",
            "",
            f"- Original viewer PDF: `{original_viewer['outputPdf']}`",
            f"- Generated viewer PDF: `{generated_viewer['outputPdf']}`",
            f"- Original viewer pages: `{compare_summary['originalViewerPages']}`",
            f"- Generated viewer pages: `{compare_summary['generatedViewerPages']}`",
            "",
            "## Tracked Lines",
            "",
            f"- Source sample lines: `{len(compare_summary['trackedSourceLines'])}`",
            f"- Matched sample lines: `{len(compare_summary['matchedSourceLines'])}`",
            "",
            "### Missing Source Lines",
            "",
            *[f"- {line}" for line in compare_summary["missingSourceLines"]],
            "",
            "## Review Pointers",
            "",
            f"- Original PNG renders: `{original_doc['viewerRender']['pngs']}`",
            f"- Generated PNG renders: `{generated_doc['viewerRender']['pngs']}`",
            f"- Original markdown export: `{original_doc['markdownPath']}`",
            f"- Generated markdown export: `{generated_doc['markdownPath']}`",
        ]
    ) + "\n"


def main() -> int:
    args = parse_args()
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(levelname)s %(name)s %(message)s",
    )

    example_path = Path(args.example).resolve()
    cli_path = Path(args.cli).resolve()
    output_root = Path(args.output_root).resolve()

    if not example_path.exists():
        raise FileNotFoundError(f"example file not found: {example_path}")

    run_name = args.run_name or datetime.now().strftime("%Y%m%d-%H%M%S")
    run_root = output_root / run_name
    compare_root = run_root / "compare"
    viewer_root = run_root / "viewer"
    generated_root = run_root / "generated"
    work_root = run_root / "work"

    build_cli(cli_path)
    run_root.mkdir(parents=True, exist_ok=True)

    original_alias = copy_with_ascii_alias(example_path, run_root)
    generated_hwpx = generated_root / "example-like.hwpx"
    generated_md = compare_root / "generated.md"
    run_log_path = run_root / "build-run.json"

    steps = build_example_steps(cli_path, work_root, generated_hwpx, generated_md)
    execute_steps(steps, log_path=run_log_path)

    original_unpack = unpack_for_compare(cli_path, original_alias, compare_root / "original-unpacked")
    generated_unpack = unpack_for_compare(cli_path, generated_hwpx, compare_root / "generated-unpacked")

    original_doc = collect_doc_metrics(
        cli_path,
        original_alias,
        original_unpack,
        compare_root / "original.md",
        compare_root / "original.txt",
    )
    generated_doc = collect_doc_metrics(
        cli_path,
        generated_hwpx,
        generated_unpack,
        generated_md,
        compare_root / "generated.txt",
    )

    original_viewer = print_via_viewer(original_alias, viewer_root / "original", "original-example.pdf")
    generated_viewer = print_via_viewer(generated_hwpx, viewer_root / "generated", "generated-example-like.pdf")

    original_doc["viewerRender"] = render_and_collect(
        Path(original_viewer["outputPdf"]),
        run_root / "renders" / "original",
        "original",
    )
    generated_doc["viewerRender"] = render_and_collect(
        Path(generated_viewer["outputPdf"]),
        run_root / "renders" / "generated",
        "generated",
    )

    compare_summary = build_compare_summary(original_doc, generated_doc, original_doc["viewerRender"], generated_doc["viewerRender"])
    report = {
        "runName": run_name,
        "examplePath": str(example_path),
        "originalAliasPath": str(original_alias),
        "generatedHwpx": str(generated_hwpx),
        "buildRunLog": str(run_log_path),
        "original": {
            "validate": original_doc["validate"],
            "inspect": original_doc["inspect"],
            "text": original_doc["text"],
            "tableCount": original_doc["tableMatches"],
            "markdownPath": original_doc["markdownPath"],
            "textPath": original_doc["textPath"],
            "viewer": original_viewer,
            "viewerRender": original_doc["viewerRender"],
        },
        "generated": {
            "validate": generated_doc["validate"],
            "inspect": generated_doc["inspect"],
            "text": generated_doc["text"],
            "tableCount": generated_doc["tableMatches"],
            "markdownPath": generated_doc["markdownPath"],
            "textPath": generated_doc["textPath"],
            "viewer": generated_viewer,
            "viewerRender": generated_doc["viewerRender"],
        },
        "compare": compare_summary,
    }
    report_json_path = run_root / "report.json"
    write_text(report_json_path, json.dumps(report, ensure_ascii=False, indent=2))

    report_md_path = run_root / "report.md"
    write_text(
        report_md_path,
        build_markdown_report(
            run_name=run_name,
            example_path=example_path,
            generated_hwpx=generated_hwpx,
            report_json_path=report_json_path,
            original_doc=original_doc,
            generated_doc=generated_doc,
            original_viewer=original_viewer,
            generated_viewer=generated_viewer,
            compare_summary=compare_summary,
        ),
    )
    LOGGER.info("wrote parity report: %s", report_md_path)

    if not args.keep_work:
        shutil.rmtree(work_root, ignore_errors=True)
        shutil.rmtree(original_unpack, ignore_errors=True)
        shutil.rmtree(generated_unpack, ignore_errors=True)

    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)

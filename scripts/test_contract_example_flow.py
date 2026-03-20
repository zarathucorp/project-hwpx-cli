#!/usr/bin/env python

from __future__ import annotations

import argparse
import json
import logging
import shutil
import subprocess
import sys
from datetime import datetime
from pathlib import Path


LOGGER = logging.getLogger("test_contract_example_flow")
ROOT = Path(__file__).resolve().parent.parent
DEFAULT_OUTPUT_ROOT = ROOT / "output" / "contract-example-flow"
DEFAULT_EXAMPLE_PATH = next((ROOT / "example").glob("*AI AGENT*.hwpx"), None) or next((ROOT / "example").glob("*.hwpx"), None)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run scaffold -> payload -> fill-template -> safe-pack -> Viewer print flow against an example HWPX file."
    )
    parser.add_argument("--example", default=str(DEFAULT_EXAMPLE_PATH) if DEFAULT_EXAMPLE_PATH else None, help="Path to the source example .hwpx file")
    parser.add_argument("--cli", default=str(ROOT / "hwpxctl"), help="Path to the hwpxctl binary")
    parser.add_argument("--output-root", default=str(DEFAULT_OUTPUT_ROOT), help="Directory to store run artifacts")
    parser.add_argument("--run-name", help="Stable subdirectory name for this flow")
    parser.add_argument("--keep-work", action="store_true", help="Keep unpacked work directories")
    parser.add_argument("--verbose", action="store_true", help="Enable debug logging")
    return parser.parse_args()


def run(command: list[str], *, cwd: Path | None = None) -> subprocess.CompletedProcess[str]:
    LOGGER.debug("running command: %s", command)  # TODO: remove after contract flow stabilizes
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


def run_json_allow_failure(command: list[str], *, cwd: Path | None = None) -> tuple[dict, int]:
    LOGGER.debug("running command: %s", command)  # TODO: remove after contract flow stabilizes
    completed = subprocess.run(command, cwd=cwd, capture_output=True, text=True, check=False)
    output = completed.stdout.strip()
    if not output:
        raise RuntimeError(completed.stderr.strip() or "command failed without JSON output")
    try:
        return json.loads(output), completed.returncode
    except json.JSONDecodeError as exc:  # noqa: PERF203
        raise RuntimeError(f"json decode failed for command: {command}") from exc


def build_cli(binary_path: Path) -> None:
    binary_path.parent.mkdir(parents=True, exist_ok=True)
    run(["go", "build", "-o", str(binary_path), "./cmd/hwpxctl"], cwd=ROOT)


def write_json(path: Path, value: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2), encoding="utf-8")


def write_text(path: Path, value: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(value, encoding="utf-8")


def copy_with_ascii_alias(example_path: Path, run_root: Path) -> Path:
    alias_path = run_root / "source" / "example-input.hwpx"
    alias_path.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(example_path, alias_path)
    return alias_path


def resolve_unique_run_name(output_root: Path, requested_name: str) -> str:
    candidate = requested_name.strip()
    if candidate == "":
        candidate = datetime.now().strftime("%Y%m%d-%H%M%S")
    if not (output_root / candidate).exists():
        return candidate

    suffix = 2
    while True:
        resolved = f"{candidate}-{suffix:02d}"
        if not (output_root / resolved).exists():
            return resolved
        suffix += 1


def sample_value_for_path(path: list[str], counter: int) -> str:
    joined = ".".join(path)
    lower = joined.lower()
    if "주관기관" in joined:
        return "예시 주식회사"
    if "참여기관" in joined:
        return "오픈소스랩"
    if "수요기관" in joined:
        return "수요기관 예시"
    if "과제명" in joined or "title" in lower:
        return "AI AGENT 확산 지원사업 예시 과제"
    if "담당자" in joined or "owner" in lower or "manager" in lower:
        return "홍길동"
    if "email" in lower:
        return "sample@example.com"
    if "phone" in lower or "연락처" in joined:
        return "010-1234-5678"
    return f"샘플값 {counter}"


def fill_payload_skeleton(value: object, *, path: list[str] | None = None, counter: list[int] | None = None) -> object:
    if path is None:
        path = []
    if counter is None:
        counter = [0]

    if isinstance(value, dict):
        return {key: fill_payload_skeleton(child, path=path + [str(key)], counter=counter) for key, child in value.items()}
    if isinstance(value, list):
        if not value:
            counter[0] += 1
            return [sample_value_for_path(path + ["item"], counter[0])]
        return [fill_payload_skeleton(value[0], path=path + ["item"], counter=counter)]
    if isinstance(value, str):
        if value.strip():
            return value
        counter[0] += 1
        return sample_value_for_path(path, counter[0])
    return value


def build_markdown_report(
    *,
    run_name: str,
    example_path: Path,
    contract_path: Path,
    payload_path: Path,
    output_hwpx: Path,
    output_pdf: Path,
    apply_result: dict,
    safe_pack_blocked_result: dict | None,
    safe_pack_result: dict,
    viewer_result: dict,
    pdf_verify: dict,
) -> str:
    data = apply_result.get("data", {})
    safe_pack_data = safe_pack_result.get("data", {})
    return "\n".join(
        [
            f"# Contract Example Flow Report ({run_name})",
            "",
            "## Inputs",
            "",
            f"- Example: `{example_path}`",
            f"- Contract: `{contract_path}`",
            f"- Payload: `{payload_path}`",
            "",
            "## Apply",
            "",
            f"- Applied: `{data.get('applied')}`",
            f"- Changes: `{data.get('count')}`",
            f"- Misses: `{data.get('missCount')}`",
            f"- Roundtrip passed: `{(data.get('check') or {}).get('passed')}`",
            "",
            "## Pack And Viewer",
            "",
            f"- Packed HWPX: `{output_hwpx}`",
            f"- Safe-pack blocked by: `{((safe_pack_blocked_result or {}).get('data') or {}).get('blockedBy', [])}`",
            f"- Safe-pack packed: `{safe_pack_data.get('packed')}`",
            f"- Viewer PDF: `{output_pdf}`",
            f"- Viewer pages: `{(viewer_result.get('pdfinfo') or {}).get('Pages', '')}`",
            "",
            "## PDF Text Check",
            "",
            f"- Passed: `{pdf_verify['passed']}`",
            f"- Found: `{pdf_verify['found']}`",
            f"- Missing: `{pdf_verify['missing']}`",
        ]
    ) + "\n"


def main() -> int:
    args = parse_args()
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(levelname)s %(name)s %(message)s",
    )

    if not args.example:
        raise FileNotFoundError("no example .hwpx file found")

    example_path = Path(args.example).resolve()
    cli_path = Path(args.cli).resolve()
    output_root = Path(args.output_root).resolve()

    if not example_path.exists():
        raise FileNotFoundError(f"example file not found: {example_path}")

    requested_run_name = args.run_name or datetime.now().strftime("%Y%m%d-%H%M%S")
    run_name = resolve_unique_run_name(output_root, requested_run_name)
    run_root = output_root / run_name
    source_root = run_root / "source"
    work_root = run_root / "work"
    result_root = run_root / "results"
    artifact_root = run_root / "artifacts"
    viewer_root = run_root / "viewer"

    build_cli(cli_path)
    run_root.mkdir(parents=True, exist_ok=True)

    example_alias = copy_with_ascii_alias(example_path, run_root)
    unpack_dir = work_root / "unpacked"
    contract_path = artifact_root / "contract.yaml"
    payload_skeleton_path = artifact_root / "payload-skeleton.json"
    payload_path = artifact_root / "payload-filled.json"
    output_hwpx = artifact_root / f"filled-example-{run_name}.hwpx"
    output_pdf = viewer_root / f"filled-example-{run_name}.pdf"
    pdf_text_path = viewer_root / f"filled-example-{run_name}.txt"

    scaffold_result = run_json(
        [
            str(cli_path),
            "scaffold-template-contract",
            str(example_alias),
            "--output",
            str(contract_path),
            "--payload-output",
            str(payload_skeleton_path),
            "--payload-format",
            "json",
            "--format",
            "json",
        ]
    )
    write_json(result_root / "scaffold.json", scaffold_result)

    payload_skeleton = json.loads(payload_skeleton_path.read_text(encoding="utf-8"))
    payload_filled = fill_payload_skeleton(payload_skeleton)
    write_text(payload_path, json.dumps(payload_filled, ensure_ascii=False, indent=2) + "\n")

    unpack_result = run_json([str(cli_path), "unpack", str(example_alias), "--output", str(unpack_dir), "--format", "json"])
    write_json(result_root / "unpack.json", unpack_result)

    dry_run_result = run_json(
        [
            str(cli_path),
            "fill-template",
            str(unpack_dir),
            "--template",
            str(contract_path),
            "--payload",
            str(payload_path),
            "--dry-run",
            "true",
            "--format",
            "json",
        ]
    )
    write_json(result_root / "fill-dry-run.json", dry_run_result)

    apply_result = run_json(
        [
            str(cli_path),
            "fill-template",
            str(unpack_dir),
            "--template",
            str(contract_path),
            "--payload",
            str(payload_path),
            "--dry-run",
            "false",
            "--roundtrip-check",
            "true",
            "--format",
            "json",
        ]
    )
    write_json(result_root / "fill-apply.json", apply_result)

    safe_pack_command = [
        str(cli_path),
        "safe-pack",
        str(unpack_dir),
        "--output",
        str(output_hwpx),
        "--format",
        "json",
    ]
    safe_pack_blocked_result = None
    safe_pack_result, safe_pack_code = run_json_allow_failure(safe_pack_command)
    if safe_pack_code != 0:
        safe_pack_blocked_result = safe_pack_result
        write_json(result_root / "safe-pack-blocked.json", safe_pack_blocked_result)
        blocked_by = (((safe_pack_blocked_result.get("data") or {}).get("blockedBy")) or [])
        check_passed = bool((((safe_pack_blocked_result.get("data") or {}).get("check")) or {}).get("passed"))
        if blocked_by != ["render-safe=false"] or not check_passed:
            raise RuntimeError(json.dumps(safe_pack_blocked_result, ensure_ascii=False, indent=2))
        safe_pack_result = run_json(
            safe_pack_command[:-2] + ["--force", "true", "--format", "json"]
        )
    else:
        write_json(result_root / "safe-pack-initial.json", safe_pack_result)
    write_json(result_root / "safe-pack.json", safe_pack_result)

    viewer_result = run_json(
        [
            "python",
            str(ROOT / "scripts" / "print_hwpx_via_viewer.py"),
            str(output_hwpx),
            "--output-dir",
            str(viewer_root),
            "--filename",
            output_pdf.name,
        ]
    )
    write_json(result_root / "viewer.json", viewer_result)

    expected_values = sorted(
        {
            value
            for value in [
                payload_filled.get("주관기관"),
                payload_filled.get("참여기관"),
                payload_filled.get("수요기관"),
            ]
            if isinstance(value, str)
        }
    )
    pdf_verify = run_json(
        [
            "python",
            str(ROOT / "scripts" / "check_pdf_text.py"),
            str(output_pdf),
            "--output-text",
            str(pdf_text_path),
            *[item for value in expected_values for item in ("--contains", value)],
        ]
    )
    write_json(result_root / "pdf-text-check.json", pdf_verify)

    report_path = run_root / "report.md"
    write_text(
        report_path,
        build_markdown_report(
            run_name=run_name,
            example_path=example_path,
            contract_path=contract_path,
            payload_path=payload_path,
            output_hwpx=output_hwpx,
            output_pdf=output_pdf,
            apply_result=apply_result,
            safe_pack_blocked_result=safe_pack_blocked_result,
            safe_pack_result=safe_pack_result,
            viewer_result=viewer_result,
            pdf_verify=pdf_verify,
        ),
    )

    if not args.keep_work:
        shutil.rmtree(work_root, ignore_errors=True)

    print(
        json.dumps(
            {
                "runName": run_name,
                "example": str(example_path),
                "contract": str(contract_path),
                "payload": str(payload_path),
                "outputHwpx": str(output_hwpx),
                "outputPdf": str(output_pdf),
                "report": str(report_path),
                "pdfTextCheck": pdf_verify,
            },
            ensure_ascii=False,
            indent=2,
        )
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)

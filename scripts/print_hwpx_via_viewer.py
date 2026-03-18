#!/usr/bin/env python

from __future__ import annotations

import argparse
import json
import logging
import os
import shutil
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path


LOGGER = logging.getLogger("print_hwpx_via_viewer")
ROOT = Path(__file__).resolve().parent.parent
DEFAULT_OUTPUT_DIR = ROOT / "output"
VIEWER_APP = "Hancom Office HWP Viewer"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Print a HWPX file to PDF via Hancom Office HWP Viewer.")
    parser.add_argument("input", help="Path to the input .hwpx file")
    parser.add_argument("--output-dir", help="Directory to save the printed PDF")
    parser.add_argument("--filename", help="Override output PDF filename")
    parser.add_argument("--keep-viewer", action="store_true", help="Keep viewer open after printing")
    parser.add_argument("--verbose", action="store_true", help="Enable debug logging")
    return parser.parse_args()


def run(command: list[str], *, env: dict[str, str] | None = None, check: bool = True) -> subprocess.CompletedProcess[str]:
    LOGGER.debug("running command: %s", command)  # TODO: remove after viewer flow stabilizes
    completed = subprocess.run(
        command,
        capture_output=True,
        text=True,
        check=False,
        env={**os.environ, **(env or {})},
    )
    if check and completed.returncode != 0:
        raise RuntimeError(completed.stderr.strip() or completed.stdout.strip() or "command failed")
    return completed


def run_osascript(script: str, *, env: dict[str, str] | None = None) -> str:
    completed = subprocess.run(
        ["osascript"],
        input=script,
        capture_output=True,
        text=True,
        check=False,
        env={**os.environ, **(env or {})},
    )
    if completed.returncode != 0:
        raise RuntimeError(completed.stderr.strip() or completed.stdout.strip() or "osascript failed")
    return completed.stdout.strip()


def is_viewer_running() -> bool:
    return run(["pgrep", "-x", VIEWER_APP], check=False).returncode == 0


def close_viewer() -> None:
    run(["osascript", "-e", f'tell application "{VIEWER_APP}" to quit'], check=False)
    time.sleep(1)
    if is_viewer_running():
        run(["pkill", "-x", VIEWER_APP], check=False)
        time.sleep(1)
    if is_viewer_running():
        raise RuntimeError("viewer process is still running after quit/pkill")


def build_output_paths(input_path: Path, output_dir_arg: str | None, filename_arg: str | None) -> tuple[Path, Path]:
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    output_dir = Path(output_dir_arg).resolve() if output_dir_arg else (DEFAULT_OUTPUT_DIR / f"viewer-print-{stamp}").resolve()
    filename = filename_arg or f"{input_path.stem}-print-{stamp}.pdf"
    output_dir.mkdir(parents=True, exist_ok=True)
    return output_dir, output_dir / filename


def open_save_panel(input_path: Path) -> None:
    run(["open", "-a", VIEWER_APP, str(input_path)])
    script = """
set appName to system attribute "VIEWER_APP"
set targetWin to system attribute "TARGET_WIN"

tell application appName to activate

tell application "System Events"
\trepeat 60 times
\t\tif exists application process appName then
\t\t\ttell application process appName
\t\t\t\tif exists window targetWin then exit repeat
\t\t\tend tell
\t\tend if
\t\tdelay 0.5
\tend repeat
\t
\ttell application process appName
\t\tset frontmost to true
\t\tif not (exists window targetWin) then error "target window missing"
\t\tclick menu item "인쇄..." of menu 1 of menu bar item "파일" of menu bar 1
\t\tdelay 1.5
\t\tset printSheet to sheet 1 of window targetWin
\t\tclick menu button 1 of group 2 of splitter group 1 of printSheet
\t\tdelay 0.5
\t\tclick menu item "PDF로 저장…" of menu 1 of menu button 1 of group 2 of splitter group 1 of printSheet
\t\tdelay 1.2
\t\tif not (exists sheet 1 of printSheet) then error "save sheet missing"
\tend tell
end tell
"""
    run_osascript(script, env={"VIEWER_APP": VIEWER_APP, "TARGET_WIN": input_path.name})


def print_to_pdf(input_path: Path, output_dir: Path, output_pdf: Path) -> dict[str, str]:
    script = """
set appName to system attribute "VIEWER_APP"
set targetWin to system attribute "TARGET_WIN"
set targetDir to system attribute "TARGET_DIR"
set targetFile to system attribute "TARGET_FILE"
set resultLines to {}

on appendLine(resultLines, lineText)
\tcopy lineText to end of resultLines
\treturn resultLines
end appendLine

tell application appName to activate
delay 0.5

tell application "System Events"
\ttell application process appName
\t\tset frontmost to true
\t\tset w to window targetWin
\t\tif not (exists sheet 1 of w) then error "print sheet missing"
\t\tset printSheet to sheet 1 of w
\t\tif not (exists sheet 1 of printSheet) then error "save sheet missing"
\t\tset saveSheet to sheet 1 of printSheet
\t\tset splitGroup to splitter group 1 of saveSheet
\t\tset resultLines to my appendLine(resultLines, "initial_filename=" & (value of text field 2 of splitGroup))
\t\tkeystroke "/"
\t\tdelay 1
\t\tset focusedElement to value of attribute "AXFocusedUIElement"
\t\tset focusRole to value of attribute "AXRole" of focusedElement
\t\tset focusValue to ""
\t\ttry
\t\t\tset focusValue to value of attribute "AXValue" of focusedElement
\t\tend try
\t\tset resultLines to my appendLine(resultLines, "focus_role=" & focusRole)
\t\tset resultLines to my appendLine(resultLines, "focus_value=" & focusValue)
\t\tif not (focusRole is "AXTextField" and focusValue is "/") then error "slash did not open path field"
\t\tset value of focusedElement to targetDir
\t\tdelay 0.2
\t\tkey code 36
\t\tdelay 1.2
\t\tset resultLines to my appendLine(resultLines, "location_after_path=" & (value of pop up button 1 of splitGroup))
\t\tset value of text field 2 of splitGroup to targetFile
\t\tdelay 0.2
\t\tset resultLines to my appendLine(resultLines, "final_filename=" & (value of text field 2 of splitGroup))
\t\tclick button "저장" of splitGroup
\t\tdelay 2
\tend tell
end tell

set AppleScript's text item delimiters to linefeed
set outputText to resultLines as text
set AppleScript's text item delimiters to ""
return outputText
"""
    output = run_osascript(
        script,
        env={
            "VIEWER_APP": VIEWER_APP,
            "TARGET_WIN": input_path.name,
            "TARGET_DIR": str(output_dir),
            "TARGET_FILE": output_pdf.name,
        },
    )
    result = {}
    for line in output.splitlines():
        if "=" not in line:
            continue
        key, value = line.split("=", 1)
        result[key] = value
    return result


def wait_for_pdf(output_pdf: Path, *, timeout_sec: float = 10) -> None:
    deadline = time.time() + timeout_sec
    while time.time() < deadline:
        if output_pdf.exists() and output_pdf.stat().st_size > 0:
            return
        time.sleep(0.5)
    raise FileNotFoundError(f"printed pdf not found: {output_pdf}")


def read_pdfinfo(output_pdf: Path) -> dict[str, str]:
    if shutil.which("pdfinfo") is None:
        return {}
    completed = run(["pdfinfo", str(output_pdf)], check=False)
    if completed.returncode != 0:
        return {}
    info = {}
    for line in completed.stdout.splitlines():
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        info[key.strip()] = value.strip()
    return info


def main() -> int:
    args = parse_args()
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(levelname)s %(name)s %(message)s",
    )

    input_path = Path(args.input).resolve()
    if not input_path.exists():
        raise FileNotFoundError(f"input file not found: {input_path}")
    if input_path.suffix.lower() != ".hwpx":
        raise ValueError(f"input must be a .hwpx file: {input_path}")

    output_dir, output_pdf = build_output_paths(input_path, args.output_dir, args.filename)
    close_viewer()
    result = {}
    pdfinfo = {}
    try:
        open_save_panel(input_path)
        result = print_to_pdf(input_path, output_dir, output_pdf)
        wait_for_pdf(output_pdf)
        pdfinfo = read_pdfinfo(output_pdf)
    finally:
        if not args.keep_viewer:
            try:
                close_viewer()
            except Exception as exc:  # noqa: BLE001
                LOGGER.warning("failed to close viewer: %s", exc)  # TODO: remove after viewer flow stabilizes

    report = {
        "input": str(input_path),
        "outputDir": str(output_dir),
        "outputPdf": str(output_pdf),
        "viewer": VIEWER_APP,
        "saveFlow": result,
        "pdfinfo": pdfinfo,
    }
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)

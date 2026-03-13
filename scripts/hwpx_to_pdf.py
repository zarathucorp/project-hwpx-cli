#!/usr/bin/env python

from __future__ import annotations

import argparse
import json
import logging
import os
import subprocess
import sys
import unicodedata
from pathlib import Path

from reportlab.lib import colors
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import mm
from reportlab.pdfbase.cidfonts import UnicodeCIDFont
from reportlab.pdfbase.pdfmetrics import registerFont
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.platypus import PageBreak, Paragraph, Preformatted, SimpleDocTemplate, Spacer, Table, TableStyle


LOGGER = logging.getLogger("hwpx_to_pdf")
DEFAULT_CLI = Path("./hwpxctl")
DEFAULT_OUTPUT_DIR = Path("output/pdf")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Convert HWPX CLI output into a reviewable PDF.")
    parser.add_argument("input", help="Path to the input .hwpx file")
    parser.add_argument("--output", help="Path to the output PDF")
    parser.add_argument("--cli", default=str(DEFAULT_CLI), help="Path to the hwpxctl binary")
    parser.add_argument("--title", help="Override PDF title")
    parser.add_argument("--verbose", action="store_true", help="Enable debug logging")
    return parser.parse_args()


def run_cli(cli_path: str, *args: str) -> str:
    command = [cli_path, *args]
    LOGGER.debug("running command: %s", command)  # TODO: remove after CLI flow stabilizes
    completed = subprocess.run(command, capture_output=True, text=True, check=False)
    if completed.returncode != 0:
        raise RuntimeError(completed.stderr.strip() or f"command failed: {' '.join(command)}")
    return completed.stdout


def normalize_text(value: str) -> str:
    return unicodedata.normalize("NFC", value)


def register_korean_font() -> str:
    system_fonts = [
        ("/System/Library/Fonts/Supplemental/AppleGothic.ttf", "AppleGothic"),
        ("/System/Library/Fonts/Supplemental/Arial Unicode.ttf", "ArialUnicode"),
        ("/usr/share/fonts/truetype/nanum/NanumGothic.ttf", "NanumGothic"),
        ("/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc", "NotoSansCJK"),
        ("/usr/share/fonts/opentype/noto/NotoSansCJKkr-Regular.otf", "NotoSansCJKkr"),
    ]

    for font_path, font_name in system_fonts:
        if os.path.exists(font_path):
            registerFont(TTFont(font_name, font_path))
            return font_name

    # Why: CID fonts provide a fallback when local TTF discovery fails.
    font_name = "HYGothic-Medium"
    registerFont(UnicodeCIDFont(font_name))
    return font_name


def build_styles(font_name: str) -> dict[str, ParagraphStyle]:
    styles = getSampleStyleSheet()
    return {
        "title": ParagraphStyle(
            "HwpxTitle",
            parent=styles["Title"],
            fontName=font_name,
            fontSize=20,
            leading=24,
            textColor=colors.HexColor("#1f2937"),
            spaceAfter=10,
        ),
        "heading": ParagraphStyle(
            "HwpxHeading",
            parent=styles["Heading2"],
            fontName=font_name,
            fontSize=12,
            leading=16,
            textColor=colors.HexColor("#111827"),
            spaceBefore=12,
            spaceAfter=6,
        ),
        "body": ParagraphStyle(
            "HwpxBody",
            parent=styles["BodyText"],
            fontName=font_name,
            fontSize=10,
            leading=15,
            wordWrap="CJK",
            textColor=colors.black,
        ),
        "mono": ParagraphStyle(
            "HwpxMono",
            parent=styles["Code"],
            fontName=font_name,
            fontSize=9,
            leading=12,
            wordWrap="CJK",
            textColor=colors.HexColor("#374151"),
        ),
    }


def build_pdf(
    output_path: Path,
    source_path: Path,
    title: str,
    inspect_data: dict,
    extracted_text: str,
) -> None:
    font_name = register_korean_font()
    styles = build_styles(font_name)

    doc = SimpleDocTemplate(
        str(output_path),
        pagesize=A4,
        rightMargin=18 * mm,
        leftMargin=18 * mm,
        topMargin=16 * mm,
        bottomMargin=16 * mm,
        title=title,
    )

    story = [
        Paragraph(normalize_text(title), styles["title"]),
        Paragraph(normalize_text(f"Source: {source_path}"), styles["body"]),
        Spacer(1, 4 * mm),
    ]

    summary_rows = [
        ["Field", "Value"],
        ["Valid", normalize_text(str(inspect_data.get("valid", False)))],
        ["Warnings", normalize_text(str(len(inspect_data.get("warnings", []))))],
        ["Entries", normalize_text(str(len(inspect_data.get("summary", {}).get("entries", []))))],
        ["Spine Items", normalize_text(str(len(inspect_data.get("summary", {}).get("spine", []))))],
        ["Sections", normalize_text(str(len(inspect_data.get("summary", {}).get("sectionPaths", []))))],
    ]
    summary_table = Table(summary_rows, colWidths=[35 * mm, 130 * mm], hAlign="LEFT")
    summary_table.setStyle(
        TableStyle(
            [
                ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#e5e7eb")),
                ("TEXTCOLOR", (0, 0), (-1, -1), colors.HexColor("#111827")),
                ("FONTNAME", (0, 0), (-1, -1), font_name),
                ("FONTSIZE", (0, 0), (-1, -1), 9),
                ("GRID", (0, 0), (-1, -1), 0.4, colors.HexColor("#d1d5db")),
                ("TOPPADDING", (0, 0), (-1, -1), 5),
                ("BOTTOMPADDING", (0, 0), (-1, -1), 5),
            ]
        )
    )

    story.extend(
        [
            Paragraph("CLI Summary", styles["heading"]),
            summary_table,
        ]
    )

    metadata = inspect_data.get("summary", {}).get("metadata", {})
    if metadata:
        metadata_rows = [["Field", "Value"]]
        for key, value in metadata.items():
            metadata_rows.append([normalize_text(key), normalize_text(str(value))])
        metadata_table = Table(metadata_rows, colWidths=[35 * mm, 130 * mm], hAlign="LEFT")
        metadata_table.setStyle(
            TableStyle(
                [
                    ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#eff6ff")),
                    ("FONTNAME", (0, 0), (-1, -1), font_name),
                    ("FONTSIZE", (0, 0), (-1, -1), 9),
                    ("GRID", (0, 0), (-1, -1), 0.4, colors.HexColor("#bfdbfe")),
                    ("TOPPADDING", (0, 0), (-1, -1), 5),
                    ("BOTTOMPADDING", (0, 0), (-1, -1), 5),
                ]
            )
        )
        story.extend([Paragraph("Metadata", styles["heading"]), metadata_table])

    warnings = inspect_data.get("warnings", [])
    if warnings:
        story.append(Paragraph("Warnings", styles["heading"]))
        for warning in warnings:
            story.append(Paragraph(normalize_text(f"- {warning}"), styles["body"]))

    story.extend([PageBreak(), Paragraph("Extracted Text", styles["heading"])])
    for line in extracted_text.splitlines():
        safe_line = normalize_text(line).replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
        if safe_line.strip():
            story.append(Paragraph(safe_line, styles["body"]))
        else:
            story.append(Spacer(1, 3 * mm))

    if not extracted_text.strip():
        story.append(Preformatted("(no text extracted)", styles["mono"]))

    doc.build(story, onFirstPage=draw_footer, onLaterPages=draw_footer)


def draw_footer(canvas, doc) -> None:
    canvas.saveState()
    canvas.setFont("Helvetica", 8)
    canvas.setFillColor(colors.HexColor("#6b7280"))
    canvas.drawRightString(190 * mm, 10 * mm, f"Page {doc.page}")
    canvas.restoreState()


def main() -> int:
    args = parse_args()
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(levelname)s %(name)s %(message)s",
    )

    input_path = Path(args.input).resolve()
    output_path = Path(args.output).resolve() if args.output else (DEFAULT_OUTPUT_DIR / f"{input_path.stem}.pdf").resolve()
    cli_path = Path(args.cli).resolve()

    if not input_path.exists():
        raise FileNotFoundError(f"input file not found: {input_path}")
    if not cli_path.exists():
        raise FileNotFoundError(f"CLI binary not found: {cli_path}")

    inspect_stdout = normalize_text(run_cli(str(cli_path), "inspect", str(input_path), "--format", "json"))
    inspect_envelope = json.loads(inspect_stdout)
    inspect_data = inspect_envelope["data"]["report"]
    extracted_stdout = normalize_text(run_cli(str(cli_path), "text", str(input_path), "--format", "json"))
    extracted_envelope = json.loads(extracted_stdout)
    extracted_text = normalize_text(extracted_envelope["data"].get("text", ""))

    output_path.parent.mkdir(parents=True, exist_ok=True)
    title = args.title or inspect_data.get("summary", {}).get("metadata", {}).get("title") or input_path.name
    build_pdf(output_path, input_path, title, inspect_data, extracted_text)
    LOGGER.info("generated pdf: %s", output_path)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001
        print(str(exc), file=sys.stderr)
        raise SystemExit(1)

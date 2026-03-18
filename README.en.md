# hwpxctl

English: [README.en.md](/Users/zarathu/projects/project-hwpx-cli/README.en.md)  
한국어: [README.md](/Users/zarathu/projects/project-hwpx-cli/README.md)

`hwpxctl` is a CLI for working with HWPX documents as ZIP/XML packages. It focuses on predictable inspection, unpack/pack workflows, paragraph and object editing, search, export, change history recording, and final render verification through Hancom Viewer PDF printing.

## Overview

- inspect HWPX package structure with `inspect`, `validate`, and `text`
- edit paragraphs, paragraph layout, lists, tables, sections, references, notes, headers/footers, equations, and shapes on unpacked directories
- control image and shape positioning
- export documents to Markdown and HTML
- search by style, object type, XML tag, attribute, and XPath
- record opt-in `historyEntry` change history
- verify final rendering through `Hancom Office HWP Viewer`

## Current Status

- most planned high-level editing, search, and export features for `v1` are implemented
- common document-authoring features such as paragraph layout, lists, and object positioning are included
- change tracking is currently `history-only`
- low-level XML/history/version access is still deferred

See [docs/roadmap.md](/Users/zarathu/projects/project-hwpx-cli/docs/roadmap.md) for the current scope and next priorities.

## Build

```bash
go build ./cmd/hwpxctl
./hwpxctl --help
```

## Quick Start

```bash
./hwpxctl inspect ./sample.hwpx
./hwpxctl unpack ./sample.hwpx --output ./work/sample
./hwpxctl append-text ./work/sample --text "Review paragraph"
./hwpxctl pack ./work/sample --output ./output/sample-edited.hwpx
python ./scripts/print_hwpx_via_viewer.py ./output/sample-edited.hwpx
```

For detailed command contracts and options, use [docs/cli-reference.md](/Users/zarathu/projects/project-hwpx-cli/docs/cli-reference.md).

## Documentation

- [docs/cli-reference.md](/Users/zarathu/projects/project-hwpx-cli/docs/cli-reference.md): command inputs, outputs, options, and JSON envelope
- [docs/agent-guide.md](/Users/zarathu/projects/project-hwpx-cli/docs/agent-guide.md): recommended invocation order for AI agents
- [docs/roadmap.md](/Users/zarathu/projects/project-hwpx-cli/docs/roadmap.md): implemented scope and next priorities
- [docs/research-notes.md](/Users/zarathu/projects/project-hwpx-cli/docs/research-notes.md): format notes and design background

## Verification Policy

- unit tests and structural validation are not treated as the final completion signal
- editing features should be verified with a real `.hwpx` artifact and a Hancom Viewer PDF print whenever possible
- the default verification script is `python ./scripts/print_hwpx_via_viewer.py <file.hwpx>`
- verification artifacts should remain under `output/` for comparison

## Project Layout

- [cmd/hwpxctl/main.go](/Users/zarathu/projects/project-hwpx-cli/cmd/hwpxctl/main.go): CLI entrypoint
- [internal/cli/cobra.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/cobra.go): subcommand routing and help wiring
- [internal/cli/root.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/root.go): shared options, error envelope, `schema`
- [internal/hwpx/core](/Users/zarathu/projects/project-hwpx-cli/internal/hwpx/core): package IO and export logic
- [internal/hwpx/shared](/Users/zarathu/projects/project-hwpx-cli/internal/hwpx/shared): shared XML editing utilities
- [scripts/print_hwpx_via_viewer.py](/Users/zarathu/projects/project-hwpx-cli/scripts/print_hwpx_via_viewer.py): Hancom Viewer PDF print verification

## Limitations

- most editing commands still operate on the first section
- change tracking currently records `historyEntry` only
- legacy `.hwp` is not supported
- low-level XML part inspection/editing is not exposed yet

## Development Notes

- prefer `--format json` or `HWPXCTL_FORMAT=json` for automation
- the default delivery flow is `implement -> generate .hwpx -> print through Hancom Viewer -> inspect output`
- keep detailed command examples in reference docs rather than expanding the root README

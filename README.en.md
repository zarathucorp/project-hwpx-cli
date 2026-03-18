# hwpxctl

English: [README.en.md](./README.en.md)
한국어: [README.md](./README.md)

`hwpxctl` is a CLI for working with HWPX documents as ZIP/XML packages. It focuses on predictable inspection, unpack/pack workflows, paragraph and table editing, search, export, change history recording, and final render verification through Hancom Viewer PDF printing.

## Overview

- inspect `.hwpx` package structure with `inspect`, `validate`, and `text`
- edit paragraphs, paragraph layout, lists, tables, sections, notes, headers/footers, hyperlinks, equations, and shapes on unpacked directories
- export documents to Markdown and HTML
- search by style, object type, XML tag, attribute, and XPath
- record opt-in `historyEntry` change history
- verify final rendering through `Hancom Office HWP Viewer`

## Current Status

- most planned high-level editing, search, and export features for `v1` are implemented
- page layout, font, cell style, merge, and border controls needed to reconstruct table-heavy forms from blank documents are available
- change tracking is currently `history-only`
- low-level XML/history/version access is still deferred

See [docs/roadmap.md](./docs/roadmap.md) for the current scope and next priorities.

## Supported Environments

- macOS: full CLI workflow including Hancom Viewer PDF print verification
- Linux / CI: CLI editing, validation, export, and tests work, but Viewer print automation is not available
- Windows / PowerShell: CLI build and basic editing work, but Viewer print automation is not available

The main difference is that `scripts/print_hwpx_via_viewer.py` depends on macOS `osascript` and `Hancom Office HWP Viewer`.

## Requirements

- with the current distribution model, `Go toolchain` must be installed first
- Go `1.26+`
- Python
- `Hancom Office HWP Viewer` for final render verification on macOS

At the moment, there is no Homebrew, apt, winget, or prebuilt release-binary distribution.  
That means `go install` or `go build` is the default installation path.

## Installation

For a public Go CLI, `go install` is the most common default path.

```bash
go install github.com/zarathucop/project-hwpx-cli/cmd/hwpxctl@latest
```

The executable is typically installed under `GOBIN` or `$(go env GOPATH)/bin`.
You can inspect the current install location with:

```bash
go env GOBIN
go env GOPATH
```

### PATH setup

#### macOS / Linux

For the current shell session:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

To persist it, add the same line to your shell config file.

- zsh: `~/.zshrc`
- bash: `~/.bashrc` or `~/.bash_profile`

If you prefer a dedicated bin directory, set `GOBIN` explicitly:

```bash
go env -w GOBIN="$HOME/.local/bin"
```

#### Windows PowerShell

A common user-level setup is:

```powershell
$goBin = "$(go env GOPATH)\bin"
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";" + $goBin, "User")
```

Then open a new PowerShell window and verify that `hwpxctl.exe` resolves from PATH.

### Build from source

For local development or modified builds:

```bash
go build -o ./hwpxctl ./cmd/hwpxctl
./hwpxctl --help
```

## Quick Start by Environment

### macOS

```bash
hwpxctl inspect ./sample.hwpx
hwpxctl unpack ./sample.hwpx --output ./work/sample
hwpxctl append-text ./work/sample --text "Review paragraph"
hwpxctl pack ./work/sample --output ./output/sample-edited.hwpx
```

### Linux / CI

```bash
hwpxctl validate ./sample.hwpx --format json
hwpxctl unpack ./sample.hwpx --output ./work/sample
hwpxctl append-text ./work/sample --text "Review paragraph"
hwpxctl pack ./work/sample --output ./output/sample-edited.hwpx
hwpxctl text ./output/sample-edited.hwpx --format json
```

### Windows PowerShell

```powershell
hwpxctl.exe inspect .\sample.hwpx
hwpxctl.exe unpack .\sample.hwpx --output .\work\sample
hwpxctl.exe append-text .\work\sample --text "Review paragraph"
hwpxctl.exe pack .\work\sample --output .\output\sample-edited.hwpx
```

## Basic Usage

### 1. Edit an existing document

```bash
hwpxctl inspect ./sample.hwpx
hwpxctl unpack ./sample.hwpx --output ./work/sample
hwpxctl set-text-style ./work/sample --paragraph 0 --font-name "Malgun Gothic" --font-size-pt 12
hwpxctl pack ./work/sample --output ./output/sample-edited.hwpx
```

### 2. Start a table-based form from a blank document

```bash
hwpxctl create --output ./work/form
hwpxctl set-page-layout ./work/form --orientation PORTRAIT --width-mm 210 --height-mm 297 --left-margin-mm 25 --right-margin-mm 25 --top-margin-mm 15 --bottom-margin-mm 15
hwpxctl add-table ./work/form --rows 4 --cols 3 --width-mm 160
hwpxctl merge-table-cells ./work/form --table 0 --start-row 0 --start-col 0 --end-row 0 --end-col 2
hwpxctl set-table-cell ./work/form --table 0 --row 0 --col 0 --text "Title" --font-name "Malgun Gothic" --font-size-pt 14 --bold true
hwpxctl normalize-table-borders ./work/form --table 0
hwpxctl pack ./work/form --output ./output/form.hwpx
```

### 3. Machine-readable automation output

```bash
hwpxctl schema
hwpxctl validate ./sample.hwpx --format json
hwpxctl find-runs-by-style ./work/sample --font-name "Malgun Gothic" --font-size-pt 12 --format json
```

### 4. Final render verification on macOS

```bash
python ./scripts/print_hwpx_via_viewer.py ./output/sample-edited.hwpx
```

For detailed command contracts and options, use [docs/cli-reference.md](./docs/cli-reference.md).

## Documentation

- [docs/cli-reference.md](./docs/cli-reference.md): command inputs, outputs, options, and JSON envelope
- [docs/agent-guide.md](./docs/agent-guide.md): recommended invocation order for AI agents
- [docs/example-table-playbook.md](./docs/example-table-playbook.md): page-by-page playbook and lessons learned for recreating `example` tables
- [docs/example-parity-harness.md](./docs/example-parity-harness.md): parity harness for comparing original and generated example-like outputs
- [docs/roadmap.md](./docs/roadmap.md): implemented scope and next priorities
- [docs/research-notes.md](./docs/research-notes.md): format notes and design background

## Public Repo Notes

- the public repository should not include original `example/*.hwpx` files when they may carry licensing or sensitive content concerns
- when a sample source is needed, prefer passing a local private path directly
- the default docs publishing target is GitHub Pages

## Verification Policy

- unit tests and structural validation are not treated as the final completion signal
- editing features should be verified with a real `.hwpx` artifact and a Hancom Viewer PDF print whenever possible
- the default verification script is `python ./scripts/print_hwpx_via_viewer.py <file.hwpx>`
- verification artifacts should remain under `output/` for comparison

## Project Layout

- [cmd/hwpxctl/main.go](./cmd/hwpxctl/main.go): CLI entrypoint
- [internal/cli/cobra.go](./internal/cli/cobra.go): subcommand routing and help wiring
- [internal/cli/root.go](./internal/cli/root.go): shared options, error envelope, `schema`
- [internal/hwpx/core](./internal/hwpx/core): package IO and export logic
- [internal/hwpx/shared](./internal/hwpx/shared): shared XML editing utilities
- [scripts/print_hwpx_via_viewer.py](./scripts/print_hwpx_via_viewer.py): Hancom Viewer PDF print verification

## Limitations

- most editing commands still operate on the first section
- Viewer PDF print automation is currently macOS-only
- change tracking currently records `historyEntry` only
- legacy `.hwp` is not supported
- low-level XML part inspection/editing is not exposed yet

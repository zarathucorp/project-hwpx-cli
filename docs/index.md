# hwpxctl

`hwpxctl`은 HWPX 문서를 ZIP/XML 패키지로 다루는 CLI입니다. inspect, validate, unpack/pack, 문단/표/섹션 편집, export, 검색, 그리고 Hancom Viewer PDF 검증까지 한 흐름으로 다룹니다.

## 어디서 시작하면 되나

- 전체 명령 계약: [CLI Reference](./cli-reference.md)
- example 표를 다시 만드는 실전 가이드: [Example Table Playbook](./example-table-playbook.md)
- 원본/생성본 비교 자동화: [Example Parity Harness](./example-parity-harness.md)
- 현재 우선순위: [Roadmap](./roadmap.md)

## 지원 환경

| 환경 | CLI 빌드/편집 | Viewer PDF 자동 인쇄 |
| --- | --- | --- |
| macOS | 가능 | 가능 |
| Linux / CI | 가능 | 불가 |
| Windows / PowerShell | 가능 | 불가 |

`scripts/print_hwpx_via_viewer.py`는 macOS `osascript`와 `Hancom Office HWP Viewer`에 의존합니다.

## 설치 전제조건

- 현재 설치 방식은 `go install` 또는 `go build` 기준입니다
- 따라서 각 환경에 `Go 1.26+`가 먼저 설치되어 있어야 합니다
- 아직 package manager나 prebuilt release binary 배포는 없습니다

## Quick Start

설치가 먼저면 보통 아래 방식부터 씁니다.

```bash
go install github.com/zarathucop/project-hwpx-cli/cmd/hwpxctl@latest
```

설치 후 `hwpxctl`이 바로 안 잡히면 `$(go env GOPATH)/bin` 또는 `GOBIN`을 PATH에 추가해야 합니다.

### macOS

```bash
hwpxctl inspect ./sample.hwpx
hwpxctl unpack ./sample.hwpx --output ./work/sample
hwpxctl append-text ./work/sample --text "검토 문단"
hwpxctl pack ./work/sample --output ./output/sample-edited.hwpx
```

### Linux / CI

```bash
hwpxctl validate ./sample.hwpx --format json
hwpxctl unpack ./sample.hwpx --output ./work/sample
hwpxctl append-text ./work/sample --text "검토 문단"
hwpxctl pack ./work/sample --output ./output/sample-edited.hwpx
hwpxctl text ./output/sample-edited.hwpx --format json
```

### Windows PowerShell

```powershell
hwpxctl.exe inspect .\sample.hwpx
hwpxctl.exe unpack .\sample.hwpx --output .\work\sample
hwpxctl.exe append-text .\work\sample --text "검토 문단"
hwpxctl.exe pack .\work\sample --output .\output\sample-edited.hwpx
```

## macOS 렌더 검증

```bash
python ./scripts/print_hwpx_via_viewer.py ./output/sample-edited.hwpx
```

## 공개 저장소 메모

- 원본 `example/*.hwpx`는 공개 저장소에 포함하지 않는 것을 기본값으로 합니다
- local private sample은 경로를 직접 지정해 사용하는 흐름을 권장합니다
- parity 하네스는 sample 경로를 직접 넘겨 사용하는 흐름을 전제로 합니다

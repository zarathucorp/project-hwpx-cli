# hwpxctl

`hwpxctl`은 HWPX 문서를 ZIP/XML 패키지로 다루는 CLI입니다. 현재 중심 방향은 low-level XML surgery 도구를 유지하면서, 기존 복합 양식을 안전하게 분석하고 채우는 `Template-First` 편집 흐름을 강화하는 것입니다.

## 어디서 시작하면 되나

- 전체 명령 계약: [CLI Reference](./cli-reference.md)
- 현재 구조 정리: [Architecture](./architecture.md)
- 현재 개발 상태: [Progress](./progress.md)
- example 표를 다시 만드는 실전 가이드: [Example Table Playbook](./example-table-playbook.md)
- 원본/생성본 비교 자동화: [Example Parity Harness](./example-parity-harness.md)
- 현재 우선순위: [Roadmap](./roadmap.md)

## 핵심 흐름

- `inspect`, `validate`, `text`로 `.hwpx` 구조 확인
- `analyze-template`, `find-targets`, `scaffold-template-contract`, `fill-template --template --payload` 기반 Template-First 흐름 지원
- unpack 디렉터리 기준으로 문단, 표, 섹션, 레이아웃, 객체 편집 지원
- 최종 검증은 macOS 기준 `Hancom Office HWP Viewer` PDF 인쇄 결과를 기준으로 함

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
go install github.com/zarathucorp/project-hwpx-cli/cmd/hwpxctl@latest
```

새 환경에서는 보통 `PATH` 반영까지 같이 해야 합니다.

### macOS / Linux PATH 설정

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

macOS `zsh`에서 영구 반영하려면 보통 아래 순서입니다.

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
command -v hwpxctl
hwpxctl --help
```

### Windows PowerShell PATH 설정

```powershell
$goBin = "$(go env GOPATH)\bin"
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";" + $goBin, "User")
```

새 PowerShell 창에서 확인합니다.

```powershell
Get-Command hwpxctl.exe
hwpxctl.exe --help
```

아래 예시는 `hwpxctl`이 이미 PATH에서 잡히는 상태를 전제로 합니다.

### macOS

```bash
go install github.com/zarathucorp/project-hwpx-cli/cmd/hwpxctl@latest
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
command -v hwpxctl
hwpxctl inspect ./sample.hwpx
hwpxctl unpack ./sample.hwpx --output ./work/sample
hwpxctl append-text ./work/sample --text "검토 문단"
hwpxctl pack ./work/sample --output ./output/sample-edited.hwpx
```

### Linux / CI

```bash
go install github.com/zarathucorp/project-hwpx-cli/cmd/hwpxctl@latest
export PATH="$(go env GOPATH)/bin:$PATH"
command -v hwpxctl
hwpxctl validate ./sample.hwpx --format json
hwpxctl unpack ./sample.hwpx --output ./work/sample
hwpxctl append-text ./work/sample --text "검토 문단"
hwpxctl pack ./work/sample --output ./output/sample-edited.hwpx
hwpxctl text ./output/sample-edited.hwpx --format json
```

### Windows PowerShell

```powershell
go install github.com/zarathucorp/project-hwpx-cli/cmd/hwpxctl@latest
$goBin = "$(go env GOPATH)\bin"
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";" + $goBin, "User")
$env:Path = [Environment]::GetEnvironmentVariable("Path", "User")
Get-Command hwpxctl.exe
hwpxctl.exe inspect .\sample.hwpx
hwpxctl.exe unpack .\sample.hwpx --output .\work\sample
hwpxctl.exe append-text .\work\sample --text "검토 문단"
hwpxctl.exe pack .\work\sample --output .\output\sample-edited.hwpx
```

## macOS 렌더 검증

```bash
python ./scripts/print_hwpx_via_viewer.py ./output/sample-edited.hwpx
```

## 문서 구분

- 방향과 구조: [Architecture](./architecture.md)
- 단계 계획: [Roadmap](./roadmap.md)
- 현재 진행 상태: [Progress](./progress.md)

## 공개 저장소 메모

- 원본 `example/*.hwpx`는 공개 저장소에 포함하지 않는 것을 기본값으로 합니다
- local private sample은 경로를 직접 지정해 사용하는 흐름을 권장합니다
- parity 하네스는 sample 경로를 직접 넘겨 사용하는 흐름을 전제로 합니다

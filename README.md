# hwpxctl

한국어: [README.md](./README.md)
English: [README.en.md](./README.en.md)

`hwpxctl`은 HWPX 문서를 ZIP/XML 패키지로 다루는 CLI입니다. 구조 점검, unpack/pack, 문단·표·섹션·참조·도형 편집, 검색, export, 변경 이력 기록, 한컴 뷰어 기반 PDF 검증까지 한 흐름으로 다루는 것을 목표로 합니다.

## 프로젝트 개요

- `.hwpx` 파일의 구조를 `inspect`, `validate`, `text`로 확인
- unpack 디렉터리 기준으로 문단, 문단 정렬/들여쓰기/간격, 목록, 표, 섹션, 각주/미주, 머리말/꼬리말, 하이퍼링크, 수식, 메모, 도형 편집
- Markdown/HTML export 지원
- 스타일, 객체, XML 태그/속성/XPath 기반 검색 지원
- opt-in `historyEntry` 변경 추적 지원
- 최종 검증은 `Hancom Office HWP Viewer` PDF 인쇄 결과 기준

## 현재 상태

- `v1` 범위의 고수준 편집/검색/export 기능은 대부분 구현됨
- 빈 문서에서 표 기반 양식을 재구성하는 데 필요한 페이지 레이아웃, 폰트, 셀 스타일, merge/border 기능이 포함됨
- 변경 추적은 현재 `history-only` 1차 구현임
- low-level XML/history/version 접근은 다음 단계로 남아 있음

세부 진행 상태는 [docs/roadmap.md](./docs/roadmap.md)에서 확인할 수 있습니다.

## 지원 환경

- macOS: CLI 편집 + `Hancom Office HWP Viewer` PDF 인쇄 검증까지 전체 흐름 지원
- Linux / CI: CLI 편집, validate, export, 테스트는 가능하지만 Viewer 자동 인쇄 검증은 지원하지 않음
- Windows / PowerShell: CLI 빌드와 기본 편집 흐름은 가능하지만 Viewer 자동 인쇄 검증 스크립트는 지원하지 않음

핵심 차이는 `scripts/print_hwpx_via_viewer.py`가 macOS의 `osascript`와 `Hancom Office HWP Viewer`에 의존한다는 점입니다.

## 요구 사항

- 현재 배포 방식 기준으로 설치 전 `Go toolchain`이 먼저 있어야 합니다
- Go `1.26+`
- Python
- macOS에서 최종 렌더 검증이 필요하면 `Hancom Office HWP Viewer`

현재는 Homebrew, apt, winget, prebuilt release binary 배포를 아직 제공하지 않습니다.  
즉 `go install` 또는 `go build`로 설치하는 흐름이 기본입니다.

## 설치

공개 저장소 기준으로는 `go install` 방식이 가장 일반적입니다.

```bash
go install github.com/zarathucop/project-hwpx-cli/cmd/hwpxctl@latest
```

설치 후 실행 파일은 보통 `GOBIN` 또는 `$(go env GOPATH)/bin` 아래에 놓입니다.
현재 설치 위치는 아래 명령으로 확인할 수 있습니다.

```bash
go env GOBIN
go env GOPATH
```

### PATH 설정

#### macOS / Linux

현재 셸에서만 바로 쓰려면:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

영구 반영은 사용하는 셸 설정 파일에 같은 줄을 추가하면 됩니다.

- zsh: `~/.zshrc`
- bash: `~/.bashrc` 또는 `~/.bash_profile`

원하면 기본 설치 위치 대신 별도 bin 디렉터리를 지정할 수도 있습니다.

```bash
go env -w GOBIN="$HOME/.local/bin"
```

#### Windows PowerShell

현재 사용자 PATH에 Go 바이너리 디렉터리를 추가하는 일반적인 방식은 아래와 같습니다.

```powershell
$goBin = "$(go env GOPATH)\bin"
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";" + $goBin, "User")
```

새 PowerShell 창을 연 뒤 `hwpxctl.exe`가 바로 실행되는지 확인합니다.

### 소스에서 직접 빌드

개발 중이거나 로컬 수정본을 바로 실행하고 싶으면 아래 방식도 쓸 수 있습니다.

```bash
go build -o ./hwpxctl ./cmd/hwpxctl
./hwpxctl --help
```

## 환경별 Quick Start

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

## 기본 사용 흐름

### 1. 기존 문서 수정

```bash
hwpxctl inspect ./sample.hwpx
hwpxctl unpack ./sample.hwpx --output ./work/sample
hwpxctl set-text-style ./work/sample --paragraph 0 --font-name "맑은 고딕" --font-size-pt 12
hwpxctl pack ./work/sample --output ./output/sample-edited.hwpx
```

### 2. 빈 문서에서 표 양식 시작

```bash
hwpxctl create --output ./work/form
hwpxctl set-page-layout ./work/form --orientation PORTRAIT --width-mm 210 --height-mm 297 --left-margin-mm 25 --right-margin-mm 25 --top-margin-mm 15 --bottom-margin-mm 15
hwpxctl add-table ./work/form --rows 4 --cols 3 --width-mm 160
hwpxctl merge-table-cells ./work/form --table 0 --start-row 0 --start-col 0 --end-row 0 --end-col 2
hwpxctl set-table-cell ./work/form --table 0 --row 0 --col 0 --text "제목" --font-name "맑은 고딕" --font-size-pt 14 --bold true
hwpxctl normalize-table-borders ./work/form --table 0
hwpxctl pack ./work/form --output ./output/form.hwpx
```

### 3. 자동화용 JSON 출력

```bash
hwpxctl schema
hwpxctl validate ./sample.hwpx --format json
hwpxctl find-runs-by-style ./work/sample --font-name "맑은 고딕" --font-size-pt 12 --format json
```

### 4. macOS 최종 렌더 검증

```bash
python ./scripts/print_hwpx_via_viewer.py ./output/sample-edited.hwpx
```

상세 명령 계약과 옵션은 [docs/cli-reference.md](./docs/cli-reference.md)에서 확인하는 것을 기본으로 합니다.

## 문서

- [docs/cli-reference.md](./docs/cli-reference.md): 명령별 입력, 출력, 옵션, JSON envelope
- [docs/agent-guide.md](./docs/agent-guide.md): AI 에이전트 호출 순서와 권장 사용 패턴
- [docs/example-table-playbook.md](./docs/example-table-playbook.md): example 표를 페이지 단위로 다시 만드는 작업형 가이드와 시행착오 메모
- [docs/example-parity-harness.md](./docs/example-parity-harness.md): example 원본과 생성본을 같은 검증 루프로 비교하는 parity 하네스 안내
- [docs/roadmap.md](./docs/roadmap.md): 구현 범위와 다음 작업 우선순위
- [docs/research-notes.md](./docs/research-notes.md): HWPX 구조 메모와 설계 배경

## 공개 저장소 메모

- 공개 저장소에는 민감하거나 저작권 이슈가 있을 수 있는 원본 `example/*.hwpx`를 포함하지 않는 것을 기본값으로 합니다
- 로컬에서 private sample을 써야 할 경우 `example/` 아래에 두고, Git에는 올리지 않는 흐름을 권장합니다
- 문서 사이트는 GitHub Pages로 자동 배포하는 구성을 기본으로 둡니다

## 검증 원칙

- 구조 검증이나 단위 테스트만으로 완료 처리하지 않습니다
- 편집 기능은 가능하면 실제 `.hwpx` 산출물을 만든 뒤 `Hancom Office HWP Viewer`로 PDF 인쇄 검증까지 수행합니다
- 기본 검증 스크립트는 `python ./scripts/print_hwpx_via_viewer.py <file.hwpx>`입니다
- 검증 산출물은 `output/` 아래에 남겨 before/after 비교가 가능하도록 유지합니다

## 프로젝트 구조

- [cmd/hwpxctl/main.go](./cmd/hwpxctl/main.go): CLI 진입점
- [internal/cli/cobra.go](./internal/cli/cobra.go): 서브커맨드 라우팅과 help 연결
- [internal/cli/root.go](./internal/cli/root.go): 공통 옵션, 에러 envelope, `schema`
- [internal/hwpx/core](./internal/hwpx/core): 패키지 읽기/쓰기와 export 핵심 로직
- [internal/hwpx/shared](./internal/hwpx/shared): 공통 XML 편집 유틸리티
- [scripts/print_hwpx_via_viewer.py](./scripts/print_hwpx_via_viewer.py): 한컴 뷰어 PDF 인쇄 검증 스크립트

## 한계

- 주요 편집 명령은 아직 첫 section 중심으로 동작합니다
- Viewer PDF 자동 인쇄 검증은 현재 macOS 전용입니다
- 변경 추적은 visible tracking이 아니라 `historyEntry` 기록 위주입니다
- `.hwp`는 지원하지 않고 `.hwpx`만 다룹니다
- low-level XML part 조회/편집 API는 아직 정식 노출하지 않았습니다

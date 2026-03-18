# hwpxctl

한국어: [README.md](/Users/zarathu/projects/project-hwpx-cli/README.md)  
English: [README.en.md](/Users/zarathu/projects/project-hwpx-cli/README.en.md)

`hwpxctl`은 HWPX 문서를 ZIP 기반 XML 패키지로 다루는 CLI입니다. 구조 점검, unpack/pack, 본문·문단 서식·표·참조·도형 편집, 검색, export, 변경 이력 기록, 한컴 뷰어 기반 렌더링 검증까지 한 흐름으로 다루는 것을 목표로 합니다.

## 프로젝트 개요

- `.hwpx` 파일의 구조를 `inspect`, `validate`, `text`로 확인
- unpack 디렉터리 기준으로 문단, 문단 정렬/들여쓰기/간격, 글머리표/번호 매기기, 표, 섹션, 각주/미주, 머리말/꼬리말, 하이퍼링크, 수식, 메모, 도형 편집
- 이미지/도형 위치 제어 지원
- Markdown/HTML export 지원
- 스타일, 객체, XML 태그/속성/XPath 기반 검색 지원
- opt-in `historyEntry` 변경 추적 지원
- 최종 검증은 `Hancom Office HWP Viewer` PDF 인쇄 결과 기준

## 현재 상태

- `v1` 범위의 고수준 편집/검색/export 기능은 대부분 구현됨
- 일반 문서 작업에서 자주 쓰는 문단 서식, 목록, 객체 위치 제어까지 포함됨
- 변경 추적은 현재 `history-only` 1차 구현임
- low-level XML/history/version 접근은 다음 단계로 남아 있음

세부 진행 상태는 [docs/roadmap.md](/Users/zarathu/projects/project-hwpx-cli/docs/roadmap.md)에서 확인할 수 있습니다.

## 설치

```bash
go build ./cmd/hwpxctl
./hwpxctl --help
```

## 빠른 시작

```bash
./hwpxctl inspect ./sample.hwpx
./hwpxctl unpack ./sample.hwpx --output ./work/sample
./hwpxctl append-text ./work/sample --text "검토 문단"
./hwpxctl pack ./work/sample --output ./output/sample-edited.hwpx
python ./scripts/print_hwpx_via_viewer.py ./output/sample-edited.hwpx
```

상세 명령 계약과 옵션은 [docs/cli-reference.md](/Users/zarathu/projects/project-hwpx-cli/docs/cli-reference.md)에서 확인하는 것을 기본으로 합니다.

## 문서

- [docs/cli-reference.md](/Users/zarathu/projects/project-hwpx-cli/docs/cli-reference.md): 명령별 입력, 출력, 옵션, JSON envelope
- [docs/agent-guide.md](/Users/zarathu/projects/project-hwpx-cli/docs/agent-guide.md): AI 에이전트 호출 순서와 권장 사용 패턴
- [docs/roadmap.md](/Users/zarathu/projects/project-hwpx-cli/docs/roadmap.md): 구현 범위와 다음 작업 우선순위
- [docs/research-notes.md](/Users/zarathu/projects/project-hwpx-cli/docs/research-notes.md): HWPX 구조 메모와 설계 배경

## 검증 원칙

- 구조 검증이나 단위 테스트만으로 완료 처리하지 않습니다.
- 편집 기능은 가능하면 실제 `.hwpx` 산출물을 만든 뒤 `Hancom Office HWP Viewer`로 PDF 인쇄 검증까지 수행합니다.
- 기본 검증 스크립트는 `python ./scripts/print_hwpx_via_viewer.py <file.hwpx>`입니다.
- 검증 산출물은 `output/` 아래에 남겨 before/after 비교가 가능하도록 유지합니다.

## 프로젝트 구조

- [cmd/hwpxctl/main.go](/Users/zarathu/projects/project-hwpx-cli/cmd/hwpxctl/main.go): CLI 진입점
- [internal/cli/cobra.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/cobra.go): 서브커맨드 라우팅과 help 연결
- [internal/cli/root.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/root.go): 공통 옵션, 에러 envelope, `schema`
- [internal/hwpx/core](/Users/zarathu/projects/project-hwpx-cli/internal/hwpx/core): 패키지 읽기/쓰기와 export 핵심 로직
- [internal/hwpx/shared](/Users/zarathu/projects/project-hwpx-cli/internal/hwpx/shared): 공통 XML 편집 유틸리티
- [scripts/print_hwpx_via_viewer.py](/Users/zarathu/projects/project-hwpx-cli/scripts/print_hwpx_via_viewer.py): 한컴 뷰어 PDF 인쇄 검증 스크립트

## 한계

- 주요 편집 명령은 아직 첫 section 중심으로 동작합니다.
- 변경 추적은 visible tracking이 아니라 `historyEntry` 기록 위주입니다.
- `.hwp`는 지원하지 않고 `.hwpx`만 다룹니다.
- low-level XML part 조회/편집 API는 아직 정식 노출하지 않았습니다.

## 개발 메모

- JSON 기반 자동화에는 `--format json` 또는 `HWPXCTL_FORMAT=json` 사용을 권장합니다.
- 새 기능은 `CLI 구현 -> 실제 .hwpx 생성 -> Hancom Viewer PDF 인쇄 -> 결과 확인` 순서로 검증합니다.
- 상세 사용 예시를 README에 계속 누적하지 않고, 필요시 레퍼런스 문서나 작업형 문서로 분리하는 방향을 유지합니다.

# hwpxctl

`hwpxctl`은 macOS/Linux 우선으로 설계한 HWPX CLI입니다. HWPX를 ZIP 기반 XML 패키지로 보고 구조를 점검하고, 텍스트를 추출하고, 압축 해제/재패킹할 수 있습니다.

문서 진입점:

- [docs/README.md](/Users/zarathu/projects/project-hwpx-cli/docs/README.md)
- [docs/cli-reference.md](/Users/zarathu/projects/project-hwpx-cli/docs/cli-reference.md)
- [docs/agent-guide.md](/Users/zarathu/projects/project-hwpx-cli/docs/agent-guide.md)
- [docs/roadmap.md](/Users/zarathu/projects/project-hwpx-cli/docs/roadmap.md)

## 명령

- `inspect <file.hwpx>`: 메타데이터, manifest, spine, section 경로 조회
- `validate <file.hwpx|directory>`: 필수 파일과 manifest/spine 참조 일관성 검증
- `text <file.hwpx>`: `Contents/section*.xml` 기준 텍스트 추출
- `unpack <file.hwpx> --output <directory>`: 작업 디렉터리로 압축 해제
- `pack <directory> --output <file.hwpx>`: 검증된 디렉터리를 다시 `.hwpx`로 패키징
- `create --output <directory>`: 편집 가능한 unpack 디렉터리 생성
- `append-text <directory> --text <text>`: 첫 section 끝에 문단 추가
- `set-paragraph-text <directory> --paragraph <n> --text <text>`: 본문 문단 텍스트 수정
- `set-text-style <directory> --paragraph <n>`: 본문 문단의 run 스타일 수정
- `delete-paragraph <directory> --paragraph <n>`: 본문 문단 삭제
- `add-section <directory>`: 문서 끝에 빈 section 추가
- `delete-section <directory> --section <n>`: spine 순서 기준 section 삭제
- `add-table <directory> --rows <n> --cols <n>`: 첫 section 끝에 표 추가
- `add-nested-table <directory> --table <n> --row <n> --col <n>`: 표 셀 안에 중첩 표 추가
- `set-table-cell <directory> --table <n> --row <n> --col <n> --text <text>`: 표 셀 텍스트 수정
- `merge-table-cells <directory> --table <n> --start-row <n> --start-col <n> --end-row <n> --end-col <n>`: 표 셀 병합
- `split-table-cell <directory> --table <n> --row <n> --col <n>`: 병합된 표 셀 분할
- `embed-image <directory> --image <file>`: 이미지 바이너리를 문서에 임베드
- `insert-image <directory> --image <file>`: 이미지를 임베드하고 본문에 배치
- `set-header <directory> --text <text>`: 첫 section에 머리말 설정
- `set-footer <directory> --text <text>`: 첫 section에 꼬리말 설정
- `set-page-number <directory>`: 첫 section에 쪽 번호 표시 설정
- `set-columns <directory> --count <n>`: 첫 section에 다단 설정
- `add-footnote <directory> --anchor-text <text> --text <text>`: 각주가 달린 문단 추가
- `add-endnote <directory> --anchor-text <text> --text <text>`: 미주가 달린 문단 추가
- `add-bookmark <directory> --name <name> --text <text>`: 책갈피 위치 문단 추가
- `add-hyperlink <directory> --target <url|#bookmark> --text <text>`: 하이퍼링크 문단 추가
- `add-heading <directory> --kind <title|heading|outline> --text <text>`: 제목/개요 문단 추가
- `insert-toc <directory>`: 제목/개요 문단 기준 기본 차례 생성
- `add-cross-reference <directory> --bookmark <name>`: 책갈피 기준 내부 참조 문단 추가
- `add-equation <directory> --script <text>`: 수식 객체 문단 추가
- `add-memo <directory> --anchor-text <text> --text <text>`: 메모가 달린 문단 추가
- `add-line <directory> --width-mm <n> --height-mm <n>`: 기본 선 도형 추가
- `add-ellipse <directory> --width-mm <n> --height-mm <n>`: 기본 타원 도형 추가
- `add-rectangle <directory> --width-mm <n> --height-mm <n>`: 기본 사각형 도형 추가
- `add-textbox <directory> --width-mm <n> --height-mm <n> --text <text>`: 기본 글상자 도형 추가
- `schema`: 명령/옵션/응답 계약을 기계적으로 조회

## 빌드

```bash
go build ./cmd/hwpxctl
./hwpxctl inspect --help
```

## 사용 예시

```bash
go run ./cmd/hwpxctl inspect ./path/to/file.hwpx
go run ./cmd/hwpxctl inspect ./path/to/file.hwpx --format json
go run ./cmd/hwpxctl validate ./path/to/file.hwpx
go run ./cmd/hwpxctl validate ./path/to/file.hwpx --format json
go run ./cmd/hwpxctl text ./path/to/file.hwpx --output ./out/file.txt
go run ./cmd/hwpxctl text ./path/to/file.hwpx --format json
go run ./cmd/hwpxctl unpack ./path/to/file.hwpx --output ./out/unpacked
go run ./cmd/hwpxctl unpack ./path/to/file.hwpx --output ./out/unpacked --format json
go run ./cmd/hwpxctl pack ./out/unpacked --output ./out/rebuilt.hwpx
go run ./cmd/hwpxctl pack ./out/unpacked --output ./out/rebuilt.hwpx --format json
go run ./cmd/hwpxctl create --output ./out/new-doc
go run ./cmd/hwpxctl append-text ./out/new-doc --text $'첫 문단\n둘째 문단'
go run ./cmd/hwpxctl set-paragraph-text ./out/new-doc --paragraph 1 --text "수정된 둘째 문단"
go run ./cmd/hwpxctl set-text-style ./out/new-doc --paragraph 1 --bold true --underline true --text-color "#C00000"
go run ./cmd/hwpxctl delete-paragraph ./out/new-doc --paragraph 0
go run ./cmd/hwpxctl add-section ./out/new-doc
go run ./cmd/hwpxctl delete-section ./out/new-doc --section 1
go run ./cmd/hwpxctl add-table ./out/new-doc --cells "항목,내용;이름,홍길동"
go run ./cmd/hwpxctl add-nested-table ./out/new-doc --table 0 --row 1 --col 1 --cells "내부1,내부2;내부3,내부4"
go run ./cmd/hwpxctl set-table-cell ./out/new-doc --table 0 --row 1 --col 1 --text "김영희"
go run ./cmd/hwpxctl merge-table-cells ./out/new-doc --table 0 --start-row 0 --start-col 0 --end-row 1 --end-col 1
go run ./cmd/hwpxctl split-table-cell ./out/new-doc --table 0 --row 0 --col 0
go run ./cmd/hwpxctl embed-image ./out/new-doc --image ./assets/logo.png
go run ./cmd/hwpxctl set-header ./out/new-doc --text "문서 제목"
go run ./cmd/hwpxctl set-footer ./out/new-doc --text "기관명"
go run ./cmd/hwpxctl set-footer ./out/new-doc --text "- {{PAGE}} / {{TOTAL_PAGE}} -"
go run ./cmd/hwpxctl set-page-number ./out/new-doc --position BOTTOM_CENTER --type DIGIT --start-page 1
go run ./cmd/hwpxctl set-columns ./out/new-doc --count 2 --gap-mm 8
go run ./cmd/hwpxctl add-footnote ./out/new-doc --anchor-text "각주가 있는 본문" --text "각주 내용"
go run ./cmd/hwpxctl add-endnote ./out/new-doc --anchor-text "미주가 있는 본문" --text "미주 내용"
go run ./cmd/hwpxctl add-bookmark ./out/new-doc --name intro --text "소개 위치"
go run ./cmd/hwpxctl add-hyperlink ./out/new-doc --target "#intro" --text "소개로 이동"
go run ./cmd/hwpxctl add-hyperlink ./out/new-doc --target "https://example.com" --text "외부 링크"
go run ./cmd/hwpxctl add-heading ./out/new-doc --kind heading --level 1 --text "소개"
go run ./cmd/hwpxctl add-heading ./out/new-doc --kind outline --level 2 --text "세부 항목"
go run ./cmd/hwpxctl insert-toc ./out/new-doc --title "목차" --max-level 2
go run ./cmd/hwpxctl add-cross-reference ./out/new-doc --bookmark heading-2 --text "소개로 이동"
go run ./cmd/hwpxctl add-equation ./out/new-doc --script "a+b"
go run ./cmd/hwpxctl add-memo ./out/new-doc --anchor-text "검토가 필요한 문장" --text $'첫 번째 메모\n두 번째 메모' --author "홍길동"
go run ./cmd/hwpxctl add-line ./out/new-doc --width-mm 50 --height-mm 10 --line-color "#2F5597"
go run ./cmd/hwpxctl add-ellipse ./out/new-doc --width-mm 40 --height-mm 20 --fill-color "#FFF2CC"
go run ./cmd/hwpxctl add-rectangle ./out/new-doc --width-mm 40 --height-mm 20 --fill-color "#FFF2CC"
go run ./cmd/hwpxctl add-textbox ./out/new-doc --width-mm 60 --height-mm 25 --text $'글상자 첫 줄\n글상자 둘째 줄'
go run ./cmd/hwpxctl schema
```

## 편집 워크플로우

```bash
go run ./cmd/hwpxctl create --output ./work/report
go run ./cmd/hwpxctl append-text ./work/report --text $'제목\n본문'
go run ./cmd/hwpxctl set-paragraph-text ./work/report --paragraph 0 --text "수정된 제목"
go run ./cmd/hwpxctl set-text-style ./work/report --paragraph 1 --italic true --text-color "#2F5597"
go run ./cmd/hwpxctl add-section ./work/report
go run ./cmd/hwpxctl add-table ./work/report --cells "항목,값;상태,진행중"
go run ./cmd/hwpxctl add-nested-table ./work/report --table 0 --row 1 --col 1 --cells "내부1,내부2;내부3,내부4"
go run ./cmd/hwpxctl merge-table-cells ./work/report --table 0 --start-row 0 --start-col 0 --end-row 1 --end-col 1
go run ./cmd/hwpxctl set-table-cell ./work/report --table 0 --row 1 --col 1 --text "병합 셀"
go run ./cmd/hwpxctl split-table-cell ./work/report --table 0 --row 0 --col 0
go run ./cmd/hwpxctl embed-image ./work/report --image ./assets/logo.png
go run ./cmd/hwpxctl set-header ./work/report --text "보고서 제목"
go run ./cmd/hwpxctl set-footer ./work/report --text "부서명"
go run ./cmd/hwpxctl set-footer ./work/report --text "- {{PAGE}} / {{TOTAL_PAGE}} -"
go run ./cmd/hwpxctl set-page-number ./work/report --position BOTTOM_CENTER --type DIGIT --start-page 1
go run ./cmd/hwpxctl set-columns ./work/report --count 2 --gap-mm 8
go run ./cmd/hwpxctl add-footnote ./work/report --anchor-text "참고 문장" --text "각주 설명"
go run ./cmd/hwpxctl add-endnote ./work/report --anchor-text "보충 문장" --text "미주 설명"
go run ./cmd/hwpxctl add-bookmark ./work/report --name summary --text "요약 위치"
go run ./cmd/hwpxctl add-hyperlink ./work/report --target "#summary" --text "요약으로 이동"
go run ./cmd/hwpxctl add-hyperlink ./work/report --target "https://example.com" --text "외부 참고 링크"
go run ./cmd/hwpxctl add-heading ./work/report --kind heading --level 1 --text "소개"
go run ./cmd/hwpxctl add-heading ./work/report --kind outline --level 2 --text "세부 항목"
go run ./cmd/hwpxctl insert-toc ./work/report --title "목차" --max-level 2
go run ./cmd/hwpxctl add-cross-reference ./work/report --bookmark heading-2 --text "소개로 이동"
go run ./cmd/hwpxctl add-equation ./work/report --script "a+b"
go run ./cmd/hwpxctl add-memo ./work/report --anchor-text "검토가 필요한 문장" --text "메모 내용"
go run ./cmd/hwpxctl add-line ./work/report --width-mm 50 --height-mm 10 --line-color "#2F5597"
go run ./cmd/hwpxctl add-ellipse ./work/report --width-mm 40 --height-mm 20 --fill-color "#FFF2CC"
go run ./cmd/hwpxctl add-rectangle ./work/report --width-mm 40 --height-mm 20 --fill-color "#FFF2CC"
go run ./cmd/hwpxctl add-textbox ./work/report --width-mm 60 --height-mm 25 --text "검토용 글상자"
go run ./cmd/hwpxctl pack ./work/report --output ./out/report.hwpx
```

- `insert-image`는 현재 한컴 뷰어 인쇄 PDF 기준으로 본문 배치까지 확인했습니다.
- `set-paragraph-text`, `delete-paragraph`는 첫 `secPr` 문단을 제외한 본문 문단 기준 0-based 인덱스를 사용합니다.
- `set-text-style`은 `--run`을 생략하면 문단의 모든 direct `hp:run`에 적용하고, 지정하면 해당 run 하나만 갱신합니다.
- `set-text-style`은 대상 run의 기존 `charPr`를 복제한 뒤 `bold`, `italic`, `underline`, `textColor`만 바꿉니다.
- `add-section`, `delete-section`은 `Contents/content.hpf` manifest/spine과 `header.xml secCnt`를 함께 갱신합니다.
- `delete-section`은 남은 section 파일과 manifest id를 다시 `section0..N` 형태로 정렬합니다.
- 기존 편집 명령은 여전히 첫 section만 직접 수정합니다.
- `add-nested-table`, `set-table-cell`, `merge-table-cells`, `split-table-cell`은 병합 상태를 반영한 논리 좌표를 사용합니다.
- 셀 분할은 현재 병합 전 가려졌던 셀 텍스트를 복원하지 않고, 비어 있는 셀로 다시 활성화합니다.
- `set-header`와 `set-footer`는 `{{PAGE}}`, `{{TOTAL_PAGE}}` 토큰을 지원합니다.
- `set-page-number`는 현재 쪽 번호 위치와 시작 번호를 제어합니다.
- `set-columns`는 첫 section의 `hp:colPr`와 `hp:secPr/@spaceColumns`를 함께 갱신합니다.
- `add-footnote`, `add-endnote`는 본문 앵커 문단과 주석 본문을 함께 생성합니다.
- `add-bookmark`는 이름 충돌을 막고 책갈피 위치 문단을 추가합니다.
- `add-hyperlink`는 URL과 `#bookmark` 내부 링크를 생성합니다.
- `add-heading`은 예제 템플릿의 `Title`, `heading N`, `개요 N` 스타일을 재사용합니다.
- `insert-toc`는 제목/개요 문단을 스캔해 기본 차례를 문서 앞부분에 생성합니다.
- `add-cross-reference`는 책갈피를 기준으로 내부 참조 링크를 추가합니다.
- `add-equation`은 한글 수식 스크립트를 `hp:equation`으로 삽입합니다.
- `add-memo`는 `memoProperties`, `memogroup`, `MEMO field`를 함께 기록합니다.
- `add-line`은 한컴 뷰어 인쇄 PDF 기준으로 보이는 기본 선 도형을 추가합니다.
- `add-ellipse`는 한컴 뷰어 인쇄 PDF 기준으로 보이는 기본 타원 도형을 추가합니다.
- `add-rectangle`는 한컴 뷰어 인쇄 PDF 기준으로 보이는 기본 사각형 도형을 추가합니다.
- `add-textbox`는 `hp:rect` 내부에 `hp:drawText`와 `hp:subList`를 함께 기록해 글상자 텍스트를 넣습니다.

## 예제 기반 통합 테스트

```bash
python ./scripts/test_example_cli.py
```

- 예제 `.hwpx`를 `inspect`, `validate`, `text`, `unpack`, `pack` 순서로 검사합니다.
- 원본과 재패킹본을 각각 PDF로 변환하고 PNG로 렌더링합니다.
- 산출물은 `output/` 아래에 저장됩니다.

## 한컴 뷰어 렌더링 검증

최종 렌더링 검증은 구조 요약 PDF가 아니라 `Hancom Office HWP Viewer`의 실제 PDF 인쇄 결과를 기준으로 확인합니다.

```bash
python ./scripts/print_hwpx_via_viewer.py ./path/to/file.hwpx
```

- 기본 저장 경로는 `output/viewer-print-YYYYMMDD-HHMMSS/` 입니다.
- 저장 시트 자동화는 `/` 입력으로 경로 입력 시트를 연 뒤 절대경로를 넣고 `Enter`, 마지막에 파일명을 바꾸는 순서를 사용합니다.
- 스크립트는 PDF 생성 후 `pdfinfo`를 읽어 JSON으로 출력합니다.
- 검증이 끝나면 뷰어 종료를 시도하고, 남아 있으면 프로세스를 정리합니다.
- 새 기능을 추가할 때마다 해당 기능이 반영된 `.hwpx`로 이 스크립트를 실행해 PDF를 만들고, 결과 렌더링을 직접 확인하는 것을 기본 검증 방식으로 사용합니다.

## 설계 메모

- HWPX 구조 요약은 [docs/research-notes.md](/Users/zarathu/projects/project-hwpx-cli/docs/research-notes.md)에 정리했습니다.
- 핵심 기준 파일은 `Contents/content.hpf`이며 `manifest`와 `spine`을 통해 section 순서를 해석합니다.
- `scripts/test_example_cli.py`와 `scripts/hwpx_to_pdf.py`는 구조/요약 검증용입니다. 최종 렌더링 확인은 위의 한컴 뷰어 인쇄 검증을 사용합니다.
- AI 에이전트용 호출은 `--format json` 또는 `HWPXCTL_FORMAT=json`을 권장합니다.
- 내부 구조는 `internal/hwpx/core`, `internal/hwpx/shared`, `internal/hwpx/<domain>` 계층으로 나뉘며, 루트 `internal/hwpx`는 façade 역할만 유지합니다.
- CLI는 `cobra` 기반입니다. 진입/라우팅은 [internal/cli/cobra.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/cobra.go), 공통 옵션/에러/스키마는 [internal/cli/root.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/root.go), 명령 메타데이터와 `schema` 출력은 [internal/cli/root.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/root.go)의 `buildSchemaDoc()`가 담당합니다.
- 실제 명령 구현은 [internal/cli/package.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/package.go), [internal/cli/schema.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/schema.go), [internal/cli/paragraph.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/paragraph.go), [internal/cli/section.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/section.go), [internal/cli/table.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/table.go), [internal/cli/media.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/media.go), [internal/cli/layout.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/layout.go), [internal/cli/note.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/note.go), [internal/cli/reference.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/reference.go), [internal/cli/object.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/object.go) 로 나뉩니다.
- 새 명령을 추가할 때는 보통 `buildSchemaDoc()`에 메타데이터를 추가하고, `lookupCommandRunner()`에 핸들러를 연결한 뒤, 해당 도메인 파일에 구현을 넣으면 `cobra` help와 `schema` 출력이 함께 갱신됩니다.

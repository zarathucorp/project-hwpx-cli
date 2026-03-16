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
- `add-table <directory> --rows <n> --cols <n>`: 첫 section 끝에 표 추가
- `set-table-cell <directory> --table <n> --row <n> --col <n> --text <text>`: 표 셀 텍스트 수정
- `embed-image <directory> --image <file>`: 이미지 바이너리를 문서에 임베드
- `set-header <directory> --text <text>`: 첫 section에 머리말 설정
- `set-footer <directory> --text <text>`: 첫 section에 꼬리말 설정
- `set-page-number <directory>`: 첫 section에 쪽 번호 표시 설정
- `add-footnote <directory> --anchor-text <text> --text <text>`: 각주가 달린 문단 추가
- `add-endnote <directory> --anchor-text <text> --text <text>`: 미주가 달린 문단 추가
- `add-bookmark <directory> --name <name> --text <text>`: 책갈피 위치 문단 추가
- `add-hyperlink <directory> --target <url|#bookmark> --text <text>`: 하이퍼링크 문단 추가
- `add-heading <directory> --kind <title|heading|outline> --text <text>`: 제목/개요 문단 추가
- `insert-toc <directory>`: 제목/개요 문단 기준 기본 차례 생성
- `add-cross-reference <directory> --bookmark <name>`: 책갈피 기준 내부 참조 문단 추가
- `schema`: 명령/옵션/응답 계약을 기계적으로 조회

## 빌드

```bash
go build ./cmd/hwpxctl
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
go run ./cmd/hwpxctl add-table ./out/new-doc --cells "항목,내용;이름,홍길동"
go run ./cmd/hwpxctl set-table-cell ./out/new-doc --table 0 --row 1 --col 1 --text "김영희"
go run ./cmd/hwpxctl embed-image ./out/new-doc --image ./assets/logo.png
go run ./cmd/hwpxctl set-header ./out/new-doc --text "문서 제목"
go run ./cmd/hwpxctl set-footer ./out/new-doc --text "기관명"
go run ./cmd/hwpxctl set-footer ./out/new-doc --text "- {{PAGE}} / {{TOTAL_PAGE}} -"
go run ./cmd/hwpxctl set-page-number ./out/new-doc --position BOTTOM_CENTER --type DIGIT --start-page 1
go run ./cmd/hwpxctl add-footnote ./out/new-doc --anchor-text "각주가 있는 본문" --text "각주 내용"
go run ./cmd/hwpxctl add-endnote ./out/new-doc --anchor-text "미주가 있는 본문" --text "미주 내용"
go run ./cmd/hwpxctl add-bookmark ./out/new-doc --name intro --text "소개 위치"
go run ./cmd/hwpxctl add-hyperlink ./out/new-doc --target "#intro" --text "소개로 이동"
go run ./cmd/hwpxctl add-hyperlink ./out/new-doc --target "https://example.com" --text "외부 링크"
go run ./cmd/hwpxctl add-heading ./out/new-doc --kind heading --level 1 --text "소개"
go run ./cmd/hwpxctl add-heading ./out/new-doc --kind outline --level 2 --text "세부 항목"
go run ./cmd/hwpxctl insert-toc ./out/new-doc --title "목차" --max-level 2
go run ./cmd/hwpxctl add-cross-reference ./out/new-doc --bookmark heading-2 --text "소개로 이동"
go run ./cmd/hwpxctl schema
```

## 편집 워크플로우

```bash
go run ./cmd/hwpxctl create --output ./work/report
go run ./cmd/hwpxctl append-text ./work/report --text $'제목\n본문'
go run ./cmd/hwpxctl add-table ./work/report --cells "항목,값;상태,진행중"
go run ./cmd/hwpxctl embed-image ./work/report --image ./assets/logo.png
go run ./cmd/hwpxctl set-header ./work/report --text "보고서 제목"
go run ./cmd/hwpxctl set-footer ./work/report --text "부서명"
go run ./cmd/hwpxctl set-footer ./work/report --text "- {{PAGE}} / {{TOTAL_PAGE}} -"
go run ./cmd/hwpxctl set-page-number ./work/report --position BOTTOM_CENTER --type DIGIT --start-page 1
go run ./cmd/hwpxctl add-footnote ./work/report --anchor-text "참고 문장" --text "각주 설명"
go run ./cmd/hwpxctl add-endnote ./work/report --anchor-text "보충 문장" --text "미주 설명"
go run ./cmd/hwpxctl add-bookmark ./work/report --name summary --text "요약 위치"
go run ./cmd/hwpxctl add-hyperlink ./work/report --target "#summary" --text "요약으로 이동"
go run ./cmd/hwpxctl add-hyperlink ./work/report --target "https://example.com" --text "외부 참고 링크"
go run ./cmd/hwpxctl add-heading ./work/report --kind heading --level 1 --text "소개"
go run ./cmd/hwpxctl add-heading ./work/report --kind outline --level 2 --text "세부 항목"
go run ./cmd/hwpxctl insert-toc ./work/report --title "목차" --max-level 2
go run ./cmd/hwpxctl add-cross-reference ./work/report --bookmark heading-2 --text "소개로 이동"
go run ./cmd/hwpxctl pack ./work/report --output ./out/report.hwpx
```

- `insert-image`는 현재 한컴 뷰어 인쇄 PDF 기준으로 본문 배치까지 확인했습니다.
- `set-header`와 `set-footer`는 `{{PAGE}}`, `{{TOTAL_PAGE}}` 토큰을 지원합니다.
- `set-page-number`는 현재 쪽 번호 위치와 시작 번호를 제어합니다.
- `add-footnote`, `add-endnote`는 본문 앵커 문단과 주석 본문을 함께 생성합니다.
- `add-bookmark`는 이름 충돌을 막고 책갈피 위치 문단을 추가합니다.
- `add-hyperlink`는 URL과 `#bookmark` 내부 링크를 생성합니다.
- `add-heading`은 예제 템플릿의 `Title`, `heading N`, `개요 N` 스타일을 재사용합니다.
- `insert-toc`는 제목/개요 문단을 스캔해 기본 차례를 문서 앞부분에 생성합니다.
- `add-cross-reference`는 책갈피를 기준으로 내부 참조 링크를 추가합니다.

## 예제 기반 통합 테스트

```bash
python ./scripts/test_example_cli.py
```

- 예제 `.hwpx`를 `inspect`, `validate`, `text`, `unpack`, `pack` 순서로 검사합니다.
- 원본과 재패킹본을 각각 PDF로 변환하고 PNG로 렌더링합니다.
- 산출물은 `output/` 아래에 저장됩니다.

## 설계 메모

- HWPX 구조 요약은 [docs/research-notes.md](/Users/zarathu/projects/project-hwpx-cli/docs/research-notes.md)에 정리했습니다.
- 핵심 기준 파일은 `Contents/content.hpf`이며 `manifest`와 `spine`을 통해 section 순서를 해석합니다.
- 검증은 구조 중심입니다. 렌더링 정확도나 한컴 UI 호환성까지 보장하지는 않습니다.
- AI 에이전트용 호출은 `--format json` 또는 `HWPXCTL_FORMAT=json`을 권장합니다.

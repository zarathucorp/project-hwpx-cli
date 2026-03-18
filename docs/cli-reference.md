# CLI Reference

`hwpxctl`은 macOS/Linux 우선의 HWPX CLI입니다.

## 빠른 시작

```bash
go build ./cmd/hwpxctl
./hwpxctl --help
./hwpxctl inspect --help
./hwpxctl schema
```

- `--help`와 각 서브커맨드 help는 `cobra`가 생성합니다.
- 명령 목록/요약/예시는 `buildSchemaDoc()` 기반 메타데이터와 맞춰 유지합니다.

## 공통 규칙

- 모든 주요 명령은 `--format text|json`을 지원합니다.
- `--output`, `-o`는 출력 파일/디렉터리 경로 옵션입니다.
- 기본 포맷은 `text`이며 `HWPXCTL_FORMAT=json`으로 기본값을 바꿀 수 있습니다.
- `schema`는 기본적으로 JSON을 출력합니다.
- 모든 mutating 명령은 `--track-changes true`, `--change-author`, `--change-summary` 공통 옵션을 지원합니다.
- `--track-changes true`를 켜면 `Contents/history.xml`과 manifest `history` item을 만들고 `historyEntry`를 append합니다.
- 현재 변경 추적은 history-only 1차 구현이며, 본문에 보이는 삽입/삭제 표시는 하지 않습니다.
- `validate --format json`은 invalid여도 구조화된 JSON error envelope를 stdout으로 출력한 뒤 종료 코드 `1`을 반환합니다.
- 잘못된 인자, 알 수 없는 명령, 필수 입력 누락은 종료 코드 `1`입니다.

## 명령 요약

| Command | Input | Output | Success stdout | Failure behavior |
| --- | --- | --- | --- | --- |
| `inspect` | `.hwpx` 파일 | text 또는 JSON | 요약 text 또는 JSON envelope | 파싱 실패 시 stderr 또는 JSON error |
| `validate` | `.hwpx` 파일 또는 unpack 디렉터리 | text 또는 JSON | 요약 text 또는 JSON envelope | invalid면 종료 코드 `1` |
| `text` | `.hwpx` 파일 | plain text, 파일, 또는 JSON | 텍스트, 파일 저장, 또는 JSON envelope | invalid/입력 오류 시 종료 코드 `1` |
| `export-markdown` | `.hwpx` 파일 또는 unpack 디렉터리 | Markdown 또는 JSON | Markdown text, 파일 저장, 또는 JSON envelope | invalid/입력 오류 시 종료 코드 `1` |
| `export-html` | `.hwpx` 파일 또는 unpack 디렉터리 | HTML 또는 JSON | HTML text, 파일 저장, 또는 JSON envelope | invalid/입력 오류 시 종료 코드 `1` |
| `unpack` | `.hwpx` 파일 | 디렉터리 또는 JSON | `Unpacked to <dir>` 또는 JSON envelope | `--output` 없으면 종료 코드 `1` |
| `pack` | unpack 디렉터리 | `.hwpx` 파일 또는 JSON | `Packed to <file>` 또는 JSON envelope | invalid 디렉터리면 종료 코드 `1` |
| `create` | 없음 | unpack 디렉터리 또는 JSON | `Created editable document ...` 또는 JSON envelope | `--output` 없으면 종료 코드 `1` |
| `append-text` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | `--text` 없으면 종료 코드 `1` |
| `add-run-text` | unpack 디렉터리 | text 또는 JSON | run 추가 결과 또는 JSON envelope | `--paragraph`, `--text` 없거나 인덱스가 잘못되면 종료 코드 `1` |
| `set-run-text` | unpack 디렉터리 | text 또는 JSON | run 교체 결과 또는 JSON envelope | `--paragraph`, `--run`, `--text` 없거나 인덱스가 잘못되면 종료 코드 `1` |
| `set-paragraph-text` | unpack 디렉터리 | text 또는 JSON | 수정 결과 또는 JSON envelope | `--paragraph`, `--text` 없으면 종료 코드 `1` |
| `set-paragraph-layout` | unpack 디렉터리 | text 또는 JSON | 문단 서식 수정 결과 또는 JSON envelope | `--paragraph` 없거나 서식 옵션이 없으면 종료 코드 `1` |
| `set-paragraph-list` | unpack 디렉터리 | text 또는 JSON | 목록 서식 수정 결과 또는 JSON envelope | `--paragraph`, `--kind` 없거나 값이 잘못되면 종료 코드 `1` |
| `set-text-style` | unpack 디렉터리 | text 또는 JSON | 스타일 수정 결과 또는 JSON envelope | 스타일 옵션이 없거나 인덱스가 잘못되면 종료 코드 `1` |
| `find-runs-by-style` | unpack 디렉터리 | text 또는 JSON | 검색 결과 목록 또는 JSON envelope | 스타일 옵션이 없으면 종료 코드 `1` |
| `replace-runs-by-style` | unpack 디렉터리 | text 또는 JSON | 스타일 기반 치환 결과 또는 JSON envelope | `--text` 또는 스타일 옵션이 없으면 종료 코드 `1` |
| `find-objects` | unpack 디렉터리 | text 또는 JSON | 객체 검색 결과 목록 또는 JSON envelope | `--type`에 지원하지 않는 값이 있으면 종료 코드 `1` |
| `find-by-tag` | unpack 디렉터리 | text 또는 JSON | 태그 검색 결과 목록 또는 JSON envelope | `--tag` 없으면 종료 코드 `1` |
| `find-by-attr` | unpack 디렉터리 | text 또는 JSON | 속성 검색 결과 목록 또는 JSON envelope | `--attr` 없으면 종료 코드 `1` |
| `find-by-xpath` | unpack 디렉터리 | text 또는 JSON | XPath 검색 결과 목록 또는 JSON envelope | `--expr` 없거나 식이 잘못되면 종료 코드 `1` |
| `delete-paragraph` | unpack 디렉터리 | text 또는 JSON | 삭제 결과 또는 JSON envelope | `--paragraph` 없으면 종료 코드 `1` |
| `add-section` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | section 템플릿 생성 실패 시 종료 코드 `1` |
| `delete-section` | unpack 디렉터리 | text 또는 JSON | 삭제 결과 또는 JSON envelope | 마지막 section 삭제나 범위 오류 시 종료 코드 `1` |
| `add-table` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | 크기 정보가 없으면 종료 코드 `1` |
| `add-nested-table` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | 대상 셀 오류나 크기 정보가 없으면 종료 코드 `1` |
| `set-table-cell` | unpack 디렉터리 | text 또는 JSON | 수정 결과 또는 JSON envelope | 범위 오류 시 종료 코드 `1` |
| `merge-table-cells` | unpack 디렉터리 | text 또는 JSON | 병합 결과 또는 JSON envelope | 잘못된 범위나 비직사각형 병합이면 종료 코드 `1` |
| `normalize-table-borders` | unpack 디렉터리 | text 또는 JSON | 정규화 결과 또는 JSON envelope | 표 인덱스 범위 오류 시 종료 코드 `1` |
| `split-table-cell` | unpack 디렉터리 | text 또는 JSON | 분할 결과 또는 JSON envelope | 병합 anchor가 아니거나 범위 오류면 종료 코드 `1` |
| `embed-image` | unpack 디렉터리 | text 또는 JSON | 임베드 결과 또는 JSON envelope | `--image` 없거나 포맷 미지원이면 종료 코드 `1` |
| `insert-image` | unpack 디렉터리 | text 또는 JSON | 삽입 결과 또는 JSON envelope | `--image` 없거나 포맷 미지원이면 종료 코드 `1` |
| `set-object-position` | unpack 디렉터리 | text 또는 JSON | 위치 수정 결과 또는 JSON envelope | `--type`, `--index` 없거나 위치 옵션이 없으면 종료 코드 `1` |
| `set-header` | unpack 디렉터리 | text 또는 JSON | 설정 결과 또는 JSON envelope | `--text` 없으면 종료 코드 `1` |
| `set-footer` | unpack 디렉터리 | text 또는 JSON | 설정 결과 또는 JSON envelope | `--text` 없으면 종료 코드 `1` |
| `set-page-number` | unpack 디렉터리 | text 또는 JSON | 설정 결과 또는 JSON envelope | 잘못된 숫자 입력 시 종료 코드 `1` |
| `set-columns` | unpack 디렉터리 | text 또는 JSON | 설정 결과 또는 JSON envelope | `--count` 없거나 0 이하이면 종료 코드 `1` |
| `add-footnote` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | `--anchor-text`, `--text` 없으면 종료 코드 `1` |
| `add-endnote` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | `--anchor-text`, `--text` 없으면 종료 코드 `1` |
| `add-bookmark` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | `--name`, `--text` 없거나 이름 충돌 시 종료 코드 `1` |
| `add-hyperlink` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | `--target`, `--text` 없거나 내부 책갈피가 없으면 종료 코드 `1` |
| `add-heading` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | `--text` 없거나 스타일을 찾지 못하면 종료 코드 `1` |
| `insert-toc` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | 제목/개요 문단이 없으면 종료 코드 `1` |
| `add-cross-reference` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | `--bookmark` 없거나 대상 책갈피가 없으면 종료 코드 `1` |
| `add-equation` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | `--script` 없으면 종료 코드 `1` |
| `add-memo` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | `--anchor-text`, `--text` 없으면 종료 코드 `1` |
| `add-line` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | 크기 정보가 없으면 종료 코드 `1` |
| `add-ellipse` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | 크기 정보가 없으면 종료 코드 `1` |
| `add-rectangle` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | 크기 정보가 없으면 종료 코드 `1` |
| `add-textbox` | unpack 디렉터리 | text 또는 JSON | 추가 결과 또는 JSON envelope | 크기 정보나 `--text`가 없으면 종료 코드 `1` |
| `schema` | 없음 | JSON 또는 text | 명령 계약 문서 | 잘못된 인자 시 종료 코드 `1` |

## JSON envelope

`--format json`일 때는 다음 envelope를 사용합니다.

```json
{
  "schemaVersion": "hwpxctl/v1",
  "command": "inspect",
  "success": true,
  "data": {},
  "error": null
}
```

실패 시:

```json
{
  "schemaVersion": "hwpxctl/v1",
  "command": "validate",
  "success": false,
  "data": {
    "inputPath": "/abs/path/broken",
    "report": {
      "valid": false
    }
  },
  "error": {
    "code": "validation_failed",
    "message": "validation failed"
  }
}
```

## inspect

패키지 메타데이터와 구조 요약을 출력합니다.

```bash
./hwpxctl inspect ./path/to/file.hwpx
./hwpxctl inspect ./path/to/file.hwpx --format json
```

`report` 필드:

- `valid`: 구조상 치명적 오류 여부
- `errors`: 필수 파일 누락, 잘못된 spine 참조 등
- `warnings`: manifest 누락, section 수 불일치 등
- `summary.entries`: 패키지 내 파일 목록
- `summary.metadata`: title, creator 등 메타데이터
- `summary.version`: `appVersion`, `hwpxVersion`
- `summary.manifest`: manifest item 목록
- `summary.spine`: spine 순서
- `summary.sectionPaths`: 실제 section XML 경로
- `summary.binaryPaths`: `BinData/` 아래 첨부 리소스 경로

## validate

`.hwpx` 파일 또는 unpack 디렉터리의 구조를 검증합니다.

```bash
./hwpxctl validate ./path/to/file.hwpx
./hwpxctl validate ./out/unpacked
./hwpxctl validate ./path/to/file.hwpx --format json
```

검증 기준:

- 필수 엔트리 존재 여부
- `Contents/content.hpf` 파싱 가능 여부
- `version.xml`, `Contents/header.xml` 파싱 가능 여부
- manifest와 spine 참조 일관성
- section 경로 해석 가능 여부

자동화에서는 종료 코드와 `data.report.valid`를 함께 확인해야 합니다.

## text

`spine` 기준으로 section 텍스트를 추출합니다.

```bash
./hwpxctl text ./path/to/file.hwpx
./hwpxctl text ./path/to/file.hwpx --output ./out/file.txt
./hwpxctl text ./path/to/file.hwpx --format json
```

동작:

- `<p>` 단위로 문단을 모읍니다
- `<t>` 텍스트 노드만 추출합니다
- `lineBreak`는 줄바꿈으로 변환합니다
- `tab`은 탭 문자로 변환합니다

JSON 예시:

```json
{
  "schemaVersion": "hwpxctl/v1",
  "command": "text",
  "success": true,
  "data": {
    "inputPath": "/abs/path/file.hwpx",
    "text": "Hello HWPX\nSecond paragraph",
    "lineCount": 2,
    "characterCount": 27
  }
}
```

제약:

- 입력은 `.hwpx` 파일만 지원합니다
- invalid 패키지에서는 추출하지 않습니다
- 스타일, 표, 주석, 레이아웃 정보는 보존하지 않습니다

## export-markdown

문단과 표를 중심으로 Markdown으로 내보냅니다.

```bash
./hwpxctl export-markdown ./path/to/file.hwpx
./hwpxctl export-markdown ./path/to/file.hwpx --output ./out/file.md
./hwpxctl export-markdown ./work/unpacked --format json
```

동작:

- `.hwpx` 파일과 unpack 디렉터리 둘 다 지원합니다
- section 순서대로 문단과 표를 읽습니다
- `Title`, `heading N`, `개요 N` 스타일은 Markdown heading으로 변환합니다
- 표는 첫 행을 header로 사용합니다
- 이미지/수식/도형은 placeholder 또는 텍스트 중심으로 내보냅니다

제약:

- 현재는 문단/표 중심 1차 구현입니다
- 각주/미주, 링크, 변경 추적의 시각 표현은 별도 변환하지 않습니다
- 복잡한 병합 표는 셀 span을 보존하지 않고 평탄화합니다

## export-html

문단과 표를 중심으로 HTML로 내보냅니다.

```bash
./hwpxctl export-html ./path/to/file.hwpx
./hwpxctl export-html ./path/to/file.hwpx --output ./out/file.html
./hwpxctl export-html ./work/unpacked --format json
```

동작:

- `.hwpx` 파일과 unpack 디렉터리 둘 다 지원합니다
- section 순서대로 문단과 표를 읽습니다
- `Title`, `heading N`, `개요 N` 스타일은 `h1`~`h6`으로 변환합니다
- 기본 inline CSS를 포함한 단일 HTML 문서를 생성합니다
- 표는 첫 행을 header row로 렌더링합니다

제약:

- 현재는 문단/표 중심 1차 구현입니다
- 이미지 바이너리를 실제 `<img>`로 풀지 않고 placeholder text로 남깁니다
- 복잡한 병합 표는 셀 span을 보존하지 않고 평탄화합니다

## unpack

`.hwpx`를 편집 가능한 디렉터리로 풉니다.

```bash
./hwpxctl unpack ./path/to/file.hwpx --output ./out/unpacked
./hwpxctl unpack ./path/to/file.hwpx --output ./out/unpacked --format json
```

JSON 성공 시에는 unpack 결과 디렉터리의 검증 보고서를 함께 돌려줍니다.

## pack

unpack된 디렉터리를 검증 후 `.hwpx`로 다시 묶습니다.

```bash
./hwpxctl pack ./out/unpacked --output ./out/rebuilt.hwpx
./hwpxctl pack ./out/unpacked --output ./out/rebuilt.hwpx --format json
```

JSON 성공 시에는 생성된 `.hwpx`에 대한 검증 보고서를 함께 돌려줍니다.

pack 전제 조건:

- `mimetype`는 저장(store) 방식으로 기록됩니다
- 나머지 파일은 일반 ZIP 압축으로 기록됩니다
- 입력 디렉터리가 invalid면 패키징을 중단합니다

## create

편집 가능한 새 HWPX 작업 디렉터리를 만듭니다.

```bash
./hwpxctl create --output ./work/new-doc
./hwpxctl create --output ./work/new-doc --format json
```

생성 결과:

- `mimetype`, `version.xml`, `settings.xml`
- `META-INF/container.xml`
- `Contents/content.hpf`, `Contents/header.xml`, `Contents/section0.xml`

제약:

- 출력은 `.hwpx` 파일이 아니라 unpack 디렉터리입니다
- 최종 파일 생성은 `pack`으로 마무리해야 합니다

## append-text

첫 번째 section 끝에 문단을 추가합니다.

```bash
./hwpxctl append-text ./work/new-doc --text "한 문단"
./hwpxctl append-text ./work/new-doc --text $'첫 문단\n둘째 문단' --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 줄바꿈이 있으면 문단 여러 개로 추가합니다
- 현재는 첫 번째 section만 편집합니다

## add-run-text

첫 번째 section의 본문 문단에 direct text run 하나를 추가합니다.

```bash
./hwpxctl add-run-text ./work/new-doc --paragraph 1 --text " (검토본)"
./hwpxctl add-run-text ./work/new-doc --paragraph 1 --run 0 --text "[머리] " --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `paragraph`는 첫 `secPr` 문단을 제외한 본문 문단 기준 0-based 인덱스입니다
- `run`을 생략하면 문단의 마지막 direct `hp:run` 뒤에 새 run을 붙입니다
- `run`을 지정하면 해당 인덱스 앞에 새 run을 삽입합니다
- 새 run의 `charPrIDRef`는 인접 run을 기준으로 상속합니다

## set-run-text

첫 번째 section의 본문 문단에서 direct text run 하나를 교체합니다.

```bash
./hwpxctl set-run-text ./work/new-doc --paragraph 1 --run 1 --text " (최종본)"
./hwpxctl set-run-text ./work/new-doc --paragraph 0 --run 0 --text "[검토]" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `paragraph`는 첫 `secPr` 문단을 제외한 본문 문단 기준 0-based 인덱스입니다
- `run`은 문단 내부 direct `hp:run` 기준 0-based 인덱스입니다
- 대상 run 내부 자식은 제거하고 `hp:t` 하나로 다시 기록합니다
- 응답에는 교체 전 텍스트를 함께 반환합니다

## set-paragraph-text

첫 번째 section의 본문 문단 텍스트를 교체합니다.

```bash
./hwpxctl set-paragraph-text ./work/new-doc --paragraph 1 --text "수정된 문단"
./hwpxctl set-paragraph-text ./work/new-doc --paragraph 1 --text "수정된 문단" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `paragraph`는 첫 `secPr` 문단을 제외한 본문 문단 기준 0-based 인덱스입니다
- 대상 문단의 문단 속성은 유지하고, 내부 내용은 단순 텍스트 run 하나로 교체합니다

## set-paragraph-layout

첫 번째 section의 본문 문단 정렬, 들여쓰기, 여백, 간격을 수정합니다.

```bash
./hwpxctl set-paragraph-layout ./work/new-doc --paragraph 1 --align CENTER --space-after-mm 4
./hwpxctl set-paragraph-layout ./work/new-doc --paragraph 2 --indent-mm 4 --left-margin-mm 8 --line-spacing-percent 180 --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `paragraph`는 첫 `secPr` 문단을 제외한 본문 문단 기준 0-based 인덱스입니다
- 현재 지원 옵션은 `align`, `indent-mm`, `left-margin-mm`, `right-margin-mm`, `space-before-mm`, `space-after-mm`, `line-spacing-percent`입니다
- 대상 문단의 현재 `paraPr`를 복제한 뒤 필요한 값만 바꾸고, 문단의 `paraPrIDRef`를 새 정의로 교체합니다
- `align`은 `LEFT`, `CENTER`, `RIGHT`, `JUSTIFY`, `DISTRIBUTE`를 지원합니다

제약:

- 현재는 첫 번째 section의 editable paragraph만 지원합니다
- 목록, 글머리표, 번호 매기기 자체는 `set-paragraph-list`로 설정합니다

## set-paragraph-list

첫 번째 section의 본문 문단에 글머리표 또는 번호 매기기를 적용합니다.

```bash
./hwpxctl set-paragraph-list ./work/new-doc --paragraph 1 --kind bullet
./hwpxctl set-paragraph-list ./work/new-doc --paragraph 2 --kind number --level 1 --start-number 3 --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `paragraph`는 첫 `secPr` 문단을 제외한 본문 문단 기준 0-based 인덱스입니다
- `kind`는 `bullet`, `number`, `none`을 지원합니다
- `level`은 0-based 목록 레벨이며 기본값은 `0`입니다
- `start-number`를 주면 번호 매기기 정의를 복제해 시작 번호를 조정합니다

제약:

- `start-number`는 `kind=number`일 때만 의미가 있습니다
- 현재는 첫 번째 section의 editable paragraph만 지원합니다

## set-text-style

첫 번째 section의 본문 문단 run 스타일을 수정합니다.

```bash
./hwpxctl set-text-style ./work/new-doc --paragraph 1 --bold true --underline true
./hwpxctl set-text-style ./work/new-doc --paragraph 1 --run 0 --italic true --text-color "#C00000" --format json
./hwpxctl set-text-style ./work/new-doc --paragraph 1 --font-name "맑은 고딕" --font-size-pt 12 --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `paragraph`는 첫 `secPr` 문단을 제외한 본문 문단 기준 0-based 인덱스입니다
- `run`을 생략하면 문단의 direct `hp:run` 전체에 같은 스타일 변경을 적용합니다
- 현재 지원 옵션은 `bold`, `italic`, `underline`, `text-color`, `font-name`, `font-size-pt`입니다
- 각 대상 run의 기존 `charPr`를 복제해 필요한 속성만 바꾸므로, 지정하지 않은 속성은 그대로 유지합니다

## find-runs-by-style

첫 번째 section의 본문 문단에서 스타일 조건에 맞는 direct run을 검색합니다.

```bash
./hwpxctl find-runs-by-style ./work/new-doc --bold true
./hwpxctl find-runs-by-style ./work/new-doc --underline true --text-color "#C00000" --format json
./hwpxctl find-runs-by-style ./work/new-doc --font-name "맑은 고딕" --font-size-pt 12 --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 현재 지원 조건은 `bold`, `italic`, `underline`, `text-color`, `font-name`, `font-size-pt`입니다
- 결과는 `paragraph`, `run`, `text`, `charPrIdRef`, 현재 스타일 상태를 함께 반환합니다
- 현재는 첫 번째 section의 direct `hp:run`만 검색합니다

## replace-runs-by-style

첫 번째 section의 본문 문단에서 스타일 조건에 맞는 direct run 텍스트를 일괄 치환합니다.

```bash
./hwpxctl replace-runs-by-style ./work/new-doc --bold true --text "[강조]"
./hwpxctl replace-runs-by-style ./work/new-doc --underline true --text-color "#C00000" --text "*검토 메모*" --format json
./hwpxctl replace-runs-by-style ./work/new-doc --font-name "맑은 고딕" --font-size-pt 12 --text "[본문]" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `--text`와 최소 하나의 스타일 조건이 필요합니다
- 현재 지원 조건은 `bold`, `italic`, `underline`, `text-color`, `font-name`, `font-size-pt`입니다
- 결과는 치환된 `paragraph`, `run`, 이전 텍스트, 새 텍스트, `charPrIdRef`를 반환합니다
- 현재는 첫 번째 section의 direct `hp:run`만 치환합니다

## find-objects

첫 번째 section의 본문 direct run 아래에서 고수준 객체를 검색합니다.

```bash
./hwpxctl find-objects ./work/new-doc
./hwpxctl find-objects ./work/new-doc --type table,textbox --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `--type`은 `table,image,equation,rectangle,line,ellipse,textbox`를 comma-separated 형태로 받습니다
- 결과는 `type`, `paragraph`, `run`, `path`, `id`, `ref`, `text`를 반환합니다
- 표는 `rows`, `cols`도 함께 반환합니다
- 현재는 첫 번째 section의 direct `hp:run` 아래만 재귀적으로 스캔합니다

## find-by-tag

첫 번째 section의 본문 direct run 아래에서 XML 태그 이름으로 요소를 검색합니다.

```bash
./hwpxctl find-by-tag ./work/new-doc --tag hp:tbl
./hwpxctl find-by-tag ./work/new-doc --tag drawText --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `--tag`는 `hp:tbl` 또는 `tbl`처럼 prefix 유무와 관계없이 비교합니다
- 결과는 `tag`, `paragraph`, `run`, `path`, `text`를 반환합니다
- 현재는 첫 번째 section의 direct `hp:run` 아래만 재귀적으로 스캔합니다

## find-by-attr

첫 번째 section의 본문 direct run 아래에서 XML 속성 이름과 값으로 요소를 검색합니다.

```bash
./hwpxctl find-by-attr ./work/new-doc --attr id --tag tbl
./hwpxctl find-by-attr ./work/new-doc --attr editable --tag drawText --value 0 --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `--attr`는 `xml:id`와 `id`처럼 prefix 유무와 관계없이 비교합니다
- `--value`는 exact match입니다
- `--tag`를 주면 특정 태그로 범위를 좁힙니다
- 결과는 `tag`, `attr`, `value`, `paragraph`, `run`, `path`, `text`를 반환합니다
- 현재는 첫 번째 section의 direct `hp:run` 아래만 재귀적으로 스캔합니다

## find-by-xpath

첫 번째 section root에서 `etree`의 XPath-like 식으로 요소를 검색합니다.

```bash
./hwpxctl find-by-xpath ./work/new-doc --expr ".//hp:tbl[@id]"
./hwpxctl find-by-xpath ./work/new-doc --expr ".//hp:drawText[@editable='0']" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `--expr`는 `etree`의 XPath-like 문법을 사용합니다
- 결과는 `tag`, `paragraph`, `run`, `path`, `text`를 반환합니다
- `paragraph`, `run`은 매칭 요소가 속한 첫 section의 본문 direct run 기준 anchor 위치입니다
- anchor를 찾을 수 없는 요소는 `paragraph=-1`, `run=-1`로 반환합니다

## delete-paragraph

첫 번째 section의 본문 문단 하나를 삭제합니다.

```bash
./hwpxctl delete-paragraph ./work/new-doc --paragraph 0
./hwpxctl delete-paragraph ./work/new-doc --paragraph 0 --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `paragraph`는 첫 `secPr` 문단을 제외한 본문 문단 기준 0-based 인덱스입니다
- 삭제 결과 JSON에는 제거된 기존 텍스트를 함께 반환합니다

## add-section

문서 끝에 빈 section 하나를 추가합니다.

```bash
./hwpxctl add-section ./work/new-doc
./hwpxctl add-section ./work/new-doc --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 현재 spine 마지막 section의 section property를 기준으로 새 `Contents/sectionN.xml`을 생성합니다
- `Contents/content.hpf` manifest/spine과 `Contents/header.xml`의 `secCnt`를 함께 갱신합니다

제약:

- 기존 편집 명령은 여전히 첫 번째 section만 직접 수정합니다

## delete-section

spine 순서 기준으로 section 하나를 삭제합니다.

```bash
./hwpxctl delete-section ./work/new-doc --section 1
./hwpxctl delete-section ./work/new-doc --section 1 --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `section`은 `spine` 기준 0-based 인덱스입니다
- 대상 section의 manifest item, spine itemref, section 파일, `header.xml secCnt`를 함께 갱신합니다
- 삭제 후 남은 section 파일과 manifest id는 다시 `section0..N` 형태로 정렬합니다

제약:

- 마지막 남은 section은 삭제할 수 없습니다

## add-table

첫 번째 section 끝에 표를 추가합니다.

```bash
./hwpxctl add-table ./work/new-doc --rows 2 --cols 3
./hwpxctl add-table ./work/new-doc --cells "항목,내용;이름,홍길동" --format json
./hwpxctl add-table ./work/new-doc --rows 3 --cols 3 --width-mm 140 --col-widths-mm 30,50,60 --row-heights-mm 10,12,14 --margin-top-mm 4 --format json
```

동작:

- `--rows`, `--cols`를 직접 주거나 `--cells`에서 자동 추론할 수 있습니다
- `--cells` 형식은 `행1열1,행1열2;행2열1,행2열2` 입니다
- `--width-mm`, `--height-mm`로 표 전체 크기를 지정할 수 있습니다
- `--col-widths-mm`, `--row-heights-mm`로 열 너비와 행 높이를 직접 지정할 수 있습니다
- `--margin-left-mm`, `--margin-right-mm`, `--margin-top-mm`, `--margin-bottom-mm`로 표 바깥 여백을 지정할 수 있습니다
- `--col-widths-mm` 값 개수는 열 개수와, `--row-heights-mm` 값 개수는 행 개수와 같아야 합니다

## add-nested-table

기존 표 셀 안에 중첩 표를 추가합니다.

```bash
./hwpxctl add-nested-table ./work/new-doc --table 0 --row 1 --col 1 --cells "내부1,내부2;내부3,내부4"
./hwpxctl add-nested-table ./work/new-doc --table 0 --row 1 --col 1 --rows 2 --cols 2 --format json
./hwpxctl add-nested-table ./work/new-doc --table 0 --row 1 --col 1 --rows 2 --cols 2 --col-widths-mm 25,35 --row-heights-mm 8,10 --format json
```

동작:

- `table`, `row`, `col`은 부모 표 셀의 0-based 논리 좌표입니다
- `--rows`, `--cols`를 직접 주거나 `--cells`에서 자동 추론할 수 있습니다
- 대상 셀의 `hp:subList` 안에 중첩 `hp:tbl`을 추가합니다
- 셀이 비어 있으면 기본 빈 문단을 제거한 뒤 중첩 표만 넣습니다
- `--width-mm`, `--height-mm`, `--col-widths-mm`, `--row-heights-mm`로 중첩 표 geometry를 지정할 수 있습니다
- `--margin-left-mm`, `--margin-right-mm`, `--margin-top-mm`, `--margin-bottom-mm`로 중첩 표 바깥 여백을 지정할 수 있습니다

제약:

- 현재는 첫 번째 section 안의 표만 대상으로 합니다
- 폭을 따로 주지 않으면 부모 셀의 너비에 맞춘 기본 폭을 사용합니다
- 중첩 표 안의 border/fill 같은 고급 스타일은 아직 지원하지 않습니다

## set-table-cell

표 셀의 텍스트를 바꿉니다.

```bash
./hwpxctl set-table-cell ./work/new-doc --table 0 --row 1 --col 1 --text "김영희"
./hwpxctl set-table-cell ./work/new-doc --table 0 --row 1 --col 1 --text "김영희" --format json
./hwpxctl set-table-cell ./work/new-doc --table 0 --row 1 --col 1 --text "제목" --font-name "맑은 고딕" --font-size-pt 14 --format json
./hwpxctl set-table-cell ./work/new-doc --table 0 --row 0 --col 0 --border-style NONE --border-left-style SOLID --border-top-style SOLID --border-left-width-mm 0.4 --border-top-width-mm 0.4 --format json
```

동작:

- `table`, `row`, `col`은 모두 0-based 인덱스입니다
- 현재는 첫 번째 section 안의 표만 대상으로 합니다
- 병합된 표에서도 논리 좌표 기준으로 대상 셀을 찾습니다
- 셀 안 기존 문단은 새 텍스트 문단 하나로 교체합니다
- `--text`와 함께 `bold`, `italic`, `underline`, `text-color`, `font-name`, `font-size-pt`를 같이 줄 수 있습니다
- 셀 스타일만 바꿀 때는 `--text` 없이 `vert-align`, `margin-*`, `fill-color`, `background-color`를 사용할 수 있습니다
- border는 전체 공통 옵션 `border-style`, `border-color`, `border-width-mm`와 면별 override `border-left-*`, `border-right-*`, `border-top-*`, `border-bottom-*`를 함께 지원합니다
- 지원 style은 `NONE`, `SOLID`, `DASH`, `DOUBLE_SLIM`이며 `DOUBLE`은 `DOUBLE_SLIM` alias로 처리합니다
- 면별 옵션이 있으면 해당 면만 override하고, 나머지 면은 공통 border 값을 그대로 사용합니다

## merge-table-cells

직사각형 범위의 표 셀을 하나로 병합합니다.

```bash
./hwpxctl merge-table-cells ./work/new-doc --table 0 --start-row 0 --start-col 0 --end-row 1 --end-col 1
./hwpxctl merge-table-cells ./work/new-doc --table 0 --start-row 0 --start-col 0 --end-row 1 --end-col 1 --format json
```

동작:

- `table`, `start-row`, `start-col`, `end-row`, `end-col`은 모두 0-based 인덱스입니다
- 현재는 첫 번째 section 안의 표만 대상으로 합니다
- 대상 범위는 하나의 직사각형이어야 하며, 기존 병합이 겹치면 오류를 반환할 수 있습니다
- 병합 후 `set-table-cell`은 병합된 셀의 논리 좌표 어디를 지정해도 anchor 셀을 갱신합니다

제약:

- 병합 과정에서 anchor가 아닌 셀의 기존 텍스트는 유지하지 않습니다

## normalize-table-borders

인접 셀의 shared edge를 정규화해서 경계선이 끊겨 보이는 경우를 줄입니다.

```bash
./hwpxctl normalize-table-borders ./work/new-doc --table 0
./hwpxctl normalize-table-borders ./work/new-doc --table 0 --format json
```

동작:

- `table`은 0-based 인덱스입니다
- 현재는 첫 번째 section 안의 표만 대상으로 합니다
- 논리 그리드를 기준으로 좌우/상하 인접 셀의 경계선을 비교합니다
- 한쪽 경계선이 더 강하면 같은 shared edge 반대편에도 같은 선을 복제합니다
- 병합된 셀도 logical span 기준으로 같은 규칙을 적용합니다

제약:

- outer perimeter 전체를 새로 설계하지는 않고 shared edge 정렬에 집중합니다
- shared edge 양쪽에 의도적으로 다른 선을 줬더라도 더 강한 쪽으로 통일됩니다

## split-table-cell

병합된 표 셀을 원래 span 크기만큼 다시 나눕니다.

```bash
./hwpxctl split-table-cell ./work/new-doc --table 0 --row 0 --col 0
./hwpxctl split-table-cell ./work/new-doc --table 0 --row 0 --col 0 --format json
```

동작:

- `table`, `row`, `col`은 모두 0-based 인덱스입니다
- 병합된 셀 내부 어느 논리 좌표를 주더라도 anchor 셀을 찾아 분할합니다
- 분할 후 각 셀은 다시 개별 `set-table-cell` 대상으로 접근할 수 있습니다

제약:

- 현재는 병합 전에 가려졌던 셀 텍스트를 복원하지 않고 빈 셀로 다시 활성화합니다

## embed-image

이미지 바이너리를 문서 패키지에 등록합니다.

```bash
./hwpxctl embed-image ./work/new-doc --image ./assets/logo.png
./hwpxctl embed-image ./work/new-doc --image ./assets/logo.png --format json
```

동작:

- `BinData/` 아래에 파일을 복사합니다
- `Contents/content.hpf` manifest와 `Contents/header.xml` binDataList를 갱신합니다
- 지원 포맷: PNG, JPG/JPEG, GIF, BMP, SVG

제약:

- 현재는 패키지 임베드만 수행합니다
- 본문 배치가 필요하면 `insert-image`를 사용해야 합니다

## insert-image

이미지를 문서에 임베드하고 첫 번째 section 본문에 배치합니다.

```bash
./hwpxctl insert-image ./work/new-doc --image ./assets/logo.png
./hwpxctl insert-image ./work/new-doc --image ./assets/logo.png --width-mm 80 --format json
```

동작:

- `embed-image`를 먼저 수행한 뒤 본문에 `hp:pic`을 추가합니다
- `--width-mm`를 주면 렌더링 폭을 밀리미터 기준으로 맞춥니다
- 현재는 첫 번째 section 끝에 그림 문단을 하나 추가합니다

## set-object-position

첫 번째 section의 이미지나 도형 위치를 수정합니다.

```bash
./hwpxctl set-object-position ./work/new-doc --type image --index 0 --treat-as-char false --x-mm 10 --y-mm 6 --format json
./hwpxctl set-object-position ./work/new-doc --type textbox --index 0 --horz-align CENTER --vert-align TOP --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `type`은 `image`, `rectangle`, `line`, `ellipse`, `textbox`를 지원합니다
- `index`는 해당 타입 안의 0-based 인덱스입니다
- 현재 지원 옵션은 `treat-as-char`, `x-mm`, `y-mm`, `horz-align`, `vert-align`입니다
- 내부적으로 대상 객체의 `hp:pos`를 찾아 필요한 속성만 갱신합니다

제약:

- 현재는 첫 번째 section에서 direct run 아래에 있는 객체만 찾습니다
- 정렬은 `horz-align=LEFT|CENTER|RIGHT`, `vert-align=TOP|CENTER|BOTTOM`만 지원합니다
- wrap, anchor 기준, z-order 같은 고급 배치 옵션은 아직 노출하지 않습니다

## set-header

첫 번째 section에 머리말 텍스트를 설정합니다.

```bash
./hwpxctl set-header ./work/new-doc --text "문서 제목"
./hwpxctl set-header ./work/new-doc --text "문서 제목 {{PAGE}} / {{TOTAL_PAGE}}" --format json
./hwpxctl set-header ./work/new-doc --text $'문서 제목\n부제목' --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 첫 번째 section의 control run에 `hp:header`를 추가하거나 교체합니다
- `--apply-page-type`으로 `BOTH`, `EVEN`, `ODD`를 지정할 수 있습니다
- `{{PAGE}}`, `{{TOTAL_PAGE}}` 토큰을 인라인 번호 control로 변환합니다

## set-footer

첫 번째 section에 꼬리말 텍스트를 설정합니다.

```bash
./hwpxctl set-footer ./work/new-doc --text "기관명"
./hwpxctl set-footer ./work/new-doc --text "- {{PAGE}} / {{TOTAL_PAGE}} -" --format json
./hwpxctl set-footer ./work/new-doc --text "기관명" --apply-page-type BOTH --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 첫 번째 section의 control run에 `hp:footer`를 추가하거나 교체합니다
- `--apply-page-type`으로 `BOTH`, `EVEN`, `ODD`를 지정할 수 있습니다
- `{{PAGE}}`, `{{TOTAL_PAGE}}` 토큰을 인라인 번호 control로 변환합니다

## add-bookmark

책갈피 위치 문단을 첫 번째 section 끝에 추가합니다.

```bash
./hwpxctl add-bookmark ./work/new-doc --name intro --text "소개 위치"
./hwpxctl add-bookmark ./work/new-doc --name intro --text "소개 위치" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 책갈피 marker와 표시 텍스트 문단을 함께 추가합니다
- 같은 이름의 책갈피가 이미 있으면 실패합니다

## add-hyperlink

하이퍼링크 문단을 첫 번째 section 끝에 추가합니다.

```bash
./hwpxctl add-hyperlink ./work/new-doc --target "#intro" --text "소개로 이동"
./hwpxctl add-hyperlink ./work/new-doc --target "https://example.com" --text "외부 링크" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- URL 또는 `#bookmark` 내부 링크를 지원합니다
- 내부 링크는 대상 책갈피가 없으면 실패합니다
- 링크 필드는 `fieldBegin/fieldEnd`와 `Command` 파라미터를 함께 기록합니다

## add-heading

제목, 제목 스타일, 개요 스타일 문단을 첫 번째 section 끝에 추가합니다.

```bash
./hwpxctl add-heading ./work/new-doc --kind heading --level 1 --text "소개"
./hwpxctl add-heading ./work/new-doc --kind outline --level 2 --text "세부 항목" --format json
./hwpxctl add-heading ./work/new-doc --kind title --text "문서 제목" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `--kind`는 `title`, `heading`, `outline`을 지원합니다
- `heading`은 `heading 1..9`, `outline`은 `개요 1..7` 스타일을 사용합니다
- 책갈피는 자동 생성되며 `--bookmark`로 직접 지정할 수 있습니다

## insert-toc

제목/개요 문단을 바탕으로 기본 차례를 문서 앞부분에 생성합니다.

```bash
./hwpxctl insert-toc ./work/new-doc
./hwpxctl insert-toc ./work/new-doc --title "목차" --max-level 2 --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `heading N`, `개요 N` 스타일 문단을 스캔합니다
- 문서 앞부분에 `TOC Heading`, `toc N` 스타일 문단을 삽입합니다
- 각 차례 항목은 해당 책갈피로 이동하는 하이퍼링크를 포함합니다

제약:

- 현재는 기본 링크형 차례만 생성합니다
- 표/그림/수식 차례는 아직 지원하지 않습니다

## add-cross-reference

책갈피를 대상으로 하는 기본 내부 참조 문단을 추가합니다.

```bash
./hwpxctl add-cross-reference ./work/new-doc --bookmark heading-2
./hwpxctl add-cross-reference ./work/new-doc --bookmark heading-2 --text "소개로 이동" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 대상 책갈피가 없으면 실패합니다
- `--text`를 생략하면 대상 문단 텍스트를 참조 문구로 사용합니다
- 현재는 하이퍼링크 기반 내부 참조 형태로 생성합니다

## add-equation

한글 수식 스크립트를 가진 수식 객체 문단을 첫 번째 section 끝에 추가합니다.

```bash
./hwpxctl add-equation ./work/new-doc --script "a+b"
./hwpxctl add-equation ./work/new-doc --script "alpha over beta" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `hp:equation`과 `hp:script`를 함께 기록합니다
- 크기와 baseline은 `0`으로 저장해 한컴 뷰어가 다시 계산하도록 둡니다
- `text` 추출에서는 원본 수식 스크립트를 반환합니다

제약:

- 현재 입력은 한글 수식 스크립트 원문입니다
- LaTeX 변환이나 수식 편집기 DSL 변환은 아직 지원하지 않습니다

## add-memo

첫 번째 section 끝에 메모가 달린 문단을 추가합니다.

```bash
./hwpxctl add-memo ./work/new-doc --anchor-text "검토가 필요한 문장" --text "메모 내용"
./hwpxctl add-memo ./work/new-doc --anchor-text "검토가 필요한 문장" --text $'첫 번째 메모\n두 번째 메모' --author "홍길동" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- `Contents/header.xml`에 `hh:memoProperties`와 기본 `hh:memoPr`를 보장합니다
- `Contents/section0.xml`에 `hp:memogroup`, `hp:memo`, `MEMO field` 앵커 문단을 함께 생성합니다
- `--author`를 주면 `fieldBegin > parameters`에 기록합니다

제약:

- 현재는 기본 메모 모양(`memoShapeIDRef="0"`)만 지원합니다
- 한컴 뷰어 인쇄 PDF에는 메모 본문이 직접 출력되지 않을 수 있습니다

## add-rectangle

첫 번째 section 끝에 기본 사각형 도형 문단을 추가합니다.

```bash
./hwpxctl add-rectangle ./work/new-doc --width-mm 40 --height-mm 20
./hwpxctl add-rectangle ./work/new-doc --width-mm 40 --height-mm 20 --fill-color "#FFF2CC" --line-color "#333333" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 밀리미터 값을 HWPUNIT으로 변환해 `hp:rect`를 생성합니다
- 기본 선/채우기/그림 위치 정보를 함께 기록합니다
- 한컴 뷰어 인쇄 PDF 기준으로 렌더링되는 사각형을 삽입합니다

제약:

- 현재는 treat-as-char 기본 사각형만 지원합니다
- 도형 안 텍스트 편집과 고급 변형은 아직 지원하지 않습니다

## add-line

첫 번째 section 끝에 기본 선 도형 문단을 추가합니다.

```bash
./hwpxctl add-line ./work/new-doc --width-mm 50 --height-mm 10
./hwpxctl add-line ./work/new-doc --width-mm 50 --height-mm 10 --line-color "#2F5597" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 밀리미터 값을 HWPUNIT으로 변환해 `hp:line`을 생성합니다
- 시작점과 끝점을 `hc:startPt`, `hc:endPt`로 기록합니다
- 한컴 뷰어 인쇄 PDF 기준으로 렌더링되는 기본 선 도형을 삽입합니다

제약:

- 현재는 treat-as-char 기본 선만 지원합니다
- 선 끝 모양, 화살표, 점선 같은 고급 옵션은 아직 지원하지 않습니다

## add-ellipse

첫 번째 section 끝에 기본 타원 도형 문단을 추가합니다.

```bash
./hwpxctl add-ellipse ./work/new-doc --width-mm 40 --height-mm 20
./hwpxctl add-ellipse ./work/new-doc --width-mm 40 --height-mm 20 --fill-color "#FFF2CC" --line-color "#333333" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 밀리미터 값을 HWPUNIT으로 변환해 `hp:ellipse`를 생성합니다
- 중심점과 축 정보를 `hc:center`, `hc:ax1`, `hc:ax2` child element로 기록합니다
- 한컴 뷰어 인쇄 PDF 기준으로 렌더링되는 기본 타원 도형을 삽입합니다

제약:

- 현재는 treat-as-char 기본 타원만 지원합니다
- 호(arc) 변환, 회전, 고급 타원 편집은 아직 지원하지 않습니다

## add-textbox

첫 번째 section 끝에 기본 글상자 도형 문단을 추가합니다.

```bash
./hwpxctl add-textbox ./work/new-doc --width-mm 60 --height-mm 25 --text "글상자 본문"
./hwpxctl add-textbox ./work/new-doc --width-mm 60 --height-mm 25 --text $'첫 줄\n둘째 줄' --fill-color "#FFF2CC" --line-color "#333333" --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 밀리미터 값을 HWPUNIT으로 변환해 `hp:rect`를 생성합니다
- 도형 내부에 `hp:drawText`, `hp:textMargin`, `hp:subList`를 함께 기록합니다
- `--text` 줄바꿈은 글상자 내부의 여러 문단으로 저장합니다
- `text` 추출에서는 글상자 내부 문단 텍스트도 함께 반환합니다

제약:

- 현재는 treat-as-char 기본 글상자만 지원합니다
- 정렬, 자동 크기 조정, 회전 같은 고급 글상자 옵션은 아직 지원하지 않습니다

## set-columns

첫 번째 section의 다단 레이아웃을 설정합니다.

```bash
./hwpxctl set-columns ./work/new-doc --count 2
./hwpxctl set-columns ./work/new-doc --count 2 --gap-mm 8 --format json
```

동작:

- 첫 번째 section의 control run에서 `hp:colPr`를 추가하거나 교체합니다
- `--count`로 단 수를 설정합니다
- `--gap-mm`를 주면 `hp:colPr/@sameGap`와 `hp:secPr/@spaceColumns`를 함께 갱신합니다

제약:

- 현재는 첫 번째 section만 직접 수정합니다
- 균등 단 폭(`sameSz=1`)만 지원합니다
- 단 나누기 선, 단별 폭, 고급 지면 배치는 아직 지원하지 않습니다

## set-page-number

첫 번째 section의 쪽 번호 표시를 설정합니다.

```bash
./hwpxctl set-page-number ./work/new-doc --position BOTTOM_CENTER --type DIGIT
./hwpxctl set-page-number ./work/new-doc --position BOTTOM_CENTER --type DIGIT --side-char - --start-page 5 --format json
```

동작:

- 첫 번째 section의 control run에 `hp:pageNum`을 추가하거나 교체합니다
- `--start-page`를 주면 `hp:startNum/@page`를 갱신합니다
- 전체 쪽수 표시는 `set-header` 또는 `set-footer`에서 `{{TOTAL_PAGE}}` 토큰으로 조합합니다

## add-footnote

첫 번째 section 끝에 각주가 달린 문단을 추가합니다.

```bash
./hwpxctl add-footnote ./work/new-doc --anchor-text "각주가 있는 문장" --text "각주 설명"
./hwpxctl add-footnote ./work/new-doc --anchor-text "각주가 있는 문장" --text $'첫 줄\n둘째 줄' --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 본문 앵커 텍스트가 있는 새 문단을 추가합니다
- 같은 문단에 `hp:footNote` control과 주석 본문 `subList`를 함께 생성합니다
- `--text`의 개행은 각주 내부의 여러 문단으로 변환합니다

## add-endnote

첫 번째 section 끝에 미주가 달린 문단을 추가합니다.

```bash
./hwpxctl add-endnote ./work/new-doc --anchor-text "미주가 있는 문장" --text "미주 설명"
./hwpxctl add-endnote ./work/new-doc --anchor-text "미주가 있는 문장" --text $'첫 줄\n둘째 줄' --format json
```

동작:

- 입력은 unpack 디렉터리입니다
- 본문 앵커 텍스트가 있는 새 문단을 추가합니다
- 같은 문단에 `hp:endNote` control과 주석 본문 `subList`를 함께 생성합니다
- `--text`의 개행은 미주 내부의 여러 문단으로 변환합니다

## schema

AI 에이전트가 명령 계약을 런타임에 조회할 때 사용합니다.

```bash
./hwpxctl schema
./hwpxctl schema --format text
```

## 에러와 종료 코드

현재는 다음 종료 코드를 사용합니다.

| Exit code | Meaning |
| --- | --- |
| `0` | 성공 |
| `1` | 잘못된 인자, invalid 문서, 파싱/입출력 오류 |

## 추천 워크플로우

구조 확인:

```bash
./hwpxctl schema
./hwpxctl inspect ./file.hwpx --format json
```

변환 전 검증:

```bash
./hwpxctl validate ./file.hwpx --format json
```

텍스트 추출:

```bash
./hwpxctl text ./file.hwpx --format json
```

수정 후 재패키징:

```bash
./hwpxctl unpack ./file.hwpx --output ./work/file --format json
./hwpxctl validate ./work/file --format json
./hwpxctl append-text ./work/file --text "추가 문단" --format json
./hwpxctl add-table ./work/file --cells "항목,값;상태,진행중" --format json
./hwpxctl insert-image ./work/file --image ./assets/logo.png --format json
./hwpxctl set-header ./work/file --text "문서 제목" --format json
./hwpxctl set-footer ./work/file --text "기관명" --format json
./hwpxctl set-page-number ./work/file --position BOTTOM_CENTER --type DIGIT --start-page 1 --format json
./hwpxctl add-footnote ./work/file --anchor-text "각주가 있는 문장" --text "각주 설명" --format json
./hwpxctl add-endnote ./work/file --anchor-text "미주가 있는 문장" --text "미주 설명" --format json
./hwpxctl pack ./work/file --output ./out/file.hwpx --format json
```

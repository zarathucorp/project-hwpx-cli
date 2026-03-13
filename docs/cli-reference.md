# CLI Reference

`hwpxctl`은 macOS/Linux 우선의 HWPX CLI입니다.

## 빠른 시작

```bash
go build ./cmd/hwpxctl
./hwpxctl --help
./hwpxctl schema
```

## 공통 규칙

- 모든 주요 명령은 `--format text|json`을 지원합니다.
- `--output`, `-o`는 출력 파일/디렉터리 경로 옵션입니다.
- 기본 포맷은 `text`이며 `HWPXCTL_FORMAT=json`으로 기본값을 바꿀 수 있습니다.
- `schema`는 기본적으로 JSON을 출력합니다.
- `validate --format json`은 invalid여도 구조화된 JSON error envelope를 stdout으로 출력한 뒤 종료 코드 `1`을 반환합니다.
- 잘못된 인자, 알 수 없는 명령, 필수 입력 누락은 종료 코드 `1`입니다.

## 명령 요약

| Command | Input | Output | Success stdout | Failure behavior |
| --- | --- | --- | --- | --- |
| `inspect` | `.hwpx` 파일 | text 또는 JSON | 요약 text 또는 JSON envelope | 파싱 실패 시 stderr 또는 JSON error |
| `validate` | `.hwpx` 파일 또는 unpack 디렉터리 | text 또는 JSON | 요약 text 또는 JSON envelope | invalid면 종료 코드 `1` |
| `text` | `.hwpx` 파일 | plain text, 파일, 또는 JSON | 텍스트, 파일 저장, 또는 JSON envelope | invalid/입력 오류 시 종료 코드 `1` |
| `unpack` | `.hwpx` 파일 | 디렉터리 또는 JSON | `Unpacked to <dir>` 또는 JSON envelope | `--output` 없으면 종료 코드 `1` |
| `pack` | unpack 디렉터리 | `.hwpx` 파일 또는 JSON | `Packed to <file>` 또는 JSON envelope | invalid 디렉터리면 종료 코드 `1` |
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
./hwpxctl pack ./work/file --output ./out/file.hwpx --format json
```

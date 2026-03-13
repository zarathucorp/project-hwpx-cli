# hwpxctl

`hwpxctl`은 macOS/Linux 우선으로 설계한 HWPX CLI입니다. HWPX를 ZIP 기반 XML 패키지로 보고 구조를 점검하고, 텍스트를 추출하고, 압축 해제/재패킹할 수 있습니다.

문서 진입점:

- [docs/README.md](/Users/zarathu/projects/project-hwpx-cli/docs/README.md)
- [docs/cli-reference.md](/Users/zarathu/projects/project-hwpx-cli/docs/cli-reference.md)
- [docs/agent-guide.md](/Users/zarathu/projects/project-hwpx-cli/docs/agent-guide.md)

## 명령

- `inspect <file.hwpx>`: 메타데이터, manifest, spine, section 경로 조회
- `validate <file.hwpx|directory>`: 필수 파일과 manifest/spine 참조 일관성 검증
- `text <file.hwpx>`: `Contents/section*.xml` 기준 텍스트 추출
- `unpack <file.hwpx> --output <directory>`: 작업 디렉터리로 압축 해제
- `pack <directory> --output <file.hwpx>`: 검증된 디렉터리를 다시 `.hwpx`로 패키징
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
go run ./cmd/hwpxctl schema
```

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

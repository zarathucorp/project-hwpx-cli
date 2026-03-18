# Example Parity Harness

`example` 문서와 CLI로 새로 만든 example-like 문서를 같은 검증 루프로 비교하는 하네스입니다.

## 목적

- `create -> build -> pack -> Viewer print -> compare` 흐름을 한 번에 실행합니다.
- 이후 기능 브랜치가 parity 개선 전/후를 같은 산출물 구조로 비교할 수 있게 합니다.
- 구조 검증이 아니라 실제 `Hancom Office HWP Viewer` PDF 출력 결과까지 남깁니다.

## 실행

```bash
python ./scripts/test_example_parity.py --run-name baseline --keep-work
```

옵션:

- `--example`: 비교 기준이 되는 원본 `.hwpx`
- `--cli`: 사용할 `hwpxctl` 바이너리 경로
- `--output-root`: 산출물 루트 디렉터리
- `--run-name`: 결과 디렉터리 이름 고정
- `--keep-work`: unpack/work 디렉터리 유지
- `--verbose`: debug 로그 출력

## 출력 구조

기본 출력 경로는 `output/example-parity/<run-name>/` 입니다.

- `source/original-example.hwpx`
  - Viewer 자동화 안정화를 위해 원본을 ASCII 이름으로 복사한 파일입니다.
- `generated/example-like.hwpx`
  - 현재 CLI 기능만으로 만든 example-like 산출물입니다.
- `viewer/original/original-example.pdf`
- `viewer/generated/generated-example-like.pdf`
- `renders/original/*.png`
- `renders/generated/*.png`
- `compare/original.md`
- `compare/generated.md`
- `compare/original.txt`
- `compare/generated.txt`
- `build-run.json`
  - 생성 단계에서 실행한 CLI 명령 로그입니다.
- `report.json`
- `report.md`

## 재사용 방식

다른 parity 기능 브랜치는 아래 순서로 재사용하면 됩니다.

1. `scripts/test_example_parity.py`의 `build_example_steps()`를 현재 기능 수준에 맞게 확장합니다.
2. 같은 `--run-name` 규칙으로 baseline/new 결과를 따로 생성합니다.
3. `report.md`의 text ratio, table ratio, viewer page ratio와 PNG 렌더를 같이 봅니다.
4. 기능 merge 전에는 생성본 Viewer PDF가 실제로 열리고 저장되는지 다시 확인합니다.

## 현재 한계

- 현재 생성 시퀀스는 기존 feasibility 시도를 하네스화한 baseline입니다.
- 문서 레이아웃 parity는 아직 낮고, 이 하네스는 그 격차를 반복 측정하는 용도입니다.
- 원본 Viewer 인쇄는 긴 유니코드 파일명 대신 ASCII 별칭 복사본으로 수행합니다.

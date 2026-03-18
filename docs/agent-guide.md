# Agent Guide

이 문서는 AI 에이전트가 `hwpxctl`을 호출할 때 토큰 사용량을 줄이고, 실패를 명확하게 처리하고, 불필요하게 XML 전체를 읽지 않도록 돕기 위한 운영 가이드입니다.

## 왜 이 CLI가 에이전트에 유리한가

Justin Poehnelt의 ["Rewrite your CLI for AI agents"](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/)가 강조하는 핵심은 다음과 같습니다.

- 서브커맨드가 작고 예측 가능해야 한다
- 성공/실패 기준이 명확해야 한다
- 출력이 기계적으로 파싱 가능해야 한다
- 런타임에 계약을 조회할 수 있어야 한다
- 큰 원문 대신 축약된 표현을 우선 제공해야 한다

`hwpxctl`은 이 기준에 맞춰 다음을 제공합니다.

- 모든 주요 명령에 `--format json`
- `schema` 명령으로 런타임 introspection
- `validate`의 구조화된 실패 응답
- `text`의 경량 본문 추출

## 권장 호출 순서

문서 내용을 바로 읽지 말고 다음 순서를 기본값으로 사용합니다.

1. `schema`: 명령 계약과 옵션을 먼저 확인
2. `inspect`: 문서 구조와 메타데이터를 확인
3. `validate`: 자동화 전에 구조적 안전성 확인
4. `text`: 전체 XML 대신 요약 가능한 텍스트 추출
5. `unpack`: 실제 수정이 필요할 때만 사용
6. `pack`: 수정 후 재검증을 통과한 경우에만 사용

## 작업별 권장 패턴

### 1. 문서가 어떤 파일인지 빠르게 파악

```bash
hwpxctl schema
hwpxctl inspect ./file.hwpx --format json
```

이 명령으로 다음을 먼저 확인합니다.

- 제목/작성자 등 최소 메타데이터
- section 개수와 실제 section 경로
- 첨부 바이너리 존재 여부
- manifest와 spine 구조

### 2. 텍스트 분석이나 요약이 목적일 때

```bash
hwpxctl validate ./file.hwpx --format json
hwpxctl text ./file.hwpx --format json
```

이 흐름이 적합한 이유:

- invalid 패키지에서 애매한 파싱 결과를 줄입니다
- XML 전체를 모델 컨텍스트에 넣지 않아도 됩니다
- section 순서가 반영된 텍스트를 바로 후속 처리에 사용할 수 있습니다

### 3. 문서 내부를 수정해야 할 때

```bash
hwpxctl unpack ./file.hwpx --output ./work/file --format json
hwpxctl validate ./work/file --format json
hwpxctl pack ./work/file --output ./out/file.hwpx --format json
```

권장 이유:

- 수정 대상은 압축된 원본보다 디렉터리 상태가 다루기 쉽습니다
- `pack` 전에 같은 `validate` 계약을 재사용할 수 있습니다
- invalid 상태를 조기에 발견할 수 있습니다

## 자동화에서 반드시 지킬 점

- `validate --format json`은 invalid여도 JSON error envelope를 stdout으로 출력하므로 종료 코드와 `data.report.valid`를 함께 확인합니다
- `schema`를 먼저 읽으면 명령과 옵션을 추측하지 않아도 됩니다
- `inspect`와 `validate`의 JSON은 사람이 읽는 설명보다 신뢰도가 높은 1차 입력으로 사용합니다
- `text` 결과는 스타일 손실이 있으므로 레이아웃 복원 근거로 사용하면 안 됩니다
- `unpack` 결과를 수정할 때는 필수 파일과 spine 참조를 깨뜨리지 않도록 주의합니다

## 현재 한계

자동화 설계 시 아래 제약을 전제로 둬야 합니다.

- 종료 코드는 아직 `0/1`만 사용합니다
- `schemaVersion`은 제공하지만 JSON Schema 파일까지는 아직 없습니다
- 세밀한 편집 명령은 많이 늘었지만, low-level XML part 단위 수정은 아직 직접 노출하지 않습니다
- 렌더링 검증 명령이 없어서 구조 검증과 시각적 동일성은 별개입니다

## 개선 선택지

### 선택지 A. 계약 자산을 더 강하게 고정

- `schema` 결과를 별도 JSON Schema 파일로 배포
- 종료 코드 의미를 세분화
- 릴리스별 golden JSON 샘플을 추가

### 선택지 B. 더 작은 조회 명령 추가

- `summary` 같은 초경량 요약 명령 추가
- `sections` 같은 section 인덱스 조회 명령 추가
- `extract --section <id>` 같은 국소 추출 명령 추가

### 선택지 C. 렌더링 검증까지 확장

- PDF/PNG 비교 기반 시각 검증 명령 추가
- `output/` 산출물을 machine-readable manifest로 묶기

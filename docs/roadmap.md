# HWPX CLI Roadmap

## 한 줄 방향

`hwpxctl`를 low-level XML surgery 도구에서, 기존 복합 양식을 안전하게 채우고 새 문서를 구조적으로 조립할 수 있는 high-level HWPX editing tool로 전환한다.

## Now / Next / Later

### Now

Template-First 최소 제품을 안정화한다.  
핵심 흐름은 `analyze-template -> find-targets -> scaffold-template-contract -> payload -> fill-template -> roundtrip-check -> safe-pack -> Viewer PDF`다.

### Next

검증과 편집 해상도를 높인다.  
우선순위는 `preview-diff`, resolver 정리, analysis 상세화다.

### Later

Create-First를 본격화한다.  
`create-from-markdown`, `create-report`, compose primitives, create-first e2e를 순차로 붙인다.

## 전환 원칙

- 기존 `Track A -> Track B -> Track C -> Track D` 순서는 유지한다.
- 기존 low-level 명령과 `--mapping` 기반 `fill-template` 흐름은 유지한다.
- `analyze-template -> find-targets -> fill-template -> roundtrip-check` 축 위에 minimal template contract를 추가한다.
- 새 contract flow는 기존 planner와 applier를 재사용한다.
- 최종 완료 기준은 계속 Viewer PDF 인쇄 결과다.

## 문제 정의

현재 `hwpxctl`는 unpacked XML을 직접 수정하는 primitive는 충분히 갖추고 있다.  
하지만 실제 사용 목표는 단순 mutation이 아니라 다음에 가깝다.

- 원본 양식의 레이아웃과 구조를 유지한 채 내용 입력
- 파란 안내문과 placeholder만 안전하게 제거
- JSON, YAML, Markdown 내용을 문서 의미 단위로 주입
- 최종적으로 Viewer/HWP에서 정상 렌더링되는 결과 확보

따라서 앞으로의 로드맵은 "명령 개수 추가"가 아니라 "실제 문서 작업 흐름 전체를 안전하게 지원하는가"를 기준으로 관리한다.

## 제품 관점

### Template-First

기존 `.hwpx` 양식을 입력으로 받아 필요한 위치만 채우는 모드다.

대표 시나리오:

- 공공 사업계획서 양식 채우기
- 기업 제출용 제안서 양식 자동 입력
- 기존 사내 표준 문서의 안내문 제거 후 본문 입력

핵심 요구:

- 수정 위치를 사람이 빠르게 찾을 수 있어야 한다.
- 좌표가 아니라 anchor, label, placeholder 기준으로 수정해야 한다.
- 안내문 제거와 본문 치환이 레이아웃을 망가뜨리지 않아야 한다.

### Create-First

기존 양식 없이 새 문서를 생성하거나 최소 스캐폴드에서 시작하는 모드다.

대표 시나리오:

- Markdown에서 새 `.hwpx` 보고서 생성
- JSON 데이터로 표 중심 문서 생성
- 표지, 목차, 본문, 부록 구조를 가진 새 보고서 scaffold 생성

핵심 요구:

- 새 문서를 의미 단위로 조립할 수 있어야 한다.
- section, heading, list, table, TOC 같은 구조를 고수준 명령으로 만들 수 있어야 한다.
- 최종 산출물이 기본적으로 render-safe해야 한다.

## 핵심 원칙

### Render-Safe First

구조 valid보다 실제 렌더링 안정성을 우선한다.

### Analyze Before Edit

복합 양식은 먼저 분석하고 그 다음 수정한다.

### Compose At Meaning Level

새 문서는 paragraph/cell 좌표가 아니라 heading, section, block, table 같은 의미 단위로 작성한다.

### Anchor Over Coordinates

기존 양식 편집은 좌표보다 label, placeholder, 근접 텍스트 기반이어야 한다.

### Safe Mutation Over Destructive Mutation

삭제보다 치환, 숨김, 내용 비우기 같은 보수적 편집을 우선한다.

### End-to-End Verification

최종 완료 기준은 XML valid가 아니라 Viewer 인쇄 결과다.

### AI-Readable Precision

검증과 분석 결과는 요약보다 machine-readable strict data를 우선한다.  
AI가 후속 판단에 사용할 수 있도록 위치, before/after, selector, section/table/cell context를 잃지 않아야 한다.

## 제품 구조

### Track A. Shared Foundation

공용 기반 계층이다.

범위:

- unpack, pack, create
- low-level mutation
- atomic write
- lock and concurrency control
- schema and structure validation

기대 결과:

- 동일 unpacked 디렉터리에서 병렬 mutation 시 XML 파손이 발생하지 않는다.
- `valid=true`와 `render-safe=true`가 분리되어 표시된다.
- multi-section 문서에서 명령별 동작 차이가 줄어든다.

### Track B. Template-First Editing

기존 양식을 분석하고 안전하게 채우는 계층이다.

범위:

- template analysis
- human-friendly discovery
- placeholder and guide detection
- safe guide removal
- mapping and contract 기반 fill

기대 결과:

- 사용자가 XML을 직접 열지 않고도 수정 위치를 찾을 수 있다.
- section/table/cell 좌표를 직접 지정하지 않고 주요 필드를 채울 수 있다.
- guide 제거와 값 치환 후에도 layout risk가 관리 가능하다.

### Track C. Create-First Composition

새 문서를 처음부터 조립하는 계층이다.

범위:

- create-from-markdown
- create-from-json
- create-report
- create-table-form
- heading, list, table, section, TOC 같은 compose primitive

기대 결과:

- 빈 문서를 만든 뒤 low-level mutation을 반복하지 않아도 된다.
- 보고서형 문서와 표 중심 문서를 데이터만으로 생성할 수 있다.

### Track D. Verification

최종 품질 게이트다.

범위:

- preview diff
- roundtrip check
- render risk hint
- safe pack
- Viewer PDF smoke test
- PDF text compare

기대 결과:

- pack 전에 실질적 수정 내역을 검토할 수 있다.
- round-trip 품질 게이트가 생긴다.
- 실제 완료 판정이 Viewer 인쇄 결과까지 연결된다.

## 단계별 계획

### Phase 1. Foundation Stabilization

목표는 low-level mutation을 계속 쓰더라도 작업 디렉터리와 기본 검증이 흔들리지 않게 만드는 것이다.

핵심 산출물:

- unpacked directory lock
- atomic XML write
- internal working file ignore
- `validate.renderSafe`, `riskHints`, `riskSignals`
- multi-section 대응 기반 정리

종료 기준:

- 병렬 mutation에서 XML 파손이 없어야 한다.
- 사용자가 "왜 문서가 위험한지"를 `validate` 결과에서 이해할 수 있어야 한다.

### Phase 2. Template-First Minimum Product

목표는 복합 양식을 사람이 분석하고, contract 또는 mapping으로 안전하게 채우는 흐름을 완성하는 것이다.

핵심 산출물:

- `analyze-template`
- `find-targets`
- placeholder and guide detection
- `remove-guides`
- `fill-template --mapping`
- `fill-template --template --payload`
- contract scaffold
- payload skeleton

종료 기준:

- example 수준의 실제 `.hwpx` 양식에서 contract flow로 입력이 가능해야 한다.
- dry-run, apply, roundtrip, Viewer PDF까지 한 흐름으로 연결되어야 한다.

### Phase 3. Verification Hardening

목표는 "되긴 한다" 수준을 넘어, 수정 결과를 사람이 검토하고 안전하게 배포할 수 있게 하는 것이다.

핵심 산출물:

- `preview-diff`
- roundtrip strict diff
- `safe-pack` 정책 정리
- Viewer smoke harness 정리
- PDF text compare

종료 기준:

- pack 전후 결과를 비교 가능한 데이터와 시각 자료로 남길 수 있어야 한다.
- Viewer 기반 smoke가 표준 개발 플로우에 재사용 가능해야 한다.

### Phase 4. Create-First Composition

목표는 새 문서를 low-level mutation이 아니라 의미 단위 compose로 만들 수 있게 하는 것이다.

핵심 산출물:

- `create-from-markdown`
- `create-from-json`
- `create-report`
- `create-table-form`
- heading, list, table, section, TOC compose primitive
- 기본 layout and style preset

종료 기준:

- 보고서형 문서와 표 중심 문서를 데이터만으로 생성할 수 있어야 한다.
- create 직후부터 render-safe 기본값을 유지해야 한다.

### Phase 5. Coverage Expansion

목표는 실제 공공 양식, 기업 문서, multi-section, object-heavy 문서까지 커버리지를 넓히는 것이다.

핵심 산출물:

- object, layout, header, footer 계열 section-aware화
- section-aware regression test 확대
- 실제 공공 양식 fixture 추가
- create-first end-to-end fixture 추가

종료 기준:

- 특정 fixture 몇 개만 통과하는 수준이 아니라, 문서 유형별 회귀 검증 체계가 생겨야 한다.

## 권장 기본 워크플로우

### Workflow 1. 기존 양식 채우기

1. 원본 양식 복사
2. `unpack`
3. `analyze-template`
4. `find-targets`
5. 필요 시 template contract 작성 또는 scaffold 생성
6. `remove-guides --dry-run`
7. `fill-template --mapping` 또는 `fill-template --template --payload`
8. `preview-diff`
9. `roundtrip-check`
10. `safe-pack`
11. Viewer PDF 인쇄

### Workflow 2. 새 문서 만들기

1. `create` 또는 `create-from-markdown`
2. `compose-*` 또는 data-driven fill
3. `validate`
4. `preview-diff`
5. `safe-pack`
6. Viewer PDF 인쇄

## 완료 판정 기준

로드맵 완료는 "명령이 많아짐"이 아니라 아래 기준으로 판단한다.

- 기존 복합 양식에서 수정 대상을 사람이 빠르게 찾을 수 있다.
- guide text와 placeholder를 안전하게 제거 또는 치환할 수 있다.
- JSON, YAML, Markdown 데이터로 주요 필드를 채울 수 있다.
- 새 문서를 의미 단위로 조립할 수 있다.
- `validate`가 render risk를 설명할 수 있다.
- pack 후 round-trip 점검이 가능하다.
- Viewer 인쇄 결과에서 문서 흐름과 레이아웃이 유지된다.

## 운영 원칙

- 세부 진행 상태, 칸반, 최근 완료 항목은 [progress.md](./progress.md)에서 관리한다.
- `roadmap.md`는 상태판이 아니라 방향, 단계, 완료 기준을 설명하는 문서로 유지한다.

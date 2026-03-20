# HWPX CLI Roadmap

## 한 줄 방향

`hwpxctl`를 low-level XML surgery 도구에서, 기존 복합 양식을 안전하게 채우고 새 문서를 구조적으로 조립할 수 있는 high-level HWPX editing tool로 전환한다.

## 전환 원칙

이미 한 차례 개편한 로드맵 순서는 유지한다. 다만 template-first 축에 `contract layer`를 끼워 넣어, 기존 구현을 버리지 않고 다음 단계로 승격한다.

- 기존 `Track A -> Track B -> Track C -> Track D` 순서는 유지한다
- 기존 low-level 명령과 `--mapping` 기반 `fill-template` 흐름은 유지한다
- `analyze-template -> find-targets -> fill-template -> roundtrip-check` 축 위에 minimal template contract를 추가한다
- 새 contract flow는 기존 planner와 applier를 재사용한다
- 최종 완료 기준은 계속 Viewer PDF 인쇄 결과다

## 왜 로드맵을 다시 짜야 하는가

기존 로드맵은 "어떤 명령이 더 필요한가" 중심이었다. 실제 사용에서 드러난 문제는 다르다. 명령 개수가 부족한 것이 아니라, 현재 명령 집합만으로는 복합 `.hwpx` 문서를 안전하게 다루기 어렵다.

실제 대상 문서는 다음 요소가 한 파일 안에 섞여 있었다.

- 여러 section
- 병합 셀과 중첩 표
- 표 내부 문단과 스타일
- 목차와 페이지 흐름
- placeholder와 작성요령
- 표지, 요약서, 본문, 사업비 표처럼 구조가 다른 페이지

현재 `hwpxctl`는 unpacked XML을 수정하는 primitive는 제공한다. 하지만 실제 사용 목표였던 아래 작업을 직접 지원하는 추상화는 부족하다.

- 원본 양식의 레이아웃과 구조를 유지한 채 내용 입력
- 파란 안내문과 placeholder만 안전하게 제거
- Markdown/JSON 내용을 문서 의미 단위로 주입
- 최종적으로 Viewer/HWP에서 정상 렌더링되는 결과 확보

따라서 앞으로의 로드맵은 "low-level write API 확장"이 아니라 "실제 문서 작업 흐름 전체를 안전하게 지원하는가"를 기준으로 관리해야 한다.

## 제품이 지원해야 하는 두 가지 시작점

### 1. Template-First

기존 `.hwpx` 양식을 입력으로 받아 필요한 위치만 채우는 흐름이다.

예시:

- 공공 사업계획서 양식 채우기
- 기업 제출용 제안서 양식 자동 입력
- 기존 사내 표준 문서의 안내문 제거 후 본문 입력

이 모드의 핵심 요구는 다음과 같다.

- 어디를 수정해야 하는지 찾기 쉬워야 한다
- 좌표가 아니라 anchor, label, placeholder 기준으로 수정해야 한다
- 안내문 제거와 본문 치환이 레이아웃을 망가뜨리지 않아야 한다

### 2. Create-First

기존 양식 없이 새 문서를 생성하거나 최소 스캐폴드에서 시작하는 흐름이다.

예시:

- Markdown에서 새 `.hwpx` 보고서 생성
- JSON 데이터로 표 중심 문서 생성
- 표지, 목차, 본문, 부록 구조를 가진 새 보고서 scaffold 생성

이 모드의 핵심 요구는 다음과 같다.

- 새 문서를 의미 단위로 조립할 수 있어야 한다
- section, heading, list, table, TOC 같은 구조를 고수준 명령으로 만들 수 있어야 한다
- 최종 산출물이 기본적으로 render-safe해야 한다

## 현재 상태 진단

### 현재 강점

- `pack`, `unpack`, `inspect`, `validate`, `text`, `export-*` 기본 흐름 보유
- 문단, run, table cell, image, shape 등 일부 low-level mutation 가능
- multi-section 지원이 일부 명령에서 확장 중
- Viewer 인쇄 기반 검증 스크립트 보유

### 현재 한계

- `validate`는 구조 valid만 보며 렌더링 안정성을 설명하지 못한다
- target discovery 비용이 매우 크다
- low-level 좌표 기반 수정만으로는 양식 유지형 편집이 어렵다
- delete 계열 명령에 safe policy가 부족하다
- 일부 명령은 section 0 전제가 남아 있다
- 동일 unpacked 디렉터리에 대한 병렬 mutation 안전성이 부족했다
- 새 문서 생성은 가능해도, 실제 작성 workflow를 위한 compose 계층이 없다

## 핵심 제품 원칙

### 1. Render-Safe First

구조 valid보다 실제 렌더링 안정성을 우선한다.

### 2. Analyze Before Edit

복합 양식은 먼저 분석하고 그 다음 수정한다.

### 3. Compose At Meaning Level

새 문서는 paragraph/cell 좌표가 아니라 heading, section, block, table 같은 의미 단위로 작성한다.

### 4. Anchor Over Coordinates

기존 양식 편집은 좌표보다 label, placeholder, 근접 텍스트 기반이어야 한다.

### 5. Safe Mutation Over Destructive Mutation

삭제보다 치환, 숨김, 내용 비우기 같은 보수적 편집을 우선한다.

### 6. End-to-End Verification

최종 완료 기준은 XML valid가 아니라 Viewer 인쇄 결과다.

### 7. AI-Readable Precision

검증과 분석 결과는 요약보다 machine-readable strict data를 우선한다. AI가 후속 판단에 사용할 수 있도록 위치, before/after, selector, section/table/cell context를 잃지 않아야 한다.

## 제품 구조 재정의

앞으로 `hwpxctl`는 아래 4개 계층으로 본다.

### A. Foundation

- unpack/pack/create
- low-level mutation
- atomic write
- lock and concurrency control
- schema/structure validation

### B. Analysis And Discovery

- document risk analysis
- section/table/paragraph/cell map
- placeholder/guide/anchor candidate detection
- 사람이 읽기 쉬운 target discovery

### C. High-Level Editing

- guide removal
- anchor-based replacement
- template fill
- template contract resolution
- document composition
- safe pack policy

### D. Verification

- preview diff
- roundtrip check
- render risk hint
- Viewer PDF smoke test

## 로드맵 트랙

### Track A. Shared Foundation

두 시작점 모두가 의존하는 공용 기반이다.

#### A1. Reliability And Concurrency

- [x] unpacked 디렉터리 lock 도입
- [x] atomic XML write 적용
- [x] internal working file을 `pack`/`validate` 대상에서 제외
- [ ] lock wait policy 또는 명시적 unlock UX 개선
- [ ] mutation transaction log 또는 recovery 정보 추가

성공 기준:

- 동일 디렉터리 병렬 mutation 시 XML 파손이 발생하지 않음
- lock 충돌 시 원인과 조치 방법이 명확히 안내됨

#### A2. Validation Beyond XML

- [x] `validate`에 `renderSafe`, `riskHints`, `riskSignals` 추가
- [ ] `roundtrip-risk` 계산
- [ ] object/control risk 계산
- [ ] risk severity 등급화
- [ ] `safe-pack`와 연동되는 차단 정책 정의

성공 기준:

- `valid=true`와 `render-safe=true`가 분리되어 표시됨
- 사용자가 "왜 실제 문서가 깨질 수 있는지"를 결과에서 이해할 수 있음

#### A3. Multi-Section Consistency

- [~] paragraph/table mutation 계열 section 지원 확대
- [ ] object/header/footer/layout 계열 전면 section-aware화
- [ ] 전 명령 공통 section selector 모델 정리
- [ ] section-aware 응답 스키마 통일

성공 기준:

- multi-section 문서에서 명령별 동작 차이가 최소화됨

### Track B. Template-First Editing

기존 양식을 분석하고 안전하게 채우는 축이다.

#### B1. Template Analysis

- [x] `analyze-template` 최소 버전 추가
- [ ] section map 상세화
- [ ] table map 상세화
- [ ] merged cell map 추가
- [ ] paragraph/run candidate 출력
- [ ] page role 후보 추정
- [ ] TOC/control/object density 분석

권장 출력:

- section index, path, preview, paragraph/table count
- table index, row/col, merged span, current text preview
- paragraph index, style summary, text preview
- placeholder, guide, anchor candidate

성공 기준:

- 사용자가 XML을 직접 열지 않고도 수정 위치를 찾을 수 있음

#### B2. Human-Friendly Target Discovery

- [x] `find-targets` 최소 버전
- [x] `find-targets --anchor`
- [x] `find-targets --near-text`
- [x] `find-targets --table-label`
- [x] `find-targets --placeholder`
- [x] style/merge/section/context summary 최소 버전 추가

예시:

- `"과제명"` anchor 후보 찾기
- `"주관기관"` 주변 셀 찾기
- `"사업비 총괄표"` 제목과 연결된 table 찾기

성공 기준:

- 사람이 `"과제명"`이나 `"주관기관"` 같은 의미 단위로 수정 대상을 찾을 수 있음

#### B2.5. Minimal Template Contract Transition

- [ ] minimal contract schema 정의
- [ ] `template_id`, `template_version`, `fingerprint`, `fields`, `tables`, `strict` 최소 필드 정의
- [ ] `analyze-template` 출력에 contract authoring용 fingerprint 후보 추가
- [ ] 경량 fingerprint 검증 추가
- [ ] `fill-template --template --payload`를 기존 planner로 컴파일하는 경로 추가
- [x] 기존 `--mapping` 경로와 결과 스키마를 유지

성공 기준:

- 기존 `mapping` 기반 흐름을 깨지 않고 contract-first 진입이 가능함
- contract와 payload가 결국 동일한 patch planning 경로를 타서 dry-run/apply/검증 결과가 일관됨

#### B3. Placeholder And Guide Detection

- [x] placeholder candidate 최소 탐지
- [x] guide text candidate 최소 탐지
- [ ] 색상 기반 guide text 탐지
- [ ] 스타일 기반 guide text 탐지
- [ ] 빈칸형 필드와 label-value 구조 탐지
- [ ] false positive를 줄이기 위한 profile 지원

성공 기준:

- 파란 안내문, 작성요령, placeholder 후보를 실사용 가능한 수준으로 분리 표시

#### B4. Safe Editing For Templates

- [x] `remove-guides --dry-run` 최소 버전
- [ ] delete 대신 clear/hide/replace-empty 정책
- [ ] safe paragraph delete
- [ ] 기존 스타일 유지형 replace
- [ ] multi-paragraph cell replace
- [ ] merged cell 보존형 텍스트 업데이트

성공 기준:

- 작성요령 제거와 값 치환 후에도 layout risk가 관리 가능함

#### B5. High-Level Fill And Compatibility

- [x] `fill-template` 최소 버전
- [x] JSON/YAML mapping spec 최소 버전
- [ ] mapping spec 정규화와 문서화
- [ ] contract-to-plan resolver
- [ ] anchor-to-value resolver
- [ ] placeholder-to-value resolver
- [ ] Markdown block to field mapper
- [ ] repeated block fill 지원

예시:

- `"과제명" -> "Open Source AI Agent 플랫폼"`
- `"주관기관" -> "예시 주식회사"`
- `"참여기관1" -> "기관 A"`

성공 기준:

- 사용자가 section/table/cell 좌표를 직접 지정하지 않고 주요 필드를 채울 수 있음
- `--mapping`과 `--template --payload`가 같은 결과 모델로 수렴함

### Track C. Create-First Composition

새 문서를 처음부터 조립하는 축이다.

#### C1. Creation Entry Points

- [ ] `create` 결과 구조 점검 강화
- [ ] `create-from-markdown`
- [ ] `create-from-json`
- [ ] `create-report`
- [ ] `create-table-form`

성공 기준:

- 사용자가 빈 문서를 만든 뒤 low-level mutation을 반복하지 않아도 됨

#### C2. Composition Primitives

- [ ] heading/list/table/paragraph block composer
- [ ] section/page break composer
- [ ] TOC scaffold
- [ ] cover/summary/body/appendix scaffold
- [ ] style preset 또는 layout preset

성공 기준:

- 새 문서를 cell/paragraph 인덱스 없이 의미 단위로 조립 가능

#### C3. Data-Driven Compose

- [ ] Markdown AST 기반 compose
- [ ] JSON schema 기반 block compose
- [ ] 반복 표/반복 섹션 생성
- [ ] appendix/attachment block 생성

성공 기준:

- 보고서형 문서와 표 중심 문서를 데이터만으로 생성 가능

#### C4. Safe Defaults For New Documents

- [ ] 기본 section/page layout preset
- [ ] 기본 TOC/page-number 정책
- [ ] 새 문서 render-safe lint
- [ ] create 직후 validate/report 자동 출력

성공 기준:

- 새 문서가 생성 직후부터 위험한 기본 상태에 빠지지 않음

### Track D. Verification And QA

두 시작점 모두에 필요한 최종 품질 게이트다.

#### D1. Preview Diff

- [ ] `preview-diff`
- [ ] paragraph/table 수준 before/after diff
- [ ] guide 제거와 placeholder 치환 요약
- [ ] 변경된 section map 요약

성공 기준:

- 사용자가 최종 pack 전에 실질적 수정 내역을 검토 가능

#### D2. Roundtrip Check

- [x] `roundtrip-check` 최소 버전
- [ ] pack 후 재-unpack 또는 재-inspect 비교
- [ ] TOC/page/section 흐름 점검
- [ ] 본문 누락 여부 점검
- [ ] 위험 변경 요약
- [ ] strict machine-readable diff 제공

성공 기준:

- `validate` 외에 round-trip quality gate가 생김
- AI가 후속 수정 판단에 사용할 수 있는 exact location diff를 얻을 수 있음

#### D3. Safe Pack

- [x] `safe-pack` 최소 버전
- [ ] 위험도 높은 변경 감지 시 warning 또는 block
- [ ] `--force` 정책 분리
- [ ] 최종 pack report에 render risk 첨부

성공 기준:

- 위험한 결과물을 무심코 pack해서 넘기는 상황을 줄임

#### D4. Viewer-Based Verification Harness

- [ ] `python ./scripts/print_hwpx_via_viewer.py` 기반 smoke test 표준화
- [ ] PDF text compare 보조 도구
- [ ] PDF snapshot compare 보조 도구
- [ ] `output/` artifact naming 규칙 정리

성공 기준:

- 실제 완료 판정이 Viewer 인쇄 결과까지 연결됨

## 권장 CLI 구조

### Low-Level Commands

계속 유지한다.

- `set-paragraph-text`
- `set-run-text`
- `set-table-cell`
- `delete-paragraph`
- `append-text`
- 기타 직접 mutation 명령

용도:

- 디버깅
- 예외 상황 수정
- 내부 엔진 검증

### Template-First Commands

- `analyze-template`
- `find-targets`
- `remove-guides`
- `fill-template`
- 향후 `fill-template --template --payload`

용도:

- 기존 양식 분석
- anchor 기반 target discovery
- 안전한 guide 제거
- 템플릿 필드 입력

### Create-First Commands

- `create`
- `create-from-markdown`
- `create-from-json`
- `create-report`
- `compose-block`
- `compose-table`

용도:

- 새 문서 시작
- 구조 중심 문서 작성
- 보고서/표 문서 scaffold 생성

### Shared Verification Commands

- `validate`
- `preview-diff`
- `roundtrip-check`
- `safe-pack`

용도:

- 구조 검증
- 수정 결과 요약
- round-trip 품질 점검
- 최종 산출물 보호

## 권장 기본 워크플로우

### Workflow 1. 기존 양식 채우기

1. 원본 양식 복사
2. `unpack`
3. `analyze-template`
4. `find-targets`
5. 필요 시 template contract 작성 또는 갱신
6. `remove-guides --dry-run`
7. `fill-template --mapping` 또는 향후 `fill-template --template --payload`
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

## 우선순위 재정렬

### Immediate P0

공용 기반부터 안정화한다.

1. unpacked dir lock + atomic write
2. `validate` risk hint 확장
3. `analyze-template` 최소 버전

상태:

- 1 완료
- 2 최소 버전 완료
- 3 최소 버전 완료

### Immediate P1

template-first의 discovery 품질을 끌어올린다.

1. section/table/cell discovery 상세화
2. fingerprint 후보와 stable selector 정보 출력
3. guide text detector 고도화
4. placeholder detector 고도화
5. `find-targets` context summary 고도화

### Immediate P2

template-first의 safe edit와 fill을 붙이고 contract layer를 연결한다.

1. `remove-guides --dry-run` 고도화
2. minimal template contract schema
3. `fill-template` dual input `--mapping` / `--template --payload`
4. `roundtrip-check` 결과를 contract flow에도 연결

상태:

- 1 최소 버전 완료
- 2 최소 버전 완료
- 3 dual input 최소 버전 완료
- 4 apply 결과 옵션 경로 완료

### Immediate P3

create-first를 low-level 생성에서 high-level compose로 확장한다.

1. `create-from-markdown`
2. heading/list/table compose primitive
3. report/table-form scaffold
4. create-safe defaults

## GitHub Issue 단위 분해

### Foundation

- [x] unpacked 디렉터리 lock 구현
- [x] atomic XML write 적용
- [x] internal working file ignore 처리
- [ ] lock wait mode 추가
- [ ] unlock/help UX 개선

### Validation

- [x] `validate`에 `renderSafe` 필드 추가
- [x] `riskHints`/`riskSignals` 추가
- [x] `section-risk` 계산기 추가
- [x] `toc-risk` 계산기 추가
- [x] `table-risk` 계산기 추가
- [x] `layout-risk` 계산기 추가
- [ ] `roundtrip-risk` 계산기 추가
- [ ] object/control risk 계산기 추가

### Template Analysis

- [x] `analyze-template` 명령 추가
- [ ] section map schema 상세화
- [ ] table map schema 상세화
- [ ] merged cell map schema 정의
- [ ] anchor candidate detector 추가
- [ ] page role detector 추가

### Discovery UX

- [x] `find-targets --anchor` 추가
- [x] `find-targets --near-text` 추가
- [x] `find-targets --table-label` 추가
- [x] `find-targets --placeholder` 추가
- [x] discovery 출력에 style/merge/section summary 최소 버전 추가

### Safe Template Editing

- [x] `remove-guides` 최소 버전 추가
- [ ] delete 대신 clear/hide 정책 구현
- [ ] safe paragraph delete 전략 추가
- [ ] merged cell 보존형 텍스트 업데이트 개선
- [ ] cell 내부 multi-paragraph replace 개선

### Template Contract Transition

- [ ] minimal contract schema 추가
- [ ] fingerprint 후보 생성기 추가
- [ ] contract validator 추가
- [ ] contract-to-plan compiler 추가
- [ ] `fill-template --template --payload` 입력 경로 추가
- [x] `--mapping`과 contract flow 공통 report 스키마 최소 버전 정리
- [x] contract scaffold generator 최소 버전 추가
- [x] scaffold payload skeleton 최소 버전 추가

### High-Level Template Fill

- [x] `fill-template` 명령 추가
- [x] mapping JSON/YAML schema 최소 버전
- [ ] mapping schema 정규화
- [ ] anchor-to-value resolver 구현
- [ ] placeholder-to-value resolver 구현
- [ ] Markdown block mapper 구현
- [ ] repeated block fill 지원

### Create-First Composition

- [ ] `create-from-markdown` 명령 추가
- [ ] `create-from-json` 명령 추가
- [ ] `create-report` 명령 추가
- [ ] `create-table-form` 명령 추가
- [ ] heading/list/table composer 구현
- [ ] TOC scaffold 구현
- [ ] cover/summary/body/appendix scaffold 구현

### Verification

- [ ] `preview-diff` 명령 추가
- [x] `roundtrip-check` 명령 추가
- [x] `safe-pack` 명령 추가
- [ ] Viewer smoke test harness 정리
- [ ] PDF text/snapshot compare 보조 도구 추가

### Coverage

- [~] paragraph/table mutation 계열 multi-section selector 확대
- [ ] object/layout/header/footer 명령 multi-section 지원
- [ ] section-aware regression test 확대
- [ ] 실제 공공 양식 fixture 추가
- [ ] create-first end-to-end fixture 추가

## 완료 판정 기준

로드맵 완료는 "명령이 많아짐"이 아니라 아래 기준으로 판단한다.

- 기존 복합 양식에서 수정 대상을 사람이 빠르게 찾을 수 있음
- guide text와 placeholder를 안전하게 제거 또는 치환할 수 있음
- JSON/YAML/Markdown 데이터로 주요 필드를 채울 수 있음
- 새 문서를 의미 단위로 조립할 수 있음
- `validate`가 render risk를 설명할 수 있음
- pack 후 round-trip 점검이 가능함
- Viewer 인쇄 결과에서 문서 흐름과 레이아웃이 유지됨

## 결론

현재 `hwpxctl`는 low-level document surgery 도구로서는 의미가 있다. 하지만 실제 공공/기업 양식 문서를 "양식 유지 상태로 자동 입력"하거나, 새 문서를 "구조적으로 안전하게 생성"하는 수준의 도구가 되려면 low-level primitive 확장만으로는 부족하다.

앞으로의 로드맵은 아래 두 축을 동시에 가져가야 한다.

- 기존 양식을 분석하고 안전하게 채우는 template-first track
- 새 문서를 의미 단위로 조립하는 create-first track

그리고 이 두 축 아래에 반드시 공통으로 깔려야 하는 기반은 다음이다.

- 분석과 탐색 기능
- safe edit 정책
- render-safe / round-trip 검증
- concurrency safety

이 방향으로 전환해야 `validate` 통과와 실제 문서 완성도 사이의 간극을 줄일 수 있다.

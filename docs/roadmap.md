# HWPX CLI Roadmap

## 한 줄 방향

`hwpxctl`를 low-level XML surgery 도구에서, 실제 복합 HWPX 양식을 레이아웃 유지 상태로 안전하게 채우는 high-level template editing tool로 전환한다.

## 왜 전면 개편이 필요한가

기존 로드맵은 기능 추가 여부 중심이었다. 하지만 실제 사용에서 드러난 핵심 문제는 "기능이 없어서"가 아니라 "현재 기능 조합만으로는 복합 양식을 안전하게 편집하기 어렵다"는 점이다.

실제 사업계획서 양식 문서는 다음 특성을 가진다.

- 여러 section
- 병합 셀과 중첩 표
- 표 내부 문단과 스타일
- 목차와 페이지 흐름
- placeholder와 작성요령
- 표지, 요약서, 본문, 사업비 표처럼 서로 다른 구조의 페이지

현재 `hwpxctl`는 unpacked XML을 편집하는 primitive는 일부 제공하지만, 실제 목표였던 "원본 양식을 유지한 채 내용만 채우고 안내문만 제거하는 작업"을 직접 지원하는 추상화가 부족하다.

이 로드맵은 더 이상 "명령 몇 개 추가"를 목표로 하지 않는다. 목표는 아래와 같다.

- 사람이 수정 대상을 쉽게 찾을 수 있어야 한다
- low-level 좌표 대신 의미 기반 target을 지정할 수 있어야 한다
- 수정 후 구조 valid뿐 아니라 render-safe 판단이 가능해야 한다
- 공공/기업 양식 같은 복합 문서를 end-to-end로 다룰 수 있어야 한다

## 제품 목표

### Primary Goal

실제 양식 `.hwpx` 문서를 입력으로 받아:

- 원본 레이아웃과 구조를 최대한 유지하고
- Markdown/JSON/YAML 내용을 적절한 위치에 채우고
- guide text와 placeholder를 안전하게 제거 또는 치환하고
- 최종적으로 HWP/HWPX 편집기와 Viewer에서 정상 렌더링되는 결과물을 생성한다

### Non-Goal

아래는 당장 우선 목표가 아니다.

- 모든 HWPX XML 요소에 대한 low-level write API 완전 노출
- 임의 문서를 완벽하게 재배치하는 generic layout engine
- 암호화 문서 지원
- 협업/버전 복원 같은 문서 관리 플랫폼 기능

## 현재 상태 진단

### 현재 강점

- `pack`, `unpack`, `inspect`, `validate`, `text`, `export-*` 기본 흐름 보유
- 문단, run, table cell, image, shape 등 일부 low-level mutation 가능
- section 편집 지원이 일부 명령에서 확장 중
- Viewer 인쇄 기반 검증 스크립트 보유

### 현재 한계

- `validate`는 구조적 valid만 보며 렌더링 안정성을 설명하지 못한다
- 복합 양식에서 target discovery 비용이 과도하다
- low-level 좌표 기반 수정만으로는 양식 유지형 편집이 어렵다
- 안내문 제거와 실제 구조 안정성 사이에 safe policy가 없다
- multi-section 지원이 명령 전반에 일관되게 적용되지 않았다
- 동일 unpacked 디렉터리 병렬 mutation에 대한 안전장치가 없다

## 핵심 제품 원칙

### 1. Render-Safe First

구조 valid보다 실제 렌더링 안정성을 우선한다.

### 2. Analyze Before Edit

복합 양식은 먼저 분석하고, 그 다음 수정한다.

### 3. Anchor Over Coordinates

사람이 쓰는 편집은 좌표보다 라벨, placeholder, 근접 텍스트 기반이어야 한다.

### 4. Safe Mutation Over Destructive Mutation

삭제보다 치환, 숨김, 내용 비우기 같은 보수적 편집을 우선한다.

### 5. End-to-End Verification

최종 완료 기준은 XML valid가 아니라 Viewer 인쇄 결과다.

## 새 로드맵 구조

기존의 feature bucket 중심 대신 다음 5개 트랙으로 재편한다.

1. Template Analysis
2. Safe Editing
3. High-Level Filling
4. Round-Trip Verification
5. Reliability And Concurrency

## Phase 0: Guardrails And Product Baseline

목표는 "위험한 문서와 위험한 수정"을 먼저 식별할 수 있게 하는 것이다.

### P0-1 문서 위험도 분석

- [ ] section 수, TOC 존재, merged cell 비율, nested table 존재, object/control density 계산
- [ ] 문서 위험도 등급 산출
- [ ] "low-level edit 위험" 경고 문구 추가

성공 기준:

- `inspect` 또는 신규 분석 명령에서 문서 위험도와 원인을 반환
- 복합 양식 문서에 대해 `toc-risk`, `section-risk`, `table-risk` 같은 힌트 제공

### P0-2 mutation 동시성 안전성

- [ ] unpacked 디렉터리 단위 lock 도입
- [ ] atomic XML write 적용
- [ ] 병렬 mutation 감지 시 fail-fast 또는 wait 정책 정의

성공 기준:

- 동일 디렉터리에 mutation 명령 병렬 실행 시 `unexpected EOF`가 재현되지 않음
- 사용자에게 lock 충돌 원인과 해제 가이드가 출력됨

### P0-3 validate 확장

- [ ] `validate` 결과에 `renderSafe` 개념 추가
- [ ] `layout-risk`, `toc-risk`, `section-risk`, `roundtrip-risk` 힌트 추가
- [ ] 구조 valid와 render-safe를 분리해 보여주기

성공 기준:

- `valid=true`여도 render risk가 있으면 명확히 경고
- 실패 메시지가 XML invalid/valid를 넘어 실제 깨질 수 있는 이유를 설명

## Phase 1: Template Analysis

목표는 "어디를 수정해야 하는지 사람이 찾기 쉽게 만드는 것"이다.

### P1-1 `analyze-template`

- [ ] section map 출력
- [ ] table map 출력
- [ ] merged cell map 출력
- [ ] paragraph/run candidate 출력
- [ ] TOC/guide text/placeholder 후보 탐지

권장 출력:

- section index, path, page role 후보
- table index, row/col, merged span, 현재 텍스트
- paragraph index, style summary, text preview
- placeholder candidate, guide candidate, anchor candidate

성공 기준:

- 복합 양식에서 사람이 XML을 직접 열지 않고도 수정 위치를 찾을 수 있음

### P1-2 사람이 쓰기 좋은 target discovery

- [ ] `find-targets --anchor`
- [ ] `find-targets --near-text`
- [ ] `find-targets --table-label`
- [ ] `find-targets --placeholder`

성공 기준:

- `"과제명"`, `"주관기관"`, `"사업비 총괄표"` 같은 라벨 기준으로 후보를 찾을 수 있음
- 결과에 section/table/cell/paragraph, 주변 텍스트, 스타일 요약이 함께 나옴

### P1-3 guide text / placeholder detector

- [ ] 색상 기반 guide text 후보 추출
- [ ] 텍스트 패턴 기반 작성요령 후보 추출
- [ ] placeholder 문법 및 빈칸형 필드 후보 탐지

성공 기준:

- 파란 안내문과 placeholder 후보를 낮은 false positive로 분리 표시

## Phase 2: Safe Editing

목표는 "지워도 valid"가 아니라 "지워도 안 깨지는" 편집 정책을 만드는 것이다.

### P2-1 `remove-guides`

- [ ] `--dry-run` 지원
- [ ] style/color/text pattern 기반 제거
- [ ] delete 대신 clear/hide/replace-empty 정책 지원
- [ ] section/table/paragraph 단위 영향 범위 요약

성공 기준:

- 작성요령 제거 후 구조 valid와 render-safe가 함께 유지됨
- 사용자가 실제 삭제 전에 영향 범위를 확인 가능

### P2-2 safe paragraph/table mutation

- [ ] `delete-paragraph` 대체 safe mode
- [ ] cell 내부 복수 문단 유지형 치환
- [ ] 기존 스타일 유지형 치환
- [ ] merged cell 보존형 텍스트 업데이트

성공 기준:

- low-level edit 없이도 실제 양식의 본문/표 내용을 안전하게 바꿀 수 있음

### P2-3 multi-section 일관 지원

- [ ] 모든 mutation 명령에 공통 section selector 적용
- [ ] section 0 기본값 의존성 제거
- [ ] section-aware JSON response 일관화

성공 기준:

- section이 여러 개인 양식에서 명령별 동작 차이가 최소화됨

## Phase 3: High-Level Fill

목표는 low-level 좌표 편집이 아니라 의미 기반 템플릿 채우기다.

### P3-1 `fill-template`

- [ ] JSON/YAML mapping spec 정의
- [ ] anchor-to-value 매핑 지원
- [ ] placeholder-to-value 매핑 지원
- [ ] safe replace policy 내장

예시:

- `"과제명" -> "Open Source AI Agent 플랫폼"`
- `"주관기관" -> "예시 주식회사"`
- `"참여기관1" -> "기관 A"`

성공 기준:

- 사용자가 section/table/cell 좌표를 직접 지정하지 않고 주요 필드를 채울 수 있음

### P3-2 Markdown to form mapping

- [ ] heading/list/table/block을 문서 구조에 매핑
- [ ] multi-line paragraph/cell 처리
- [ ] 표 형태 Markdown을 사업비/요약 표에 매핑

성공 기준:

- 수행계획서 Markdown을 바로 필드 데이터로 변환해 문서에 입력 가능

### P3-3 반복 블록 지원

- [ ] 참여기관 N건 반복 입력
- [ ] 인력/예산 row 확장 또는 치환
- [ ] repeated anchor block 설계

성공 기준:

- 반복 표를 low-level 수작업 없이 채울 수 있음

## Phase 4: Round-Trip Verification

목표는 `pack` 이후 결과가 실제로 안전한지 점검하는 것이다.

### P4-1 `preview-diff`

- [ ] paragraph/table 수준 before/after diff
- [ ] guide 제거와 placeholder 치환 결과 요약
- [ ] 변경된 section map 요약

성공 기준:

- 사용자가 최종 pack 전에 실질적 수정 내역을 검토 가능

### P4-2 `roundtrip-check`

- [ ] pack 후 재-unpack 또는 재-inspect 비교
- [ ] TOC/page/section 흐름 점검
- [ ] 본문 누락 여부 점검
- [ ] 위험 변경 요약

성공 기준:

- `validate` 외에 round-trip quality gate가 생김

### P4-3 `safe-pack`

- [ ] 위험도 높은 변경 감지 시 warning 또는 block
- [ ] `--force` 정책 분리
- [ ] 최종 pack report에 render risk 첨부

성공 기준:

- 위험한 결과물을 무심코 pack해서 넘기는 상황을 줄임

### P4-4 Viewer 기반 검증 하네스

- [ ] `python ./scripts/print_hwpx_via_viewer.py` 기반 smoke test 표준화
- [ ] PDF text/snapshot 비교 보조 도구 추가
- [ ] output 아티팩트 관리 규칙 정리

성공 기준:

- 실제 문서 완료 판정이 Viewer 인쇄 결과까지 포함됨

## Phase 5: Examples And Productization

목표는 실제 복합 양식 기준으로 재현 가능한 사용 흐름을 제공하는 것이다.

### P5-1 end-to-end example

- [ ] 실제 공공 양식 샘플 1건 기준 예제 제공
- [ ] input JSON/Markdown, mapping YAML, output HWPX, output PDF 함께 제공
- [ ] before/after 비교 자료 제공

성공 기준:

- 새 사용자가 "어떻게 써야 하는지"를 예제로 바로 이해 가능

### P5-2 operator guide

- [ ] 복합 양식 편집 권장 절차 문서화
- [ ] low-level command를 써야 하는 경우와 쓰지 말아야 하는 경우 정리
- [ ] 리스크 대응 가이드 작성

성공 기준:

- 사용자 문서만 읽고도 안전한 작업 순서를 따라갈 수 있음

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

### High-Level Commands

새 제품 방향의 중심이다.

- `analyze-template`
- `find-targets`
- `remove-guides`
- `fill-template`
- `preview-diff`
- `roundtrip-check`
- `safe-pack`

용도:

- 실제 양식 자동 입력
- 사람 친화적 target discovery
- 안전성 우선 workflow

## 권장 기본 워크플로우

1. 원본 양식 복사
2. `unpack`
3. `analyze-template`
4. `remove-guides --dry-run`
5. `fill-template`
6. `preview-diff`
7. `roundtrip-check`
8. `safe-pack`
9. Viewer PDF 인쇄
10. 결과 비교 후 필요 시 mapping 수정

## 즉시 착수 우선순위

### Immediate P0

1. unpacked dir lock + atomic write
2. `validate` risk hint 확장
3. `analyze-template` 최소 버전

### Immediate P1

1. guide text detector
2. placeholder detector
3. section/table/cell discovery 출력

### Immediate P2

1. `remove-guides --dry-run`
2. `fill-template` 최소 버전
3. `roundtrip-check` 최소 버전

## GitHub Issue 단위 분해

### Reliability

- [ ] unpacked 디렉터리 lock 구현
- [ ] atomic XML write 적용
- [ ] 병렬 mutation error regression test 추가

### Validation

- [ ] `validate`에 `renderSafe` 필드 추가
- [ ] `layout-risk` 계산기 추가
- [ ] `toc-risk` 계산기 추가
- [ ] `section-risk` 계산기 추가
- [ ] `roundtrip-risk` 계산기 추가

### Analysis

- [ ] `analyze-template` 명령 추가
- [ ] section map schema 정의
- [ ] table map schema 정의
- [ ] merged cell map schema 정의
- [ ] placeholder candidate detector 추가
- [ ] guide candidate detector 추가
- [ ] anchor candidate detector 추가

### Discovery UX

- [ ] `find-targets --anchor` 추가
- [ ] `find-targets --near-text` 추가
- [ ] `find-targets --table-label` 추가
- [ ] discovery 출력에 style/merge/section summary 추가

### Safe Editing

- [ ] `remove-guides` 명령 추가
- [ ] delete 대신 clear/hide 정책 구현
- [ ] safe paragraph delete 전략 추가
- [ ] merged cell 보존형 텍스트 업데이트 개선
- [ ] cell 내부 multi-paragraph replace 개선

### High-Level Fill

- [ ] `fill-template` 명령 추가
- [ ] mapping YAML schema 설계
- [ ] anchor-to-value resolver 구현
- [ ] placeholder-to-value resolver 구현
- [ ] Markdown block mapper 구현
- [ ] repeated block fill 지원

### Verification

- [ ] `preview-diff` 명령 추가
- [ ] `roundtrip-check` 명령 추가
- [ ] `safe-pack` 명령 추가
- [ ] Viewer smoke test harness 정리
- [ ] PDF text/snapshot compare 보조 도구 추가

### Coverage

- [ ] 전체 mutation 명령의 multi-section selector 통일
- [ ] section-aware regression test 확대
- [ ] 실제 공공 양식 fixture 추가
- [ ] end-to-end example 문서 추가

## 완료 판정 기준

로드맵 완료는 "명령이 많아짐"이 아니라 아래 기준으로 판단한다.

- 복합 양식에서 수정 대상을 사람이 빠르게 찾을 수 있음
- guide text와 placeholder를 안전하게 제거/치환할 수 있음
- JSON/YAML/Markdown 데이터로 주요 필드를 채울 수 있음
- `validate`가 render risk를 설명할 수 있음
- pack 후 round-trip 점검이 가능함
- Viewer 인쇄 결과에서 문서 흐름과 레이아웃이 유지됨

## 결론

기존 로드맵은 low-level 기능 추가 관점에서는 유효했지만, 실제 사용 문제를 해결하는 데는 초점이 맞지 않는다. 앞으로의 로드맵은 "무엇을 더 편집할 수 있는가"보다 "실제 양식을 얼마나 안전하게 채울 수 있는가"를 중심으로 관리해야 한다.

따라서 `hwpxctl`의 다음 단계는 XML 편집 기능 확장이 아니라:

- template analysis
- safe editing
- high-level fill
- round-trip verification
- concurrency safety

이 다섯 축으로 재편하는 것이 맞다.

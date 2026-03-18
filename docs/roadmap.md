# HWPX CLI Roadmap

현재 구현 상태:

- 본문 추가
- 문단 정렬/들여쓰기/간격
- 글머리표/번호 매기기
- 본문 run 스타일 적용
- 표 생성, 셀 수정, 셀 병합/분할, 중첩 표
- 이미지 임베드 및 본문 배치
- 이미지/도형 위치 제어
- 머리말/꼬리말 및 쪽 번호
- 머리말/꼬리말 제거
- 각주/미주
- 책갈피 및 하이퍼링크
- 제목/개요 문단, 차례, 기본 내부 참조
- 수식
- 메모
- 선/타원/사각형/글상자
- 다단 편집
- opt-in historyEntry 변경 추적
- HTML/Markdown export
- pack/unpack/validate/inspect/text

## P0 Top Priority: Example Parity

최우선 목표는 `create`부터 시작해 `example/[활용 분야 신청서(HWP)] 2026년 오픈소스 AI·SW 개발·활용 지원사업.hwpx`와 **동일한 문서**를 CLI만으로 재현 가능한 수준까지 끌어올리는 것이다.

현재 판정:

- 부분 재현은 가능
- 동일 문서 재현은 아직 불가
- 원본은 표 79개 중심의 복합 폼 문서라, 표/페이지/스타일의 정밀 제어가 더 필요

P0 완료 기준:

- `create` 기반 새 문서에서 example과 동일한 구조를 CLI로 구성 가능
- `text` 기준 핵심 본문이 실질적으로 동일
- `Hancom Office HWP Viewer` PDF 인쇄 결과에서 페이지 수와 주요 레이아웃이 기대 범위 안에 들어옴
- 검증 스크립트는 반드시 `python ./scripts/print_hwpx_via_viewer.py <generated.hwpx>`를 포함

P0 기능 분해:

- [ ] P0-1 섹션/페이지 레이아웃 제어
  - 목표: `pagePr`, margin, landscape, page border fill 계열 제어
  - 필요 이유: example은 A4 landscape와 여백/페이지 경계 설정에 의존
- [ ] P0-2 표 geometry 제어
  - 목표: 열 너비, 행 높이, 표 전체 width/height, table/inline margin 제어
  - 필요 이유: example은 22x13 대형 표와 다수의 세부 크기 제어를 사용
- [ ] P0-3 셀 스타일 제어
  - 목표: cell margin, vertical align, border fill, background, 기본 텍스트 정렬 제어
  - 필요 이유: example은 셀별 시각 스타일과 채움/선 차이가 크다
- [ ] P0-4 셀 내부 문단/런 스타일 확장
  - 목표: 표 셀 안 문단 정렬, 문단 스타일 참조, 텍스트 스타일 조합을 더 정밀하게 적용
  - 필요 이유: example은 폼 라벨/본문/안내 문구가 서로 다른 스타일을 사용
- [ ] P0-5 example parity 검증 하네스
  - 목표: `create -> build example-like doc -> pack -> viewer print -> compare` 흐름 자동화
  - 필요 이유: 기능이 생겨도 최종 acceptance는 한컴 뷰어 PDF 기준이어야 함

## Progress Snapshot

완료된 큰 묶음:

- M1 ~ M5 완료
- M6는 history-only 1차 변경 추적까지 완료
- P1 Paragraph Formatting은 정렬/들여쓰기/간격, bullet/number까지 완료
- P1 Editing은 문서/섹션/표 편집 핵심 기능 완료
- P1 Text Styling은 run 추가/교체, 스타일 적용, 스타일 검색/치환까지 완료
- P1 Shapes And Layout은 위치 제어 포함 완료
- P2 Search And Analysis는 XPath 검색까지 완료
- P2 Export는 1차 구현 완료

바로 확인할 수 있는 현재 미완료 핵심:

- example 동일 문서 재현용 정밀 layout/table/cell/style 제어
- low-level XML/history/version 접근
- 템플릿 분석 CLI
- 문서 비교/구조 점검 도구

## Recommended Next Steps

1. P0-1 섹션/페이지 레이아웃 제어
2. P0-2 표 geometry 제어
3. P0-3 셀 스타일 제어
4. P0-4 셀 내부 문단/런 스타일 확장
5. P0-5 example parity 검증 하네스
6. low-level XML/history/version 접근
7. 템플릿 분석 CLI
8. 문서 비교/구조 점검 도구

## Milestones

### M1

- [x] 머리말/꼬리말
- [x] 쪽 번호
- [x] 전체 쪽수 표기

성공 기준:

- 텍스트 머리말/꼬리말 삽입
- 현재 쪽과 시작 번호 지정
- 한컴 뷰어 인쇄 PDF에서 반복 표시 확인
- `{{PAGE}}`, `{{TOTAL_PAGE}}` 조합이 한컴 뷰어 인쇄 PDF에서 실제 숫자로 치환됨

### M2

- [x] 각주/미주

성공 기준:

- 본문 앵커와 주석 본문 생성
- 쪽 하단 배치 확인
- 텍스트 추출 시 순서 보존

### M3

- [x] 책갈피
- [x] 하이퍼링크

성공 기준:

- 책갈피 생성과 이름 충돌 처리
- URL 링크와 문서 내 책갈피 링크 생성
- 한컴 뷰어 인쇄 PDF에서 링크 주석 생성 확인

### M4

- [x] 제목 스타일/개요
- [x] 차례
- [x] 상호 참조

성공 기준:

- 제목 기반 차례 생성
- 책갈피 기반 기본 내부 참조 생성
- 표/그림/수식 참조는 후속 단계로 분리

### M5

- [x] 수식

성공 기준:

- 단일 수식 삽입
- 본문 배치와 한컴 뷰어 렌더링 확인

### M6

- [x] 메모
- [x] 기본 도형(사각형)
- [x] 글상자
- [x] 변경 추적

성공 기준:

- 기본 사각형 도형 삽입과 한컴 뷰어 인쇄 PDF 렌더링 확인
- 메모 생성과 문서 열림 확인
- 글상자 텍스트와 한컴 뷰어 인쇄 PDF 렌더링 확인
- opt-in `historyEntry` 기록 후 한컴 뷰어 문서 열림과 PDF 인쇄 확인
- 보이는 삽입/삭제 추적은 후속 단계

## Deferred TODO

- [x] 기본값은 이력 미기록 유지
- [x] 필요한 사용자만 명시적으로 켤 수 있는 opt-in 옵션 추가
- [x] 1차는 mutating CLI에 `historyEntry`만 남기는 방식 적용
- [ ] 2차는 텍스트 계열 명령에 한해 `insertBegin/deleteBegin` 기반 visible tracking 검토
- [x] 표/이미지/머리말 같은 구조 변경은 현재 history-only로 제한

## Python-hwpx Comparison Backlog

`python-hwpx` 대비 현재 CLI에 아직 없는 범위를 backlog로 정리한다.

### P1 Editing

- [x] 문단 삭제
- [x] 문단 텍스트 수정
- [x] 섹션 추가
- [x] 섹션 삭제
- [x] 머리말 제거
- [x] 꼬리말 제거
- [x] 표 셀 병합
- [x] 표 셀 분할
- [x] 중첩 표

준비 메모:

- 문단 삭제/수정은 현재 `append-text` 흐름과 같은 section 편집기에서 확장 가능
- 섹션 추가/삭제는 `content.hpf` manifest/spine과 `header.xml secCnt`를 함께 갱신하고, 기존 편집 명령은 당분간 첫 section 기준을 유지
- 표 병합/분할은 셀 주소와 span을 논리 좌표 기준으로 맞추고, 가려진 셀 텍스트 복원은 후속 과제로 둔다

### P1 Text Styling

- [x] run 단위 텍스트 추가
- [x] run 단위 텍스트 교체
- [x] bold/italic/underline 스타일 적용
- [x] 텍스트 색상 적용
- [x] 스타일 기반 run 검색
- [x] 스타일 기반 선택 치환

준비 메모:

- 현재 `set-text-style`로 direct run 기준 `charPr` 복제 후 스타일 적용까지는 완료
- 후속 과제는 export, low-level access, 템플릿 분석으로 좁혀짐

### P1 Paragraph Formatting

- [x] 문단 정렬
- [x] 문단 들여쓰기/좌우 여백
- [x] 문단 앞/뒤 간격
- [x] 줄간격
- [x] 글머리표
- [x] 번호 매기기

준비 메모:

- 현재는 첫 section의 editable paragraph 기준으로 동작
- paragraph별 `paraPr`를 복제해서 필요한 값만 바꾸는 방식이라 기존 문서 스타일을 최대한 유지
- 번호 매기기의 임의 시작값은 numbering definition clone으로 처리

### P1 Shapes And Layout

- [x] 선 도형
- [x] 타원 도형
- [x] 글상자
- [x] 다단 편집
- [x] 이미지/도형 위치 제어

준비 메모:

- 선/타원은 현재 사각형 구현 패턴을 복제해 확장 가능
- 글상자는 도형보다 텍스트 컨테이너 구조 검증이 먼저 필요
- 다단 편집은 section 첫 문단의 `hp:colPr` 편집 명령으로 분리하는 편이 안전
- 위치 제어 1차는 `treatAsChar`, `x/y`, 수평/수직 정렬만 노출

### P2 Search And Analysis

- [x] 객체 검색 CLI
- [x] 태그 기반 검색
- [x] 속성 기반 검색
- [x] XPath 기반 검색
- [ ] 템플릿 분석 CLI
- [ ] 문서 비교/구조 점검 도구

준비 메모:

- 읽기 전용 기능이라 문서 손상 리스크가 낮아 중간에 병렬 진행 가능
- 기존 `inspect`와 겹치지 않게 출력 계약을 먼저 정리

### P2 Export

- [x] HTML 내보내기
- [x] Markdown 내보내기

준비 메모:

- 현재는 문단/표 중심 block 모델과 placeholder 기반 1차 구현 완료
- 후속 보강은 이미지 실제 추출, 각주/링크/병합 표 표현 개선

### P2 Low-level Access

- [ ] master page 조회
- [ ] history 파트 조회
- [ ] version 파트 조회
- [ ] 저수준 XML part 출력/편집 API
- [ ] namespace 정규화/호환성 처리

준비 메모:

- CLI로 바로 노출하기보다 내부 package API를 먼저 만드는 쪽이 확장성에 유리

## Candidate Backlog From Web Workflows

웹 기반 한글 문서 서비스에서 반복적으로 보이는 요구사항을 현재 CLI 기준의 후보 기능으로 정리한다.

- [ ] 파일/문서 버전 조회 및 복원
- [ ] 본문 검색 강화
- [ ] 필드/태그 기반 템플릿 채우기
- [ ] 구조화 추출(JSON/tree/object export)
- [ ] 숨은 객체/고급 요소 조회

메모:

- 한컴독스는 파일 버전 관리와 본문 검색을 기본 기능으로 제공한다
- Polaris Converter는 문서 검색, 메모, 출력과 같은 읽기/열람 기능을 강조한다
- Polaris AI DataInsight는 텍스트뿐 아니라 이미지/표/차트/객체 속성 추출을 강조한다
- 한컴싸인/Polaris Docs 계열은 템플릿, 필드, 이력, 협업 흐름을 전면에 둔다

### Out Of Scope For Now

- [ ] 암호화된 HWPX 지원 여부 검토

준비 메모:

- `python-hwpx`도 암호화 파일은 지원하지 않으므로 우선순위는 낮음

## Execution Order

1. P0-1: 섹션/페이지 레이아웃 제어
2. P0-2: 표 geometry 제어
3. P0-3: 셀 스타일 제어
4. P0-4: 셀 내부 문단/런 스타일 확장
5. P0-5: example parity 검증 하네스
6. M1: 머리말/꼬리말 + 쪽 번호
7. M2: 각주/미주
3. M3: 책갈피 + 하이퍼링크
4. M4: 제목 스타일/개요 + 차례 + 상호 참조
5. M5: 수식
6. M6: 메모 + 기본 도형 + 글상자 + history-only 변경 추적 완료

## Notes

- 구현 순서는 실제 문서 작성 빈도와 기능 의존성을 기준으로 잡는다.
- 검증 기준은 구조 검증과 한컴 뷰어 인쇄 PDF를 함께 사용한다.
- 새 기능은 가능하면 CLI 명령, 자동 테스트, 수동 인쇄 검증을 한 세트로 추가한다.
- 변경 추적은 OWPML 히스토리 구조가 커서 별도 마일스톤으로 분리할 수 있다.
- 변경 추적이 들어가더라도 기본 동작은 현재처럼 문서만 수정하고, 이력 기록은 opt-in으로 유지한다.

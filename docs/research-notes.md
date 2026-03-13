# HWPX CLI Research Notes

## What the Hancom articles imply

- [HWP format structure](https://tech.hancom.com/%ed%95%9c-%ea%b8%80-%eb%ac%b8%ec%84%9c-%ed%8c%8c%ec%9d%bc-%ed%98%95%ec%8b%9d-hwp-%ed%8f%ac%eb%a7%b7-%ea%b5%ac%ec%a1%b0-%ec%82%b4%ed%8e%b4%eb%b3%b4%ea%b8%b0/) shows why legacy `.hwp` is harder: it is a binary CFB container with record-oriented streams, so extraction requires header parsing and record-size handling.
- [HWPX format structure](https://tech.hancom.com/hwpxformat/) establishes the better CLI target for macOS/Linux: HWPX is a ZIP package of XML parts, with `mimetype`, `version.xml`, `settings.xml`, `Contents/`, `BinData/`, and `META-INF/`.
- The same article highlights that `Contents/content.hpf` is the package index. `metadata` holds title/creator data, `manifest` maps package items, and `spine` defines reading order.
- `Contents/header.xml` stores shared document properties and style mappings. `Contents/section*.xml` stores body content by section, and text is primarily carried in `<hp:t>` nodes under paragraph runs.
- [Python HWP parsing](https://tech.hancom.com/python-hwp-parsing-1/) is useful only as a contrast: binary HWP readers must decode tagged records, variable-length payloads, and stream positions carefully.
- [Python HWPX parsing](https://tech.hancom.com/python-hwpx-parsing-1/) suggests the practical extraction workflow for a CLI: open ZIP, extract namespaces, read `header.xml`, read `content.hpf`, resolve `spine`, then walk section XML and binary attachments.

## MVP scope

- Inspect package metadata, manifest, spine, sections, and binary payload paths
- Validate structural integrity of `.hwpx` files or unpacked directories
- Extract plain text in spine order
- Unpack `.hwpx` archives to editable directories
- Repack validated directories back into `.hwpx`

## Representative Hangul features from official sources

웹 검색 기준으로 아래한글의 대표 편집 기능 후보를 다음처럼 정리할 수 있다.

- 본문 작성/문단 편집: 한컴 도움말의 텍스트 입력·문단 관련 기능군이 가장 기본적인 편집 축이다.
- 표 생성/표 편집: 표 만들기, 셀 편집, 셀 병합/분할, 표 계산은 실무 문서에서 가장 빈도가 높다.
- 그림 삽입: 보고서/공문/제안서에서 로고, 캡처, 증빙 이미지 배치가 필수적이다.
- 머리말/꼬리말, 쪽 번호: 반복 레이아웃과 인쇄 문서 구성에서 핵심이다.
- 각주/미주: 법무, 학술, 보고 문서에서 자주 쓰인다.
- 수식: 교육/연구/기술 문서에서 중요하다.
- 차트/도형/글상자: 시각 보조 요소로 자주 쓰이지만, 본문/표/그림보다 구조가 복잡하다.
- 검토/주석/메모: 협업 편집에서 의미가 크지만 단일 문서 생성 CLI의 1차 우선순위는 아니다.

참고한 공식 자료:

- [한컴 도움말 모바일 목록](https://help.hancom.com/hoffice/multi/mobile/index.htm)
- [한컴 기술 블로그: HWPX format](https://tech.hancom.com/hwpxformat/)

## Editing roadmap

대표 기능을 HWPX XML 난이도와 현재 코드 구조 기준으로 나누면 다음 순서가 적절하다.

### Phase 1

- 새 문서 생성
- 본문 문단 추가
- 표 추가
- 표 셀 수정
- 이미지 바이너리 임베드

### Phase 2

- 본문 내 `<hp:pic>` 배치 XML 생성
- 머리말/꼬리말
- 쪽 번호
- 각주/미주

### Phase 3

- 수식
- 글상자/도형
- 차트
- 검토/주석

## Executed in this iteration

- `create`: 편집 가능한 unpack 디렉터리 생성
- `append-text`: 첫 section 끝에 문단 추가
- `add-table`: 기본 테두리 표 추가
- `set-table-cell`: 표 셀 텍스트 교체
- `embed-image`: `BinData`/manifest/header binDataList 등록

제약:

- 이미지의 실제 본문 배치는 아직 미구현이다
- 편집 대상은 현재 첫 번째 section 기준이다
- 표 스타일은 기본 테두리 중심이다

## Why Node.js for this repo

## Why Go for this repo

- macOS/Linux CLI 배포에 일반적으로 많이 쓰이고 단일 바이너리 배포가 쉽다
- 표준 라이브러리만으로 ZIP/XML 처리와 테스트 구성이 가능하다
- Windows 지원을 나중에 붙일 때도 크로스 컴파일 경로가 단순하다

## Deliberately out of scope for v0.1

- Legacy `.hwp` binary parsing
- Rendering fidelity checks against Hancom Office
- Windows packaging or installer support

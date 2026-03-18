# Example Table Playbook

`example` 문서의 표를 새 빈 문서에서 다시 만들 때 쓴 작업형 가이드입니다.  
목적은 “한 번에 전체 문서를 맞추기”가 아니라 “페이지 단위로, XML 근거를 확인하면서 표를 재현하기”입니다.

공개 저장소에는 원본 `example/*.hwpx`가 포함되지 않을 수 있습니다.  
이 경우 이 문서는 로컬 private sample을 기준으로 읽는 것을 전제로 합니다.

## 언제 이 문서를 보나

- `create`부터 시작해서 `example`과 비슷한 표를 새로 만들고 싶을 때
- Viewer 렌더만 봐서는 왜 안 맞는지 감이 안 잡힐 때
- `merge-table-cells`, `set-table-cell`, `normalize-table-borders`를 어떤 순서로 써야 할지 정리할 때

## 권장 흐름

1. 먼저 페이지 하나만 고릅니다
2. 같은 변형인지 확인합니다
3. 원본 XML에서 merge map, row/col 크기, borderFill, fill을 뽑습니다
4. `create -> page layout -> title paragraph -> add-table -> merge -> cell style -> text -> normalize -> pack -> Viewer print` 순서로 만듭니다
5. PDF와 XML을 같이 봅니다

한 번에 전체 문서를 만들려고 하면 실패 원인이 섞입니다.  
실제로는 `1페이지씩`, 더 좁게는 `표 1개씩` 맞추는 쪽이 훨씬 안정적입니다.

## 가장 많이 헷갈렸던 점

### 1. 같은 페이지처럼 보여도 변형이 다를 수 있습니다

`example`에는 같은 구조의 표라도 `총괄`, `주관`, `참여` 변형이 따로 있습니다.  
제목 문구가 다르면 렌더 비교가 계속 어긋납니다.

예:

- `ㅇ   비목별 소요명세 (총괄)`
- `ㅇ   비목별 소요명세 (주관 : ○○○○)`
- `ㅇ   비목별 소요명세 (참여 : ○○○○), 참여기관 없는 경우 표삭제`

시각 비교 전에 먼저 XML이나 본문 추출로 **같은 변형을 보고 있는지** 확인해야 합니다.

### 2. 빈 외곽선처럼 보이는 건 border보다 merge 문제일 때가 많습니다

렌더에서 외곽이 비어 보이면 보통 두 경우입니다.

- 실제 border가 약함
- 원본의 `rowSpan/colSpan`이 아직 덜 반영됨

후자가 더 흔했습니다.  
특히 `%` 열, 그룹 제목 셀, 합계행 주변은 merge가 다르면 Viewer에서 “빈 외곽선”처럼 보입니다.

### 3. visual-only 비교는 한계가 큽니다

표가 얼추 비슷해 보여도 XML은 전혀 다를 수 있습니다.

- row 높이
- col 폭
- `inMargin`
- `borderFillIDRef`
- fill color
- 문단 정렬

이 값들이 다르면 같은 표처럼 보이기 어렵습니다.  
그래서 렌더로 이상을 찾고, 원인은 XML로 확인하는 방식이 필요합니다.

## 현재 기능으로 가능한 것

- 페이지 크기와 여백 조정
- 표 행/열 크기 지정
- 직사각형 merge
- 셀 단위 텍스트 입력
- 셀 내부 정렬
- 폰트명, 폰트 크기
- 셀 채움색
- 공통 border와 면별 border override
- shared edge / perimeter 정규화

## 최근 merge 관련 기능에서 바뀐 점

- `merge-table-cells`
  - 병합 영역 perimeter의 강한 선을 anchor 셀 border로 승격합니다
  - 병합 영역 안의 fill color도 anchor 셀 borderFill로 승격합니다
- `split-table-cell`
  - 분할 후 새로 되살아나는 셀은 anchor 셀의 borderFill과 기본 문단/글자 스타일을 복제합니다
- `normalize-table-borders`
  - shared edge뿐 아니라 표 perimeter도 같이 정규화합니다

이 기능들 덕분에 병합된 셀의 외곽과 회색 헤더 블록이 전보다 훨씬 덜 깨집니다.

## 추천 작업 순서

### 1. 페이지 설정부터 맞춥니다

```bash
./hwpxctl create --output ./work/page20
./hwpxctl set-page-layout ./work/page20 \
  --orientation PORTRAIT \
  --width-mm 210 --height-mm 297 \
  --left-margin-mm 25 --right-margin-mm 25 \
  --top-margin-mm 15 --bottom-margin-mm 15 \
  --header-margin-mm 15 --footer-margin-mm 15
```

여백이 다르면 같은 표 폭이어도 전체 인상이 달라집니다.

### 2. 제목 문단과 단위 문구를 표 밖에 둡니다

원본은 표 제목이 표 내부가 아니라 별도 문단인 경우가 많습니다.

```bash
./hwpxctl append-text ./work/page20 --text $'(단위: 천원)\nㅇ   비목별 소요명세 (총괄)'
./hwpxctl set-paragraph-layout ./work/page20 --paragraph 0 --align RIGHT
./hwpxctl set-text-style ./work/page20 --paragraph 0 --font-name "맑은 고딕" --font-size-pt 10
./hwpxctl set-paragraph-layout ./work/page20 --paragraph 1 --align LEFT
./hwpxctl set-text-style ./work/page20 --paragraph 1 --font-name "맑은 고딕" --font-size-pt 10
```

### 3. 표는 먼저 빈 그리드로 만들고 merge는 나중에 적용합니다

```bash
./hwpxctl add-table ./work/page20 \
  --rows 17 --cols 8 \
  --width-mm 159.45 \
  --col-widths-mm 25.58,26.11,20.95,17.96,17.96,17.96,17.97,14.96 \
  --row-heights-mm 8.75,4.16,25.52,8.52,8.50,52.74,8.50,8.50,9.08,8.50,9.03,17.02,8.52,8.50,8.52,8.50,7.51
```

그 다음 원본 XML에서 확인한 merge map을 넣습니다.

```bash
./hwpxctl merge-table-cells ./work/page20 --table 0 --start-row 0 --start-col 0 --end-row 1 --end-col 0
./hwpxctl merge-table-cells ./work/page20 --table 0 --start-row 0 --start-col 3 --end-row 0 --end-col 5
./hwpxctl merge-table-cells ./work/page20 --table 0 --start-row 5 --start-col 0 --end-row 10 --end-col 0
```

핵심은 “눈으로 보기 좋은 merge”가 아니라 **원본 XML의 merge 좌표**를 그대로 따라가는 것입니다.

### 4. border/fill은 merge 후 적용하는 쪽이 안전합니다

```bash
./hwpxctl set-table-cell ./work/page20 --table 0 --row 0 --col 0 --fill-color "#D6D6D6"
./hwpxctl set-table-cell ./work/page20 --table 0 --row 0 --col 0 --border-top-style SOLID --border-top-width-mm 0.4
./hwpxctl set-table-cell ./work/page20 --table 0 --row 0 --col 0 --border-bottom-style DOUBLE --border-bottom-width-mm 0.5
```

면별 border override가 필요하면 `border-left-*`, `border-right-*`, `border-top-*`, `border-bottom-*`를 씁니다.

### 5. 텍스트와 정렬은 마지막에 세부 보정합니다

```bash
./hwpxctl set-table-cell ./work/page20 --table 0 --row 2 --col 1 --text "보수" --align LEFT --font-name "맑은 고딕" --font-size-pt 10
./hwpxctl set-table-cell-layout ./work/page20 --table 0 --row 2 --col 1 --paragraph 0 --align LEFT
```

`set-table-cell`만으로 충분하지 않은 경우 `set-table-cell-layout`, `set-table-cell-text-style`를 같이 씁니다.

### 6. 마지막에 border normalization을 한 번 더 돌립니다

```bash
./hwpxctl normalize-table-borders ./work/page20 --table 0
```

이 명령은 merge를 대신하지 않습니다.  
하지만 shared edge와 perimeter를 정리해줘서 Viewer에서 선이 덜 끊겨 보입니다.

### 7. 최종 검증은 Viewer PDF입니다

```bash
./hwpxctl pack ./work/page20 --output ./output/page20.hwpx
python ./scripts/print_hwpx_via_viewer.py ./output/page20.hwpx
```

렌더가 이상하면 다시 XML로 돌아갑니다.

## XML에서 우선 확인할 값

표 하나를 따라 만들 때 최소한 아래는 먼저 확인하는 게 좋습니다.

- 표 제목 문구와 변형 이름
- `rowCnt`, `colCnt`
- 각 주요 셀의 `rowSpan`, `colSpan`
- 주요 헤더 셀의 `borderFillIDRef`
- 회색 셀의 `fillBrush/winBrush faceColor`
- 표 `inMargin`
- 대표적인 row 높이와 col 폭

## page20 예산표에서 실제로 효과가 컸던 수정

- `참여` 변형이 아니라 `총괄` 변형을 기준으로 다시 잡기
- 헤더 회색 채움 16개를 XML 기준으로 반영하기
- `민간부담금` 헤더와 좌측 그룹 제목 셀 merge를 원본 좌표대로 맞추기
- `normalize-table-borders`를 마지막에 한 번 더 적용하기

## 아직 자동으로 안 되는 것

- 원본 문서에서 merge map 자동 추출
- 같은 구조의 다른 변형을 자동 선택
- page 단위 전체 생성 시퀀스 자동 합성

즉 현재는 “도구는 갖춰졌고, 원본 XML을 보고 수동으로 조립하는 단계”입니다.

## 추천 사용 방식

1. 먼저 [example-parity-harness.md](./example-parity-harness.md)로 전체 parity 흐름을 봅니다
2. 실제 표 하나를 만들 땐 이 문서 순서대로 진행합니다
3. 렌더가 이상하면 [cli-reference.md](./cli-reference.md)로 돌아가 옵션 계약을 다시 확인합니다

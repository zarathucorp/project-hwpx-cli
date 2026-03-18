# HWPX CLI Documentation

이 폴더는 `hwpxctl`을 사람과 AI 에이전트가 모두 예측 가능하게 사용할 수 있도록 정리한 문서 모음입니다.

문서 구성은 Justin Poehnelt의 ["Rewrite your CLI for AI agents"](https://justin.poehnelt.com/posts/rewrite-your-cli-for-ai-agents/)에서 강조한 다음 원칙을 현재 구현 범위에 맞게 적용했습니다.

- 명령의 입력과 출력을 명확하게 고정한다
- 구조화된 출력(JSON)을 우선 문서화한다
- 런타임에 계약을 조회할 수 있게 한다
- 큰 문서를 직접 읽기 전에 요약 가능한 하위 명령을 먼저 사용한다
- 사람이 읽는 설명과 에이전트가 실행할 계약을 분리한다

## 문서 맵

- [CLI Reference](/Users/zarathu/projects/project-hwpx-cli/docs/cli-reference.md): 명령별 입력, 출력, 종료 코드, 예시
- [Agent Guide](/Users/zarathu/projects/project-hwpx-cli/docs/agent-guide.md): AI 에이전트가 `hwpxctl`을 호출할 때의 권장 순서와 제약
- [Roadmap](/Users/zarathu/projects/project-hwpx-cli/docs/roadmap.md): 현재 구현 범위, 남은 작업, 우선순위
- [Research Notes](/Users/zarathu/projects/project-hwpx-cli/docs/research-notes.md): HWPX 포맷 조사 메모와 MVP 범위

## 현재 CLI가 잘하는 일

- `.hwpx`를 ZIP 기반 XML 패키지로 해석
- 구조 점검과 필수 파일 검증
- `spine` 기준 섹션 텍스트 추출
- 편집 가능한 디렉터리로 압축 해제
- 검증 가능한 디렉터리를 `.hwpx`로 재패키징
- unpack 디렉터리에 대한 본문/표/섹션/참조/주석/도형 편집
- macOS 기준 PDF 인쇄 자동화

## 현재 CLI가 보장하지 않는 일

- 렌더링 정확도 보장
- 한컴 UI와의 완전한 호환성 보장
- 레거시 `.hwp` 파싱
- 세밀한 XML 편집 명령 제공

## 권장 읽기 순서

사람이 빠르게 파악하려면 [CLI Reference](/Users/zarathu/projects/project-hwpx-cli/docs/cli-reference.md)부터 읽으면 됩니다.

AI 에이전트 호출 규칙까지 함께 정리하려면 [Agent Guide](/Users/zarathu/projects/project-hwpx-cli/docs/agent-guide.md)를 이어서 확인하면 됩니다.

개발자가 CLI 구조를 이어서 수정하려면 다음 파일 순서가 가장 빠릅니다.

- [internal/cli/cobra.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/cobra.go): `cobra` 루트/서브커맨드 구성과 help 진입점
- [internal/cli/root.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/root.go): 공통 옵션, 에러 envelope, `buildSchemaDoc()`
- [internal/cli/package.go](/Users/zarathu/projects/project-hwpx-cli/internal/cli/package.go): inspect/validate/text/unpack/pack/create
- 도메인별 `internal/cli/*.go`: 편집 명령 구현

새 명령 추가 기본 절차:

1. 도메인 파일에 핸들러 구현
2. `buildSchemaDoc()`에 명령 메타데이터 추가
3. `lookupCommandRunner()`에 핸들러 연결
4. `go test ./...`로 help/JSON envelope 회귀 확인

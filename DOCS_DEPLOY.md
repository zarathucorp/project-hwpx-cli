# Docs Deploy

문서 사이트 배포 방식과 수동 실행 방법을 정리한 메모입니다.

## 현재 정책

- GitHub Pages 문서 배포는 매 commit 자동 실행이 아니라 수동 실행입니다.
- 워크플로 파일은 `.github/workflows/deploy-docs.yml`입니다.
- 트리거는 `workflow_dispatch`만 사용합니다.

이유:

- 문서 수정이 잦을 때 불필요한 GitHub Actions 실행을 줄이기 위함
- private repository 또는 제한된 무료 사용량 환경에서 소모를 예측 가능하게 유지하기 위함

## GitHub UI에서 수동 실행

1. GitHub 저장소의 `Actions` 탭으로 이동
2. `deploy-docs` 워크플로 선택
3. `Run workflow` 클릭
4. 필요하면 branch/ref 선택 후 실행

## gh CLI로 수동 실행

사전 조건:

- `gh` 설치
- `gh auth login` 완료
- 해당 저장소에 대한 Actions 실행 권한 보유

기본 실행:

```bash
gh workflow run deploy-docs.yml
```

특정 브랜치 기준 실행:

```bash
gh workflow run deploy-docs.yml --ref main
```

실행 목록 확인:

```bash
gh run list --workflow deploy-docs.yml
```

가장 최근 실행 상태 실시간 확인:

```bash
gh run watch
```

## 추천 운영 방식

다음 중 하나를 기본값으로 권장합니다.

- 문서 변경을 몇 개 묶은 뒤 필요할 때만 수동 배포
- 릴리스 직전이나 `main` 반영 후에만 수동 배포
- 외부에 바로 보여줘야 하는 변경이 생겼을 때만 배포

## 참고

- 문서 로컬 검증:

```bash
python -m mkdocs build --strict
```

- 워크플로 정의 파일:

`/Users/zarathu/projects/project-hwpx-cli/.github/workflows/deploy-docs.yml`

# Pre-Commit Checks

이 문서는 로컬 커밋 전에 실행하는 기본 검증 묶음을 설명한다.

## 범위

기본 pre-commit 검증은 빠르고 재현 가능한 테스트만 실행한다.

- Go formatting check
- root module `go test ./...`
- `clients/*` provider client module `go test ./...`
- `clients/*` provider client module `go mod verify`
- `git diff --check`

실제 외부 API 를 호출하는 live/e2e 테스트는 기본 pre-commit 검증에 포함하지 않는다.
해당 테스트는 provider client 문서와 `docs/development/e2e/README.md` 에 적힌 build
tag 와 환경변수 gate 로 별도 실행한다.

## 수동 실행

```bash
make pre-commit
```

또는 스크립트를 직접 실행한다.

```bash
scripts/check/pre-commit.sh
```

## Git hook 설치

repo-managed hook 을 로컬 checkout 에 연결한다.

```bash
make install-hooks
```

이 명령은 현재 checkout 의 `core.hooksPath` 를 `.githooks` 로 설정한다.
이후 `git commit` 을 실행하면 `.githooks/pre-commit` 이 `make pre-commit` 과 같은
검증 묶음을 실행한다.

## Live/E2E Tests

KRX, KIS, Datago 처럼 실제 인증키와 외부 API 를 쓰는 테스트는 커밋 전 기본
검증에서 분리한다. 외부 API 상태, quota, 네트워크, 승인 범위에 영향을 받기
때문이다.

라이브 통합 테스트는 필요할 때 명시적으로 실행한다.

```bash
MWOSA_KRX_AUTH_KEY="..." make test-e2e-krx
```

인증키는 환경변수나 `.gitignore` 된 repo root `.env` 로만 전달하고, 코드, fixture,
로그에 남기지 않는다.

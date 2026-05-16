# Live/E2E Tests

이 문서는 실제 외부 API 와 인증키를 사용하는 opt-in 테스트를 설명한다.

기본 검증인 `go test ./...`, `make pre-commit`, `make verify` 는 live/e2e 테스트를
실행하지 않는다. 외부 API quota, 네트워크 상태, provider 승인 범위, 거래소 서비스
상태에 따라 결과가 달라질 수 있기 때문이다.

## KRX

KRX live/e2e 테스트는 `e2e` build tag, `KRX_E2E=1`, `MWOSA_KRX_AUTH_KEY` 가 모두
준비된 경우에만 실행된다. gate 가 꺼져 있거나 인증키가 없으면 테스트는 skip 한다.
`make test-e2e-krx*` 타깃은 repo root 의 `.env` 파일을 자동으로 읽는다. `.env` 는
`.gitignore` 에 포함되어 있으므로 커밋하지 않는다.

```bash
MWOSA_KRX_AUTH_KEY="..."
```

```bash
MWOSA_KRX_AUTH_KEY="..." make test-e2e-krx-client
MWOSA_KRX_AUTH_KEY="..." make test-e2e-krx-cli
MWOSA_KRX_AUTH_KEY="..." make test-e2e-krx
```

client e2e 를 직접 실행한다.

```bash
cd clients/krx

KRX_E2E=1 \
MWOSA_KRX_AUTH_KEY="..." \
go test -tags=e2e -count=1 ./...
```

전체 31개 KRX API 승인 범위를 얕게 확인하려면 client e2e 에 `KRX_E2E_ALL=1` 을
추가한다.

```bash
cd clients/krx

KRX_E2E=1 \
KRX_E2E_ALL=1 \
MWOSA_KRX_AUTH_KEY="..." \
go test -tags=e2e -count=1 ./...
```

CLI e2e 를 직접 실행한다.

```bash
KRX_E2E=1 \
MWOSA_KRX_AUTH_KEY="..." \
go test -tags=e2e -count=1 ./testing/e2e
```

CLI e2e 는 임시 config 와 임시 SQLite database 를 사용한다. `get krx`,
`sync krx`, `sync daily`, `backfill daily`, `get daily` 흐름을 실제 API 로 실행해
raw API 승인 확인부터 canonical daily 저장/조회까지 검증한다. JSON 같은
machine-readable 결과는 stdout 으로, backfill progress 같은 diagnostics 는 stderr 로
나가는지도 함께 확인한다.

느린 네트워크나 첫 `go run` 컴파일 때문에 시간이 더 필요하면 `KRX_E2E_TIMEOUT` 을
Go duration 형식으로 지정한다.

```bash
KRX_E2E_TIMEOUT=3m \
MWOSA_KRX_AUTH_KEY="..." \
make test-e2e-krx
```

## Secret Handling

- 인증키는 환경변수로만 전달한다.
- 인증키 값을 코드, 문서, fixture, config 예시, 로그에 남기지 않는다.
- 외부 API 응답 전체를 fixture 로 저장하지 않는다.
- 실패 로그를 공유할 때도 `MWOSA_KRX_AUTH_KEY` 값이 포함되지 않았는지 먼저 확인한다.

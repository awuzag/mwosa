# krx Client Workspace

`clients/krx` 는 KRX Data Marketplace OPEN API 를 호출하는 독립 Go client module 이다.

이 모듈은 `mwosa` CLI, provider adapter, storage, Cobra 에 의존하지 않는다. KRX
endpoint path, `AUTH_KEY` header, provider-native `OutBlock_1` parsing, remote error
context 만 소유한다.

현재 구현 범위는 KRX OPEN API 문서에 수집된 전체 31개 API 다.

- 지수: `krx_dd_trd`, `kospi_dd_trd`, `kosdaq_dd_trd`, `bon_dd_trd`,
  `drvprod_dd_trd`
- 주식: `stk_bydd_trd`, `ksq_bydd_trd`, `knx_bydd_trd`, `sw_bydd_trd`,
  `sr_bydd_trd`, `stk_isu_base_info`, `ksq_isu_base_info`,
  `knx_isu_base_info`
- 증권상품: `etf_bydd_trd`, `etn_bydd_trd`, `elw_bydd_trd`
- 채권: `kts_bydd_trd`, `bnd_bydd_trd`, `smb_bydd_trd`
- 파생상품: `fut_bydd_trd`, `eqsfu_stk_bydd_trd`, `eqkfu_ksq_bydd_trd`,
  `opt_bydd_trd`, `eqsop_bydd_trd`, `eqkop_bydd_trd`
- 일반상품: `oil_bydd_trd`, `gold_bydd_trd`, `ets_bydd_trd`
- ESG: `esg_etp_info`, `sri_bond_info`, `esg_index_info`

## Docs

- [docs/README.md](docs/README.md): KRX OPEN API 전체 31개 서비스의 수집 문서
- [docs/services.md](docs/services.md): 서비스 목록, `api_id`, `path`, `BO_ID`,
  endpoint 요약
- [docs/apis/](docs/apis/): 서비스별 요청/응답 필드 명세

## Go client

```go
client, err := krx.New(
	krx.WithAuthKey(os.Getenv("KRX_AUTH_KEY")),
)
if err != nil {
	return err
}

rows, err := client.ETF(ctx, "20250131")
```

기본 base URL 은 `https://data-dbg.krx.co.kr/svc/apis` 이다. 테스트나 로컬 검증에서는
`WithBaseURL(server.URL)` 로 교체한다. 샘플 endpoint 를 명시적으로 쓰려면
`WithSampleBaseURL(...)` 을 사용한다.

## Tests

```bash
go test ./...
go mod verify
```

라이브 e2e 테스트는 `e2e` build tag 와 `KRX_E2E=1` 환경변수 gate 뒤에 둔다.
기본 `go test ./...`, root `make pre-commit`, root `make verify` 에서는 실행하지
않는다. 실제 KRX OPEN API 승인 상태, 네트워크, quota 에 영향을 받기 때문이다.

```bash
cd clients/krx

KRX_E2E=1 \
MWOSA_KRX_AUTH_KEY="..." \
go test -tags=e2e -count=1 ./...
```

기본 라이브 e2e 는 `20240415` 기준 `etf_bydd_trd` 와 `stk_bydd_trd` 를 호출해
응답 성공, `row_count > 0`, 기본 식별 필드 존재를 확인한다. 전체 31개 API 승인
범위를 얕게 확인하려면 `KRX_E2E_ALL=1` 을 추가한다.
느린 네트워크에서는 `KRX_E2E_TIMEOUT=30s` 처럼 Go duration 형식으로 timeout 을
늘릴 수 있다.

```bash
cd clients/krx

KRX_E2E=1 \
KRX_E2E_ALL=1 \
MWOSA_KRX_AUTH_KEY="..." \
go test -tags=e2e -count=1 ./...
```

샘플 endpoint e2e 는 별도 인증키인 `KRX_SAMPLE_AUTH_KEY` 를 사용한다. 모든 인증키는
환경변수 또는 repo root 의 `.gitignore` 된 `.env` 에서만 읽고, 코드, 문서, fixture,
로그에 값을 남기지 않는다. 외부 API 응답 전체도 fixture 로 저장하지 않는다.

## Scripts

- [scripts/apply-all-services.browser.js](scripts/apply-all-services.browser.js):
  로그인된 KRX OPEN API 브라우저 탭에서 전체 31개 서비스를 12개월로 일괄 신청하는
  브라우저 콘솔용 스크립트

## 일괄 신청 스크립트 사용법

1. Chrome 에서 KRX OPEN API 에 로그인한다.
2. `https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd` 또는 같은
   origin 의 KRX OPEN API 페이지를 연다.
3. DevTools Console 또는 Sources > Snippets 에
   `scripts/apply-all-services.browser.js` 내용을 붙여 넣고 실행한다.
4. 확인 창에서 신청 범위와 기간을 확인한 뒤 진행한다.

스크립트 기본값:

- 신청 범위: `KRX 시리즈 일별시세정보` 부터 `ESG 지수` 까지 전체 31개 서비스
- 기간: `12M` 화면 옵션에 해당하는 `1Y`
- 신청 목적: `개인 연구`

실행 결과는 화면의 `<pre id="krx-apply-log">` 와 브라우저 콘솔에 함께 출력된다.

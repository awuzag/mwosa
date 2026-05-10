# krx Client Workspace

`clients/krx` 는 KRX Data Marketplace OPEN API 를 호출하는 독립 Go client module 이다.

이 모듈은 `mwosa` CLI, provider adapter, storage, Cobra 에 의존하지 않는다. KRX
endpoint path, `AUTH_KEY` header, provider-native `OutBlock_1` parsing, remote error
context 만 소유한다.

현재 구현 범위는 ETP 일별매매정보 3개 API 다.

- `ETF(ctx, baseDate)`: `etf_bydd_trd`
- `ETN(ctx, baseDate)`: `etn_bydd_trd`
- `ELW(ctx, baseDate)`: `elw_bydd_trd`

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

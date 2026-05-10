# kis Provider

## 개요

`kis` provider 는 한국투자증권 KIS Developers OpenAPI 를 `mwosa` 의 국내 시장
핵심 provider 로 연결하는 adapter 다.

기존 `datago` provider 가 D-1 영업일 기준의 일별 snapshot 수집에 강하다면,
`kis` provider 는 현재가, 기간별 시세, 분봉, 호가, 체결, 종목 기본정보, 실시간
시세를 맡는다. 주문과 계좌 API 도 문서 수집 범위에는 포함하지만, `mwosa` 의
초기 provider 역할에는 넣지 않는다.

## 수집 문서

KIS Developers 포털의 국내주식 8개 카테고리와 SDK 구현에 필요한 OAuth 문서를
수집했다.

| 파일 | 역할 |
| --- | --- |
| `clients/kis/docs/domestic-stock-openapi-catalog.json` | 요청/응답 필드, 예시, TR ID 를 포함한 수집 catalog |
| `clients/kis/docs/domestic-stock-api-inventory.md` | 카테고리별 API 목록과 TR ID 요약 |

수집 범위는 다음과 같다.

| scope | collection | count | 비고 |
| --- | --- | ---: | --- |
| SDK dependency | OAuth인증 | 3 | REST token, token revoke, websocket approval key |
| 국내주식 | 주문/계좌 | 23 | 문서만 수집, provider v1 제외 |
| 국내주식 | 기본시세 | 21 | REST quote, daily, minute, ETF/ETN quote 후보 |
| 국내주식 | ELW 시세 | 22 | v1 후순위 |
| 국내주식 | 업종/기타 | 14 | sector/index 보강 후보 |
| 국내주식 | 종목정보 | 26 | instrument role 후보 |
| 국내주식 | 시세분석 | 29 | screening/enrichment 후보 |
| 국내주식 | 순위분석 | 22 | market ranking 후보 |
| 국내주식 | 실시간시세 | 29 | websocket 후보 |

국내주식 API 는 총 186개이고, OAuth 문서를 포함하면 수집 catalog 는 189개 API 를
담는다.

## Provider id 와 group

provider id 는 `kis` 로 둔다.

초기 group 은 KIS 포털 카테고리 이름을 그대로 쓰기보다 `mwosa` 역할과 인증/호출
특성이 드러나도록 나눈다.

| group | 상태 | 원천 카테고리 | 주요 operation | capability |
| --- | --- | --- | --- | --- |
| `domesticStockQuotation` | core | 기본시세 | 현재가, 일/주/월/년, 분봉, 호가, 체결 | `quote`, `daily_bar`, `intraday_bar`, `orderbook`, `trade` |
| `domesticStockInstrument` | core | 종목정보 | 상품기본조회, 주식기본조회 | `instrument` |
| `domesticStockRealtime` | planned | 실시간시세 | 실시간체결가, 실시간호가 | `realtime_quote`, `realtime_orderbook`, `realtime_trade` |
| `domesticStockRanking` | planned | 순위분석, 시세분석 | 거래량, 등락률, 투자자, 프로그램 등 | `market_scan` |
| `domesticStockAccount` | deferred | 주문/계좌 | 주문, 잔고, 매수가능, 손익 | provider v1 제외 |

주문/계좌 API 는 같은 KIS credential 을 쓰더라도 `mwosa` 의 market-data provider 와
분리해서 다룬다. 자동매매나 주문 실행은 별도 실행/risk plane 이 필요하므로
문서 catalog 에만 보존한다.

## SDK 우선순위

`clients/kis` 는 독립 Go module 로 둔다. HTTP client 기본 스택은
`github.com/go-resty/resty/v2` 로 한다.

SDK 는 resty 가 제공하는 request building, JSON binding, retry hook, middleware
성격의 hook 을 활용한다. 직접 구현하는 부분은 KIS endpoint, domain 선택, OAuth,
header/TR ID 정책, provider-native response parsing, remote error mapping 같은
KIS 계약으로 제한한다.

v1 SDK 는 아래 REST API 부터 구현한다.

| 역할 | API | endpoint | TR ID |
| --- | --- | --- | --- |
| 현재가 | 주식현재가 시세 | `/uapi/domestic-stock/v1/quotations/inquire-price` | `FHKST01010100` |
| 기간별 일봉 | 국내주식기간별시세(일/주/월/년) | `/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice` | `FHKST03010100` |
| 당일 분봉 | 주식당일분봉조회 | `/uapi/domestic-stock/v1/quotations/inquire-time-itemchartprice` | `FHKST03010200` |
| 호가 | 주식현재가 호가/예상체결 | `/uapi/domestic-stock/v1/quotations/inquire-asking-price-exp-ccn` | `FHKST01010200` |
| 종목 기본정보 | 상품기본조회 | `/uapi/domestic-stock/v1/quotations/search-info` | `CTPF1604R` |
| 주식 기본정보 | 주식기본조회 | `/uapi/domestic-stock/v1/quotations/search-stock-info` | `CTPF1002R` |
| ETF/ETN 현재가 | ETF/ETN 현재가 | `/uapi/etfetn/v1/quotations/inquire-price` | `FHPST02400000` |

websocket 은 REST client 와 분리해서 나중에 붙인다. 실시간 API 는
`/oauth2/Approval` 로 접속키를 받고, `/tryitout/<TR_ID>` 문서의 request/response
field catalog 를 바탕으로 별도 session manager 를 만든다.

### Resty 사용 기준

- `resty.Client` 는 SDK `Client` 안에 감싸고 public API 로 직접 노출하지 않는다.
- 공통 header 는 resty request hook 또는 SDK 내부 helper 로 주입한다.
- retry 는 resty retry 기능을 사용하되, KIS business error 인 `rt_cd != "0"` 은
  status code retry 와 분리해서 판단한다.
- endpoint method 는 typed input/output 을 유지하고, `map[string]string` 기반
  호출은 SDK 바깥으로 노출하지 않는다.
- 테스트는 resty 를 실제 HTTP stack 으로 사용하되 `httptest` server 로 요청 path,
  query, header, body, error mapping 을 검증한다.

## Adapter 방향

`providers/kis` adapter 는 SDK 결과를 provider role interface 와 canonical model 로
변환한다.

- service layer 는 KIS endpoint 나 TR ID 를 직접 알지 않는다.
- SDK error 는 provider, group, operation, tr_id, status, rt_cd, msg_cd 맥락을
  포함해 반환한다.
- 실계좌 domain 과 모의투자 domain 은 SDK config 에서 명시적으로 선택한다.
- `datago` 와 겹치는 일봉 데이터는 freshness 와 목적이 다르므로 router profile 에
  반영한다.
- 주문/계좌 API 는 초기 registry 에 등록하지 않는다.

## 관련 문서

- `clients/kis/docs/feature-plan.md`
- `docs/architectures/provider/README.md`
- `docs/providers/README.md`

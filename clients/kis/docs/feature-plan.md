# KIS Provider Feature Plan

## 목적

한국투자증권 KIS Developers OpenAPI 를 `mwosa` 의 국내 주식 핵심 provider 로
연동한다.

1차 목표는 포털 문서 수집과 API 표면 정리다. 2차 목표는 이 문서를 기반으로
독립 Go SDK 인 `clients/kis` 를 만들고, 이후 `providers/kis` adapter 로 `mwosa`
provider registry 에 연결하는 것이다.

## 배경

현재 `mwosa` 는 국내 ETF/주식 일별 snapshot 에서 `datago` provider 를 먼저
사용한다. 이 경로는 대량 수집과 D-1 영업일 데이터에 적합하지만, 현재가, 분봉,
호가, 체결, 실시간 시세에는 맞지 않는다.

KIS provider 는 이 빈 곳을 채운다. 국내 리서치 흐름에서 사용자가 기대하는
`quote`, `daily`, `intraday`, `orderbook`, `trade`, `instrument` 조회를 KIS SDK
위에 얹고, service layer 에는 provider-neutral role 로만 노출한다.

## 문서 수집 범위

KIS 포털에서 국내주식 8개 카테고리와 SDK 의존 OAuth 문서를 수집했다.

- 원천: `https://apiportal.koreainvestment.com/apiservice-category`
- 수집 catalog: `clients/kis/docs/domestic-stock-openapi-catalog.json`
- 요약 inventory: `clients/kis/docs/domestic-stock-api-inventory.md`
- 국내주식 API 수: 186
- OAuth 포함 총 API 수: 189

수집 catalog 는 API 이름, endpoint, method, 실전/모의 domain, TR ID, 요청 예시,
응답 예시, 요청/응답 field catalog 를 보관한다. 구현 중에는 catalog 를 손으로
해석해 코드에 옮기고, 원천 문서가 바뀌면 같은 방식으로 다시 수집한다.

## v1 범위

v1 은 시장 데이터 provider 로 제한한다.

| capability | KIS API 후보 | mwosa 역할 |
| --- | --- | --- |
| 현재가 | 주식현재가 시세 | `quote.Snapshotter` |
| 기간별 일봉 | 국내주식기간별시세(일/주/월/년) | `dailybar.Fetcher` |
| 당일 분봉 | 주식당일분봉조회 | intraday role 추가 후보 |
| 호가 | 주식현재가 호가/예상체결 | orderbook role 추가 후보 |
| 체결 | 주식현재가 체결, 시간대별체결 | trade role 추가 후보 |
| 종목정보 | 상품기본조회, 주식기본조회 | `instrument.Searcher` 또는 `instrument.Resolver` |
| ETF/ETN 현재가 | ETF/ETN 현재가 | `quote.Snapshotter` for ETF/ETN |

실시간 websocket 은 문서와 인증 흐름만 v1 에서 정리하고, SDK 구현은 REST client 와
분리된 후속 단계로 둔다.

## 제외 범위

- 주문 실행
- 계좌/잔고/손익 조회
- 자동매매 판단
- 실시간 websocket session manager 구현
- KIS 문서 전체의 OpenAPI schema 자동 생성기
- provider adapter 전에 service layer 계약을 크게 바꾸는 작업

주문/계좌 API 는 문서 catalog 에 남겨두되, `mwosa` provider v1 에 등록하지 않는다.
이 범위는 나중에 실행/risk/portfolio plane 이 명확해졌을 때 다시 연다.

## 구현 단계

### 1. 문서 수집과 정규화

- 국내주식 카테고리별 API 목록을 수집한다.
- OAuth token 과 websocket approval 문서를 SDK 의존 문서로 함께 수집한다.
- endpoint, TR ID, request field, response field, 모의투자 지원 여부를 catalog 로
  보존한다.
- SDK MVP 후보 endpoint 를 표시한다.

### 2. `clients/kis` Go SDK

- 독립 Go module 로 만든다.
- HTTP client 기본 스택은 `github.com/go-resty/resty/v2` 로 한다.
- 생성자는 필수값을 명확한 `Config` 구조체로 받고, 환경 선택과 부가 동작은
  `With...` 함수형 옵션으로 받는다.
- `Config` 는 `AppKey`, `AppSecret`, `AccessToken` 처럼 대부분의 호출에 필요한
  필수 provider-native 값을 소유한다.
- token 발급은 민감정보 저장 방식이 정해지기 전까지 explicit method 로 둔다.
- resty 는 request building, JSON binding, retry hook, 공통 hook 기반으로 사용한다.
- SDK 내부는 `authorization`, `appkey`, `appsecret`, `tr_id`, `custtype` 같은 공통
  header 정책을 한 곳에서 관리한다.
- response parser 는 `rt_cd`, `msg_cd`, `msg1`, `output`, `output1`, `output2`
  같은 KIS envelope 차이를 endpoint 별 typed result 로 해석한다.
- KIS business error 와 HTTP/network retry 는 분리한다.
- 기본 단위 테스트는 resty client 와 `httptest` server 로 작성하고, 실제 KIS 호출은
  e2e test 로 분리한다.

### 3. `providers/kis` adapter

- SDK result 를 canonical model 로 정규화한다.
- provider profile 에 market, security type, freshness, 실전/모의 지원 여부를
  명시한다.
- `quote` 와 `daily_bar` 부터 registry 에 등록한다.
- 분봉, 호가, 체결은 role interface 가 정리된 뒤 등록한다.

## 완료 기준

- 수집 catalog 와 inventory 가 repo 에 저장되어 있다.
- v1 SDK endpoint 후보와 제외 범위가 문서화되어 있다.
- `clients/kis` 구현자가 endpoint, TR ID, request/response field 를 repo 안에서
  확인할 수 있다.
- provider adapter 가 어떤 role 부터 등록할지 결정되어 있다.
- 주문/계좌 API 가 v1 에 들어오지 않는 이유가 명확하다.

## 열어둘 질문

- `intraday_bar`, `orderbook`, `trade` role 을 기존 provider/core 아래에 어떻게
  추가할 것인가?
- KIS token 저장은 CLI config 에 둘 것인가, OS credential store 를 붙일 것인가?
- 실전/모의 domain 을 provider profile 로 분리할 것인가, SDK config 차이로만 둘
  것인가?
- KIS 일봉과 datago 일봉이 겹칠 때 router 기본 우선순위를 어떻게 둘 것인가?
- websocket 은 SDK 안의 별도 package 로 둘 것인가, 별도 client module 로 뺄 것인가?

# KIS Provider

## 개요

`kis` provider 는 한국투자증권 KIS Developers OpenAPI 를 `mwosa` 의 국내 시장
조회 provider 로 연결한다. 이번 구현 범위는 거래를 발생시키지 않는 조회 기능
중, `mwosa` core role 로 표현할 수 있는 시장 데이터 조회 기능을 우선 등록한다.

주문, 정정, 취소, 계좌 실거래 실행은 등록하지 않는다. 계좌/잔고 조회도 현재
client 에 구현되어 있지 않고 실행/risk/portfolio 경계가 아직 정해지지 않았으므로
이번 provider capability 에 넣지 않는다.

## Provider id 와 group

| provider id | group | capability | 비고 |
| --- | --- | --- | --- |
| `kis` | `domesticStockQuotation` | `quote_snapshot`, `daily_bar`, `intraday_bar`, `orderbook`, `trades` | 국내 주식/ETF/ETN 현재가, 일봉, 분봉, 10단계 호가, 시장 체결 조회 |
| `kis` | `domesticStockInstrument` | `instrument` | 상품/주식 기본정보 기반의 정확한 코드 조회 |

KIS 고유 API 이름은 public CLI verb 로 노출하지 않고 provider profile 의
operation metadata 로만 남긴다.

## KIS client 메서드 매핑

| client method | endpoint | 거래 발생 여부 | canonical resource 후보 | provider capability | 구현 상태 |
| --- | --- | --- | --- | --- | --- |
| `Token` | `/oauth2/tokenP` | 없음 | auth dependency | credential bootstrap | 내부 사용 |
| `Price` | `/uapi/domestic-stock/v1/quotations/inquire-price` | 없음 | quote snapshot | `quote_snapshot` / `price` | 구현 |
| `ETFETNPrice` | `/uapi/etfetn/v1/quotations/inquire-price` | 없음 | quote snapshot | `quote_snapshot` / `etfetnPrice` | 구현 |
| `Daily` | `/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice` | 없음 | daily bar | `daily_bar` / `daily` | 구현 |
| `Product` | `/uapi/domestic-stock/v1/quotations/search-info` | 없음 | instrument | `instrument` / `product` | 구현 |
| `Stock` | `/uapi/domestic-stock/v1/quotations/search-stock-info` | 없음 | instrument | `instrument` / `stock` | 구현 |
| `Intraday` | `/uapi/domestic-stock/v1/quotations/inquire-time-itemchartprice` | 없음 | intraday minute bar | `intraday_bar` / `intraday` | 구현 |
| `Orderbook` | `/uapi/domestic-stock/v1/quotations/inquire-asking-price-exp-ccn` | 없음 | orderbook snapshot | `orderbook` / `orderbook` | 구현 |
| `Trades` | `/uapi/domestic-stock/v1/quotations/inquire-ccnl` | 없음 | recent market trade prints | `trades` / `trades` | 구현 |
| `TimeTrades` | `/uapi/domestic-stock/v1/quotations/inquire-time-itemconclusion` | 없음 | time-filtered market trade prints | `trades` / `timeTrades` | 구현 |

`trades` 는 실제 주문 체결내역이나 계좌 execution 이 아니라 시장에서 발생한
체결 print 조회다. 계좌/잔고/손익 조회와 주문, 매수, 매도, 정정, 취소는
execution/risk/portfolio plane 경계가 정해질 때까지 이 provider capability 에
등록하지 않는다.

## CLI 표면

지원되는 public command 는 provider-neutral resource 이름을 사용한다.

```text
mwosa get quote 005930 --provider kis --security-type stock -o json
mwosa get quote 069500 --provider kis --security-type etf -o json
mwosa get intraday 005930 --provider kis --security-type stock --at 141200 -o json
mwosa get orderbook 005930 --provider kis --security-type stock -o json
mwosa list trades 005930 --provider kis --security-type stock -o json
mwosa list trades 005930 --provider kis --security-type stock --at 141200 -o json
mwosa ensure daily 005930 --provider kis --security-type stock --from 2026-05-01 --to 2026-05-08 -o json
mwosa inspect instrument 005930 --provider kis --security-type stock -o json
mwosa list instruments 069500 --provider kis --security-type etf -o json
mwosa doctor provider kis -o json
```

`kis` 일봉은 심볼 단위 조회다. `sync daily` 또는 `backfill daily` 처럼 provider
전체 배치를 수집하는 command 는 현재 KIS adapter 의 범위가 아니다.
분봉, 호가, 시장 체결 조회도 provider live/read-through 조회이며 현재 로컬
canonical storage 에 저장하지 않는다. `--at` 은 `HHMMSS` 또는 `HH:MM:SS` 형식의
범용 시간 anchor 로 받고, KIS adapter 내부에서 provider 요청 형식으로 바꾼다.

## 인증

기본 설정 경로는 기존 provider config 패턴을 따른다.

```text
mwosa login provider kis --app-key <key> --app-secret <secret>
```

지원되는 환경변수 fallback:

| field | env |
| --- | --- |
| `auth.app_key` | `MWOSA_KIS_APP_KEY`, `KIS_APP_KEY`, `APP_KEY` |
| `auth.app_secret` | `MWOSA_KIS_APP_SECRET`, `KIS_APP_SECRET`, `APP_SECRET` |
| `auth.access_token` | `MWOSA_KIS_ACCESS_TOKEN`, `KIS_ACCESS_TOKEN` |
| `virtual` | `MWOSA_KIS_VIRTUAL`, `KIS_VIRTUAL` |

`access_token` 이 없으면 첫 provider 호출 시 OAuth token 을 발급해서 client 내부에
저장한다. 이때 발급받은 token 은 CLI 실행이 끝나도 재사용할 수 있도록 별도 SQLite
파일에 저장한다. 파일은 market-data canonical DB 인 `mwosa.db` 또는 `--database`
파일과 섞지 않고, 같은 data directory 의 `provider-token-cache.sqlite` 에 둔다.

명시적으로 `auth.access_token`, `MWOSA_KIS_ACCESS_TOKEN`, `KIS_ACCESS_TOKEN` 이
주어지면 그 token 을 우선 사용하고 cache 를 읽거나 갱신하지 않는다. 이는 임시로
발급받은 token 을 직접 검증하거나 운영 환경에서 외부 secret 관리자가 token 을
주입하는 흐름을 보수적으로 존중하기 위한 정책이다.

cache key 는 `provider_id`, `auth_scope`, `environment`, app key hash 로 구성한다.
app key 원문은 저장하지 않는다. `virtual=true` 인 모의투자 환경과 실전 환경은 서로
다른 token 으로 취급하므로 token 이 교차 재사용되지 않는다.

만료 판단은 KIS tokenP 응답의 `access_token_token_expired` 를 우선 사용한다. 해당
값이 없으면 `expires_in` 과 발급 시각으로 보조 계산한다. 만료 2분 전부터는 만료된
token 으로 보고 `/oauth2/tokenP` 를 다시 호출한 뒤 cache 를 갱신한다. cache miss 는
정상 경로이며 tokenP 발급으로 이어진다. cache read/write 실패는 성공처럼 숨기지 않고
명시적인 error 로 반환한다.

secret 값과 token 값은 doctor/login/list/config 출력에 표시하지 않는다. 출력에는
configured 여부만 남긴다.

# KIS Provider

## 개요

`kis` provider 는 한국투자증권 KIS Developers OpenAPI 를 `mwosa` 의 국내 시장
조회 provider 로 연결한다. 이번 구현 범위는 거래를 발생시키지 않는 조회 기능
중, 현재 `mwosa` core role 로 표현할 수 있는 기능을 우선 등록한다.

주문, 정정, 취소, 계좌 실거래 실행은 등록하지 않는다. 계좌/잔고 조회도 현재
client 에 구현되어 있지 않고 실행/risk/portfolio 경계가 아직 정해지지 않았으므로
이번 provider capability 에 넣지 않는다.

## Provider id 와 group

| provider id | group | capability | 비고 |
| --- | --- | --- | --- |
| `kis` | `domesticStockQuotation` | `quote_snapshot`, `daily_bar` | 국내 주식/ETF/ETN 현재가와 심볼 단위 일봉 조회 |
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
| `Intraday` | `/uapi/domestic-stock/v1/quotations/inquire-time-itemchartprice` | 없음 | intraday bar | TODO: `intraday_bar` | 보류 |
| `Orderbook` | `/uapi/domestic-stock/v1/quotations/inquire-asking-price-exp-ccn` | 없음 | orderbook | TODO: `orderbook` | 보류 |
| `Trades` | `/uapi/domestic-stock/v1/quotations/inquire-ccnl` | 없음 | market trades | TODO: `trades` | 보류 |
| `TimeTrades` | `/uapi/domestic-stock/v1/quotations/inquire-time-itemconclusion` | 없음 | market trades | TODO: `trades` | 보류 |

보류 항목은 조회 API 이지만 아직 `providers/core` 에 범용 role 과 출력 모델이 없다.
KIS 전용 CLI verb 를 만들지 않기 위해 이번 범위에서는 문서화된 TODO 로 남긴다.

## CLI 표면

지원되는 public command 는 provider-neutral resource 이름을 사용한다.

```text
mwosa get quote 005930 --provider kis --security-type stock -o json
mwosa get quote 069500 --provider kis --security-type etf -o json
mwosa ensure daily 005930 --provider kis --security-type stock --from 2026-05-01 --to 2026-05-08 -o json
mwosa inspect instrument 005930 --provider kis --security-type stock -o json
mwosa list instruments 069500 --provider kis --security-type etf -o json
mwosa doctor provider kis -o json
```

`kis` 일봉은 심볼 단위 조회다. `sync daily` 또는 `backfill daily` 처럼 provider
전체 배치를 수집하는 command 는 현재 KIS adapter 의 범위가 아니다.

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
저장한다. secret 값은 doctor/login 출력에서 configured 여부만 보이고 값은 출력하지
않는다.

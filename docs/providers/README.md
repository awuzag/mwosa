# Provider Compatibility Guide

이 문서는 `mwosa` 가 어떤 금융 데이터 provider 와 호환되는 방향으로 설계되는지 설명한다.

`mwosa` 의 목표는 provider 별 API 차이를 CLI 바깥으로 드러내지 않고, 사용자가 같은 명령어로 시세, 종목 정보, 재무, 공시, 뉴스, 거시 지표를 조회할 수 있게 하는 것이다.

```text
mwosa inspect AAPL --provider yahoo-finance
mwosa get quote 005930 --provider kis
mwosa get daily 491820 --prefer-provider datago
```

이 문서는 현재 구현 완료 목록이 아니라 **호환 예정 provider map** 이다. 실제 지원 여부는 이후 `mwosa list providers` 와 `mwosa inspect provider <name>` 에서 확인할 수 있게 한다.

## 최소 provider 요구사항

`mwosa` 로 실제 시장 데이터를 조회하려면 최소 1개 이상의 provider 가 활성화되어 있어야 한다.

provider 가 하나도 없어도 `help`, `version`, `config`, `list providers` 같은 로컬 명령은 실행할 수 있다. 하지만 `inspect`, `get`, `ensure`, `sync`, `screen`, `compare`, `calc` 처럼 시장 데이터가 필요한 명령은 해당 capability 를 제공하는 provider 가 필요하다.

예를 들어 `mwosa inspect 005930` 은 현재가와 종목 정보를 제공할 수 있는 provider 가 필요하고, `mwosa get macro CPI` 는 macro capability 를 제공하는 provider 가 필요하다.

## 호환성 상태

| 상태 | 의미 |
| --- | --- |
| `core` | `mwosa` 가 우선적으로 호환을 맞출 provider 다. |
| `planned` | core 이후 확장 대상으로 둘 provider 다. |
| `reference` | 가격 데이터보다 식별자, 공시, macro, 뉴스 같은 보조 데이터를 위한 provider 다. |
| `deferred` | 장기적으로 열어두지만 초기 주식/ETF 리서치에는 후순위인 provider 다. |

## Provider 이름 규칙

CLI 에서 사용하는 provider 이름은 짧은 lowercase 이름을 기본으로 한다. 핵심 provider 는 가능하면 `-` 나 `/` 를 붙이지 않고 하나의 id 로 둔다. 단, 공공데이터포털처럼 활용신청과 credential 단위가 사용자에게 별도로 드러나는 경우에는 `datago-corpfin` 같은 계열 하위 provider 를 둘 수 있다.

| 예시 | 설명 |
| --- | --- |
| `kis` | 한국투자증권 KIS Developers |
| `datago` | 공공데이터포털 |
| `yahoo-finance` | Yahoo Finance 계열 market data |
| `alpha-vantage` | Alpha Vantage API |
| `sec-edgar` | SEC EDGAR filing data |

## Provider group 규칙

provider 가 하나의 API 묶음만 의미한다고 가정하지 않는다. 공공데이터포털처럼 같은 사이트 안에서도 API 활용신청, 인증 범위, 도메인, 응답 구조가 나뉘면 provider 아래에 group 을 둔다.

group 은 사용자가 매번 입력하는 이름이 아니라 registry, auth 점검, provenance, fallback 설명에 쓰는 내부 단위다.

```text
provider: datago
group: securitiesProductPrice
operations: getETFPriceInfo, getETNPriceInfo, getELWPriceInfo

provider: datago
group: stockPrice
operations: getStockPriceInfo
```

provider 와 group 을 한 문자열로 합쳐 `datago/securitiesProductPrice` 나 `datago-securities-product-price` 처럼 만들지 않는다. CLI 에서는 보통 `--provider datago` 를 유지하고, 세부 API 범위는 `inspect provider datago` 또는 `inspect auth` 에서 group 단위로 보여준다. 별도 활용신청 단위는 `datago-corpfin` 처럼 하위 provider 로 분리하되, 그 안에서도 group 은 provenance 와 role registration 단위로 남긴다.

## 국내 시장 호환 계획

국내 주식과 ETF 리서치는 `mwosa` 의 초기 핵심 범위다.

| provider | 상태 | 호환 범위 | 주요 capability | 비고 |
| --- | --- | --- | --- | --- |
| `kis` | `core` | 한국 주식, ETF, ETN | `quote`, `candles`, `instrument`, `intraday_bar`, `orderbook`, `trades` | 국내 현재가, 일봉, 분봉, 호가, 시장 체결 조회의 핵심 provider 로 둔다. 주문/계좌 기능은 초기 범위에서 제외한다. |
| `datago` | `core` | 한국 ETF, ETN, ELW, 주식 | `candles`, `instrument` | group별 공공 OpenAPI 를 묶는 provider 다. 실시간 quote provider 로 보지 않는다. |
| `krx` | `planned` | 한국 거래소, 시장, 종목, 지수 | `instrument`, `candles`, `index`, `market` | 국내 종목 reference 와 시장 calendar 보강용이다. |
| `kiwoom` | `planned` | 한국 주식, ETF | `quote`, `candles`, `orderbook`, `trades` | 국내 실시간/브로커 데이터 확장 대상으로 둔다. 여기서 `trades` 는 시장 체결 데이터이며 계좌 체결내역이 아니다. |
| `opendart` | `reference` | 한국 상장사 공시 | `filings`, `company_registry`, `financials`, `fundamentals` | 기업 공시와 재무 원천 자료 보강용이다. 기존 `dart` 표기는 구현 id가 아니라 이전 자리 표시자로 본다. |
| `ecos` | `reference` | 한국 거시 지표, 환율, 금리 | `macro`, `fx`, `rates` | 한국 macro context 보강용이다. |

## 국내 provider group 계획

| provider | group | 상태 | 주요 operation | capability | 비고 |
| --- | --- | --- | --- | --- | --- |
| `datago` | `securitiesProductPrice` | `core` | `getETFPriceInfo`, `getETNPriceInfo`, `getELWPriceInfo` | `candles`, `instrument` | 금융위원회 증권상품시세정보 OpenAPI 다. |
| `datago` | `stockPrice` | `core` | `getStockPriceInfo` | `candles`, `instrument` | 금융위원회 주식시세정보 OpenAPI 다. |
| `datago` | `krxListedInstrument` | `planned` | KRX상장종목정보 | `instrument` | 종목코드와 `crno` 같은 reference identifier 를 canonical instrument store 에 저장하는 source 로 둔다. |
| `datago` | `corporateFinancial` | `planned` | 기업 재무 정보 | `fundamentals`, `financials` | `crno` 데이터 의존성을 가진다. |
| `kis` | `domesticStockQuotation` | `core` | `price`, `daily`, `etfetnPrice`, `etfComponentStockPrice`, `intraday`, `orderbook`, `trades`, `timeTrades` | `quote`, `candles`, `composition`, `intraday_bar`, `orderbook`, `trades` | KIS client 의 국내 주식/ETF/ETN 현재가, ETF 구성종목시세 기반 구성 종목 조회, 심볼 단위 일봉, 분봉, 10단계 호가, 시장 체결 조회를 연결한다. |
| `kis` | `domesticStockInstrument` | `core` | `product`, `stock` | `instrument` | 상품기본조회와 주식기본조회 기반의 정확한 코드 조회를 연결한다. |
| `krx` | `etpDailyTrade` | `planned` | KRX ETF/ETN/ELW 일별매매정보 | `candles`, `instrument` | KRX OpenAPI 승인 후 보조 후보로 둔다. |
| `krx` | `stockDailyTrade` | `planned` | KRX 주식 일별매매정보 | `candles`, `instrument` | datago 와 중복되는 범위는 router 우선순위로 다룬다. |

## 글로벌 시장 호환 계획

글로벌 provider 는 국내 MVP 이후 확장한다. 단, symbol coverage 검증과 JSON 출력 schema 설계에는 초기부터 일부를 사용할 수 있다.

| provider | 상태 | 호환 범위 | 주요 capability | 비고 |
| --- | --- | --- | --- | --- |
| `yahoo-finance` | `planned` | 글로벌 주식, ETF, 지수 | `quote`, `candles`, `instrument`, `fundamentals` | 넓은 symbol coverage 검증용으로 유용하다. 공식 계약형 API 로 보기 어려우므로 운영 의존도는 낮게 둔다. |
| `alpha-vantage` | `planned` | 글로벌 주식, ETF, FX, crypto | `quote`, `candles`, `fundamentals`, `fx` | 단순 HTTP API 와 샘플이 풍부해 adapter 실험에 적합하다. |
| `polygon` | `planned` | 미국 주식, 옵션, FX, crypto | `quote`, `candles`, `trades`, `news` | 미국 시장 고품질 시세 확장 대상이다. |
| `finnhub` | `planned` | 글로벌 주식, ETF, 뉴스, 이벤트 | `quote`, `candles`, `fundamentals`, `earnings`, `news` | 시세와 뉴스/이벤트를 함께 다루는 확장 대상이다. |
| `fmp` | `planned` | 글로벌 주식, ETF, 재무 | `quote`, `fundamentals`, `financials`, `earnings` | 정규화된 재무제표와 밸류에이션 데이터 확장 대상이다. |
| `tiingo` | `planned` | 미국/글로벌 주식, ETF, 뉴스 | `candles`, `news`, `fundamentals` | EOD 와 뉴스 보강 대상이다. |
| `twelve-data` | `planned` | 글로벌 주식, ETF, FX, crypto | `quote`, `candles`, `fx` | 다양한 시장의 candle adapter 확장 대상이다. |
| `eodhd` | `planned` | 글로벌 주식, ETF, 지수 | `candles`, `fundamentals`, `splits`, `dividends` | 글로벌 EOD 확장 대상이다. |
| `stooq` | `planned` | 글로벌 지수, 주식, FX | `candles`, `index`, `fx` | 무료 EOD 보조 provider 로 검토한다. |
| `nasdaq-data-link` | `planned` | curated datasets | `candles`, `macro`, `fundamentals` | dataset 별 라이선스와 비용을 별도로 확인한다. |

## 공시와 재무 데이터 호환 계획

공시와 재무 데이터는 `inspect`, `get filings`, `get financials`, `get fundamentals` 같은 명령에서 보조 provider 로 사용한다.

| provider | 상태 | 호환 범위 | 주요 capability | 비고 |
| --- | --- | --- | --- | --- |
| `datago-corpfin` | `core` | 한국 기업 재무정보, KRX 상장종목 식별자 | `financials` | KRX 상장종목정보 API 로 종목명/회사명을 우선 법인등록번호로 해석하고, 보조로 단축코드/ISIN 을 지원한 뒤 금융위원회 기업 재무정보 API 를 조회한다. |
| `opendart` | `reference` | 한국 상장사 공시 | `filings`, `company_registry`, `financials`, `fundamentals` | 국내 기업 리서치의 기본 filing provider 다. `corp_code`와 상장회사 `stock_code`를 별도 식별자로 보존한다. |
| `sec-edgar` | `reference` | 미국 상장사 공시 | `filings`, `instrument` | 10-K, 10-Q, 8-K 같은 미국 공시용이다. |
| `fmp` | `planned` | 글로벌 재무 | `financials`, `fundamentals`, `earnings` | raw filing 보다 normalized financial API 로 본다. |
| `finnhub` | `planned` | 글로벌 재무, 이벤트 | `fundamentals`, `earnings`, `news` | 실적 이벤트와 뉴스 context 보강용이다. |

## 거시 지표와 환율 호환 계획

거시 지표는 종목 시세 provider 와 분리해서 다룬다.

| provider | 상태 | 호환 범위 | 주요 capability | 비고 |
| --- | --- | --- | --- | --- |
| `ecos` | `reference` | 한국 macro, 환율, 금리 | `macro`, `fx`, `rates` | 한국 시장 context 를 위한 기본 macro provider 다. |
| `fred` | `reference` | 미국 macro, 금리 | `macro`, `rates` | 미국 금리, 물가, 고용, 경기 지표용이다. |
| `world-bank` | `reference` | 글로벌 macro | `macro` | 국가별 장기 macro 비교용이다. |
| `oecd` | `reference` | 글로벌 macro | `macro` | 선진국 macro 비교용이다. |
| `ecb` | `reference` | 유럽 환율, 금리 | `macro`, `fx`, `rates` | EUR 기준 환율과 유럽 macro 보강용이다. |

## 뉴스 호환 계획

뉴스 provider 는 가격 데이터처럼 정본으로 저장하기보다, 특정 시점의 research context 로 취급한다.

| provider | 상태 | 호환 범위 | 주요 capability | 비고 |
| --- | --- | --- | --- | --- |
| `finnhub` | `planned` | 글로벌 시장 뉴스 | `news`, `earnings` | 종목 뉴스와 이벤트 보강용이다. |
| `newsapi` | `planned` | 일반 뉴스 | `news` | 금융 특화 데이터가 아니므로 필터링이 필요하다. |
| `gdelt` | `planned` | 글로벌 뉴스 이벤트 | `news`, `events` | 거시 이벤트 흐름을 보는 확장 대상이다. |
| `naver-news` | `planned` | 한국 뉴스 | `news` | 국내 종목 뉴스 확장 대상이며 사용 조건 확인이 필요하다. |

## 식별자와 reference 호환 계획

식별자 provider 는 가격을 주는 provider 가 아니라 symbol resolution 을 돕는 provider 다.

| provider | 상태 | 호환 범위 | 주요 capability | 비고 |
| --- | --- | --- | --- | --- |
| `datago-corpfin` | `core` | KRX 상장종목 기본정보 | `financials` internal resolution | `krxListedInfo/getItemInfo` 로 단축코드, ISIN, 종목명에서 법인등록번호를 찾고 Datago 재무제표 조회에 사용한다. |
| `openfigi` | `reference` | 글로벌 금융상품 식별자 | `instrument` | ticker, FIGI, exchange, security type 매핑 보강용이다. |

## Crypto 호환 계획

초기 목표가 주식/ETF 리서치라면 crypto 는 후순위다. 다만 CLI resource model 은 crypto provider 를 나중에 붙일 수 있게 열어둔다.

| provider | 상태 | 호환 범위 | 주요 capability | 비고 |
| --- | --- | --- | --- | --- |
| `coingecko` | `deferred` | crypto spot | `quote`, `candles`, `instrument` | crypto 기본 시세 확장 대상이다. |
| `binance` | `deferred` | crypto exchange | `quote`, `candles`, `orderbook`, `trades` | 거래소 종속 symbol 체계를 별도로 다뤄야 한다. |
| `coinbase` | `deferred` | crypto exchange | `quote`, `candles`, `trades` | USD 기반 crypto 시세 확장 대상이다. |

## Aggregator 호환 계획

| provider | 상태 | 호환 범위 | 주요 capability | 비고 |
| --- | --- | --- | --- | --- |
| `openbb` | `planned` | 여러 데이터 provider 통합 | `quote`, `candles`, `fundamentals`, `news`, `macro` | 직접 provider 구현 전 탐색에는 유용하지만, CLI core 에 강하게 결합하지 않는다. |

## Capability 별 호환 요약

| capability | 우선 provider | 확장 provider |
| --- | --- | --- |
| `quote` | `kis` | `kiwoom`, `yahoo-finance`, `alpha-vantage`, `polygon`, `finnhub`, `twelve-data` |
| `composition` | `kis` | `krx`, `yahoo-finance`, `openfigi` |
| `candles` | `kis`, `datago` | `krx`, `yahoo-finance`, `alpha-vantage`, `polygon`, `stooq`, `eodhd` |
| `instrument` | `datago`, `krx`, `kis` | `openfigi`, `yahoo-finance`, `fmp` |
| `intraday_bar` | `kis` | `kiwoom`, `polygon`, `twelve-data` |
| `orderbook` | `kis` | `kiwoom`, `binance` |
| `trades` | `kis` | `kiwoom`, `polygon`, `binance`, `coinbase` |
| `fundamentals` | `opendart` | `fmp`, `finnhub`, `alpha-vantage`, `tiingo` |
| `financials` | `datago-corpfin`, `opendart` | `fmp`, `sec-edgar`, `finnhub`, `alpha-vantage` |
| `filings` | `opendart`, `sec-edgar` | `finnhub` |
| `news` | `finnhub` | `newsapi`, `gdelt`, `naver-news` |
| `macro` | `ecos`, `fred` | `world-bank`, `oecd`, `ecb` |
| `fx` | `ecos` | `alpha-vantage`, `twelve-data`, `ecb`, `stooq` |

## Composition storage 방향

`composition` capability 는 ETF 같은 상품의 구성 종목 정보를 canonical aggregate 로
다룬다. provider adapter 는 provider-native 구성 종목 응답에서 구성 정보만
`Composition` 으로 옮기고, canonical storage 는 raw 응답이나 KIS DTO 를 저장하지 않는다.

저장소는 `market_v2`, `instrument_v2`, `provider_source_v2` 를 공유 식별자 계층으로
재사용한다. 구성 aggregate 는 `composition_observation_v1` 과 `composition_member_v1` 로
저장한다. 구성 정보 조회 결과에 현재가, 등락, 거래량 같은 값이 함께 들어오더라도
composition 저장 대상에는 포함하지 않는다. 따라서 기본 constituents 출력과 저장 aggregate
에는 provider raw field 나 API 전용 `etf_snapshot` 모델이 섞이지 않는다.

## Compatibility contract

각 provider adapter 는 가능한 한 같은 public CLI 경험을 제공해야 한다.

- provider 고유 필드는 canonical output 을 깨지 않는 extension 으로 둔다.
- provider 가 지원하지 않는 capability 는 명확히 `unsupported` 로 표기한다.
- provider 별 rate limit, 지연 시간, 최신성은 `inspect provider <name>` 에서 확인할 수 있게 한다.
- 같은 capability 를 여러 provider 가 지원하면 `--provider` 는 강제 선택, `--prefer-provider` 는 우선 선택으로 동작한다.
- 장애, quota 초과, symbol 미지원은 조용히 삼키지 않고 결과에 드러낸다.

## 관련 문서

- `docs/adr/0003-provider-group-role-registration.md`
- `docs/providers/datago/README.md`
- `docs/providers/krx/README.md`
- `docs/providers/opendart/README.md`

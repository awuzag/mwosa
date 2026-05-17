# OpenDART Provider

`opendart` provider 는 한국 기업의 공시, 회사 식별자, 재무제표를 다룬다.
가격 provider 가 아니므로 `daily_bar`, `quote`, `intraday_bar` 역할은 제공하지
않는다.

## 식별자

OpenDART 의 기본 키는 `corp_code` 다. KRX 종목코드와 같은 값이 아니다.

| 필드 | 의미 |
| --- | --- |
| `corp_code` | OpenDART 공시대상회사 고유번호다. 재무제표와 공시 API 호출에 사용한다. |
| `stock_code` | 상장회사에만 제공되는 6자리 종목코드다. KRX symbol 과 같은 축으로 다루되 `corp_code` 와 별도로 보존한다. |
| `corp_name` | OpenDART 회사명이다. |
| `corp_eng_name` | OpenDART 영문 회사명이다. |
| `modify_date` | OpenDART 회사 정보 최종 변경일이다. |

현재 구현은 `corpCode.xml` 을 받아 전환용 `opendart_companies` 와 canonical
회사 식별자 graph 에 함께 upsert 한다.

- `company_v1`
- `company_identifier_v1`
- `instrument_company_link_v1`

`stock_code` 로 공시나 재무를 조회할 때는 먼저 `corp_code` 로 해석한 뒤
OpenDART API 를 호출한다. `corp_code`, `stock_code`, `crno`, `isin` 같은 값은
canonical key 가 아니라 identifier 로 다룬다.

## 현재 CLI 범위

```bash
mwosa doctor provider opendart -o json
mwosa inspect provider opendart -o json

mwosa sync companies --provider opendart --listed-only -o json
mwosa search companies 삼성전자 -o table
mwosa inspect company 005930 -o json
mwosa get company-identifiers 005930 -o table

mwosa list filings 005930 --provider opendart --from 2025-01-01 --to 2025-12-31 -o json
mwosa list filings --corp-code 00126380 --provider opendart --from 2025-01-01 -o json
mwosa get filing 20250318000935 --provider opendart -o json
mwosa get filing 20250318000935 --provider opendart --include-payload -o json

mwosa sync financials statements 005930 --provider opendart --from 2023 --to 2025 --period quarter -o json
mwosa get financials statements 005930 --year 2025 --period quarter --statement income_statement -o table
mwosa calc financials metrics 005930 --window 3y --period quarter -o json
mwosa get financials metrics 005930 --window 3y --period quarter -o table
mwosa calc financials valuation 005930 --as-of latest -o json
mwosa get financials valuation 005930 --as-of latest -o table

mwosa sync financials dividends 005930 --provider opendart --from 2023 --to 2025 -o json
mwosa get financials dividends 005930 --window 3y -o table
mwosa sync financials facts 005930 --provider opendart --from 2023 --to 2025 -o json
mwosa get financials facts 005930 --year 2025 -o table
mwosa get financials health 005930 --window 3y -o table

mwosa sync events 005930 --provider opendart --from 2023-01-01 -o json
mwosa list events 005930 --provider opendart --from 2023-01-01 -o table

mwosa inspect stock 005930 --section profile,investment,financials,dividends -o table
mwosa inspect stock 005930 --section all -o json
```

OpenDART API catalog 는 일반 사용자 workflow 가 아니라 진단용 surface 로 둔다.

```bash
mwosa list provider-apis opendart -o table
mwosa get provider-raw-snapshots --provider opendart -o table
mwosa get provider-raw opendart corpCode -o json
```

machine-readable 출력은 stdout 으로만 나가며, 인증키와 진단 정보는 결과에
섞지 않는다.

`get filing <rcept-no>` 는 OpenDART `document.xml` file response 를 조회한다.
기본 출력은 provider, operation, 접수번호, content type, byte size, sha256 같은
metadata 만 포함한다. `--include-payload` 를 지정하면 JSON/NDJSON 출력에만 base64
payload 를 포함할 수 있다. binary raw bytes 를 stdout 에 직접 쓰지 않는다.

## 인증

인증키는 `OPENDART_API_KEY` 를 우선 사용한다. 로컬 설정 파일에는
`providers.opendart.auth.api_key` 로 저장할 수 있지만, 출력에는 raw key 값을
표시하지 않는다.

```bash
export OPENDART_API_KEY="..."
mwosa doctor provider opendart -o json
```

## 현재 범위와 이후 범위

현재 범위는 provider foundation, 회사 식별자 매핑, 공시 검색, 단일회사 전체
재무제표 저장/조회, financial metric 계산, valuation snapshot 계산, periodic
report fact, material event, stock summary 조회다. 전체 85개 OpenDART API 를 한
번에 canonical schema 로 정규화하지 않는다.

canonical 로 안정화되지 않은 응답은 기존 `provider_raw_snapshots` 흐름을
재사용한다. 공시 원문 문서는 `get filing` 의 metadata-first raw file escape hatch
로 제공한다. XBRL full parser, 추가 periodic report fact, 주식 수/자기주식 세부
확장, 최대주주/임원 지분 변화 확장, 이벤트성 공시 coverage 확대는 후속 범위로
둔다.

## 데이터 의존 순서

저장된 canonical 데이터를 읽는 명령은 선행 sync/calc 가 필요하다.

| 조회 명령 | 선행 명령 |
| --- | --- |
| `get financials statements` | `sync financials statements` |
| `get financials metrics`, `get financials health` | `calc financials metrics` |
| `get financials valuation` | 저장된 daily price/market cap 과 `calc financials valuation` |
| `get financials dividends` | `sync financials dividends` |
| `get financials facts` | `sync financials facts` |
| `list events`, `inspect stock --section events` | `sync events` |

missing data 는 빈 성공으로 숨기지 않고 `stored ... not found` 계열 error 또는
`missing` section 으로 드러낸다. 계산 불가는 `uncomputable_reason` 으로 남긴다.

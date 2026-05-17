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

초기 구현은 `corpCode.xml` 을 받아 `opendart_companies` 에 upsert 한다.
`stock_code` 로 공시나 재무를 조회할 때는 먼저 `corp_code` 로 해석한 뒤
OpenDART API 를 호출한다.

## 초기 CLI 범위

```bash
mwosa doctor provider opendart -o json
mwosa inspect provider opendart -o json

mwosa sync companies --provider opendart --listed-only -o json
mwosa search companies 005930 --provider opendart -o json

mwosa list filings 005930 --provider opendart --from 2025-01-01 --to 2025-12-31 -o json
mwosa list filings --corp-code 00126380 --provider opendart --from 2025-01-01 -o json
mwosa get financials 005930 --provider opendart --year 2025 -o json
```

OpenDART API catalog 는 일반 사용자 workflow 가 아니라 진단용 surface 로 둔다.

```bash
mwosa list provider-apis opendart -o table
```

machine-readable 출력은 stdout 으로만 나가며, 인증키와 진단 정보는 결과에
섞지 않는다.

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
재무제표 조회다. 전체 85개 OpenDART API 를 한 번에 canonical schema 로
정규화하지 않는다.

canonical 로 안정화되지 않은 응답은 기존 `provider_raw_snapshots` 흐름을
재사용한다. 공시 원문 문서, XBRL full parser, 지배구조/이벤트 전체 API
정규화는 후속 범위로 둔다.

# krx Provider Implementation Notes

## 서비스 신청 모델

KRX OPEN API 상세 페이지는 서비스마다 `BO_ID` 를 가진다. `API 이용신청` 동작도
현재 상세 페이지의 `BO_ID` 와 이용 기간, 이용 목적을 제출하는 구조다. 따라서
`mwosa` 에서는 `krx` provider 하나를 켰다고 해서 모든 KRX operation 이 승인된
것으로 보면 안 된다.

구현 방향:

- provider id 는 `krx` 로 둔다.
- 서비스 승인 단위는 `api_id` 또는 `BO_ID` 로 별도 추적한다.
- role registration 과 raw 호출은 승인/활성화된 서비스만 허용한다.
- 승인되지 않은 서비스 호출은 조용히 빈 결과로 만들지 말고 `unsupported` 또는
  `not configured` 계열 error 로 드러낸다.

## 인증과 호출 메모

상세 페이지의 샘플 예제는 인증키를 HTTP request header 의 `AUTH_KEY` 로 전달한다.
샘플 화면에 노출되는 테스트 키는 문서화하지 않고, 실제 provider 설정에서는 사용자
발급 키만 받는다.

상세 페이지에서 확인되는 샘플 URL 패턴:

```text
https://data-dbg.krx.co.kr/svc/sample/apis/{path}/{api_id}
AUTH_KEY: <issued-auth-key>
```

실제 운영 endpoint 는 이용신청 후 받은 개발 명세서에서 다시 확인해야 한다. client
구현 시에는 샘플 URL 과 운영 URL 을 config 로 분리할 수 있게 둔다.

## group 후보

KRX 화면의 `path` 는 client package 안에서 1차 group 으로 쓰기 좋다. 다만 실제
활용신청은 더 세밀한 서비스별 단위이므로 group 과 service scope 를 분리한다.

| group | `path` | 주요 API | capability 후보 |
| --- | --- | --- | --- |
| `indexDailyTrade` | `idx` | `krx_dd_trd`, `kospi_dd_trd`, `kosdaq_dd_trd` | `index`, `daily_bar` |
| `stockDailyTrade` | `sto` | `stk_bydd_trd`, `ksq_bydd_trd`, `knx_bydd_trd` | `daily_bar`, `instrument` |
| `stockInstrument` | `sto` | `stk_isu_base_info`, `ksq_isu_base_info`, `knx_isu_base_info` | `instrument` |
| `etpDailyTrade` | `etp` | `etf_bydd_trd`, `etn_bydd_trd`, `elw_bydd_trd` | `daily_bar`, `instrument` |
| `bondDailyTrade` | `bon` | `kts_bydd_trd`, `bnd_bydd_trd`, `smb_bydd_trd` | `daily_bar`, `instrument` |
| `derivativesDailyTrade` | `drv` | `fut_bydd_trd`, `opt_bydd_trd`, 주식선물/옵션 계열 | `daily_bar`, `instrument` |
| `commodityDailyTrade` | `gen` | `oil_bydd_trd`, `gold_bydd_trd`, `ets_bydd_trd` | `daily_bar`, `market` |
| `esgReference` | `esg` | `esg_etp_info`, `sri_bond_info`, `esg_index_info` | `instrument`, `index`, `reference` |

## datago 와의 관계

`datago` 와 `krx` 는 일부 데이터 범위가 겹칠 수 있다. 특히 ETF/ETN/ELW 와 주식
일별 데이터는 canonical `daily_bar` 관점에서 중복 source 가 될 수 있다.

처리 방향:

- `datago` 는 공공데이터포털 API provider 로 유지한다.
- `krx` 는 KRX OPEN API provider 로 분리한다.
- provenance 에 `provider`, `group`, `api_id`, `BO_ID` 를 남긴다.
- 같은 symbol/date 데이터가 여러 provider 에서 들어오면 storage upsert 정책과
  provider priority 를 별도로 결정한다.

## 구현된 v1 범위

client module 은 31개 서비스를 모두 typed client 로 구현했다. provider adapter 는
현재 `mwosa` 의 국내 리서치 목적에 직접 맞는 ETP 일별매매정보와 주식 일별매매정보,
주식 종목기본정보를 canonical role 로 연결한다.

v1 후보:

- `etpDailyTrade`: `etf_bydd_trd`, `etn_bydd_trd`, `elw_bydd_trd`
- `stockDailyTrade`: `stk_bydd_trd`, `ksq_bydd_trd`, `knx_bydd_trd`
- `stockInstrument`: `stk_isu_base_info`, `ksq_isu_base_info`,
  `knx_isu_base_info`

raw snapshot 후보:

- 지수, 채권, 파생상품, 일반상품, ESG 서비스는 `mwosa get krx <api-id>` 로
  조회하고 `mwosa sync krx <api-id>` 로 `provider_raw_snapshots` table 에 저장한다.
- 이 데이터는 canonical schema 와 security type 모델이 정리된 뒤 별도 role/storage 로
  승격한다.

## client module 후보

`RULE.md` 의 provider client 분리 원칙에 맞추면 KRX 도 독립 Go module 로 시작하는
편이 좋다.

```text
github.com/awuzag/krx
providers/krx
storage/providerraw
```

client module 이 소유할 것:

- base URL 과 sample URL 전환
- `AUTH_KEY` header 설정
- `path` / `api_id` 별 request builder
- KRX native response parsing
- 서비스별 remote error mapping
- fake HTTP transport 또는 `httptest` 기반 단위 테스트

provider adapter 가 소유할 것:

- `mwosa` provider config 를 client config 로 변환
- 승인된 `api_id` 만 role registry 에 등록
- KRX native record 를 canonical model 로 normalize
- provider priority, provenance, unsupported capability 설명

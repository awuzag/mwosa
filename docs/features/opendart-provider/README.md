# OpenDART Provider 구현 목표

## 목적

`opendart` provider 는 한국 기업의 공시, 재무제표, 지배구조, 자본정책,
주요 이벤트를 `mwosa` 안에서 조회하고 분석하기 위한 feature 다.

가격, 거래량, 시가총액 같은 시장 데이터는 KRX, Datago, KIS provider 가 맡고,
OpenDART 는 기업이 제출한 공시와 재무 원천 자료를 맡는다. 이 둘을 결합하면
단순 가격 모멘텀을 넘어 다음 질문에 답할 수 있다.

- 매출과 이익이 실제로 성장하고 있는가?
- 가격 상승이 재무 성과와 자본 효율로 설명되는가?
- 부채, 감사의견, 자금조달, 소송 같은 위험 신호가 있는가?
- 최대주주, 임원, 주요주주 지분 변화가 기업 판단에 영향을 주는가?
- 배당, 자기주식, 유상증자, CB/BW 발행 같은 자본정책이 주주에게 우호적인가?

이 문서는 1차 구현 범위만 정하는 문서가 아니다. OpenDART provider 로 최종적으로
갖춰야 할 구현 목표를 기록한다. 여기에 적힌 항목은 순서대로 실행하기 위한
계획이 아니라, `mwosa` 에 필요한 최종 상태의 목록이다.

## 원천 데이터

OpenDART 공식 개발가이드 기준 API 는 2026-05-14 로컬 조사 문서 기준 85개다.
현재 별도 SDK repository 인 `/Users/danghamo/Documents/gituhb/opendart` 는
공식 개발가이드에서 추출한 OpenAPI 문서와 생성된 SDK 표면을 가진다.

| 공식 그룹 | API 수 | mwosa 관점 |
| --- | ---: | --- |
| 공시정보 | 4 | 회사 식별자, 기업개황, 공시 검색, 원문 문서 |
| 정기보고서 주요정보 | 30 | 배당, 주식 수, 자기주식, 주주, 임직원, 감사, 보수, 자금 사용 |
| 정기보고서 재무정보 | 7 | 주요 계정, 전체 재무제표, XBRL, 재무지표 |
| 지분공시 종합정보 | 2 | 대량보유, 임원ㆍ주요주주 소유 보고 |
| 주요사항보고서 주요정보 | 36 | 증자, 감자, CB/BW/EB, 자기주식, 합병, 분할, 소송, 부도, 영업양수도 |
| 증권신고서 주요정보 | 6 | 지분증권, 채무증권, 합병, 분할, 주식교환ㆍ이전 |

관련 로컬 문서:

- `/Users/danghamo/Documents/gituhb/opendart/README.md`
- `/Users/danghamo/Documents/gituhb/opendart/docs/apis/official-inventory.md`
- `/Users/danghamo/Documents/gituhb/opendart/docs/apis/README.md`

## 핵심 방향

OpenDART provider 는 가격 provider 가 아니다. `daily_bar`, `quote`,
`intraday_bar` 역할을 제공하지 않고, 기업 reference 와 공시 기반 분석 역할을
제공한다.

초기 provider id 는 `opendart` 를 우선 후보로 둔다. 기존
`docs/providers/README.md` 에 있는 `dart` 표기는 이전 자리 표시자로 보고,
실제 구현 전에 provider id 를 `opendart` 로 통일할지, CLI alias 로 `dart` 를
남길지 결정한다.

2026-05-17 초기 구현에서는 provider id 를 `opendart` 로 확정하고 `dart` alias 는
추가하지 않았다. 이전 문서의 `dart` 표기는 구현 id 가 아니라 자리 표시자로 본다.

OpenDART 의 `corp_code` 는 KRX 종목코드가 아니다. 상장회사에는 `stock_code` 가
함께 제공되지만, API 호출의 기본 키는 `corp_code` 다. 따라서 provider 는
식별자 매핑을 먼저 안정화해야 한다.

```text
KRX symbol / stock_code
  -> OpenDART corp_code
  -> filings / financials / ownership / events
  -> KRX, Datago, KIS market data 와 결합
```

## Provider Capability 목표

OpenDART 전체 API 를 한 번에 canonical schema 로 밀어 넣지 않는다. 기능을
분석 목적과 데이터 성격에 맞게 나누고, canonical 로 안정화된 영역과
provider-native raw 영역을 함께 둔다.

| capability 후보 | 설명 |
| --- | --- |
| `company_registry` | `corp_code`, 회사명, 영문명, `stock_code`, 최종변경일 매핑 |
| `filings` | 공시 검색, 접수번호, 보고서명, 접수일, 정정 여부, 공시 유형 |
| `financials` | 주요 계정, 전체 재무제표, 연결/별도 구분, 보고서 코드 |
| `financial_metrics` | 성장성, 수익성, 안정성, 현금흐름 품질, 밸류에이션 입력값 |
| `company_facts` | 배당, 주식 수, 자기주식, 최대주주, 임직원, 감사의견 |
| `ownership` | 대량보유, 임원ㆍ주요주주 소유 보고 |
| `material_events` | 주요사항보고서 기반 이벤트 스트림 |
| `registration_events` | 증권신고서 기반 발행, 합병, 분할 이벤트 |
| `document_raw` | 공시 원문, XBRL, file/XML 응답 보관 |
| `raw_api_snapshot` | 아직 canonical 화하지 않은 전체 API provider-native 보관 |

## 분석 목표

OpenDART 데이터는 단독으로 투자 판단을 끝내기보다, 시장 데이터와 결합해
스크리닝과 리서치 품질을 높이는 데 쓴다.

| 분석 축 | 볼 데이터 | 예시 질문 |
| --- | --- | --- |
| 성장성 | 매출, 영업이익, 순이익, 재무지표 | 전년 동기 대비 성장하는가? 성장 속도가 둔화되는가? |
| 수익성 | 영업이익률, 순이익률, ROE, ROA | 매출 성장과 이익률이 함께 좋아지는가? |
| 안정성 | 부채, 자본, 미상환 채무증권, 감사의견 | 성장 기업이 재무 부담을 감당할 수 있는가? |
| 현금흐름 품질 | 영업현금흐름, 순이익, 자본적 지출 후보 | 이익이 현금으로 전환되고 있는가? |
| 자본정책 | 배당, 자기주식, 증자, 감자, CB/BW/EB | 주주가치에 우호적인 정책인가? 희석 위험이 있는가? |
| 지배구조 | 최대주주, 대량보유, 임원ㆍ주요주주 지분 | 지분 구조가 바뀌고 있는가? 내부자 지분 변화가 있는가? |
| 이벤트 리스크 | 소송, 부도, 영업정지, 회생, 합병, 분할 | 가격 추세 뒤에 구조적 이벤트가 있는가? |
| 밸류에이션 | 재무 metric + 시가총액 | 비싼 성장주인지, 싼 가치주인지, 업종 내 상대 위치는 어떤지 |

## 전체 구현 목표

아래 항목은 `opendart` provider 가 최종적으로 갖춰야 할 기능과 경계를 나열한
것이다. 실제 작업 순서는 의존성, 현재 코드 상태, 사용자가 먼저 필요한 분석
흐름에 맞춰 따로 정한다.

### Provider 경계와 인증

- provider id 는 `opendart` 를 우선 후보로 둔다. 기존 `dart` 표기를 alias 로
  둘지 제거할지는 구현 전에 결정한다.
- OpenDART SDK 는 `github.com/ev3rlit/opendart` root package 를 사용한다.
- provider config 와 auth loading 은 기존 mwosa provider 패턴에 맞춘다.
- `OPENDART_API_KEY` 기반 doctor 를 제공한다.
- API key 값은 stdout, stderr, error message 에 노출하지 않는다.
- SDK error 는 mwosa error context 로 감싼다.
- live OpenDART 호출은 기본 테스트와 CI 필수 경로에 넣지 않는다.
- `inspect provider opendart` 에서 지원 group, operation, 호출 제한 주의사항,
  raw/canonical 지원 범위를 확인할 수 있게 한다.

### Company Registry

- `corpCode.xml` ZIP/XML 응답을 SDK 를 통해 가져온다.
- `corp_code`, `corp_name`, `corp_eng_name`, `stock_code`, `modify_date` 를 저장한다.
- 상장회사만 필터링할 수 있게 한다.
- KRX symbol 로 `corp_code` 를 찾는 조회 경로를 제공한다.
- 동일 `stock_code` 변경, 상장폐지, 회사명 변경 같은 이력을 어떻게 볼지 기록한다.
- `corp_code` 와 `stock_code` 차이가 CLI help 또는 문서에 명확히 남아야 한다.

### Filing Index

- 회사, 기간, 공시유형, 상세유형으로 공시 목록을 조회한다.
- `rcept_no`, `corp_code`, `stock_code`, `corp_name`, `report_nm`, `rcept_dt`,
  `flr_nm`, `rm` 을 보존한다.
- 정정, 첨부정정, 철회, 최종보고서 여부를 screen/filter 에 쓸 수 있게 한다.
- filing index 는 원문 문서 저장과 재무제표 조회의 연결점으로 사용한다.
- 특정 종목의 최근 공시 목록과 특정 기간의 정기보고서, 주요사항보고서를
  구분해서 조회할 수 있어야 한다.

### Financials

- 단일회사 주요계정, 단일회사 전체 재무제표, 단일회사 주요 재무지표를 연결한다.
- 다중회사 재무 API 는 대량 수집 또는 비교 조회가 필요할 때 사용할 수 있게 한다.
- 연결 재무제표를 우선 사용하고, 연결이 없을 때 별도 재무제표를 fallback
  후보로 둔다. fallback 은 조용히 숨기지 않고 provenance 에 남긴다.
- `bsns_year`, `reprt_code`, `fs_div`, `sj_div`, `account_id`, `account_nm`,
  `thstrm_amount`, `frmtrm_amount` 같은 원천 필드를 보존한다.
- account name 은 provider-native 로 남기고, screen surface 는 canonical
  metric 이름을 사용한다.
- 매출, 영업이익, 순이익, 자산, 부채, 자본 같은 핵심 계정이 canonical metric
  후보로 정리되어야 한다.

### Fundamentals And Valuation

- `revenue_growth_yoy`, `operating_income_growth_yoy`, `net_income_growth_yoy`
  같은 성장 지표를 계산한다.
- `operating_margin`, `net_margin`, `roe`, `roa`, `debt_to_equity` 를 계산한다.
- KRX/Datago 시가총액과 결합해 `per`, `pbr`, `psr` 후보를 만든다.
- 업종 또는 market segment 안에서 percentile 기반 상대 평가를 할 수 있게 한다.
- 가격 모멘텀 스크리닝 결과에 성장, 품질, 밸류에이션 metric 을 붙일 수 있어야 한다.
- 계산값마다 사용한 보고서, 기준일, 연결/별도 여부, market data 기준일이 남아야 한다.
- 세부 점수화는 `docs/features/dart-financial-valuation/README.md` 와 연결한다.

### Company Facts

정기보고서 주요정보 중 분석에 바로 쓰이는 사실 데이터를 제공한다.

- 배당에 관한 사항
- 주식의 총수 현황
- 자기주식 취득 및 처분 현황
- 최대주주 현황과 최대주주 변동현황
- 직원 현황
- 회계감사인의 명칭 및 감사의견
- 공모/사모자금의 사용내역
- 임원 보수와 이사회 관련 항목

각 fact 는 원천 API, 접수번호, 사업연도, 보고서 코드 provenance 를 가져야 한다.
screen 조건에서는 배당, 자기주식, 감사의견, 직원 수 변화 같은 fact 를 사용할 수
있어야 한다.

### Ownership And Material Events

- 대량보유와 임원ㆍ주요주주 소유 보고를 ownership event 로 저장한다.
- 유상증자, 감자, CB/BW/EB 발행, 자기주식, 합병, 분할, 소송, 부도,
  영업정지, 회생절차를 material event 로 저장한다.
- 증권신고서 기반 지분증권, 채무증권, 합병, 분할, 주식교환ㆍ이전 정보도
  registration event 후보로 보존한다.
- event 는 가격 데이터와 time-series 로 join 할 수 있게 `event_date`,
  `rcept_dt`, `rcept_no`, `corp_code`, `stock_code` 를 가진다.
- 이벤트 해석은 조심한다. 예를 들어 유상증자는 항상 나쁜 신호가 아니므로,
  provider 는 사실과 금액, 수량, 일정만 제공하고 평가는 별도 score layer 에 둔다.
- screen 결과에서 최근 dilution, lawsuit, default, merger 같은 event flag 를
  확인할 수 있어야 한다.

### Documents And XBRL

- `document.xml` 원문 파일을 `rcept_no` 기준으로 조회한다.
- 기본 출력은 metadata 로 두고, 사용자가 명시적으로 요청할 때만 base64 payload 를
  포함한다.
- `fnlttXbrl.xml` 은 file response 로 보존하고, full parser 는 별도 장기 과제로 둔다.
- JSON API 와 file/XML API 를 같은 출력/저장 모델로 억지로 합치지 않는다.
- 파일 응답은 metadata 출력과 payload 출력 계약을 분리한다.
- 대용량 원문은 canonical query path 에 끼워 넣지 않고, 사용자가 명시적으로
  요청할 때 내려받는다.
- 기본 `go test ./...` 는 파일 API live 호출에 의존하지 않아야 한다.

### Full Raw API Coverage

- 85개 API 전체를 provider-native raw 조회와 snapshot 으로 제공한다.
- canonical 로 안정화되지 않은 API 도 `raw_api_snapshot` 으로 조회/저장할 수
  있게 한다.
- operation 목록, 요청 파라미터, canonical support label 을 inspect 에 노출한다.
- raw coverage 는 canonical data model 완성과 별개로 다룬다.
- 새 OpenDART API 가 추가되면 SDK generator 와 docs inventory 를 갱신한 뒤
  provider raw catalog 를 갱신한다.
- OpenDART 공식 API inventory 와 mwosa raw operation catalog 의 차이를 확인할
  수 있어야 한다.
- canonical 미지원 API 도 조용히 누락되지 않고 raw 지원 또는 unsupported 로
  명확히 드러나야 한다.

## CLI 방향

정확한 command 는 구현 시점에 기존 mwosa verb-first 구조에 맞춰 확정한다.
아래는 의도만 보여주는 초안이다.

```bash
mwosa doctor provider opendart -o json
mwosa inspect provider opendart -o json

mwosa sync companies --provider opendart --listed-only -o json
mwosa search companies 005930 --provider opendart -o json

mwosa list filings 005930 --provider opendart --from 2025-01-01 --to 2025-12-31 -o json
mwosa list filings --corp-code 00126380 --provider opendart --from 2025-01-01 -o json
mwosa get filing 20250318000935 --provider opendart -o json

mwosa sync financials 005930 --provider opendart --year 2025 --report annual -o json
mwosa get financials 005930 --provider opendart --year 2025 -o json

mwosa list events 005930 --provider opendart --from 2025-01-01 --to 2025-12-31 -o json
mwosa get provider-raw opendart fnlttSinglAcnt --corp-code 00126380 --business-year 2025 --report-code 11011 -o json
```

CLI 는 stdout/stderr 계약을 지킨다. JSON, NDJSON, CSV 는 stdout 으로만 결과를
출력하고, 진행 상황과 경고는 stderr 로 보낸다.

2026-05-17 현재 구현된 실제 CLI 범위는 아래와 같다.

```bash
mwosa list provider-apis opendart -o json
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

이 범위는 전체 85개 API canonicalization 이 아니라 provider foundation, 회사
식별자 매핑, 공시 검색, 단일회사 전체 재무제표, 계산 metric, valuation snapshot,
periodic report fact, material event, stock summary 흐름을 검증하기 위한 범위다.
`provider-apis`, `provider-raw`, `provider-raw-snapshots`, `get filing` 은 raw/API
catalog 와 file response escape hatch 이며, 일반 workflow 의 중심 command 로 두지
않는다.

## 현재 저장/분석 표면

현재 구현은 OpenDART 전용 schema 를 늘리기보다 provider-neutral canonical storage
를 중심으로 둔다.

| 영역 | 현재 표면 |
| --- | --- |
| 회사 식별자 | `company_v1`, `company_identifier_v1`, `instrument_company_link_v1` |
| 전환 경로 | `opendart_companies` |
| 재무제표 | `financial_statement_v1`, `financial_line_item_v1` |
| 재무지표 | `financial_metric_v1` |
| 밸류에이션 | `valuation_snapshot_v1` |
| 배당/사실 | `company_fact_v1` |
| 이벤트 | `company_event_v1` |
| 종합 조회 | `inspect stock` |
| screen 입력 | `stock_daily_metrics`, `stock_daily_fundamentals` |

`stock_daily_metrics` 와 `stock_daily_fundamentals` dataset 은 저장된 daily bar 에
`financial_metrics`, `valuation`, `fundamental_scores`, `company_facts`,
`company_events` 를 붙여 가격 모멘텀, 재무 품질, 가치 지표를 함께 screen 할 수
있게 한다.

## 현재 의존 순서

저장된 canonical 데이터를 읽는 명령은 선행 sync/calc 가 필요하다.

| 조회 명령 | 선행 명령 |
| --- | --- |
| `inspect company`, `get company-identifiers` | `sync companies --provider opendart` |
| `get financials statements` | `sync financials statements` |
| `get financials metrics`, `get financials health` | `calc financials metrics` |
| `get financials valuation` | 저장된 daily price/market cap 과 `calc financials valuation` |
| `get financials dividends` | `sync financials dividends` |
| `get financials facts` | `sync financials facts` |
| `list events`, `inspect stock --section events` | `sync events` |

missing data 는 빈 성공처럼 숨기지 않는다. 저장 데이터가 없으면 조회 명령은 명시적
error 를 반환하고, `inspect stock` 은 section 별 `missing` reason 을 함께 출력한다.
계산 불가는 `uncomputable_reason` 으로 보존한다.

## 남은 결정점

- `fnlttXbrl` file response 보존과 XBRL full parser 의 경계를 분리한다.
- 추가 periodic report facts 는 canonical mapping 이 정해진 항목부터
  `company_fact_v1` 로 확장한다.
- 주식 수, 자기주식, 최대주주, 임원 지분 변화는 screen 에 필요한 key 를 먼저
  정한 뒤 세부 fact 로 늘린다.
- 이벤트성 공시 coverage 는 현재 material event mapping 을 기준으로 operation 별
  canonical support label 을 추가한다.
- canonical 미지원 API 는 `provider_raw_snapshots` 또는 raw file escape hatch 로
  유지하고, unsupported capability 는 명시적 error 로 드러낸다.

## 저장 모델 방향

OpenDART 데이터는 갱신 주기와 provenance 가 서로 다르므로 daily bar 테이블에
섞지 않는다.

| 저장 후보 | 성격 |
| --- | --- |
| `opendart_companies` | `corp_code` 중심 회사 registry |
| `instrument_identifiers` | KRX symbol, ISIN, DART `corp_code`, 법인등록번호 연결 |
| `filings` | 공시 검색 결과와 접수번호 index |
| `financial_statements` | 보고서 단위 재무제표 header |
| `financial_line_items` | 계정별 금액과 원천 필드 |
| `financial_metrics` | screen 에 쓰는 canonical 계산 metric |
| `company_facts` | 배당, 감사의견, 임직원, 자기주식 등 정기보고서 fact |
| `ownership_events` | 대량보유, 임원ㆍ주요주주 소유 보고 |
| `material_events` | 주요사항보고서 event |
| `provider_raw_snapshots` | canonical 미지원 또는 원문 보존용 provider-native payload |

테이블 이름과 schema 는 구현 시점에 기존 storage 구조와 migration 방식을 보고
확정한다.

## 제외 범위

아래는 OpenDART provider 구현 목표에서 분리한다.

- OpenDART 데이터만으로 매수/매도 추천 문구 생성
- XBRL 전체 taxonomy parser 완성
- 주석, 세그먼트, 연결 내부거래 제거 같은 고급 회계 분석
- 금융업 전용 회계 모델의 완전한 정규화
- 실시간 공시 알림 시스템
- 모든 주요사항보고서 이벤트의 긍정/부정 자동 판정
- 기본 테스트에서 실제 OpenDART 서버 호출

## 리스크와 주의점

- OpenDART 는 공시 원천 자료다. 데이터의 정확성과 완전성 판단은 원문 보고서
  확인을 필요로 할 수 있다.
- 정정공시가 존재하므로 같은 기간의 최신 값만 단순 upsert 하면 이력이 사라질
  수 있다.
- 보고서 코드와 회계 기준, 연결/별도 구분을 함께 보존하지 않으면 시계열 비교가
  왜곡된다.
- `status=013` 처럼 조회 데이터 없음은 실패와 정상 빈 결과의 경계를 명확히
  다뤄야 한다.
- file/XML API 는 JSON API 와 응답 처리, 저장, 테스트 전략이 다르다.
- OpenDART 호출 제한과 사용자 인증키 상태를 provider doctor 에서 드러내야 한다.

## 관련 문서

- `docs/providers/README.md`
- `docs/architectures/provider/README.md`
- `docs/architectures/layers/README.md`
- `docs/features/dart-financial-valuation/README.md`
- `/Users/danghamo/Documents/gituhb/opendart/README.md`
- `/Users/danghamo/Documents/gituhb/opendart/docs/apis/official-inventory.md`

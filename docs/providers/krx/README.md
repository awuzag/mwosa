# krx Provider

## 개요

`krx` provider 는 KRX Data Marketplace OPEN API 를 `mwosa` 에 연결하는
provider 다. KRX OPEN API 는 하나의 통합 API 보다는 서비스별 API 로
나뉘며, 상세 페이지의 `API 이용신청` 도 각 서비스의 `BO_ID` 를 기준으로
진행된다.

이 문서는 2026-05-10 기준 KRX OPEN API 서비스 목록과 각 서비스 상세 페이지를
확인해 만든 조사 문서에서 출발했다. 현재 `clients/krx` 독립 Go module 은 31개
API typed client 를 제공하고, `providers/krx` adapter 는 그중 canonical 로 자연스럽게
흡수 가능한 일부 API 를 `daily_bar` / `instrument` role 로 연결한다. 나머지 API 는
provider-native raw snapshot 으로 조회하고 저장할 수 있다.

## 문서

- [services.md](services.md): KRX OPEN API 전체 서비스 목록, `api_id`, `path`,
  `BO_ID`, 상세 페이지 링크
- [implementation-notes.md](implementation-notes.md): `mwosa` provider/client
  설계에 반영할 인증, 신청 단위, capability 매핑 메모
- [../../../clients/krx/README.md](../../../clients/krx/README.md):
  KRX OPEN API 31개 typed client module
- [../../../clients/krx/scripts/apply-all-services.browser.js](../../../clients/krx/scripts/apply-all-services.browser.js):
  KRX OPEN API 전체 31개 서비스를 12개월로 일괄 신청하는 브라우저 콘솔용 스크립트

## 원천

- 서비스 목록:
  <https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd>
- 상세 페이지 예:
  <https://openapi.krx.co.kr/contents/OPP/USES/service/OPPUSES003_S2.cmd?BO_ID=nrEpCLaZpoLCTzPUMxuF>

## 전체 서비스 요약

KRX OPEN API 서비스 목록에서 확인한 서비스는 총 31개다.

| 구분 | 서비스 수 | `path` | 주된 성격 |
| --- | ---: | --- | --- |
| 지수 | 5 | `idx` | KRX/KOSPI/KOSDAQ/채권/파생상품 지수 일별 정보 |
| 주식 | 8 | `sto` | 유가증권, 코스닥, 코넥스 매매정보와 종목기본정보 |
| 증권상품 | 3 | `etp` | ETF, ETN, ELW 일별매매정보 |
| 채권 | 3 | `bon` | 국채전문유통, 일반채권, 소액채권 매매정보 |
| 파생상품 | 6 | `drv` | 선물, 주식선물, 옵션, 주식옵션 매매정보 |
| 일반상품 | 3 | `gen` | 석유, 금, 배출권 시장 매매정보 |
| ESG | 3 | `esg` | ESG 증권상품, 사회책임투자채권, ESG 지수 정보 |

## 구현 상태

`mwosa` 의 현재 방향이 국내 주식/ETF 리서치와 로컬 일별 데이터 저장에 있으므로
canonical 저장은 주식/ETP 일별매매정보, 주식 종목기본정보, 지수 일별시세정보부터 연결한다.

| 상태 | API |
| --- | --- |
| `daily_bar` + `instrument` | `etf_bydd_trd`, `etn_bydd_trd`, `elw_bydd_trd` |
| `daily_bar` | `stk_bydd_trd`, `ksq_bydd_trd`, `knx_bydd_trd` |
| `instrument` | `stk_isu_base_info`, `ksq_isu_base_info`, `knx_isu_base_info` |
| `index_bar` | `krx_dd_trd`, `kospi_dd_trd`, `kosdaq_dd_trd`, `drvprod_dd_trd` |
| `raw snapshot` | 위 canonical API 를 포함한 전체 31개 API |

`raw snapshot` 은 `provider_raw_snapshots` table 에 provider, group, `api_id`,
base date, row count, canonical support label, provider-native JSON payload 를 저장한다.
채권지수(`bon_dd_trd`), 채권, 파생상품, 일반상품, ESG 처럼 아직 canonical schema 가
분리되지 않은 데이터는 이 경로로 보관한다. 주식처럼 거래 가능한 증권은
`daily_bar` / `instrument` 로, KOSPI 같은 벤치마크 지수는 `index_bar` 로 분리한다.

## CLI

```bash
mwosa list krx-apis -o json
mwosa get krx etf_bydd_trd --as-of 20240415 -o json
mwosa sync krx krx_dd_trd --as-of 20240415 -o json

mwosa get index KOSPI --provider krx --as-of 20240415 -o json
mwosa sync index KOSPI --provider krx --as-of 20240415 -o json
mwosa sync index --provider krx --as-of 20240415 -o json

mwosa sync daily --provider krx --security-type etf --as-of 20240415 -o json
mwosa backfill daily --provider krx --security-type stock --from 20240415 --to 20240416 -o json
mwosa list instruments 삼성전자 --provider krx --security-type stock -o json
```

`list krx-apis` 는 각 `api_id` 의 짧은 설명과 canonical 저장 지원 범위를 함께
보여준다. `get krx` 는 provider-native 응답을 stdout 으로 출력한다. `sync krx` 는
같은 응답을 provider-native snapshot table 에 저장하고 저장 결과만 stdout 으로
출력한다.

## 신청 단위 메모

상세 페이지에는 각 서비스별 `BO_ID` 와 `API 이용신청` 버튼이 존재한다. 따라서
`krx` provider 설정은 provider 전체 키 하나만으로 모든 operation 을 등록했다고
가정하면 안 된다. 현재 설정은 서비스별 `enabled` 값을 두며, 비활성화된 API 를
호출하면 빈 성공 대신 unsupported error 를 반환한다.

```json
{
  "providers": {
    "krx": {
      "enabled": true,
      "auth": {
        "auth_key": "..."
      },
      "services": {
        "etf_bydd_trd": { "enabled": true },
        "etn_bydd_trd": { "enabled": true },
        "elw_bydd_trd": { "enabled": true }
      }
    }
  }
}
```

## 관련 문서

- [../README.md](../README.md)
- [../datago/README.md](../datago/README.md)

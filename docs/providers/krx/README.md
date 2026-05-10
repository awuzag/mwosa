# krx Provider

## 개요

`krx` provider 는 KRX Data Marketplace OPEN API 를 `mwosa` 에 연결하기 위한
planned provider 다. KRX OPEN API 는 하나의 통합 API 보다는 서비스별 API 로
나뉘며, 상세 페이지의 `API 이용신청` 도 각 서비스의 `BO_ID` 를 기준으로
진행된다.

이 문서는 2026-05-10 기준 KRX OPEN API 서비스 목록과 각 서비스 상세 페이지를
확인해 만든 개발 전 조사 문서다. 실제 client 구현 전에는 필요한 서비스별로
이용신청 상태와 개발 명세서 다운로드 파일을 다시 확인해야 한다.

## 문서

- [services.md](services.md): KRX OPEN API 전체 서비스 목록, `api_id`, `path`,
  `BO_ID`, 상세 페이지 링크
- [implementation-notes.md](implementation-notes.md): `mwosa` provider/client
  설계에 반영할 인증, 신청 단위, capability 매핑 메모
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

## 초기 구현 후보

`mwosa` 의 현재 방향이 국내 주식/ETF 리서치와 로컬 일별 데이터 저장에 있으므로
초기 `krx` provider 는 아래 순서로 좁게 시작하는 편이 좋다.

| 우선순위 | 서비스 | `api_id` | capability |
| ---: | --- | --- | --- |
| 1 | ETF 일별매매정보 | `etf_bydd_trd` | `daily_bar`, `instrument` |
| 1 | ETN 일별매매정보 | `etn_bydd_trd` | `daily_bar`, `instrument` |
| 1 | ELW 일별매매정보 | `elw_bydd_trd` | `daily_bar`, `instrument` |
| 2 | 유가증권 일별매매정보 | `stk_bydd_trd` | `daily_bar`, `instrument` |
| 2 | 코스닥 일별매매정보 | `ksq_bydd_trd` | `daily_bar`, `instrument` |
| 2 | 코넥스 일별매매정보 | `knx_bydd_trd` | `daily_bar`, `instrument` |
| 3 | 유가증권/코스닥/코넥스 종목기본정보 | `*_isu_base_info` | `instrument` |
| 4 | KRX/KOSPI/KOSDAQ 시리즈 일별시세정보 | `*_dd_trd` | `index`, `daily_bar` |

## 신청 단위 메모

상세 페이지에는 각 서비스별 `BO_ID` 와 `API 이용신청` 버튼이 존재한다. 따라서
`krx` provider 설정은 provider 전체 키 하나만으로 모든 operation 을 등록했다고
가정하면 안 된다. 실제 구현에서는 서비스별 승인 상태를 반영해 등록 가능한 role
만 활성화해야 한다.

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

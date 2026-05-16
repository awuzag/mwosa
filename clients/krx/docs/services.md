# KRX OPEN API Service Catalog

수집일: 2026-05-10

이 문서는 KRX OPEN API 서비스 목록 페이지와 각 상세 페이지에서 수집한 서비스 식별자, endpoint, 요청/응답 필드 문서 링크를 정리한다.
상세 필드 명세는 `apis/<category>/<api_id>.md` 파일에 서비스별로 저장한다.

## 호출 공통 규칙

- 인증키는 query parameter가 아니라 HTTP request header `AUTH_KEY`로 전달한다.
- 상세 화면의 샘플 endpoint는 `https://data-dbg.krx.co.kr/svc/sample/apis/{path}/{api_id}` 형태다.
- 다운로드 명세서가 표시하는 endpoint는 `https://data-dbg.krx.co.kr/svc/apis/{path}/{api_id}` 형태다.
- 서비스별 이용신청은 `BO_ID` 단위다. provider 구현에서 승인되지 않은 `api_id`는 빈 성공으로 처리하지 않는다.

## 전체 서비스

| 구분 | 서비스 | API ID | Path | BO_ID | 등록일 | 수정일 | 문서 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 지수 | KRX 시리즈 일별시세정보 | `krx_dd_trd` | `idx` | `SsgXTEspyJESKvyXZtCU` | 2020/09/15 | 2026/01/16 | [krx_dd_trd](apis/index/krx_dd_trd.md) |
| 지수 | KOSPI 시리즈 일별시세정보 | `kospi_dd_trd` | `idx` | `EREKZauXnMmxyIlqzeDN` | 2020/09/15 | 2026/01/16 | [kospi_dd_trd](apis/index/kospi_dd_trd.md) |
| 지수 | KOSDAQ 시리즈 일별시세정보 | `kosdaq_dd_trd` | `idx` | `nimebcamqFNIPNcRrHoO` | 2020/09/15 | 2026/01/16 | [kosdaq_dd_trd](apis/index/kosdaq_dd_trd.md) |
| 지수 | 채권지수 시세정보 | `bon_dd_trd` | `idx` | `vMxIKCtPBUeRytCqkoFv` | 2022/07/04 | 2026/01/16 | [bon_dd_trd](apis/index/bon_dd_trd.md) |
| 지수 | 파생상품지수 시세정보 | `drvprod_dd_trd` | `idx` | `rPBjbLtScMwmSXWDOYPd` | 2022/07/04 | 2026/01/16 | [drvprod_dd_trd](apis/index/drvprod_dd_trd.md) |
| 주식 | 유가증권 일별매매정보 | `stk_bydd_trd` | `sto` | `JvJFzlAENzZlPBDNGAWC` | 2020/09/22 | 2026/01/16 | [stk_bydd_trd](apis/stock/stk_bydd_trd.md) |
| 주식 | 코스닥 일별매매정보 | `ksq_bydd_trd` | `sto` | `hZjGpkllgCBCWqeTsYFj` | 2020/09/22 | 2026/01/16 | [ksq_bydd_trd](apis/stock/ksq_bydd_trd.md) |
| 주식 | 코넥스 일별매매정보 | `knx_bydd_trd` | `sto` | `HSiRvxGSYnvaKuAuqpqp` | 2020/09/22 | 2026/01/16 | [knx_bydd_trd](apis/stock/knx_bydd_trd.md) |
| 주식 | 신주인수권증권 일별매매정보 | `sw_bydd_trd` | `sto` | `erXKnEAzTqcGnkcoSdGA` | 2020/09/22 | 2026/01/16 | [sw_bydd_trd](apis/stock/sw_bydd_trd.md) |
| 주식 | 신주인수권증서 일별매매정보 | `sr_bydd_trd` | `sto` | `YieGrzzJtKhbaNLuKmhz` | 2022/07/04 | 2026/01/16 | [sr_bydd_trd](apis/stock/sr_bydd_trd.md) |
| 주식 | 유가증권 종목기본정보 | `stk_isu_base_info` | `sto` | `PiwgMdTwmsenXhmqqxuj` | 2022/05/06 | 2026/01/16 | [stk_isu_base_info](apis/stock/stk_isu_base_info.md) |
| 주식 | 코스닥 종목기본정보 | `ksq_isu_base_info` | `sto` | `CifLHplnUFMgpHIMMPXs` | 2022/05/06 | 2026/01/16 | [ksq_isu_base_info](apis/stock/ksq_isu_base_info.md) |
| 주식 | 코넥스 종목기본정보 | `knx_isu_base_info` | `sto` | `COgTLqgmGlqyJvaEFNIc` | 2022/05/06 | 2026/01/16 | [knx_isu_base_info](apis/stock/knx_isu_base_info.md) |
| 증권상품 | ETF 일별매매정보 | `etf_bydd_trd` | `etp` | `nrEpCLaZpoLCTzPUMxuF` | 2020/09/22 | 2026/01/16 | [etf_bydd_trd](apis/etp/etf_bydd_trd.md) |
| 증권상품 | ETN 일별매매정보 | `etn_bydd_trd` | `etp` | `VujebrcOsZQMybnUuwLk` | 2020/09/22 | 2026/01/16 | [etn_bydd_trd](apis/etp/etn_bydd_trd.md) |
| 증권상품 | ELW 일별매매정보 | `elw_bydd_trd` | `etp` | `brBhSEuDCUNpmfsCslfM` | 2020/09/22 | 2026/01/16 | [elw_bydd_trd](apis/etp/elw_bydd_trd.md) |
| 채권 | 국채전문유통시장 일별매매정보 | `kts_bydd_trd` | `bon` | `CEnOyORzHgXWpdbUfWyf` | 2020/09/22 | 2026/01/16 | [kts_bydd_trd](apis/bond/kts_bydd_trd.md) |
| 채권 | 일반채권시장 일별매매정보 | `bnd_bydd_trd` | `bon` | `JfStBNhXISpVVfBHgspT` | 2020/09/22 | 2026/01/16 | [bnd_bydd_trd](apis/bond/bnd_bydd_trd.md) |
| 채권 | 소액채권시장 일별매매정보 | `smb_bydd_trd` | `bon` | `yrTTOsXuYzHprbWLuYzd` | 2020/09/22 | 2026/01/16 | [smb_bydd_trd](apis/bond/smb_bydd_trd.md) |
| 파생상품 | 선물 일별매매정보 (주식선물外) | `fut_bydd_trd` | `drv` | `ilaVYOabbaicHbKTsqga` | 2020/09/22 | 2026/01/16 | [fut_bydd_trd](apis/derivatives/fut_bydd_trd.md) |
| 파생상품 | 주식선물(유가) 일별매매정보 | `eqsfu_stk_bydd_trd` | `drv` | `JzVvQnspImpuqtZlFWpJ` | 2020/09/22 | 2026/01/16 | [eqsfu_stk_bydd_trd](apis/derivatives/eqsfu_stk_bydd_trd.md) |
| 파생상품 | 주식선물(코스닥) 일별매매정보 | `eqkfu_ksq_bydd_trd` | `drv` | `henfdJADfLTCUCBWIRCj` | 2020/09/22 | 2026/01/16 | [eqkfu_ksq_bydd_trd](apis/derivatives/eqkfu_ksq_bydd_trd.md) |
| 파생상품 | 옵션 일별매매정보 (주식옵션外) | `opt_bydd_trd` | `drv` | `AoTvuFpukvuBsfypkZbq` | 2020/09/22 | 2026/01/16 | [opt_bydd_trd](apis/derivatives/opt_bydd_trd.md) |
| 파생상품 | 주식옵션(유가) 일별매매정보 | `eqsop_bydd_trd` | `drv` | `fwWKgzbevDVtAoECgkpA` | 2020/09/22 | 2026/01/16 | [eqsop_bydd_trd](apis/derivatives/eqsop_bydd_trd.md) |
| 파생상품 | 주식옵션(코스닥) 일별매매정보 | `eqkop_bydd_trd` | `drv` | `AFNbHSizSPnEssZoUqiS` | 2020/09/22 | 2026/01/16 | [eqkop_bydd_trd](apis/derivatives/eqkop_bydd_trd.md) |
| 일반상품 | 석유시장 일별매매정보 | `oil_bydd_trd` | `gen` | `rTvrZvAFKfcaLPOggJtW` | 2020/09/22 | 2026/01/16 | [oil_bydd_trd](apis/commodity/oil_bydd_trd.md) |
| 일반상품 | 금시장 일별매매정보 | `gold_bydd_trd` | `gen` | `sxveSnWzWNzWxQASsgEG` | 2020/09/22 | 2026/01/16 | [gold_bydd_trd](apis/commodity/gold_bydd_trd.md) |
| 일반상품 | 배출권 시장 일별매매정보 | `ets_bydd_trd` | `gen` | `IZiYdcgRQFMeENJPEMKG` | 2020/09/22 | 2026/01/16 | [ets_bydd_trd](apis/commodity/ets_bydd_trd.md) |
| ESG | ESG 증권상품 | `esg_etp_info` | `esg` | `dpRoGGhdnfSZSrMFtUCz` | 2025/12/26 | 2026/03/30 | [esg_etp_info](apis/esg/esg_etp_info.md) |
| ESG | 사회책임투자채권 정보 | `sri_bond_info` | `esg` | `MwsSXzVIceQhMSJUeCdp` | 2023/11/15 | 2026/01/16 | [sri_bond_info](apis/esg/sri_bond_info.md) |
| ESG | ESG 지수 | `esg_index_info` | `esg` | `WgFYvEvsseQMARfMVZCq` | 2025/12/26 | 2026/03/30 | [esg_index_info](apis/esg/esg_index_info.md) |

## 카테고리별 요약

| 구분 | 서비스 수 | Path |
| --- | ---: | --- |
| 지수 | 5 | `idx` |
| 주식 | 8 | `sto` |
| 증권상품 | 3 | `etp` |
| 채권 | 3 | `bon` |
| 파생상품 | 6 | `drv` |
| 일반상품 | 3 | `gen` |
| ESG | 3 | `esg` |

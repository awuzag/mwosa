# KRX OPEN API Service Catalog

수집일: 2026-05-10

이 문서는 KRX OPEN API 서비스 목록과 각 상세 페이지에서 확인한 서비스 식별자를
정리한다. 상세 페이지의 샘플 URL 은 `https://data-dbg.krx.co.kr` 아래
`/svc/sample/apis/{path}/{api_id}` 형태로 노출된다. 운영 호출 URL 과 승인 범위는
서비스별 개발 명세서와 실제 이용신청 상태로 다시 확인해야 한다.

## 컬럼

| 컬럼 | 의미 |
| --- | --- |
| 구분 | KRX OPEN API 화면의 서비스 카테고리 |
| 서비스 | KRX 화면의 API 명 |
| `api_id` | 샘플 테스트와 예제에서 쓰는 API ID |
| `path` | 샘플 API 경로의 도메인 구분 |
| 샘플 경로 | 상세 페이지에 노출된 샘플 API path |
| `BO_ID` | 서비스 상세/이용신청 식별자 |
| 데이터 시작 | 설명 문구에서 확인한 제공 시작일 |
| 등록일 | 상세 페이지 등록일 |
| 수정일 | 상세 페이지 최근 수정일 |

## 지수

| 서비스 | `api_id` | `path` | 샘플 경로 | `BO_ID` | 데이터 시작 | 등록일 | 수정일 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| KRX 시리즈 일별시세정보 | `krx_dd_trd` | `idx` | `/svc/sample/apis/idx/krx_dd_trd` | `SsgXTEspyJESKvyXZtCU` | 2010-01-04 | 2020-09-15 | 2026-01-16 |
| KOSPI 시리즈 일별시세정보 | `kospi_dd_trd` | `idx` | `/svc/sample/apis/idx/kospi_dd_trd` | `EREKZauXnMmxyIlqzeDN` | 2010-01-04 | 2020-09-15 | 2026-01-16 |
| KOSDAQ 시리즈 일별시세정보 | `kosdaq_dd_trd` | `idx` | `/svc/sample/apis/idx/kosdaq_dd_trd` | `nimebcamqFNIPNcRrHoO` | 2010-01-04 | 2020-09-15 | 2026-01-16 |
| 채권지수 시세정보 | `bon_dd_trd` | `idx` | `/svc/sample/apis/idx/bon_dd_trd` | `vMxIKCtPBUeRytCqkoFv` | 2010-01-04 | 2022-07-04 | 2026-01-16 |
| 파생상품지수 시세정보 | `drvprod_dd_trd` | `idx` | `/svc/sample/apis/idx/drvprod_dd_trd` | `rPBjbLtScMwmSXWDOYPd` | 2010-01-04 | 2022-07-04 | 2026-01-16 |

## 주식

| 서비스 | `api_id` | `path` | 샘플 경로 | `BO_ID` | 데이터 시작 | 등록일 | 수정일 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 유가증권 일별매매정보 | `stk_bydd_trd` | `sto` | `/svc/sample/apis/sto/stk_bydd_trd` | `JvJFzlAENzZlPBDNGAWC` | 2010-01-04 | 2020-09-22 | 2026-01-16 |
| 코스닥 일별매매정보 | `ksq_bydd_trd` | `sto` | `/svc/sample/apis/sto/ksq_bydd_trd` | `hZjGpkllgCBCWqeTsYFj` | 2010-01-04 | 2020-09-22 | 2026-01-16 |
| 코넥스 일별매매정보 | `knx_bydd_trd` | `sto` | `/svc/sample/apis/sto/knx_bydd_trd` | `HSiRvxGSYnvaKuAuqpqp` | 2013-07-01 | 2020-09-22 | 2026-01-16 |
| 신주인수권증권 일별매매정보 | `sw_bydd_trd` | `sto` | `/svc/sample/apis/sto/sw_bydd_trd` | `erXKnEAzTqcGnkcoSdGA` | 2010-01-04 | 2020-09-22 | 2026-01-16 |
| 신주인수권증서 일별매매정보 | `sr_bydd_trd` | `sto` | `/svc/sample/apis/sto/sr_bydd_trd` | `YieGrzzJtKhbaNLuKmhz` | 2010-02-12 | 2022-07-04 | 2026-01-16 |
| 유가증권 종목기본정보 | `stk_isu_base_info` | `sto` | `/svc/sample/apis/sto/stk_isu_base_info` | `PiwgMdTwmsenXhmqqxuj` | 2010-01-04 | 2022-05-06 | 2026-01-16 |
| 코스닥 종목기본정보 | `ksq_isu_base_info` | `sto` | `/svc/sample/apis/sto/ksq_isu_base_info` | `CifLHplnUFMgpHIMMPXs` | 2010-01-04 | 2022-05-06 | 2026-01-16 |
| 코넥스 종목기본정보 | `knx_isu_base_info` | `sto` | `/svc/sample/apis/sto/knx_isu_base_info` | `COgTLqgmGlqyJvaEFNIc` | 2013-07-01 | 2022-05-06 | 2026-01-16 |

## 증권상품

| 서비스 | `api_id` | `path` | 샘플 경로 | `BO_ID` | 데이터 시작 | 등록일 | 수정일 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| ETF 일별매매정보 | `etf_bydd_trd` | `etp` | `/svc/sample/apis/etp/etf_bydd_trd` | `nrEpCLaZpoLCTzPUMxuF` | 2010-01-04 | 2020-09-22 | 2026-01-16 |
| ETN 일별매매정보 | `etn_bydd_trd` | `etp` | `/svc/sample/apis/etp/etn_bydd_trd` | `VujebrcOsZQMybnUuwLk` | 2014-11-17 | 2020-09-22 | 2026-01-16 |
| ELW 일별매매정보 | `elw_bydd_trd` | `etp` | `/svc/sample/apis/etp/elw_bydd_trd` | `brBhSEuDCUNpmfsCslfM` | 2010-01-04 | 2020-09-22 | 2026-01-16 |

## 채권

| 서비스 | `api_id` | `path` | 샘플 경로 | `BO_ID` | 데이터 시작 | 등록일 | 수정일 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 국채전문유통시장 일별매매정보 | `kts_bydd_trd` | `bon` | `/svc/sample/apis/bon/kts_bydd_trd` | `CEnOyORzHgXWpdbUfWyf` | 2010-01-04 | 2020-09-22 | 2026-01-16 |
| 일반채권시장 일별매매정보 | `bnd_bydd_trd` | `bon` | `/svc/sample/apis/bon/bnd_bydd_trd` | `JfStBNhXISpVVfBHgspT` | 2010-01-04 | 2020-09-22 | 2026-01-16 |
| 소액채권시장 일별매매정보 | `smb_bydd_trd` | `bon` | `/svc/sample/apis/bon/smb_bydd_trd` | `yrTTOsXuYzHprbWLuYzd` | 2010-01-04 | 2020-09-22 | 2026-01-16 |

## 파생상품

| 서비스 | `api_id` | `path` | 샘플 경로 | `BO_ID` | 데이터 시작 | 등록일 | 수정일 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 선물 일별매매정보 (주식선물 외) | `fut_bydd_trd` | `drv` | `/svc/sample/apis/drv/fut_bydd_trd` | `ilaVYOabbaicHbKTsqga` | 2010-01-04 | 2020-09-22 | 2026-01-16 |
| 주식선물(유가) 일별매매정보 | `eqsfu_stk_bydd_trd` | `drv` | `/svc/sample/apis/drv/eqsfu_stk_bydd_trd` | `JzVvQnspImpuqtZlFWpJ` | 2010-01-04 | 2020-09-22 | 2026-01-16 |
| 주식선물(코스닥) 일별매매정보 | `eqkfu_ksq_bydd_trd` | `drv` | `/svc/sample/apis/drv/eqkfu_ksq_bydd_trd` | `henfdJADfLTCUCBWIRCj` | 2015-08-03 | 2020-09-22 | 2026-01-16 |
| 옵션 일별매매정보 (주식옵션 외) | `opt_bydd_trd` | `drv` | `/svc/sample/apis/drv/opt_bydd_trd` | `AoTvuFpukvuBsfypkZbq` | 2010-01-04 | 2020-09-22 | 2026-01-16 |
| 주식옵션(유가) 일별매매정보 | `eqsop_bydd_trd` | `drv` | `/svc/sample/apis/drv/eqsop_bydd_trd` | `fwWKgzbevDVtAoECgkpA` | 2010-01-04 | 2020-09-22 | 2026-01-16 |
| 주식옵션(코스닥) 일별매매정보 | `eqkop_bydd_trd` | `drv` | `/svc/sample/apis/drv/eqkop_bydd_trd` | `AFNbHSizSPnEssZoUqiS` | 2017-06-26 | 2020-09-22 | 2026-01-16 |

## 일반상품

| 서비스 | `api_id` | `path` | 샘플 경로 | `BO_ID` | 데이터 시작 | 등록일 | 수정일 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 석유시장 일별매매정보 | `oil_bydd_trd` | `gen` | `/svc/sample/apis/gen/oil_bydd_trd` | `rTvrZvAFKfcaLPOggJtW` | 2012-03-30 | 2020-09-22 | 2026-01-16 |
| 금시장 일별매매정보 | `gold_bydd_trd` | `gen` | `/svc/sample/apis/gen/gold_bydd_trd` | `sxveSnWzWNzWxQASsgEG` | 2014-03-24 | 2020-09-22 | 2026-01-16 |
| 배출권 시장 일별매매정보 | `ets_bydd_trd` | `gen` | `/svc/sample/apis/gen/ets_bydd_trd` | `IZiYdcgRQFMeENJPEMKG` | 2015-01-12 | 2020-09-22 | 2026-01-16 |

## ESG

| 서비스 | `api_id` | `path` | 샘플 경로 | `BO_ID` | 데이터 시작 | 등록일 | 수정일 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| ESG 증권상품 | `esg_etp_info` | `esg` | `/svc/sample/apis/esg/esg_etp_info` | `dpRoGGhdnfSZSrMFtUCz` | 2020-01-02 | 2025-12-26 | 2026-03-30 |
| 사회책임투자채권 정보 | `sri_bond_info` | `esg` | `/svc/sample/apis/esg/sri_bond_info` | `MwsSXzVIceQhMSJUeCdp` | 2019-01-01 | 2023-11-15 | 2026-01-16 |
| ESG 지수 | `esg_index_info` | `esg` | `/svc/sample/apis/esg/esg_index_info` | `WgFYvEvsseQMARfMVZCq` | 2020-01-02 | 2025-12-26 | 2026-03-30 |

## 상세 페이지

서비스 상세 페이지는 모두 KRX OPEN API 사이트의 `OPPUSES*_S2.cmd` 경로를
사용하고, query string 의 `BO_ID` 로 개별 서비스를 식별한다. 전체 상세 URL 은
아래처럼 조합한다.

```text
https://openapi.krx.co.kr/contents/OPP/USES/service/{screen}_S2.cmd?BO_ID={BO_ID}
```

`screen` 은 카테고리별로 다르다.

| 구분 | 상세 화면 |
| --- | --- |
| 지수 | `OPPUSES001` |
| 주식 | `OPPUSES002` |
| 증권상품 | `OPPUSES003` |
| 채권 | `OPPUSES004` |
| 파생상품 | `OPPUSES005` |
| 일반상품 | `OPPUSES006` |
| ESG | `OPPUSES007` |

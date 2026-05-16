# 신주인수권증권 일별매매정보

수집일: 2026-05-10

## 개요

- 구분: 주식
- API ID: `sw_bydd_trd`
- Path: `sto`
- BO_ID: `erXKnEAzTqcGnkcoSdGA`
- BO version: `1.0`
- 등록일: 2020/09/22
- 최근 수정일: 2026/01/16
- 설명: 유가증권/코스닥시장에 상장되어 있는 신주인수권증권의 매매정보 제공 ('10년01월04일 데이터부터 제공)

## Endpoint

- 명세 endpoint: `https://data-dbg.krx.co.kr/svc/apis/sto/sw_bydd_trd`
- 샘플 테스트 endpoint: `https://data-dbg.krx.co.kr/svc/sample/apis/sto/sw_bydd_trd`
- 인증: HTTP request header `AUTH_KEY`에 사용자가 발급받은 인증키를 전달한다.
- 응답 형식: 상세 화면의 테스트 탭은 `json`과 `xml` 요청 타입을 제공한다.

```http
GET /svc/apis/sto/sw_bydd_trd?basDd=20200414 HTTP/1.1
Host: data-dbg.krx.co.kr
AUTH_KEY: <issued-auth-key>
```

## 요청 필드

| 항목명 | 이름 | Type | Size | Sample |
| --- | --- | --- | ---: | --- |
| 기준일자 | `basDd` | `string` | 8 | `20200414` |

## 응답 필드

| 항목명 | 출력명 | Type | Size | Format | Sample |
| --- | --- | --- | ---: | --- | --- |
| 기준일자 | `BAS_DD` | `string` |  | `` | `` |
| 시장구분 | `MKT_NM` | `string` |  | `` | `` |
| 종목코드 | `ISU_CD` | `string` |  | `` | `` |
| 종목명 | `ISU_NM` | `string` |  | `` | `` |
| 종가 | `TDD_CLSPRC` | `string` |  | `###0` | `` |
| 대비 | `CMPPREVDD_PRC` | `string` |  | `###0` | `` |
| 등락률 | `FLUC_RT` | `string` |  | `###0.00` | `` |
| 시가 | `TDD_OPNPRC` | `string` |  | `###0` | `` |
| 고가 | `TDD_HGPRC` | `string` |  | `###0` | `` |
| 저가 | `TDD_LWPRC` | `string` |  | `###0` | `` |
| 거래량 | `ACC_TRDVOL` | `string` |  | `###0` | `` |
| 거래대금 | `ACC_TRDVAL` | `string` |  | `###0` | `` |
| 시가총액 | `MKTCAP` | `string` |  | `###0` | `` |
| 상장증권수 | `LIST_SHRS` | `string` |  | `###0` | `` |
| 행사가격 | `EXER_PRC` | `string` |  | `###0` | `` |
| 존속기간_시작일 | `EXST_STRT_DD` | `string` |  | `` | `` |
| 존속기간_종료일 | `EXST_END_DD` | `string` |  | `` | `` |
| 목적주권_종목코드 | `TARSTK_ISU_SRT_CD` | `string` |  | `` | `` |
| 목적주권_종목명 | `TARSTK_ISU_NM` | `string` |  | `` | `` |
| 목적주권_종가 | `TARSTK_ISU_PRSNT_PRC` | `string` |  | `###0` | `` |

## 수집 원천

- 상세 페이지: <https://openapi.krx.co.kr/contents/OPP/USES/service/OPPUSES002_S2.cmd?BO_ID=erXKnEAzTqcGnkcoSdGA>
- 서비스 목록: <https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd>
- 개발 명세서 다운로드: `POST https://openapi.krx.co.kr/contents/OPP/USES/service/downloadApiDoc.cmd` with `path=sto`, `BO_ID=erXKnEAzTqcGnkcoSdGA`, `BO_VER=1.0`

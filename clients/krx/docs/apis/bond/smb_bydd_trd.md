# 소액채권시장 일별매매정보

수집일: 2026-05-10

## 개요

- 구분: 채권
- API ID: `smb_bydd_trd`
- Path: `bon`
- BO_ID: `yrTTOsXuYzHprbWLuYzd`
- BO version: `1.0`
- 등록일: 2020/09/22
- 최근 수정일: 2026/01/16
- 설명: 소액채권시장에 상장되어있는 채권의 매매정보 제공 ('10년01월04일 데이터부터 제공)

## Endpoint

- 명세 endpoint: `https://data-dbg.krx.co.kr/svc/apis/bon/smb_bydd_trd`
- 샘플 테스트 endpoint: `https://data-dbg.krx.co.kr/svc/sample/apis/bon/smb_bydd_trd`
- 인증: HTTP request header `AUTH_KEY`에 사용자가 발급받은 인증키를 전달한다.
- 응답 형식: 상세 화면의 테스트 탭은 `json`과 `xml` 요청 타입을 제공한다.

```http
GET /svc/apis/bon/smb_bydd_trd?basDd=20200414 HTTP/1.1
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
| 종가_가격 | `CLSPRC` | `string` |  | `###0.0` | `` |
| 종가_대비 | `CMPPREVDD_PRC` | `string` |  | `###0.0` | `` |
| 종가_수익률 | `CLSPRC_YD` | `string` |  | `###0.000` | `` |
| 시가_가격 | `OPNPRC` | `string` |  | `###0.0` | `` |
| 시가_수익률 | `OPNPRC_YD` | `string` |  | `###0.000` | `` |
| 고가_가격 | `HGPRC` | `string` |  | `###0.0` | `` |
| 고가_수익률 | `HGPRC_YD` | `string` |  | `###0.000` | `` |
| 저가_가격 | `LWPRC` | `string` |  | `###0.0` | `` |
| 저가_수익률 | `LWPRC_YD` | `string` |  | `###0.000` | `` |
| 거래량 | `ACC_TRDVOL` | `string` |  | `###0` | `` |
| 거래대금 | `ACC_TRDVAL` | `string` |  | `###0` | `` |

## 수집 원천

- 상세 페이지: <https://openapi.krx.co.kr/contents/OPP/USES/service/OPPUSES004_S2.cmd?BO_ID=yrTTOsXuYzHprbWLuYzd>
- 서비스 목록: <https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd>
- 개발 명세서 다운로드: `POST https://openapi.krx.co.kr/contents/OPP/USES/service/downloadApiDoc.cmd` with `path=bon`, `BO_ID=yrTTOsXuYzHprbWLuYzd`, `BO_VER=1.0`

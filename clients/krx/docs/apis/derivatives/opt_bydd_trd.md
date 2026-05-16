# 옵션 일별매매정보 (주식옵션外)

수집일: 2026-05-10

## 개요

- 구분: 파생상품
- API ID: `opt_bydd_trd`
- Path: `drv`
- BO_ID: `AoTvuFpukvuBsfypkZbq`
- BO version: `1.0`
- 등록일: 2020/09/22
- 최근 수정일: 2026/01/16
- 설명: 파생상품시장의 옵션 중 주식옵션을 제외한 옵션의 매매정보 제공 ('10년01월04일 데이터부터 제공)

## Endpoint

- 명세 endpoint: `https://data-dbg.krx.co.kr/svc/apis/drv/opt_bydd_trd`
- 샘플 테스트 endpoint: `https://data-dbg.krx.co.kr/svc/sample/apis/drv/opt_bydd_trd`
- 인증: HTTP request header `AUTH_KEY`에 사용자가 발급받은 인증키를 전달한다.
- 응답 형식: 상세 화면의 테스트 탭은 `json`과 `xml` 요청 타입을 제공한다.

```http
GET /svc/apis/drv/opt_bydd_trd?basDd=20200414 HTTP/1.1
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
| 상품구분 | `PROD_NM` | `string` |  | `` | `` |
| 권리유형(CALL/PUT) | `RGHT_TP_NM` | `string` |  | `` | `` |
| 종목코드 | `ISU_CD` | `string` |  | `` | `` |
| 종목명 | `ISU_NM` | `string` |  | `` | `` |
| 종가 | `TDD_CLSPRC` | `string` |  | `###0.00#` | `` |
| 대비 | `CMPPREVDD_PRC` | `string` |  | `###0.00#` | `` |
| 시가 | `TDD_OPNPRC` | `string` |  | `###0.00#` | `` |
| 고가 | `TDD_HGPRC` | `string` |  | `###0.00#` | `` |
| 저가 | `TDD_LWPRC` | `string` |  | `###0.00#` | `` |
| 내재변동성 | `IMP_VOLT` | `string` |  | `###0.00#` | `` |
| 익일정산가 | `NXTDD_BAS_PRC` | `string` |  | `###0.00#` | `` |
| 거래량 | `ACC_TRDVOL` | `string` |  | `###0` | `` |
| 거래대금 | `ACC_TRDVAL` | `string` |  | `###0` | `` |
| 미결제약정 | `ACC_OPNINT_QTY` | `string` |  | `###0` | `` |

## 수집 원천

- 상세 페이지: <https://openapi.krx.co.kr/contents/OPP/USES/service/OPPUSES005_S2.cmd?BO_ID=AoTvuFpukvuBsfypkZbq>
- 서비스 목록: <https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd>
- 개발 명세서 다운로드: `POST https://openapi.krx.co.kr/contents/OPP/USES/service/downloadApiDoc.cmd` with `path=drv`, `BO_ID=AoTvuFpukvuBsfypkZbq`, `BO_VER=1.0`

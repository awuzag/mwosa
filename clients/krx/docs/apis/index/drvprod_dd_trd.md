# 파생상품지수 시세정보

수집일: 2026-05-10

## 개요

- 구분: 지수
- API ID: `drvprod_dd_trd`
- Path: `idx`
- BO_ID: `rPBjbLtScMwmSXWDOYPd`
- BO version: `1.0`
- 등록일: 2022/07/04
- 최근 수정일: 2026/01/16
- 설명: 파생상품지수의 시세정보를 제공 ('10년01월04일 데이터부터 제공)

## Endpoint

- 명세 endpoint: `https://data-dbg.krx.co.kr/svc/apis/idx/drvprod_dd_trd`
- 샘플 테스트 endpoint: `https://data-dbg.krx.co.kr/svc/sample/apis/idx/drvprod_dd_trd`
- 인증: HTTP request header `AUTH_KEY`에 사용자가 발급받은 인증키를 전달한다.
- 응답 형식: 상세 화면의 테스트 탭은 `json`과 `xml` 요청 타입을 제공한다.

```http
GET /svc/apis/idx/drvprod_dd_trd?basDd=20200414 HTTP/1.1
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
| 계열구분 | `IDX_CLSS` | `string` |  | `` | `` |
| 지수명 | `IDX_NM` | `string` |  | `` | `` |
| 종가 | `CLSPRC_IDX` | `string` |  | `###0.00` | `` |
| 대비 | `CMPPREVDD_IDX` | `string` |  | `###0.00` | `` |
| 등락률 | `FLUC_RT` | `string` |  | `###0.00` | `` |
| 시가 | `OPNPRC_IDX` | `string` |  | `###0.00` | `` |
| 고가 | `HGPRC_IDX` | `string` |  | `###0.00` | `` |
| 저가 | `LWPRC_IDX` | `string` |  | `###0.00` | `` |

## 수집 원천

- 상세 페이지: <https://openapi.krx.co.kr/contents/OPP/USES/service/OPPUSES001_S2.cmd?BO_ID=rPBjbLtScMwmSXWDOYPd>
- 서비스 목록: <https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd>
- 개발 명세서 다운로드: `POST https://openapi.krx.co.kr/contents/OPP/USES/service/downloadApiDoc.cmd` with `path=idx`, `BO_ID=rPBjbLtScMwmSXWDOYPd`, `BO_VER=1.0`

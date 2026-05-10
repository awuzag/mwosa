# 사회책임투자채권 정보

수집일: 2026-05-10

## 개요

- 구분: ESG
- API ID: `sri_bond_info`
- Path: `esg`
- BO_ID: `MwsSXzVIceQhMSJUeCdp`
- BO version: `1.0`
- 등록일: 2023/11/15
- 최근 수정일: 2026/01/16
- 설명: 사회책임투자채권 정보를 제공 ('19년01월01일 데이터부터 제공)

## Endpoint

- 명세 endpoint: `https://data-dbg.krx.co.kr/svc/apis/esg/sri_bond_info`
- 샘플 테스트 endpoint: `https://data-dbg.krx.co.kr/svc/sample/apis/esg/sri_bond_info`
- 인증: HTTP request header `AUTH_KEY`에 사용자가 발급받은 인증키를 전달한다.
- 응답 형식: 상세 화면의 테스트 탭은 `json`과 `xml` 요청 타입을 제공한다.

```http
GET /svc/apis/esg/sri_bond_info?basDd=20200414 HTTP/1.1
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
| 발행기관 | `ISUR_NM` | `string` |  | `` | `` |
| 표준코드 | `ISU_CD` | `string` |  | `` | `` |
| 채권종류 | `SRI_BND_TP_NM` | `string` |  | `` | `` |
| 종목명 | `ISU_NM` | `string` |  | `` | `` |
| 상장일 | `LIST_DD` | `string` |  | `` | `` |
| 발행일 | `ISU_DD` | `string` |  | `` | `` |
| 상환일 | `REDMPT_DD` | `string` |  | `` | `` |
| 표면이자율 | `ISU_RT` | `string` |  | `###0.00000` | `` |
| 발행금액 | `ISU_AMT` | `string` |  | `###0` | `` |
| 상장금액 | `LIST_AMT` | `string` |  | `###0` | `` |
| 채권유형 | `BND_TP_NM` | `string` |  | `` | `` |

## 수집 원천

- 상세 페이지: <https://openapi.krx.co.kr/contents/OPP/USES/service/OPPUSES007_S2.cmd?BO_ID=MwsSXzVIceQhMSJUeCdp>
- 서비스 목록: <https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd>
- 개발 명세서 다운로드: `POST https://openapi.krx.co.kr/contents/OPP/USES/service/downloadApiDoc.cmd` with `path=esg`, `BO_ID=MwsSXzVIceQhMSJUeCdp`, `BO_VER=1.0`

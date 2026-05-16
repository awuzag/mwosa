# 코넥스 종목기본정보

수집일: 2026-05-10

## 개요

- 구분: 주식
- API ID: `knx_isu_base_info`
- Path: `sto`
- BO_ID: `COgTLqgmGlqyJvaEFNIc`
- BO version: `1.0`
- 등록일: 2022/05/06
- 최근 수정일: 2026/01/16
- 설명: 코넥스 종목기본정보 ('13년07월01일 데이터부터 제공)

## Endpoint

- 명세 endpoint: `https://data-dbg.krx.co.kr/svc/apis/sto/knx_isu_base_info`
- 샘플 테스트 endpoint: `https://data-dbg.krx.co.kr/svc/sample/apis/sto/knx_isu_base_info`
- 인증: HTTP request header `AUTH_KEY`에 사용자가 발급받은 인증키를 전달한다.
- 응답 형식: 상세 화면의 테스트 탭은 `json`과 `xml` 요청 타입을 제공한다.

```http
GET /svc/apis/sto/knx_isu_base_info?basDd=20200414 HTTP/1.1
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
| 표준코드 | `ISU_CD` | `string` |  | `` | `` |
| 단축코드 | `ISU_SRT_CD` | `string` |  | `` | `` |
| 한글 종목명 | `ISU_NM` | `string` |  | `` | `` |
| 한글 종목약명 | `ISU_ABBRV` | `string` |  | `` | `` |
| 영문 종목명 | `ISU_ENG_NM` | `string` |  | `` | `` |
| 상장일 | `LIST_DD` | `string` |  | `` | `` |
| 시장구분 | `MKT_TP_NM` | `string` |  | `` | `` |
| 증권구분 | `SECUGRP_NM` | `string` |  | `` | `` |
| 소속부 | `SECT_TP_NM` | `string` |  | `` | `` |
| 주식종류 | `KIND_STKCERT_TP_NM` | `string` |  | `` | `` |
| 액면가 | `PARVAL` | `string` |  | `` | `` |
| 상장주식수 | `LIST_SHRS` | `string` |  | `` | `` |

## 수집 원천

- 상세 페이지: <https://openapi.krx.co.kr/contents/OPP/USES/service/OPPUSES002_S2.cmd?BO_ID=COgTLqgmGlqyJvaEFNIc>
- 서비스 목록: <https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd>
- 개발 명세서 다운로드: `POST https://openapi.krx.co.kr/contents/OPP/USES/service/downloadApiDoc.cmd` with `path=sto`, `BO_ID=COgTLqgmGlqyJvaEFNIc`, `BO_VER=1.0`

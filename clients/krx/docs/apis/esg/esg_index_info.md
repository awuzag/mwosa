# ESG 지수

수집일: 2026-05-10

## 개요

- 구분: ESG
- API ID: `esg_index_info`
- Path: `esg`
- BO_ID: `WgFYvEvsseQMARfMVZCq`
- BO version: `1.0`
- 등록일: 2025/12/26
- 최근 수정일: 2026/03/30
- 설명: ESG 지수 정보를 제공 ('20년01월02일 데이터부터 제공)

## Endpoint

- 명세 endpoint: `https://data-dbg.krx.co.kr/svc/apis/esg/esg_index_info`
- 샘플 테스트 endpoint: `https://data-dbg.krx.co.kr/svc/sample/apis/esg/esg_index_info`
- 인증: HTTP request header `AUTH_KEY`에 사용자가 발급받은 인증키를 전달한다.
- 응답 형식: 상세 화면의 테스트 탭은 `json`과 `xml` 요청 타입을 제공한다.

```http
GET /svc/apis/esg/esg_index_info HTTP/1.1
Host: data-dbg.krx.co.kr
AUTH_KEY: <issued-auth-key>
```

## 요청 필드

| 항목명 | 이름 | Type | Size | Sample |
| --- | --- | --- | ---: | --- |
| 기준일자 | `basDd` | `string` |  | `` |

## 응답 필드

| 항목명 | 출력명 | Type | Size | Format | Sample |
| --- | --- | --- | ---: | --- | --- |
| 기준일자 | `BAS_DD` | `string` |  | `` | `` |
| 지수명 | `IDX_NM` | `string` |  | `` | `` |
| 현재가 | `CLSPRC_IDX` | `string` |  | `` | `` |
| 전일비 | `PRV_DD_CMPR` | `string` |  | `` | `` |
| 등락률 | `UPDN_RATE` | `string` |  | `` | `` |
| 구성종목수 | `TRD_ISU_CNT` | `string` |  | `` | `` |
| 거래량(천주) | `ACC_TRDVOL` | `string` |  | `` | `` |
| 거래대금(백만원) | `ACC_TRDVAL` | `string` |  | `` | `` |

## 수집 원천

- 상세 페이지: <https://openapi.krx.co.kr/contents/OPP/USES/service/OPPUSES007_S2.cmd?BO_ID=WgFYvEvsseQMARfMVZCq>
- 서비스 목록: <https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd>
- 개발 명세서 다운로드: `POST https://openapi.krx.co.kr/contents/OPP/USES/service/downloadApiDoc.cmd` with `path=esg`, `BO_ID=WgFYvEvsseQMARfMVZCq`, `BO_VER=1.0`

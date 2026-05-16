# ETN 일별매매정보

수집일: 2026-05-10

## 개요

- 구분: 증권상품
- API ID: `etn_bydd_trd`
- Path: `etp`
- BO_ID: `VujebrcOsZQMybnUuwLk`
- BO version: `1.0`
- 등록일: 2020/09/22
- 최근 수정일: 2026/01/16
- 설명: ETN(상장지수증권)의 매매정보 제공 ('14년11월17일 데이터부터 제공)

## Endpoint

- 명세 endpoint: `https://data-dbg.krx.co.kr/svc/apis/etp/etn_bydd_trd`
- 샘플 테스트 endpoint: `https://data-dbg.krx.co.kr/svc/sample/apis/etp/etn_bydd_trd`
- 인증: HTTP request header `AUTH_KEY`에 사용자가 발급받은 인증키를 전달한다.
- 응답 형식: 상세 화면의 테스트 탭은 `json`과 `xml` 요청 타입을 제공한다.

```http
GET /svc/apis/etp/etn_bydd_trd?basDd=20200414 HTTP/1.1
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
| 종목코드 | `ISU_CD` | `string` |  | `` | `` |
| 종목명 | `ISU_NM` | `string` |  | `` | `` |
| 종가 | `TDD_CLSPRC` | `string` |  | `###0` | `` |
| 대비 | `CMPPREVDD_PRC` | `string` |  | `###0` | `` |
| 등락률 | `FLUC_RT` | `string` |  | `###0.00` | `` |
| 지표가치(IV) | `PER1SECU_INDIC_VAL` | `string` |  | `###0.00` | `` |
| 시가 | `TDD_OPNPRC` | `string` |  | `###0` | `` |
| 고가 | `TDD_HGPRC` | `string` |  | `###0` | `` |
| 저가 | `TDD_LWPRC` | `string` |  | `###0` | `` |
| 거래량 | `ACC_TRDVOL` | `string` |  | `###0` | `` |
| 거래대금 | `ACC_TRDVAL` | `string` |  | `###0` | `` |
| 시가총액 | `MKTCAP` | `string` |  | `###0` | `` |
| 지표가치총액 | `INDIC_VAL_AMT` | `string` |  | `###0` | `` |
| 상장증권수 | `LIST_SHRS` | `string` |  | `###0` | `` |
| 기초지수_지수명 | `IDX_IND_NM` | `string` |  | `` | `` |
| 기초지수_종가 | `OBJ_STKPRC_IDX` | `string` |  | `###0.00` | `` |
| 기초지수_대비 | `CMPPREVDD_IDX` | `string` |  | `###0.00` | `` |
| 기초지수_등락률 | `FLUC_RT_IDX` | `string` |  | `###0.00` | `` |

## 수집 원천

- 상세 페이지: <https://openapi.krx.co.kr/contents/OPP/USES/service/OPPUSES003_S2.cmd?BO_ID=VujebrcOsZQMybnUuwLk>
- 서비스 목록: <https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd>
- 개발 명세서 다운로드: `POST https://openapi.krx.co.kr/contents/OPP/USES/service/downloadApiDoc.cmd` with `path=etp`, `BO_ID=VujebrcOsZQMybnUuwLk`, `BO_VER=1.0`

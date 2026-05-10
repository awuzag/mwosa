# KRX OPEN API Docs

수집일: 2026-05-10

`clients/krx/docs` 는 KRX Data Marketplace OPEN API client 구현을 위해 공식 서비스 목록과 상세 페이지에서 수집한 API 명세를 보관한다.

## 구성

- [services.md](services.md): 31개 서비스 목록, `api_id`, `path`, `BO_ID`, endpoint 요약
- [services.json](services.json): 같은 내용을 기계가 읽기 쉬운 JSON으로 저장한 manifest
- [apis/](apis/): 서비스별 요청 필드와 응답 필드 문서

## 수집 범위

- 공식 서비스 목록에서 확인한 전체 서비스 수: 31개
- 서비스 상세 페이지의 embedded BLD 명세에서 요청 block과 출력 block을 추출했다.
- 샘플 인증키 값은 문서에 보관하지 않는다. 실제 호출은 사용자가 발급받은 `AUTH_KEY` header를 사용한다.
- 상세 페이지의 `개발 명세서 다운로드`는 `downloadApiDoc.cmd`에 `path`, `BO_ID`, `BO_VER`를 제출하면 `Spec.docx`를 반환한다.

## 초기 client 구현 후보

| 우선 | API ID | 문서 | 이유 |
| ---: | --- | --- | --- |
| 1 | `etf_bydd_trd` | [ETF 일별매매정보](apis/etp/etf_bydd_trd.md) | ETP 일별매매정보 |
| 2 | `etn_bydd_trd` | [ETN 일별매매정보](apis/etp/etn_bydd_trd.md) | ETP 일별매매정보 |
| 3 | `elw_bydd_trd` | [ELW 일별매매정보](apis/etp/elw_bydd_trd.md) | ETP 일별매매정보 |
| 4 | `stk_bydd_trd` | [유가증권 일별매매정보](apis/stock/stk_bydd_trd.md) | 주식시장 일별매매정보 |
| 5 | `ksq_bydd_trd` | [코스닥 일별매매정보](apis/stock/ksq_bydd_trd.md) | 주식시장 일별매매정보 |
| 6 | `knx_bydd_trd` | [코넥스 일별매매정보](apis/stock/knx_bydd_trd.md) | 주식시장 일별매매정보 |

## 원천

- KRX OPEN API 서비스 목록: <https://openapi.krx.co.kr/contents/OPP/INFO/service/OPPINFO004.cmd>
- 상세 페이지: `https://openapi.krx.co.kr/contents/OPP/USES/service/{screen}_S2.cmd?BO_ID={BO_ID}`
- 샘플 endpoint: `https://data-dbg.krx.co.kr/svc/sample/apis/{path}/{api_id}`
- 명세 endpoint: `https://data-dbg.krx.co.kr/svc/apis/{path}/{api_id}`

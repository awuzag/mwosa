---
id: API.A.XX-YY
title: API 이름
type: service-design-api-endpoint
status: draft
tags: [service-design, api, endpoint]
source: local
created: YYYY-MM-DD
updated: YYYY-MM-DD
service_design: SD.A.XX
api_design: SD.A.XX40
domain_model: SD.A.XX10
persistence: SD.A.XX20
service: SD.A.XX30
---

# API.A.XX-YY API 이름

## 기본 정보

| 항목 | 값 |
| --- | --- |
| Method / Path | `METHOD /api/v1/resources` |
| operationId | `operationName` |
| 역할 | 이 Endpoint가 보장하는 업무 결과 |
| API 유형 | Command 또는 Query |
| 인증 | 사용하는 인증 컨텍스트 |
| 권한 | 소유권·role·permission 판정 기준 |
| 노출 범위 | public, internal, operator, local/test 중 하나 |
| 멱등성 | 필수 여부와 key 이름. 상세 의미는 아래 멱등성 섹션에서 정의 |
| 캐시 | `no-store`, ETag, TTL 또는 해당 없음 |
| 호환성 | API 버전, deprecation 여부 |

## HTTP 계약 원장

- 완전한 OpenAPI 문서: [openapi/openapi.yaml](openapi/openapi.yaml)
- 이 Endpoint의 Path Item: [openapi/paths/API_A_XX_01_endpoint.yaml](openapi/paths/API_A_XX_01_endpoint.yaml)
- 공통 component: [schemas](openapi/components/schemas.yaml), [parameters](openapi/components/parameters.yaml), [headers](openapi/components/headers.yaml), [responses](openapi/components/responses.yaml), [security schemes](openapi/components/security-schemes.yaml)

Method/Path, parameter, request/response schema, required/nullable, content type, response header, HTTP 상태, 오류 body와 wire 예시는 OpenAPI를 기준으로 한다.

## 연관 문서

| 구분 | 식별자와 경로 |
| --- | --- |
| 요구사항 | `REQ.A.XX`, `blueprint/00-requirements/...` |
| UC | `UC.A.XX`, `blueprint/30-uc/...` |
| BC | `BC.A.XX`, `blueprint/40-event-storming-bounded-context/...` |
| 도메인 | `SD.A.XX10`, `../A_XX_10-domain-model/...` |
| 영속성 | `SD.A.XX20`, `../A_XX_20-persistence/...` |
| 서비스 | `SD.A.XX30`, `../A_XX_30-service/...` |
| 시퀀스 | `SCN.A.XX-YY`, `blueprint/80-sequence/...` |

## 책임과 경계

- 보장하는 업무 결과:
- 요청 안에서 하지 않는 일:
- 다른 Context 또는 외부 시스템의 책임:

## 보안과 개인정보

- 인증과 소유 증명의 업무 의미:
- 권한 판정 위치와 기준:
- 존재 여부와 내부 실패 사유의 공개 원칙:
- 로그·trace·metric·응답에 남기지 않을 값:
- 정확한 security AND/OR 조합은 OpenAPI를 따른다.

## 처리 규칙

1. 요청 컨텍스트와 공개 입력을 검증한다.
2. Command 또는 Query Handler를 호출한다.
3. 도메인 규칙과 외부 의존성 결과를 적용한다.
4. 응답 모델 또는 공개 오류로 변환한다.

목록 Query라면 pagination, filter, sort와 최대 조회 크기를 적고, 비동기 Command라면 수락·상태 조회·완료 조건을 적는다.

## 상태 변경과 트랜잭션

- 시작 상태:
- 성공 종료 상태:
- 실패·만료 상태:
- 같은 로컬 트랜잭션에 저장할 대상:
- Outbox/Event 저장과 발행 시점:
- 외부 호출과 부수 효과의 경계:
- Query라면 상태를 바꾸지 않는다는 보장:

## 멱등성과 동시성

- 멱등 범위:
- 요청 fingerprint와 민감 정보 처리:
- 같은 key·같은 요청의 replay 결과:
- 같은 key·다른 요청의 충돌 처리:
- IdempotencyRecord TTL과 복구 기간:
- row lock, version, unique constraint 또는 중복 Event 방지:
- 해당 없으면 그 이유:

## 예외와 복구 규칙

정확한 HTTP 상태, error code, ProblemDetails schema와 예시는 OpenAPI를 기준으로 한다.

| 업무 조건 | 공개 원칙 | 클라이언트 복구 |
| --- | --- | --- |
| 입력 또는 상태 선행조건 불충족 | 내부 식별자와 상세 원인을 숨긴다. | 입력 또는 최신 상태를 다시 확인한다. |
| 동시 변경 또는 멱등 충돌 | 중복 실행을 만들지 않는다. | 원래 key를 사용하거나 최신 상태를 조회한다. |
| 필수 의존성 장애 | 성공 형태로 응답하지 않는다. | `Retry-After` 또는 정책에 따라 재시도한다. |

## 도메인과 서비스 매핑

| 계층 | 매핑 |
| --- | --- |
| Command / Query Handler |  |
| Aggregate / Entity |  |
| Repository / Read Model |  |
| Port / Adapter |  |
| Domain / Integration Event |  |

## 관측성과 운영

- 로그:
- Metric:
- Trace:
- Audit:
- Rate limit 기준과 공개할 재시도 정보:
- Timeout과 재시도 정책:
- Cache와 body sampling 정책:

## 검증 항목

- OpenAPI lint와 bundle이 성공한다.
- 정상 요청과 각 공개 오류 예시가 schema를 통과한다.
- 인증·권한·소유 증명 실패가 정한 공개 오류로 변환된다.
- 같은 멱등 key 재요청과 다른 payload 충돌을 검증한다.
- 동시에 실행해도 Aggregate와 부수 효과가 중복되지 않는다.
- 트랜잭션 실패 시 부분 저장이나 유실된 Event가 남지 않는다.
- 민감 정보가 응답, 로그, trace, metric label에 남지 않는다.

## 연관 시퀀스

- 시퀀스 문서: `SCN.A.XX-YY`
- 관련 API:
- 여러 참여자의 Mermaid 다이어그램은 `80-sequence` 문서에서 관리한다.

## 호환성과 변경 정책

- 하위 호환 추가:
- breaking change 처리:
- deprecation 공지와 제거 조건:

## 확인 필요

- 아직 OpenAPI 계약으로 확정하지 않은 결정만 기록한다.

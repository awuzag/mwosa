---
id: service-design-template-api
title: A_XX_40 API 설계 템플릿
type: template-index
status: active
tags: [service-design, api, openapi, template]
source: local
created: 2026-07-09
updated: 2026-07-10
---

# A_XX_40 API 설계 템플릿

## 역할

HTTP 계약은 OpenAPI로, 업무 처리와 설계 판단은 Markdown으로 나누어 관리한다. 여러 API와 Context가 참여하는 처리 순서는 `80-sequence` 문서를 원장으로 사용한다.

## 원장 구분

| 원장 | 담당 내용 |
| --- | --- |
| `openapi/openapi.yaml` | 공개 API 진입 문서, 서버와 security scheme 등록 |
| `openapi/paths/*.yaml` | Method/Path, parameter, request body, response, HTTP 상태, 오류, 예시 |
| `openapi/components/*.yaml` | 재사용 schema, parameter, response header, response, security scheme |
| `api-endpoint-process.md` | 책임과 경계, 상태 변경, 트랜잭션, 멱등성, 동시성, 보안 근거, 운영 판단 |
| `80-sequence/**` | 여러 API·서비스·Context가 참여하는 처리 순서 |

이벤트 payload를 기계 판독 가능한 계약으로 관리할 때는 OpenAPI에 섞지 않고 별도 event schema 또는 AsyncAPI를 사용한다.

## 폴더 구조

```text
A_XX_40-api/
├── README.md
├── api-endpoint-process.md
└── openapi/
    ├── openapi.yaml
    ├── redocly.yaml
    ├── paths/
    │   └── API_A_XX_01_endpoint.yaml
    └── components/
        ├── schemas.yaml
        ├── parameters.yaml
        ├── headers.yaml
        ├── responses.yaml
        └── security-schemes.yaml
```

운영 코드 생성에서 제외해야 하는 개발 전용 API가 있다면 `dev.openapi.yaml`을 별도 진입 문서로 만들고 운영 bundle과 분리한다.

## Endpoint 설계 노트 필수 섹션

| 섹션 | 작성 내용 |
| --- | --- |
| 기본 정보 | API ID, operationId, Method/Path, 역할, Command/Query, 노출 범위, 캐시와 호환성 |
| HTTP 계약 원장 | 완전한 OpenAPI 진입 문서와 해당 Path Item 링크 |
| 연관 문서 | 요구사항, UC, 도메인, 영속성, 서비스, 시퀀스 |
| 책임과 경계 | API가 보장하는 결과, 하지 않는 일, 다른 Context 책임 |
| 보안과 개인정보 | 소유·권한 판정 근거, 정보 노출 방지, 민감 정보 처리 |
| 처리 규칙 | Application Service 처리와 주요 업무 분기 |
| 상태 변경과 트랜잭션 | 시작·종료 상태, 원자 저장 대상, Outbox와 부수 효과 |
| 멱등성과 동시성 | scope, fingerprint, replay, TTL, lock/version, 충돌 처리 |
| 예외와 복구 규칙 | 업무 실패 조건, 공개 원칙, 클라이언트 복구 방법 |
| 도메인과 서비스 매핑 | Handler, Aggregate, Repository, Port/Adapter, Event, Read Model |
| 관측성과 운영 | 로그, metric, trace, audit, rate limit, timeout, cache |
| 검증 항목 | 정상·오류·보안·멱등·동시성·장애 검증 상황 |
| 연관 시퀀스 | `80-sequence` 문서와 관련 API 목록 |
| 호환성과 변경 정책 | 버전, deprecation, breaking change 처리 |
| 확인 필요 | 아직 계약으로 확정하지 않은 결정만 기록 |

## 작성 규칙

- Request/Response 필드 표, JSON 예시, header 형식, HTTP 상태와 오류 body는 OpenAPI에만 작성한다.
- Markdown의 예외 섹션은 status/code를 복제하지 않고 업무 조건과 복구 의미를 설명한다.
- OpenAPI에는 미확정 결정을 넣지 않는다. 미확정 사항은 Markdown의 `확인 필요`에 둔다.
- `x-api-id`, `x-use-case`, `x-service-design`, `x-sequences`, `x-design-doc`로 추적성을 유지한다.
- 쿠키 인증으로 상태를 변경하는 Endpoint는 Session cookie, CSRF token과 허용 Origin을 AND 조건으로 검증하고 `401`·`403` 계약을 함께 정의한다.
- 단일 Endpoint의 내부 알고리즘이 꼭 필요할 때만 설계 노트에 다이어그램을 둔다. 여러 참여자의 처리 순서는 `80-sequence`로 분리한다.
- 목록 API의 pagination, filter, sort, 최대 page size는 OpenAPI parameter와 설계 노트의 Read Model 규칙에서 함께 검토한다.
- `202 Accepted`를 사용하면 상태 조회 위치, 재시도 간격, 종료 상태와 만료 정책을 함께 정의한다.

## 검증

`archive` 저장소 루트에서 실행한다.

```bash
npx --yes @redocly/cli@2.38.0 lint \
  --config blueprint/50-service-design/.template/A_XX_40-api/openapi/redocly.yaml \
  blueprint/50-service-design/.template/A_XX_40-api/openapi/openapi.yaml

npx --yes @redocly/cli@2.38.0 bundle \
  blueprint/50-service-design/.template/A_XX_40-api/openapi/openapi.yaml \
  --output /tmp/A_XX.openapi.bundle.yaml
```

코드 생성기는 여러 조각을 직접 읽지 않고 검증을 통과한 bundle을 입력으로 사용한다. bundle은 생성물이며 직접 수정하지 않는다.

## 템플릿

- [Endpoint 설계 노트](api-endpoint-process.md)
- [OpenAPI 진입 문서](openapi/openapi.yaml)
- [OpenAPI Path Item](openapi/paths/API_A_XX_01_endpoint.yaml)

---
id: SD.A.XX20
title: 영속성 설계 이름
type: service-design-persistence
status: draft
tags: [persistence, database, schema, repository]
source: local
created: YYYY-MM-DD
updated: YYYY-MM-DD
bounded_context: BC.A.XX
service_design: SD.A.XX
domain_model: SD.A.XX10
service: SD.A.XX30
api: SD.A.XX40
---

# 영속성 설계 이름

## 역할

이 문서가 다루는 저장 책임, 데이터베이스 스키마, 읽기/쓰기 전략을 한두 문장으로 설명한다.

## 연관 태그

🏷️ BC 참조: [BC.A.XX](../../../40-event-storming-bounded-context/BC_A_XX_name.md) | 도메인 참조: [SD.A.XX10](../A_XX_10-domain-model/README.md) | 서비스 참조: [SD.A.XX30](../A_XX_30-service/README.md) | API 참조: [SD.A.XX40](../A_XX_40-api/README.md) | 시나리오 참조: [SCN.A.XX](../../../80-sequence/SCN_A_XX_name.md)

## 저장 모델

| 테이블/컬렉션 | 역할 | 연결 도메인 | 비고 |
| --- | --- | --- | --- |
|  |  |  |  |

## 스키마

| 저장 모델 | 필드 | 타입 | 제약 | 설명 |
| --- | --- | --- | --- | --- |
|  |  |  |  |  |

## Aggregate 매핑

| 도메인 모델 | 저장 모델 | 매핑 방식 | 주의점 |
| --- | --- | --- | --- |
| [SD.A.XX10](../A_XX_10-domain-model/README.md) |  |  |  |

## Repository 설계 근거

| Repository 메서드 | 저장 모델 | 쿼리 기준 | 반환/저장 대상 |
| --- | --- | --- | --- |
|  |  |  |  |

## 쓰기 전략

| 작업 | 트랜잭션 경계 | 정합성 기준 | 실패 처리 |
| --- | --- | --- | --- |
|  |  |  |  |

## 읽기 전략

| 화면/API | 조회 모델 | 인덱스 | 캐시 여부 |
| --- | --- | --- | --- |
|  |  |  |  |

## 인덱스

| 저장 모델 | 인덱스 | 목적 |
| --- | --- | --- |
|  |  |  |

## 마이그레이션

-

## 확인 필요

-

# ecos Provider

## 개요

`ecos` provider 는 한국은행 ECOS 의 거시경제 지표를 `mwosa` 의 `macro`
canonical 모델로 연결하기 위한 provider 다. 첫 대상은 ECOS 100대 통계지표이며,
CLI 요청 preset 은 `key-statistics` 로 둔다.

현재 단계에서는 role, service, storage, CLI 경계를 먼저 만들고 실제 외부 ECOS
API live 호출은 필수 범위에 넣지 않는다. 추후 `github.com/awuzag/ecos` client 를
붙일 때는 `providers/ecos.Client` 를 통해 adapter 가 선별한 canonical record 만
넘긴다.

## canonical 매핑 원칙

ECOS 100대 통계지표는 시장 지수 OHLC 가 아니다. 따라서 KOSPI/KOSDAQ 같은
시장 지수 일별 시세는 계속 `index_bar` 에 저장하고, 금리, 환율, 물가, 통화량,
고용, GDP 같은 경제 지표는 `macro_indicator` 와 `macro_observation` 에 저장한다.

기본 매핑은 다음 원칙을 따른다.

| ECOS 성격 | canonical 위치 |
| --- | --- |
| 통계표/항목 코드 | `macro_indicator.source_code` 와 provider document |
| 사람이 읽는 지표명 | `macro_indicator.name`, `friendly_name` |
| 분류 | `macro_indicator.category` |
| 주기 | `macro_indicator.frequency` |
| 단위/배율 | `macro_indicator.unit`, `scale` |
| 기준 시점 | `macro_observation.period` |
| 공표 시각 | `macro_observation.published_at` |
| 수집 시각 | `macro_observation.collected_at` |
| 수정 차수 | `macro_observation.revision` |

## raw JSON 미저장

local canonical storage 는 ECOS raw response 전체를 보관하지 않는다. raw payload 를
그대로 저장하면 같은 값의 중복 정본이 생기고, canonical field 와 provider-native
field 중 무엇을 기준으로 삼아야 하는지 흐려진다.

provider 별 부가 정보는 adapter 가 선별한 작은 provider document 로 저장한다.
예를 들어 통계표 코드, 항목 코드, 주기 코드, 원천 경로처럼 나중에 재조회나
진단에 필요한 작은 문서만 `macro_indicator_provider_doc.document_json` 에 둔다.
이 JSON 문서는 SQLite `TEXT` 로 저장하되 저장소 schema 생성 단계에서 `json_valid`
수준의 검증을 둔다.

## CLI

```bash
mwosa list macro-indicators --provider ecos --preset key-statistics -o table
mwosa sync macro key-statistics --provider ecos -o json
mwosa sync macro ecos.base-rate --provider ecos --from 2024-01 --to 2024-12 -o json
mwosa get macro ecos.base-rate --from 2024-01 --to 2024-12 -o json
```

`list macro-indicators --provider ecos` 는 provider 에서 indicator metadata 를 조회한다.
`sync macro key-statistics` 는 metadata, source, provider document 를 canonical
저장소에 저장한다. 개별 지표 관측값은 indicator id 와 기간을 지정해 별도로
동기화한다. `get macro` 는 저장된 관측값을 읽는다.

## 관련 문서

- [../README.md](../README.md)
- [../../architectures/provider/README.md](../../architectures/provider/README.md)
- [../../architectures/layers/README.md](../../architectures/layers/README.md)

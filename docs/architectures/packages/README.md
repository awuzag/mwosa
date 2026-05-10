# Package Architecture

## 목적

이 문서는 `mwosa` repository 안에서 독립 재사용 패키지를 어떻게 둘지 논의하기 위한 기준이다.

`packages/` 는 CLI 실행 흐름의 레이어가 아니라, 여러 레이어가 가져다 쓸 수 있는 코어 라이브러리 영역이다. provider client module 처럼 외부 데이터 소스에 묶인 구현체도 아니고, Cobra command 나 service use case 자체도 아니다.

## 기본 구분

| 구분 | 의미 |
| --- | --- |
| CLI layer | command, handler, service, presentation 처럼 `mwosa` 실행 흐름을 구성하는 레이어다. |
| provider client module | 특정 외부 provider API 를 호출하고 provider-native response 를 파싱하는 독립 Go module 이다. |
| provider adapter | provider client 를 `mwosa` role interface 와 canonical data 로 연결하는 CLI 내부 연결 지점이다. |
| core package | provider, CLI, storage 와 분리해서 재사용할 수 있는 순수 기능 패키지다. |

`packages/` 는 마지막 항목인 core package 를 담는다.

## Core package 후보

현재 문서에서 다루는 core package 후보는 투자 리서치 보조지표 계산 패키지와
백테스트 시뮬레이션 엔진이다.

```text
packages/
  indicators/
  backtest/
```

`packages/indicators` 는 추세, 모멘텀, 변동성, 거래량, 수익률, 리스크 같은 투자 보조지표 계산을 담당한다. MACD 와 일목균형표는 검토 대상 중 일부일 뿐이다. 이 패키지는 시장 데이터를 가져오지 않고, 저장소를 읽지 않고, CLI 출력 형식을 알지 않는다.

`packages/backtest` 는 과거 market data 를 시간 순서대로 흘리고, 전략 룰셋,
주문, 체결 가정, 포트폴리오 상태 변화를 결정론적으로 실행하는 core package 다.
이 패키지는 provider, storage, YAML 파일, Cobra, presentation 을 알지 않는다.

## 의존 방향

```text
command/service/storage/provider
  -> packages/indicators
  -> packages/backtest

packages/indicators
  -> Go standard library
  -> package-local optional dependencies

packages/backtest
  -> Go standard library
  -> packages/indicators, if the backtest package owns indicator evaluation
  -> package-local optional dependencies
```

금지 방향:

- `packages/indicators -> command`
- `packages/indicators -> providers`
- `packages/indicators -> storage`
- `packages/indicators -> presentation`
- `packages/indicators -> Cobra`
- `packages/backtest -> command`
- `packages/backtest -> providers`
- `packages/backtest -> storage`
- `packages/backtest -> presentation`
- `packages/backtest -> Cobra`

`packages/indicators` 나 `packages/backtest` 가 외부 라이브러리를 사용하더라도 그 타입은 패키지 public API 로 노출하지 않는다. 외부 라이브러리는 구현 세부사항이고, `mwosa` 쪽 service 는 core package 가 정의한 입력과 출력만 본다.

## Go workspace 기준

`packages/` 아래의 core package 는 독립 Go module 로 둘 수 있다.

```text
packages/indicators/go.mod
packages/backtest/go.mod  # not decided yet
```

이 방식은 provider client module 과 같은 Go workspace 전략을 따른다. root `go.work` 가 CLI module, provider client module, core package module 을 함께 묶고, 각 module 은 자기 테스트를 독립적으로 가진다.

`packages/backtest` 는 규모가 커질 가능성이 높지만, 하위 폴더와 독립 module
분리는 레이어 책임과 public contract 를 먼저 정한 뒤 결정한다.

독립 module 로 고정할지는 아래 조건을 기준으로 판단한다.

- CLI 없이도 테스트와 benchmark 를 돌릴 가치가 있다.
- provider, storage, presentation 과 독립적으로 재사용된다.
- 외부 라이브러리 교체 가능성을 패키지 안에 숨겨야 한다.
- 다른 도구나 agent tool 에서 같은 계산을 가져다 쓸 수 있다.

## 관련 문서

- `docs/architectures/packages/indicators/README.md`
- `docs/architectures/packages/backtest/README.md`
- `docs/architectures/layers/README.md`
- `docs/architectures/tech-stack/README.md`

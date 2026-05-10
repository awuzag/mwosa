# Strategy Asset Schema References

## 목적

이 문서는 `mwosa` 백테스터의 전략 에셋 스키마를 설계하기 전에 참고한 외부
사례를 정리한다.

중요한 결론은 두 가지다.

- QuantConnect/LEAN 은 YAML 전략 스키마를 중심으로 설계되어 있지 않다.
- KIS backtester 의 `.kis.yaml` 은 `mwosa` 의 `StrategySpec` 초안에 직접 참고할
  수 있다.
- `mwosa` 는 전략 스펙과 실행 스펙을 별도 `kind` 로 분리하되, 작성 편의성을
  위해 `---` 로 나뉜 하나의 YAML stream 안에 함께 둘 수 있다.

`mwosa` 에서는 전략 자체와 실행 조건을 분리한다.

```text
StrategySpec
  = 진입, 청산, 지표, 파라미터, 리스크 룰셋

BacktestRunSpec
  = 어떤 전략을 어떤 universe, 기간, 자본, 비용 가정으로 실행할지
```

```yaml
kind: Strategy
schema_version: 1
name: trend-pullback
---
kind: BacktestRun
schema_version: 1
name: trend-pullback-2024-2026
strategy:
  name: trend-pullback
```

## QuantConnect / LEAN

### 확인한 사실

공식 문서 기준으로 QuantConnect/LEAN 은 사용자가 전략을 YAML 로 정의하는
스키마를 제공하지 않는다. 알고리즘은 Python 또는 C# 코드로 작성하고,
프로젝트와 엔진 실행 설정은 JSON 파일과 CLI parameter 로 관리한다.

확인한 공식 문서:

- [LEAN CLI project configuration](https://www.quantconnect.com/docs/v2/lean-cli/projects/configuration)
- [LEAN local platform configuration](https://www.quantconnect.com/docs/v2/local-platform/development-environment/configuration)
- [Algorithm Framework overview](https://www.quantconnect.com/docs/v1/algorithm-framework/overview)
- [Algorithm Engine key concepts](https://www.quantconnect.com/docs/v2/writing-algorithms/key-concepts/algorithm-engine)
- [QuantConnect/lean-cli README](https://github.com/QuantConnect/lean-cli)

### 참고할 점

QuantConnect project 의 `config.json` 은 프로젝트 설명, 알고리즘 언어, 라이브러리,
파라미터 같은 실행 설정을 담는다. `parameters` 는 key/value 객체이며, CLI
backtest 에서는 `--parameter` 로 실행 시점의 값을 넘길 수 있다.

LEAN engine 쪽 `lean.json` 은 로컬 엔진 실행 설정이다. 데이터 provider,
result handler, data feed handler, transaction handler 같은 엔진 구성 요소를
JSON 으로 지정한다.

전략 구조 관점에서는 Algorithm Framework 가 더 중요하다.

```text
Universe Selection
  -> Alpha Creation
  -> Portfolio Construction
  -> Risk Management
  -> Execution
```

이 흐름은 `mwosa` 의 trading engine layer 와 거의 같은 문제를 다룬다.

```text
UniverseSelector
  -> EntryRule / ExitRule
  -> PositionSizer
  -> RiskManager
  -> OrderExecutor
```

다만 QuantConnect 에서는 이 모듈을 YAML data asset 으로 선언하기보다 Python/C#
코드와 framework model 로 조립한다.

### `mwosa` 에 가져올 판단

QuantConnect 에서 직접 가져올 것은 YAML shape 가 아니라 아래 설계 원칙이다.

- 전략 정의와 실행 파라미터는 분리한다.
- 실행 시점의 파라미터 override 를 허용한다.
- backtest, optimization, live 가 같은 알고리즘/프레임워크 모델을 공유한다.
- streaming engine 관점을 유지해 future data leakage 를 막는다.
- universe, signal, portfolio target, risk, execution 을 한 덩어리로 섞지 않는다.

## KIS Backtester `.kis.yaml`

### 로컬 출처

조사한 로컬 경로:

- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/kis_backtest/file/schema.py`
- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/kis_backtest/file/loader.py`
- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/kis_backtest/core/schema.py`
- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/kis_backtest/core/strategy.py`
- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/backend/schemas/backtest.py`
- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/kis_mcp/tools/backtest.py`
- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/low_mdd.yml`
- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/examples/turtle_low_mdd.kis.yaml`
- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/kis_backtest/file/templates/*.kis.yaml`
- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/.lean-workspace/projects/bt_momentum/config.json`
- `/Users/danghamo/Documents/gituhb/open-trading-api/backtester/.lean-workspace/projects/bt_momentum/lean-config.json`

### YAML 구조

KIS 전략 파일은 전략 자체를 아래처럼 정의한다.

```yaml
version: "1.0"

metadata:
  name: "SMA 골든/데드 크로스"
  description: "단기 SMA가 장기 SMA를 상향 돌파하면 매수"
  author: "kis_backtest"
  tags:
    - trend
    - sma

strategy:
  id: sma_crossover
  category: trend
  params:
    fast_period:
      default: 5
      min: 2
      max: 50
      type: int
      description: "단기 SMA 기간"
  indicators:
    - id: sma
      alias: fast
      params:
        period: $fast_period
    - id: sma
      alias: slow
      params:
        period: 20
  entry:
    logic: AND
    conditions:
      - indicator: fast
        operator: cross_above
        compare_to: slow
  exit:
    logic: OR
    conditions:
      - indicator: fast
        operator: cross_below
        compare_to: slow

risk:
  stop_loss:
    enabled: true
    percent: 5.0
```

스키마 구성:

| 영역 | 역할 |
| --- | --- |
| `version` | 전략 파일 버전 |
| `metadata` | 이름, 설명, 작성자, 태그 |
| `strategy.id` | 전략 식별자 |
| `strategy.category` | trend, momentum, mean_reversion 등 분류 |
| `strategy.params` | 사용자 조정 가능 파라미터와 기본값, 범위 |
| `strategy.indicators` | 지표 ID, alias, output, params |
| `strategy.entry` | 진입 조건 그룹 |
| `strategy.exit` | 청산 조건 그룹 |
| `risk` | 손절, 익절, trailing stop, max position size |

조건 구조:

| 필드 | 역할 |
| --- | --- |
| `logic` | `AND` 또는 `OR` |
| `conditions` | 조건 목록 |
| `indicator` | 왼쪽 지표 alias 또는 가격 예약어 |
| `operator` | `greater_than`, `less_than`, `cross_above`, `cross_below` 등 |
| `compare_to` | 비교 대상 지표 alias |
| `value` | 숫자 또는 `$param_name` |
| `output` / `compare_output` | multi-output indicator 의 출력 선택 |
| `compare_scalar` / `compare_operation` | 지표 값에 스칼라 연산 적용 |
| `candlestick` / `signal` | 캔들스틱 패턴 조건 |

### 실행 요청 구조

KIS backtester 는 전략 파일과 실행 요청을 분리한다.

`backend/schemas/backtest.py` 의 `BacktestRequest` 는 아래 실행 조건을 받는다.

| 필드 | 역할 |
| --- | --- |
| `strategy_id` | 저장된 전략 또는 preset id |
| `symbols` | 실행 종목 목록 |
| `start_date` | 백테스트 시작일 |
| `end_date` | 백테스트 종료일 |
| `initial_capital` | 초기 자본 |
| `param_overrides` | 전략 파라미터 override |
| `commission_rate` | 수수료율 |
| `tax_rate` | 거래세율 |
| `slippage` | 슬리피지 |

MCP tool 의 `run_backtest` 도 같은 분리를 따른다. `yaml_content` 로 전략을 받고,
`symbols`, `start_date`, `end_date`, `initial_capital`, `commission_rate`,
`tax_rate`, `slippage` 는 실행 요청으로 따로 받는다.

LEAN project 로 넘길 때도 실행 설정은 `config.json` 의 `parameters` 로 들어간다.
예시 프로젝트에는 `symbols`, `start_date`, `end_date`, `initial_capital`,
`commission_rate`, `tax_rate` 가 parameters 에 들어 있다.

### 참고할 점

좋은 점:

- 전략 에셋과 실행 조건을 이미 분리하고 있다.
- indicator alias 를 조건에서 참조하도록 해 rule tree 를 데이터로 표현한다.
- `$param_name` 참조와 override 흐름이 있다.
- YAML 입력을 `KisStrategyFile -> StrategyDefinition -> StrategySchema` 로
  정규화한다.
- operator alias 를 표준 operator 로 변환한다.
- 가격 예약어(`close`, `open`, `high`, `low`, `volume`)와 indicator alias 를
  같은 조건 체계에서 다룬다.

주의할 점:

- `risk` 가 전략 파일에 들어 있어 전략 룰셋과 실행 정책이 일부 섞여 있다.
- `stop_loss`, `take_profit`, `trailing_stop` 은 entry/exit rule 과 risk rule
  사이의 경계를 다시 정해야 한다.
- `symbols`, `date`, `capital`, `commission`, `slippage` 는 전략 파일 밖에 있어야
  한다.
- 현재 조건 group 은 `AND`/`OR` 중심이라 `NOT`, nested group, reusable rule asset
  확장은 별도 설계가 필요하다.

## `mwosa` Go 스키마 초안

아래 Go type 은 구현 확정안이 아니라, 위 레퍼런스를 `mwosa` 의
`StrategySpec`/`BacktestRunSpec` 분리로 옮긴 초안이다.

```go
type StrategySpec struct {
	SchemaVersion int               `json:"schema_version" yaml:"schema_version"`
	ID            string            `json:"id" yaml:"id"`
	Name          string            `json:"name" yaml:"name"`
	Description   string            `json:"description,omitempty" yaml:"description,omitempty"`
	Category      string            `json:"category,omitempty" yaml:"category,omitempty"`
	Tags          []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Params        map[string]Param  `json:"params,omitempty" yaml:"params,omitempty"`
	Indicators    []IndicatorSpec   `json:"indicators,omitempty" yaml:"indicators,omitempty"`
	Entry         RuleGroupSpec     `json:"entry" yaml:"entry"`
	Exit          RuleGroupSpec     `json:"exit" yaml:"exit"`
	Risk          *RiskPolicySpec    `json:"risk,omitempty" yaml:"risk,omitempty"`
}

type Param struct {
	Default     any      `json:"default" yaml:"default"`
	Type        string   `json:"type,omitempty" yaml:"type,omitempty"`
	Min         *float64 `json:"min,omitempty" yaml:"min,omitempty"`
	Max         *float64 `json:"max,omitempty" yaml:"max,omitempty"`
	Step        *float64 `json:"step,omitempty" yaml:"step,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

type IndicatorSpec struct {
	ID     string         `json:"id" yaml:"id"`
	Alias  string         `json:"alias,omitempty" yaml:"alias,omitempty"`
	Name   string         `json:"name,omitempty" yaml:"name,omitempty"`
	Output string         `json:"output,omitempty" yaml:"output,omitempty"`
	Params map[string]any `json:"params,omitempty" yaml:"params,omitempty"`
}

type RuleGroupSpec struct {
	Logic      string          `json:"logic" yaml:"logic"` // AND, OR
	Conditions []ConditionSpec `json:"conditions" yaml:"conditions"`
}

type ConditionSpec struct {
	Indicator        string   `json:"indicator,omitempty" yaml:"indicator,omitempty"`
	IndicatorOutput  string   `json:"indicator_output,omitempty" yaml:"indicator_output,omitempty"`
	Operator         string   `json:"operator,omitempty" yaml:"operator,omitempty"`
	CompareTo        string   `json:"compare_to,omitempty" yaml:"compare_to,omitempty"`
	CompareOutput    string   `json:"compare_output,omitempty" yaml:"compare_output,omitempty"`
	CompareScalar    *float64 `json:"compare_scalar,omitempty" yaml:"compare_scalar,omitempty"`
	CompareOperation string   `json:"compare_operation,omitempty" yaml:"compare_operation,omitempty"`
	Value            any      `json:"value,omitempty" yaml:"value,omitempty"`
	Tag              string   `json:"tag,omitempty" yaml:"tag,omitempty"`
}

type RiskPolicySpec struct {
	StopLossPct        *float64 `json:"stop_loss_pct,omitempty" yaml:"stop_loss_pct,omitempty"`
	TakeProfitPct      *float64 `json:"take_profit_pct,omitempty" yaml:"take_profit_pct,omitempty"`
	TrailingStopPct    *float64 `json:"trailing_stop_pct,omitempty" yaml:"trailing_stop_pct,omitempty"`
	MaxSymbolWeightPct *float64 `json:"max_symbol_weight_pct,omitempty" yaml:"max_symbol_weight_pct,omitempty"`
	MaxPositions       *int     `json:"max_positions,omitempty" yaml:"max_positions,omitempty"`
}
```

실행 조건은 별도 type 으로 둔다.

```go
type BacktestRunSpec struct {
	SchemaVersion  int                  `json:"schema_version" yaml:"schema_version"`
	StrategyRef    string               `json:"strategy_ref" yaml:"strategy_ref"`
	ParamOverrides map[string]any      `json:"param_overrides,omitempty" yaml:"param_overrides,omitempty"`
	Universe       UniverseRunSpec      `json:"universe" yaml:"universe"`
	Data           DataRunSpec          `json:"data" yaml:"data"`
	Portfolio      PortfolioRunSpec     `json:"portfolio" yaml:"portfolio"`
	Execution      ExecutionProfileSpec `json:"execution" yaml:"execution"`
	Report         ReportRunSpec        `json:"report,omitempty" yaml:"report,omitempty"`
}

type UniverseRunSpec struct {
	Symbols []string `json:"symbols,omitempty" yaml:"symbols,omitempty"`
	Ref     string   `json:"ref,omitempty" yaml:"ref,omitempty"`
}

type DataRunSpec struct {
	Market       string `json:"market" yaml:"market"`
	SecurityType string `json:"security_type" yaml:"security_type"`
	Timeframe    string `json:"timeframe" yaml:"timeframe"`
	From         string `json:"from" yaml:"from"`
	To           string `json:"to" yaml:"to"`
}

type PortfolioRunSpec struct {
	InitialCash float64 `json:"initial_cash" yaml:"initial_cash"`
	Currency    string  `json:"currency,omitempty" yaml:"currency,omitempty"`
}

type ExecutionProfileSpec struct {
	Fill          string   `json:"fill" yaml:"fill"` // next_open 등
	CommissionBps *float64 `json:"commission_bps,omitempty" yaml:"commission_bps,omitempty"`
	TaxBps        *float64 `json:"tax_bps,omitempty" yaml:"tax_bps,omitempty"`
	SlippageBps   *float64 `json:"slippage_bps,omitempty" yaml:"slippage_bps,omitempty"`
}

type ReportRunSpec struct {
	Metrics []string `json:"metrics,omitempty" yaml:"metrics,omitempty"`
}
```

## `mwosa` 설계 결정

초기 결정:

- `StrategySpec` 은 저장 가능한 전략 에셋이다.
- `BacktestRunSpec` 은 실행 요청이다.
- `symbols`, `from`, `to`, `initial_cash`, `commission`, `slippage` 는
  `StrategySpec` 에 넣지 않는다.
- `risk` 는 첫 버전에서는 `StrategySpec` 에 두되, 나중에 `RiskPolicy` asset 으로
  분리할 수 있게 만든다.
- `sizing` 은 첫 버전에서는 `StrategySpec` 에 둔다.
- `universe.symbols` 는 첫 버전에서는 `StrategySpec` 에 둔다.
- `EntryRule` / `ExitRule` 은 `techan` 식 rule 조합 모델을 따른다.
- operator 는 내부 enum 으로 정규화한다.
- `StrategySpec` 은 YAML/JSON 모두 같은 Go type 으로 파싱할 수 있게 한다.
- YAML 은 바로 실행하지 않고
  `StrategySpec + BacktestRunSpec -> validation -> StrategyPlan` 으로 compile 한다.

남은 질문:

- `risk` 를 전략 에셋에 둘지, 실행 profile 에 둘지, 별도 asset 으로 둘지
- `universe` 를 저장된 universe asset 으로 분리할지
- 조건 group 에 nested `all`/`any`/`not` 을 처음부터 넣을지
- 지표 alias 와 tag namespace 를 어떻게 충돌 없이 관리할지

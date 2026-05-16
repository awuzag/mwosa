# RULE.md

이 문서는 mwosa의 짧은 프로그래밍 규칙입니다. 아키텍처 문서보다
짧게 유지합니다. 설명이 길어지는 내용은 이 문서에 모두 풀어쓰기보다
관련 문서로 연결합니다.

## 핵심 원칙

- 레이어 경계를 지킵니다. CLI는 의존성을 조립하고, service는 role
  interface와 repository interface만 사용하며, provider 구현 세부사항은
  service로 새지 않게 합니다.
- provider client는 독립적으로 관리합니다. `clients` 아래의
  provider client는 자체 `go.mod`와 테스트를 가진 별도 Go 모듈입니다.
- 변경 범위는 좁게 유지합니다. 현재 경계 안에서 해결할 수 있는 요청에
  불필요한 대형 리팩터링을 붙이지 않습니다.
- 실패를 빈 성공처럼 숨기지 않습니다. unsupported capability, route
  없음, remote error, storage error, invalid input은 명시적인 error로
  반환하고, 가능한 경우 provider, group, operation, market,
  security type, symbol, date 맥락을 포함합니다.

## 함수 설계

- 함수 이름은 미니멀하게 둡니다. 패키지명, 타입명, 수신자에서 이미
  드러나는 문맥을 함수 이름에 반복하지 않습니다.
- I/O, remote call, storage, 오래 걸리는 작업처럼 취소와 timeout이 필요한
  함수는 `context.Context`를 첫 인자로 받습니다. 순수 계산이나 단순 값
  변환 함수에는 억지로 context를 넣지 않습니다.
- 함수 인자는 필수 값과 자주 바뀌는 값을 먼저 드러냅니다. 선택적 설정,
  확장 가능한 설정, 호출자별 조정값은 일반 인자로 계속 늘리지 않습니다.
- 인자가 복잡해지거나 선택값이 늘어나면 `With...` 형태의 함수형 옵션
  패턴을 우선 사용합니다. option은 명시적으로 검증하고, 잘못된 option을
  조용히 무시하지 않습니다.

## 에러 처리

- 직접 작성하는 Go 코드는 error 생성, wrapping, joining에
  `github.com/samber/oops`를 사용합니다.
- 새 error를 만들 때 `fmt.Errorf`, `errors.New`, `errors.Join`을
  사용하지 않습니다. error 판별을 위한 `errors.Is`, `errors.As`는
  사용할 수 있습니다.
- 생성 코드는 이 규칙에서 제외합니다. 생성 코드는 다시 생성될 수 있으므로
  직접 수정하지 않습니다.
- 새 error는 `oops.New` 또는 `oops.Errorf`를 사용합니다. 하위 레이어
  error를 원인으로 보존해야 할 때는 `Wrap` 또는 `Wrapf`를 사용합니다.
  cleanup 과정에서 여러 error를 보존해야 할 때는 `oops.Join`을
  사용합니다.
- 같은 함수 안에서 domain과 context가 반복되면 재사용 가능한 builder를
  먼저 만듭니다. builder는 `.New`, `.Errorf`, `.Wrap`, `.Wrapf`,
  `.Join`, `.Recover`, `.Recoverf` 같은 종료 메서드로 끝냅니다.

```go
errb := oops.
	In("dailybar_repository").
	With(
		"market", query.Market,
		"security_type", query.SecurityType,
		"symbol", query.Symbol,
		"from", query.From,
		"to", query.To,
	)

client, err := r.database.Client(ctx)
if err != nil {
	return nil, errb.Wrap(err)
}
```

- context는 CLI 경계에서 한 번에 몰아서 붙이지 않습니다. 호출자는 자신이
  알고 있는 요청 맥락을 붙이고, 호출받는 쪽은 자신이 알고 있는 domain
  맥락을 붙입니다.
- `With(...)`는 구조화된 context이며, 항상 사람이 읽는 메시지를 대체하지는
  않습니다. 테스트나 CLI 사용자가 `provider=datago`,
  `group=securitiesProductPrice`, `operation=getETFPriceInfo`,
  `status=502` 같은 필드를 error 문자열에서 직접 확인해야 한다면 해당
  필드를 메시지에도 남깁니다.

## Storage

- 로컬 canonical storage 방향은 SQLite입니다.
- SQLite 접근은 Bun 기반으로 관리합니다. schema와 index는 Bun model과
  schema 생성 코드에서 관리합니다.
- database runtime 접근은 lazy하게 처리합니다. storage handle 생성만으로
  SQLite를 열지 않고, 실제 첫 사용 시점에 열 수 있습니다. cleanup은
  command 단위에서 닫습니다.
- repository 구현체는 export하지 않습니다. 생성자만 export하고,
  service layer repository interface를 반환합니다.
- repository 생성 시점에 결정되는 invariant는 생성자에서 검증합니다.
  모든 repository 메서드에서 반복 방어하지 않습니다.

## Providers

- provider id와 provider group은 분리된 개념입니다. datago의 provider id는
  `datago`이고, 첫 group은 `securitiesProductPrice`입니다.
- provider group을 provider id에 `-` 또는 `/`로 붙이지 않습니다.
- provider adapter는 provider-native 응답을 canonical model 방향으로
  변환합니다.
- provider client는 endpoint path, service key, pagination,
  provider-native parsing, remote error context를 소유합니다.
- 외부 API 테스트는 fake HTTP transport 또는 `httptest`를 사용합니다.
  단위 테스트는 실제 public API 호출에 의존하지 않습니다.

## Testing

- 단위 테스트에서는 `github.com/stretchr/testify`를 적극적으로 사용합니다.
  실패 시 이후 검증이 의미 없으면 `require`를 우선하고, 같은 상태에서 여러
  값을 함께 확인할 때는 `assert`를 사용할 수 있습니다.
- 테스트 helper는 실패를 숨기지 않습니다. helper 안에서 테스트를 중단해야
  한다면 `t.Helper()`와 `require`로 실패 위치를 호출자 기준으로 드러냅니다.
- 기본 `go test ./...`는 빠르고 재현 가능한 단위 테스트와 가벼운 통합
  테스트를 대상으로 합니다. 실제 외부 API 호출이나 사용자의 로컬 환경에
  강하게 묶인 테스트는 기본 경로에 넣지 않습니다.
- 단위 테스트는 함수, 메서드, 작은 패키지 단위를 검증합니다. 외부 의존성은
  interface fake, stub, fake HTTP transport, `httptest`로 대체합니다.
- 통합 테스트는 repository와 SQLite, service와 provider adapter,
  provider client와 `httptest.Server`처럼 여러 컴포넌트의 연결을
  검증합니다. 재현 가능하고 빠른 통합 테스트는 기본 테스트에 포함할 수
  있습니다.
- 빌드된 CLI 실행, 외부 프로세스, 실제 DB 서버, 실제 provider API처럼
  느리거나 환경 의존성이 큰 검증은 `integration` 또는 `e2e` build tag로
  분리합니다.
- e2e 테스트는 사용자가 만나는 경계에서 검증합니다. CLI는 `os/exec`로
  바이너리를 실행해 exit code, stdout, stderr, output shape를 확인합니다.

## CLI

- CLI는 verb-first 구조를 유지하고 `README.md`와 일관되게 둡니다.
- machine-readable output은 예측 가능해야 합니다. JSON output은 `jq`로
  다루기 쉬운 구조를 우선하고, human table output은 간결해도 됩니다.
- stdout은 command 결과에 사용합니다. stderr는 diagnostics, progress,
  log에 사용합니다.

## Documentation

- 아키텍처 계약은 `docs/architectures` 아래에 둡니다.
- provider 목록과 provider별 구현 계획은 `docs/providers` 아래에 둡니다.
- `docs/providers/provider-package-contract.md`는 다시 만들지 않습니다. 이미
  삭제된 문서입니다.

# JsonLogic Relation

Strategy rule 모델은 [JsonLogic](https://jsonlogic.com/) 과 매우 가깝다.
JsonLogic 은 rule 을 JSON 으로 직렬화하고 front-end, back-end, database 사이에서
공유할 수 있는 작은 AST 로 본다. `eval` 을 쓰지 않고, setter, loop, side effect
없이 하나의 결정을 내리는 구조라는 점도 backtest strategy asset 과 잘 맞는다.

따라서 strategy rule 은 JsonLogic 의 원칙을 참고한다.

| 원칙 | Strategy spec 적용 |
| --- | --- |
| operator 는 node 의 key 다 | `gt`, `lt`, `all`, `indicator` 같은 함수 이름을 key 로 둔다 |
| 한 rule node 는 하나의 operator 만 가진다 | `{gt: [...]}` 처럼 단일 연산자 형태로 검증한다 |
| 값은 배열 args 로 전달한다 | `gt: [{price: close}, {ref: trend}]` 처럼 표현한다 |
| rule 은 데이터다 | YAML 로 저장, DB 등록, UI builder 변환이 가능해야 한다 |
| 실행은 안전해야 한다 | `eval` 없이 registry 에 등록된 operator 만 compile 한다 |

다만 JsonLogic 을 그대로 가져오지는 않는다. 백테스터에는 시간 축과 시장 상태가
있기 때문에 `var` 하나로 모든 값을 읽기보다 `price`, `indicator`, `position`,
`portfolio`, `tag` 같은 도메인 operator 를 둔다. `and`, `or`, `>`, `<` 같은 기호형
operator 는 읽기 좋은 alias 로 `all`, `any`, `gt`, `lt` 를 제공하고, 필요하면
compile 단계에서 JsonLogic 스타일 alias 와 상호 변환할 수 있게 둔다.


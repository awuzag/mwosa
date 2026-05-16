# Backtest Examples

이 폴더는 `mwosa` 백테스터 예제를 시나리오별 하위 폴더로 나눠 보관한다. 각
예제 폴더에는 실행 YAML 과 `mwosa` 로 시작하는 CLI 명령을 담은 `README.md` 가
있다.

명령은 현재 브랜치의 CLI command surface 를 기준으로 한다. 로컬에 설치된
`mwosa` 가 오래된 버전이면 현재 브랜치로 다시 빌드/설치한 뒤 실행한다.

| Example | Description |
| --- | --- |
| [`sma-cross`](sma-cross/README.md) | 단일 ETF SMA cross 백테스트 smoke 예제 |
| [`turtle-breakout`](turtle-breakout/README.md) | Donchian breakout 과 ATR stop 을 쓰는 터틀 스타일 추세 추종 예제 |
| [`universe-pipeline`](universe-pipeline/README.md) | 월간 universe pipeline 과 유동성/모멘텀 ranking 예제 |
| [`relative-strength-rotation`](relative-strength-rotation/README.md) | 주간 상대강도 ETF rotation 예제 |
| [`dual-momentum-rotation`](dual-momentum-rotation/README.md) | 월간 dual momentum 현금 대기형 rotation 예제 |
| [`evaluation-grid`](evaluation-grid/README.md) | 기간/파라미터 grid evaluation 예제 |
| [`evaluation-walk-forward`](evaluation-walk-forward/README.md) | walk-forward train/test evaluation 예제 |

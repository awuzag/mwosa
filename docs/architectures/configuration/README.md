# Configuration Architecture

## 목적

이 문서는 `mwosa` CLI 의 설정 파일과 로컬 데이터 파일 기본 경로를 정리한다.

초기 구현에서는 사용자가 별도 경로를 지정하지 않아도 안전하게 실행되는 기본값이 필요하다. 기본값은 실행 위치에 따라 바뀌면 안 되고, 설정 파일과 SQLite 데이터베이스 파일은 서로 다른 성격의 파일로 다룬다.

## 기본 방향

`mwosa` 는 config 와 data 를 분리한다.

| 구분 | 의미 | 기본 위치 |
| --- | --- | --- |
| config | 사용자가 읽고 수정할 수 있는 앱 설정 | 사용자 설정 디렉터리 |
| data | 앱이 생성하고 갱신하는 로컬 데이터 | 사용자 데이터 디렉터리 |

이 구분은 XDG base directory 관례를 기준으로 삼는다. `mwosa` 는 GUI 앱보다 터미널 CLI 에 가깝기 때문에, macOS 도 Linux 와 같은 Unix 계열 CLI 기본값을 우선한다.

## 기본 경로

초기 기본 경로는 아래처럼 둔다. config file 이름은 현재 후보인 `config.json` 으로 예시를 든다. 최종 file format 이 바뀌면 파일명만 함께 조정한다.

| OS | config file | database file |
| --- | --- | --- |
| Linux | `~/.config/mwosa/config.json` | `~/.local/share/mwosa/mwosa.db` |
| macOS | `~/.config/mwosa/config.json` | `~/.local/share/mwosa/mwosa.db` |
| Windows | `%AppData%\mwosa\config.json` | `%LocalAppData%\mwosa\mwosa.db` |

Linux 와 macOS 에서는 사용자가 `XDG_CONFIG_HOME` 또는 `XDG_DATA_HOME` 을 지정했다면 그 값을 우선한다. 예를 들어 `XDG_CONFIG_HOME=/custom/config` 이면 config file 은 `/custom/config/mwosa/config.json` 이 된다.

macOS 에서도 CLI 도구는 `~/.config` 아래에 설정을 두는 경우가 많다. data 는 config 와 같은 디렉터리에 섞지 않고, XDG 의 data 대응 경로인 `~/.local/share` 아래에 둔다. SQLite 파일은 cache 가 아니라 사용자 로컬 앱 데이터의 정본이므로 cache 디렉터리에 두지 않는다.

macOS native 앱 관례를 더 강하게 따르는 배포 형태가 필요해지면 `~/Library/Application Support/mwosa` 를 다시 검토할 수 있다. 현재 기준에서는 CLI 사용자의 dotfiles, backup, shell scripting 경험을 우선해 Unix 계열 기본 경로를 통일한다.

Windows 는 roaming profile 에 따라 동기화될 수 있는 config 는 `%AppData%` 아래에 두고, SQLite database 는 머신 로컬 성격이 강하므로 `%LocalAppData%` 아래에 둔다.

## 우선순위

CLI 실행 시 경로 결정 우선순위는 명시적인 입력을 가장 높게 둔다.

### Config file

1. `--config`
2. `MWOSA_CONFIG`
3. OS 기본 config 경로

### Database file

1. `--database`
2. `MWOSA_DATABASE`
3. config file 의 `database.path`
4. OS 기본 data 경로

`--database` 와 `MWOSA_DATABASE` 는 실험, 테스트, 임시 실행에서 기본 database 를 우회하기 위한 탈출구다. 일반 설치 사용자는 별도 지정 없이 OS 기본 data 경로를 사용한다.

## 개발 실행 경로

`go run ./cmd/mwosa ...` 처럼 Go development build 로 실행할 때는 기본 경로를 현재 작업 디렉터리의 `.mwosa` 아래로 둔다.

| file | development default |
| --- | --- |
| config file | `$PWD/.mwosa/config.json` |
| database file | `$PWD/.mwosa/mwosa.db` |

이 규칙은 명시 경로보다 낮은 우선순위다. 즉 `--config`, `MWOSA_CONFIG`, `--database`, `MWOSA_DATABASE` 를 지정하면 해당 값이 계속 우선한다. 릴리스 바이너리와 설치된 CLI 는 기존 OS 기본 경로를 사용한다.

## Init 동작

`mwosa init config` 는 아래 작업을 수행한다.

- config 디렉터리를 만든다.
- data 디렉터리를 만든다.
- config file 이 없으면 기본 config file 을 생성한다.
- database file 은 실제 저장소 접근 시점에 생성되도록 둔다.

config file 은 사용자가 검토할 수 있는 최소 설정만 담는다. provider 인증 정보나 민감 정보는 별도 정책이 확정되기 전까지 환경변수 우선을 유지한다.

## Inspect 동작

`mwosa inspect config` 는 최종 적용된 경로를 보여준다.

출력에는 적어도 아래 항목을 포함한다.

- config file path
- database file path
- 각 경로가 결정된 source: flag, env, config file, default
- config file 존재 여부
- data directory 존재 여부

사용자가 설정 문제를 디버깅할 때 실행 위치와 기본 경로를 혼동하지 않도록, 상대 경로를 그대로 숨기지 않고 최종 절대 경로를 출력한다.

## 구현 기준

Go 구현은 별도 설정 framework 없이 표준 라이브러리와 명시적인 `config` package 로 시작한다.

`config` package 는 아래 책임을 가진다.

- OS 별 기본 config path 계산
- OS 별 기본 data path 계산
- 환경변수 override 적용
- config file parsing
- CLI flag 로 받은 명시 경로와 config file 값 병합

command layer 는 flag 를 선언하고 값을 전달한다. storage layer 는 이미 결정된 database path 만 받는다. provider adapter 는 전체 config object 에서 provider 관련 값만 읽는다.

## 아직 정하지 않음

- config file format 의 최종 확정: `json`, `yaml`, `toml`
- provider 인증 정보 저장 방식
- config file 자동 migration 정책
- 여러 profile 또는 workspace 별 config 분리 여부

## 관련 문서

- `docs/architectures/tech-stack/README.md`
- `docs/architectures/layers/README.md`
- `README.md`

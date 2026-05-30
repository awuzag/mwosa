# CLI Release Guide

## 목적

이 문서는 `mwosa` 를 사용자가 설치할 수 있는 CLI 앱으로 배포하는 절차를
정리한다.

배포 기준은 브랜치가 아니라 SemVer 태그다. 작업 브랜치와 릴리스 브랜치
전략은 `docs/development/git-branching/README.md` 를 따른다.

## 1차 설치 경로

Go 가 설치된 사용자는 태그 기준으로 CLI 를 설치한다.

```bash
go install github.com/awuzag/mwosa/cmd/mwosa@v0.2.0
```

현재 개발 중인 커밋을 직접 확인할 때는 로컬 workspace 영향을 끊고 빌드한다.

```bash
GOWORK=off go build ./cmd/mwosa
./mwosa version
```

`GOWORK=off` 빌드가 통과해야 외부 사용자의 `go install ...@version` 도 같은
모듈 해석으로 동작한다.

## 바이너리 배포

Go 가 없는 사용자는 GitHub Release 에 첨부된 OS/CPU 별 archive 를 받는다.
릴리스 태그가 push 되면 GitHub Actions 가 GoReleaser 를 실행해서 다음 대상의
바이너리를 만든다.

| OS | CPU |
| --- | --- |
| macOS | amd64, arm64 |
| Linux | amd64, arm64 |
| Windows | amd64, arm64 |

macOS 와 Linux 는 `tar.gz`, Windows 는 `zip` archive 로 배포한다. 각 릴리스는
`checksums.txt` 를 함께 제공한다.

## 릴리스 절차

릴리스 준비는 `main` 에 검증된 변경이 들어간 뒤 시작한다.

```bash
git switch main
git pull --ff-only
go test ./...
GOWORK=off go build ./cmd/mwosa

git switch -c release/v0.1
git push -u origin release/v0.1
```

릴리스 브랜치에서는 배포를 막는 변경만 반영한다.

- 버전 문자열과 릴리스 노트 정리
- 설치 문서와 completion 문서 보정
- 릴리스를 막는 버그 수정
- provider client 모듈 버전 정리

준비가 끝나면 태그를 만든다.

```bash
git tag v0.1.0
git push origin v0.1.0
```

태그 push 후 `.github/workflows/release.yml` 이 실행되고, GitHub Release 에
archive 와 checksum 이 업로드된다.

## provider client 모듈

`clients/*` 는 독립 Go 모듈이다. 루트 CLI 가 client 모듈을 import 한다면
루트 `go.mod` 는 해당 client 모듈을 명시적으로 require 해야 한다.

client 코드가 바뀐 릴리스를 준비할 때는 루트 모듈과 client 모듈의 버전 관계를
먼저 정리한다. 같은 릴리스 번호로 맞추는 경우 client 모듈 태그는 Go 모듈의
중첩 모듈 규칙에 맞게 별도로 만든다.

```bash
git tag clients/datago-etp/v0.1.0
git tag v0.1.0
git push origin clients/datago-etp/v0.1.0 v0.1.0
```

루트 `go.mod` 가 아직 pseudo-version 을 가리키는 상태라면 해당 커밋이 원격에서
조회 가능해야 한다. 릴리스 전에는 아래 명령으로 외부 설치 모듈 해석을 확인한다.

```bash
GOWORK=off go test ./...
GOWORK=off go build ./cmd/mwosa
```

## 다음 설치 채널

GitHub Release archive 가 안정화되면 같은 산출물을 기준으로 설치 채널을
늘린다.

1. `install.sh`, `install.ps1` 로 OS/CPU 감지 후 GitHub Release archive 설치
2. Homebrew tap 으로 macOS/Linuxbrew 설치 지원
3. Scoop bucket 으로 Windows 개발자 설치 지원
4. WinGet 으로 Windows 일반 사용자 설치 지원
5. `.deb`, `.rpm` 패키지와 Linux repository 검토

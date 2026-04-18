# conoha - ConoHa VPS3 CLI

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

[日本語](README.md) | [English](README-en.md)

ConoHa VPS3 API용 커맨드라인 인터페이스입니다. Go로 작성된 싱글 바이너리로, 에이전트 친화적 설계를 채택하고 있습니다.

**[문서 사이트](https://crowdy.github.io/conoha-cli-pages/)** — 가이드, 실전 배포 예제, 커맨드 레퍼런스

> **참고**: 이 도구는 VPS3 전용입니다. 구 VPS2용 CLI(hironobu-s/conoha-vps, miyabisun/conoha-cli)와는 호환되지 않습니다.

## 특징

- 싱글 바이너리, 크로스 플랫폼 지원 (Linux / macOS / Windows)
- 복수 프로필 지원 (`gh auth` 스타일)
- 구조화된 출력 (`--format json/yaml/csv/table`)
- 에이전트 친화적 설계 (`--no-input`, 결정적 종료 코드, stderr/stdout 분리)
- 토큰 자동 갱신 (만료 5분 전 자동 재인증)
- Claude Code 스킬 연동 (`conoha skill install`로 인프라 구축 레시피 설치)

## 설치

### Scoop (Windows)

```powershell
scoop bucket add crowdy https://github.com/crowdy/crowdy-bucket
scoop install conoha
```

### 소스에서 빌드

```bash
go install github.com/crowdy/conoha-cli@latest
```

### 릴리스 바이너리

[Releases](https://github.com/crowdy/conoha-cli/releases) 페이지에서 다운로드하거나, 아래 명령어를 사용하세요:

**Linux (amd64)**

```bash
VERSION=$(curl -s https://api.github.com/repos/crowdy/conoha-cli/releases/latest | grep tag_name | cut -d '"' -f4)
curl -Lo conoha.tar.gz "https://github.com/crowdy/conoha-cli/releases/download/${VERSION}/conoha-cli_${VERSION#v}_linux_amd64.tar.gz"
tar xzf conoha.tar.gz conoha
sudo mv conoha /usr/local/bin/
rm conoha.tar.gz
```

**macOS (Apple Silicon)**

```bash
VERSION=$(curl -s https://api.github.com/repos/crowdy/conoha-cli/releases/latest | grep tag_name | cut -d '"' -f4)
curl -Lo conoha.tar.gz "https://github.com/crowdy/conoha-cli/releases/download/${VERSION}/conoha-cli_${VERSION#v}_darwin_arm64.tar.gz"
tar xzf conoha.tar.gz conoha
sudo mv conoha /usr/local/bin/
rm conoha.tar.gz
```

**Windows (amd64)**

```powershell
$version = (Invoke-RestMethod https://api.github.com/repos/crowdy/conoha-cli/releases/latest).tag_name
$v = $version -replace '^v', ''
Invoke-WebRequest -Uri "https://github.com/crowdy/conoha-cli/releases/download/$version/conoha-cli_${v}_windows_amd64.zip" -OutFile conoha.zip
Expand-Archive conoha.zip -DestinationPath .
Remove-Item conoha.zip
```

> **Tip**: [Scoop](https://scoop.sh/)이 이미 설치되어 있다면 `%PATH%`를 따로 수정하는 것보다 아래 명령으로 `shims` 디렉터리에 배치하는 편이 간단합니다:
>
> ```cmd
> move conoha.exe %USERPROFILE%\scoop\shims\
> ```

## 빠른 시작

```bash
# 로그인 (테넌트 ID, 사용자명, 비밀번호 입력)
conoha auth login

# 인증 상태 확인
conoha auth status

# 서버 목록 조회
conoha server list

# JSON 형식으로 출력
conoha server list --format json

# 서버 상세 정보 (ID 또는 서버명으로 지정 가능)
conoha server show <server-id-or-name>

# 서버 이름 변경
conoha server rename <server-id-or-name> new-name
```

## 명령어 목록

| 명령어 | 설명 |
|--------|------|
| `conoha auth` | 인증 관리 (login / logout / status / list / switch / token / remove) |
| `conoha server` | 서버 관리 (list / show / create / delete / start / stop / reboot / resize / rebuild / rename / console / ips / metadata / ssh / deploy / attach-volume / detach-volume) |
| `conoha flavor` | 플레이버 조회 (list / show) |
| `conoha keypair` | SSH 키페어 관리 (list / create / delete) |
| `conoha volume` | 블록 스토리지 관리 (list / show / create / delete / types / backup) |
| `conoha image` | 이미지 관리 (list / show / delete) |
| `conoha network` | 네트워크 관리 (network / subnet / port / security-group / qos) |
| `conoha lb` | 로드 밸런서 관리 (lb / listener / pool / member / healthmonitor) |
| `conoha dns` | DNS 관리 (domain / record) |
| `conoha storage` | 오브젝트 스토리지 (container / ls / cp / rm / publish) |
| `conoha identity` | 아이덴티티 관리 (credential / subuser / role) |
| `conoha app` | 앱 배포 및 관리 (init / deploy / logs / status / stop / restart / env / destroy / reset / list) |
| `conoha config` | CLI 설정 관리 (show / set / path) |
| `conoha skill` | Claude Code 스킬 관리 (install / update / remove) |

## Claude Code 스킬

ConoHa CLI에는 Claude Code용 인프라 구축 스킬이 포함되어 있습니다. 설치하면 Claude Code에서 자연어로 인프라 구축을 지시할 수 있습니다.

### 설치

```bash
# 스킬 설치
conoha skill install

# 스킬 업데이트
conoha skill update

# 스킬 삭제
conoha skill remove
```

### 사용법

Claude Code에서 다음과 같이 지시하면 스킬이 자동으로 트리거됩니다:

```
> ConoHa에 서버 만들어줘
> k8s 클러스터 구축해줘
> 앱을 배포해줘
```

### 레시피 목록

| 레시피 | 설명 |
|--------|------|
| Docker Compose 앱 배포 | `conoha app deploy`를 통한 컨테이너 앱 배포 |
| 커스텀 스크립트 배포 | 스타트업 스크립트를 이용한 서버 구성 |
| Kubernetes 클러스터 | k3s를 이용한 클러스터 구축 (coming soon) |
| OpenStack 플랫폼 | DevStack을 이용한 플랫폼 구축 (coming soon) |
| Slurm HPC 클러스터 | Slurm을 이용한 HPC 클러스터 구축 (coming soon) |

자세한 내용은 [conoha-cli-skill](https://github.com/crowdy/conoha-cli-skill)을 참조하세요.

## 설정

설정 파일은 `~/.config/conoha/`에 저장됩니다:

| 파일 | 설명 | 퍼미션 |
|------|------|--------|
| `config.yaml` | 프로필 설정 | 0600 |
| `credentials.yaml` | 비밀번호 | 0600 |
| `tokens.yaml` | 토큰 캐시 | 0600 |

### 환경 변수

| 변수 | 설명 |
|------|------|
| `CONOHA_PROFILE` | 사용할 프로필명 |
| `CONOHA_TENANT_ID` | 테넌트 ID |
| `CONOHA_USERNAME` | API 사용자명 |
| `CONOHA_PASSWORD` | API 비밀번호 |
| `CONOHA_TOKEN` | 인증 토큰 (직접 지정) |
| `CONOHA_FORMAT` | 출력 형식 |
| `CONOHA_CONFIG_DIR` | 설정 디렉토리 경로 |
| `CONOHA_NO_INPUT` | 비대화 모드 (`1` 또는 `true`) |
| `CONOHA_ENDPOINT` | API 엔드포인트 오버라이드 |
| `CONOHA_ENDPOINT_MODE` | `int`로 내부 API 모드 (서비스명을 경로에 추가) |
| `CONOHA_DEBUG` | 디버그 로깅 (`1` 또는 `api`) |

우선순위: 환경 변수 > 플래그 > 프로필 설정 > 기본값

### 글로벌 플래그

```
--profile    사용할 프로필
--format     출력 형식 (table / json / yaml / csv)
--no-input   대화형 프롬프트 비활성화
--quiet      불필요한 출력 억제
--verbose    상세 출력
--no-color   컬러 출력 비활성화
```

## 종료 코드

| 코드 | 의미 |
|------|------|
| 0 | 성공 |
| 1 | 일반 에러 |
| 2 | 인증 실패 |
| 3 | 리소스 없음 |
| 4 | 검증 에러 |
| 5 | API 에러 |
| 6 | 네트워크 에러 |
| 10 | 사용자 취소 |

## 에이전트 연동

이 CLI는 스크립트 및 AI 에이전트에서의 활용을 고려하여 설계되었습니다:

```bash
# 비대화 모드로 JSON 출력
conoha server list --format json --no-input

# 스크립팅을 위한 토큰 취득
TOKEN=$(conoha auth token)

# 종료 코드로 에러 핸들링
conoha server show abc123 || echo "Exit code: $?"
```

## 개발

```bash
make build     # 바이너리 빌드
make test      # 테스트 실행
make lint      # 린터 실행
make clean     # 산출물 삭제
```

## API 문서

- [ConoHa VPS3 API 레퍼런스](https://doc.conoha.jp/reference/api-vps3/)

## 라이선스

Apache License 2.0 - 자세한 내용은 [LICENSE](LICENSE)를 참조하세요.

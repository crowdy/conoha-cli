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
| `conoha app` | 앱 배포 및 관리 (init / deploy / rollback / logs / status / stop / restart / env / destroy / list) |
| `conoha proxy` | conoha-proxy 리버스 프록시 관리 (boot / reboot / start / stop / restart / remove / logs / details / services) |
| `conoha config` | CLI 설정 관리 (show / set / path) |
| `conoha skill` | Claude Code 스킬 관리 (install / update / remove) |

## 앱 배포

`conoha app` 은 같은 VPS 에서 공존할 수 있는 두 가지 배포 모드를 제공합니다. `conoha app init` 시점에 서버 측 마커 (`/opt/conoha/<name>/.conoha-mode`) 가 기록되고, 이후의 `deploy` / `status` / `logs` / `stop` / `restart` / `destroy` / `rollback` 은 자동으로 그 모드로 동작합니다. `--proxy` / `--no-proxy` 플래그는 마커를 덮어쓰되, 불일치 시 에러가 납니다 (모드 전환은 `destroy` → 재 `init`).

| 모드 | 기본 | 용도 | 레이아웃 | `conoha.yml` | `conoha proxy boot` | DNS / TLS |
|---|:-:|---|---|:-:|:-:|:-:|
| **proxy** (blue/green) | ✓ | 도메인 + Let's Encrypt TLS 공개 앱 | `/opt/conoha/<name>/<slot>/` blue/green 슬롯 | 필수 | 필수 | 필수 |
| **no-proxy** (flat) |  | 테스트, 사내/개발 VPS, 비 HTTP 서비스, 취미 앱 | `/opt/conoha/<name>/` 평면 단일 디렉터리 | 불필요 | 불필요 | 불필요 |

### proxy 모드 (기본): conoha-proxy 기반 blue/green

[conoha-proxy](https://github.com/crowdy/conoha-proxy) 가 Let's Encrypt HTTPS, Host 헤더 라우팅, drain 윈도우 내 즉시 롤백을 제공합니다.

1. 리포지토리 루트에 `conoha.yml` 작성:

   ```yaml
   name: myapp                   # DNS-1123 라벨 (소문자 영숫자 + 하이픈, 1-63 자)
   hosts:
     - app.example.com           # 하나 이상, 중복 불가
   web:
     service: web                # compose 파일의 서비스명과 일치해야 함
     port: 8080                  # 컨테이너 listen 포트 (1-65535)
   # --- 선택 ---
   compose_file: docker-compose.yml   # 생략 시 conoha-docker-compose.yml → docker-compose.yml → compose.yml 순으로 자동 검출
   accessories: [db, redis]           # web 과 같은 네트워크에 붙는 부속 서비스
   health:
     path: /healthz
     interval_ms: 1000
     timeout_ms: 500
     healthy_threshold: 2
     unhealthy_threshold: 3
   deploy:
     drain_ms: 5000                   # 구 슬롯을 내릴 때까지의 drain 윈도우 (ms)
   ```

2. VPS 에 프록시 컨테이너 부팅:

   ```bash
   conoha proxy boot my-server --acme-email ops@example.com
   ```

3. DNS A 레코드를 VPS 로 향하게 하기 (Let's Encrypt HTTP-01 검증에 필요).

4. 프록시에 앱을 등록하고 배포:

   ```bash
   conoha app init my-server
   conoha app deploy my-server
   ```

5. 롤백 (drain 윈도우 내에서만, 이전 슬롯으로 즉시 전환):

   ```bash
   conoha app rollback my-server
   ```

`deploy --slot <id>` 로 슬롯 ID 를 고정할 수 있습니다 (규칙: `[a-z0-9][a-z0-9-]{0,63}`, 기본값은 git short SHA 또는 timestamp). 기존 슬롯명을 재사용하면 작업 디렉터리를 정리한 뒤 재전개합니다.

### no-proxy 모드: 평면 단일 슬롯

`conoha.yml` / proxy / DNS 없이도 가능한 최단 경로. `docker-compose.yml` 만 있으면 됩니다. SSH 로 `docker compose up -d --build` 를 실행하는 것과 동등하며, TLS / Host 기반 라우팅이 필요 없는 용도 (테스트, 사내 도구, 비 HTTP 서비스, 취미 배포) 에 적합합니다.

```bash
# 초기화 (Docker / Compose 설치만 수행, proxy 없음)
conoha app init my-server --app-name myapp --no-proxy

# 배포 (현재 디렉터리 tar → 업로드 → /opt/conoha/myapp/ 에 전개 → docker compose up -d --build)
conoha app deploy my-server --app-name myapp --no-proxy
```

이후의 `status` / `logs` / `stop` / `restart` / `destroy` 는 서버 마커에서 자동 판별되므로 `--no-proxy` 를 반복할 필요가 없습니다 (다시 넘겨도 에러는 아니며 no-op):

```bash
conoha app status my-server --app-name myapp
conoha app logs my-server --app-name myapp --follow
conoha app destroy my-server --app-name myapp
```

no-proxy 모드에는 blue/green 스왑이 없으므로 `rollback` 은 사용할 수 없습니다 (실행 시 "rollback is not supported in no-proxy mode" 에러가 발생). 이전 커밋으로 되돌리려면 `git checkout <sha> && conoha app deploy --no-proxy --app-name <app> <server>` 로 재배포하세요.

### 모드 전환

기존 앱의 모드를 바꾸려면 한 번 제거한 뒤 반대 모드로 재 init 합니다:

```bash
conoha app destroy my-server --app-name myapp            # 마커와 작업 디렉터리 제거
conoha app init my-server --app-name myapp --no-proxy    # 반대 모드로 재초기화
```

같은 VPS 위에서도 `<app-name>` 이 다르면 proxy / no-proxy 를 나란히 공존시킬 수 있습니다.

### 주요 플래그

| 플래그 | 명령 | 설명 |
|---|---|---|
| `--app-name <name>` | `destroy` / `status` / `logs` / `stop` / `restart` / `env` 에서는 항상, `init` / `deploy` / `rollback` 에서는 `--no-proxy` 와 함께 쓸 때 필수 | 앱 이름. 생략 시 TTY 있으면 대화 프롬프트, 비 TTY 환경에서는 지정 필수 |
| `--proxy` / `--no-proxy` | 모든 lifecycle 명령 | `init` 에서는 마커에 기록할 모드를 선택, 그 외에서는 마커를 덮어쓰기 (불일치 시 에러) |
| `--slot <id>` | `deploy` | 슬롯 ID 고정 (proxy 모드에서만 의미) |
| `--drain-ms <ms>` | `rollback` | 롤백 drain 윈도우 오버라이드 (0 = proxy 기본값) |
| `--follow` / `-f` | `logs` | 실시간 스트리밍 |
| `--service <name>` | `logs` | 특정 서비스만 |
| `--tail <n>` | `logs` | 출력 줄 수 (기본 100) |
| `--data-dir <path>` | proxy 를 호출하는 명령 | 서버 측 proxy 데이터 디렉터리 (기본 `/var/lib/conoha-proxy`) |

### 환경 변수 관리

배포를 가로질러 유지되는 환경 변수는 서버 측에서 관리합니다 (두 모드 공통):

```bash
conoha app env set my-server --app-name myapp DATABASE_URL=postgres://...
conoha app env list my-server --app-name myapp
conoha app env get my-server --app-name myapp DATABASE_URL
conoha app env unset my-server --app-name myapp DATABASE_URL
```

배포 시 `.env` 는 **리포지토리에 커밋된 `.env` → 서버 측 `/opt/conoha/<app>.env.server` (즉 `conoha app env set` 값) 순으로 이어붙여** 조립됩니다. 따라서 서버 측 값이 뒤에 오는 원칙에 따라 우선합니다. 리포지토리에 커밋한 `.env` 도 `docker compose` 에 그대로 전달됩니다.

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

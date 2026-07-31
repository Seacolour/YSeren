# YSeren Docker Headless

当前目录提供 YSeren Headless 的本地容器化验证骨架。它复用现有 Go
Core 和内嵌 Web Player，不复制媒体文件，也不引入另一套服务实现。

Docker Hub 上已有手动推送的 `seacolour/yseren:dev-amd64` 实验镜像。它只
包含 `linux/amd64`，不是稳定发布，也没有 `latest` 标签。当前不可变摘要为：

```text
sha256:ac7c0f4c08a734c8b05d575e0667a6e91cf1790d7d55a978a6b32527c66f5232
```

本地构建仍是当前首选验证路径；实验镜像用于少量测试，不代表真实 NAS 或
arm64 设备已经通过验证。

## 适用范围

- Linux 容器，当前目标架构为 `linux/amd64` 和 `linux/arm64`。
- 媒体通过只读 bind mount 直接访问，原文件仍留在宿主机原目录。
- 播放继续由局域网设备上的浏览器承担。
- YSeren 没有账号或访问密码，只应在可信局域网中开放端口。

Docker 不能让镜像跨越 Linux 内核或 CPU 架构运行，也不替代 Windows
Desktop 和 Android 应用。Docker Desktop 访问 Windows/macOS 文件时还会经过
宿主机文件共享层，性能和权限行为可能不同于原生 Linux/NAS。

## 使用 Compose 本地构建

PowerShell：

```powershell
$env:YSEREN_MEDIA_PATH = 'D:/Media/Videos'
docker compose up --build -d
docker compose ps
```

本仓库开发环境也可以直接使用测试目录：

```powershell
$env:YSEREN_MEDIA_PATH = 'D:/Code/yseren/resource'
docker compose up --build -d
```

启动后打开 <http://localhost:1479/>。同一局域网的其他设备应使用 Docker
宿主机的局域网地址，例如 `http://192.168.1.20:1479/`。

宿主机的 `1479` 端口已被占用时，可以覆盖映射端口：

```powershell
$env:YSEREN_HOST_PORT = '14880'
docker compose up --build -d
```

此时应打开 <http://localhost:14880/>，容器内部仍监听 `1479`。

停止并移除本地容器：

```powershell
docker compose down
```

## 使用 Docker Hub 实验镜像

当前实验镜像只支持 amd64。使用仓库中的 Compose 配置时：

```powershell
$env:YSEREN_MEDIA_PATH = 'D:/Media/Videos'
$env:YSEREN_IMAGE = 'docker.io/seacolour/yseren:dev-amd64'
docker compose pull yseren
docker compose up -d --no-build
```

需要锁定已经验证的内容时，可以直接使用摘要而不是可变标签：

```text
docker.io/seacolour/yseren@sha256:ac7c0f4c08a734c8b05d575e0667a6e91cf1790d7d55a978a6b32527c66f5232
```

请勿自行推断或使用 `latest`，因为仓库目前没有发布该标签。

Compose 会把：

- `YSEREN_MEDIA_PATH` 挂载到容器内 `/media`，并设为只读；
- [`docker/yseren.yaml`](./yseren.yaml) 挂载到 `/config/yseren.yaml`；
- `YSEREN_HOST_PORT`（默认 `1479`）映射到容器 TCP `1479`；
- 容器根文件系统设为只读，并移除全部 Linux capabilities。

YAML 中必须使用容器内路径 `/media`，不能写宿主机的 `D:/Media`、
`/mnt/media` 等路径。需要多个媒体源时，可以添加多个只读挂载，并在
YAML 中分别使用其容器内路径。

## 直接构建和运行

```powershell
docker build `
  --build-arg VERSION=dev `
  --build-arg VCS_REF=local `
  -t yseren:local .

docker run --rm --name yseren-local `
  -p 1479:1479 `
  -v 'D:/Media/Videos:/media:ro' `
  yseren:local
```

镜像默认读取 `/config/yseren.yaml`。Dockerfile 已包含指向 `/media` 的默认
配置；需要自定义时再额外挂载配置文件：

```powershell
-v 'D:/path/to/yseren.yaml:/config/yseren.yaml:ro'
```

## 当前容器约束

### 地址展示

bridge 网络中的 YSeren 只能看到容器自己的 `172.x.x.x` 地址，因此
`/api/status` 或终端输出的局域网地址可能不是外部设备应使用的地址。外部
设备应使用“宿主机局域网 IP + 映射端口”。正式发布镜像前，建议增加显式
`advertise_url`/`YSEREN_ADVERTISE_URL` 配置，避免普通用户复制错误地址。

### 文件权限与软链接

容器默认以 UID/GID `10001:10001` 的非 root 用户运行。Linux/NAS 上的媒体
目录必须允许该用户读取，或者在 Compose 中按部署环境覆盖 `user`。宿主机
软链接只有在目标也挂载到容器并保持可解析时才能使用；越出媒体源范围的
链接仍会被 YSeren 安全过滤。

### HTTPS 与健康检查

运行层包含 CA 证书，以保证版本检查可以访问 GitHub HTTPS。镜像使用
`/api/status` 进行 Docker `HEALTHCHECK`，并通过 `SIGTERM` 触发现有的优雅
退出逻辑。

## 正式发布前的剩余工作

1. 增加可配置的外部访问地址，并验证 Desktop/Headless 不受影响。
2. 为 GitHub Actions 配置 Docker Hub PAT，并在主分支实际运行多架构工作流。
3. 在至少一种真实 NAS/arm64 设备上验证目录权限、性能和持续运行。
4. 确认稳定镜像的签名策略；工作流已配置生成 provenance 和 SBOM attestations。

## GitHub Actions 发布策略

[`docker-publish.yml`](../.github/workflows/docker-publish.yml) 会先构建 amd64
镜像并运行容器级 smoke，再发布 `linux/amd64`、`linux/arm64` 清单：

- 手动运行时只接受 `dev-*` 标签；留空则使用 `dev-<commit SHA>`。
- 精确的 `vMAJOR.MINOR.PATCH` Git tag 会生成 `vMAJOR.MINOR.PATCH` 和
  `MAJOR.MINOR.PATCH` 两个镜像标签。
- 工作流不会创建 `latest`、主版本或次版本等可变稳定标签。
- 如果对应版本标签已经存在，工作流会拒绝覆盖并直接失败。

仓库需要配置两个 Actions secrets：

- `DOCKERHUB_USERNAME`：Docker Hub 用户名（当前为 `seacolour`）。
- `DOCKERHUB_TOKEN`：单独创建、具有仓库读写权限的 Docker Hub access token。

不要从 Docker Desktop 提取或复制其登录凭据；应在 Docker Hub 单独创建 PAT，
再通过 GitHub 仓库的 Actions secrets 页面保存。工作流合并到默认分支并配置
secrets 后，才应手动发布新的多架构开发标签。

# Docker Mirror Go - Docker Registry Proxy

一个高性能的 Docker 镜像加速代理服务，使用 Go 语言编写，用于解决获取 Docker 官方镜像无法正常访问的问题。


## 特性

- 🚀 **高性能**: 使用 Go 语言编写，性能优异
- 🐳 **多平台支持**: 支持 Docker Hub、GCR、Quay.io、GHCR 等多个镜像仓库
- 📦 **多架构**: 支持 AMD64 和 ARM64 架构
- 🔒 **安全**: 支持 Docker 仓库认证和 token 缓存
- ☁️ **云原生**: 支持 Docker 容器化部署
- 🔄 **自动构建**: 集成 GitHub Actions 自动构建和发布

## 快速开始

### 使用预构建的二进制文件

从 [Releases](https://github.com/woodchen-ink/docker-mirror-go/releases) 页面下载适合你系统的二进制文件：

```bash
# 下载并运行 (以 Linux AMD64 为例)
wget https://github.com/woodchen-ink/docker-mirror-go/releases/latest/download/docker-mirror-go-linux-amd64
chmod +x docker-mirror-go-linux-amd64
./docker-mirror-go-linux-amd64
```

### 使用 Docker

```bash
# 使用 GitHub Container Registry
docker run -p 8080:8080 ghcr.io/woodchen-ink/docker-mirror-go:latest

# 或者自己构建
docker build -t docker-mirror-go .
docker run -p 8080:8080 docker-mirror-go
```

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/woodchen-ink/docker-mirror-go.git
cd docker-mirror-go

# 安装依赖
go mod download

# 构建
go build -o docker-mirror-go .

# 运行
./docker-mirror-go
```

## 使用方法

启动服务后，将你的 Docker daemon 配置为使用代理：

### 方法一：配置 Docker daemon

编辑 `/etc/docker/daemon.json`：

```json
{
  "registry-mirrors": ["http://your-domain:8080"]
}
```

然后重启 Docker：

```bash
sudo systemctl restart docker
```

### 方法二：直接使用代理

```bash
# 拉取镜像时指定代理
docker pull your-domain:8080/library/nginx
docker pull your-domain:8080/library/redis

# 支持的仓库
docker pull your-domain:8080/gcr/google-containers/pause
docker pull your-domain:8080/quay/prometheus/prometheus
docker pull your-domain:8080/ghcr/actions/runner
```

## 配置

### 环境变量

- `PORT`: 服务监听端口 (默认: 8080)

### 支持的仓库

| 前缀 | 目标仓库 |
|------|----------|
| (无) | Docker Hub (registry-1.docker.io) |
| gcr | Google Container Registry (gcr.io) |
| k8sgcr | Kubernetes GCR (k8s.gcr.io) |
| quay | Quay.io |
| ghcr | GitHub Container Registry (ghcr.io) |

## 开发

### 本地开发

```bash
# 安装依赖
go mod download

# 运行
go run .

# 测试
go test ./...

# 格式化代码
go fmt ./...
```

### 项目结构

```
.
├── main.go                 # 主入口
├── internal/
│   ├── handler/           # HTTP 处理器
│   ├── backend/           # 后端代理逻辑
│   └── token/             # Token 管理和缓存
├── Dockerfile             # Docker 构建文件
├── .github/workflows/     # GitHub Actions 工作流
└── go.mod                 # Go 模块文件
```


## 许可证

MIT OR Apache-2.0

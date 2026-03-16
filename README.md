# proxytool

A lightweight CLI proxy manager for Linux servers. Downloads subscription links, tests node latency, and runs a local proxy via the [mihomo](https://github.com/MetaCubeX/mihomo) (Clash-Meta) engine. Controls Docker and system-wide proxy settings.

[![build](https://github.com/yoyofly3143/proxytool/actions/workflows/build.yml/badge.svg)](https://github.com/yoyofly3143/proxytool/actions/workflows/build.yml)

## 特性

- 支持 Clash YAML 和 V2Ray base64 订阅格式（vmess/ss/trojan/vless）
- 自动测速选最优节点
- 一键启停代理（基于 mihomo 内核，首次运行自动下载）
- 控制 Docker 代理（`/etc/docker/daemon.json`）
- 控制系统代理（`/etc/environment`）
- 静态编译，支持 CentOS 7+（kernel 3.10, glibc 2.17+）
- 无需安装 Python/Node.js 等运行时

## 安装

```bash
# amd64 服务器
wget https://github.com/yoyofly3143/proxytool/releases/latest/download/proxytool-linux-amd64 -O proxytool
chmod +x proxytool
sudo mv proxytool /usr/local/bin/

# arm64 服务器
wget https://github.com/yoyofly3143/proxytool/releases/latest/download/proxytool-linux-arm64 -O proxytool
chmod +x proxytool
sudo mv proxytool /usr/local/bin/
```

## 使用

### 1. 添加订阅

```bash
# Clash 格式订阅
proxytool sub add clash "https://api.wcc.best/sub?target=clash&url=..."

# V2Ray 格式订阅
proxytool sub add v2ray "https://qcpzz.pages.dev/zwz?sub=sub.mot.cloudns.biz"
```

### 2. 更新节点

```bash
proxytool sub update          # 更新所有订阅
proxytool sub update clash    # 只更新指定订阅
```

### 3. 查看和测速节点

```bash
proxytool node list           # 列出所有节点
proxytool node test           # 并发测速(TCP延迟)，从快到慢排列
proxytool node test -t 10     # 设置超时为10秒
```

### 4. 选择节点（可选）

```bash
proxytool node select "HK-01"   # 手动选择节点
# 不选则 proxy start 时自动选最快节点
```

### 5. 启停代理

```bash
proxytool proxy start         # 启动代理（HTTP:7890, SOCKS5:7891）
proxytool proxy stop          # 停止代理
proxytool proxy restart       # 重启代理
proxytool proxy status        # 查看状态
```

### 6. Docker 代理

```bash
proxytool docker enable       # 写入 /etc/docker/daemon.json
proxytool docker disable      # 清除 Docker 代理
proxytool docker status

# 之后重启 Docker 生效:
systemctl restart docker
```

### 7. 系统全局代理

```bash
proxytool system enable       # 写入 /etc/environment
proxytool system disable      # 清除代理变量
proxytool system status

# 使当前 shell 生效:
source /etc/environment
```

### 8. 查看总体状态

```bash
proxytool status
```

## 典型工作流（拉取 Docker 镜像）

```bash
# 第一次使用
proxytool sub add clash "<clash订阅URL>"
proxytool sub update
proxytool proxy start          # 自动测速选最优节点

# 开启 Docker 代理
proxytool docker enable
systemctl restart docker

# 拉镜像（Docker Desktop 会走代理）
docker pull nginx

# 完成后关闭
proxytool docker disable
systemctl restart docker
proxytool proxy stop
```

## 配置文件位置

| 文件 | 说明 |
|------|------|
| `~/.proxytool/config.json` | 工具配置（订阅、端口、选中节点） |
| `~/.proxytool/subs/` | 订阅缓存 |
| `~/.proxytool/mihomo/config.yaml` | 当前代理配置 |
| `~/.proxytool/mihomo/mihomo.pid` | 进程 PID |
| `~/.proxytool/mihomo/mihomo.log` | mihomo 日志 |

## 端口

| 类型 | 默认端口 |
|------|----------|
| HTTP 代理 | 7890 |
| SOCKS5 代理 | 7891 |

## 构建

```bash
# 需要 Go 1.21+（仓库 go.mod 以 Go 1.22 为基准）
make linux-amd64   # 构建 amd64 二进制
make linux-arm64   # 构建 arm64 二进制
make all           # 构建两者
```

## 注意事项

- `proxytool system enable/disable` 修改 `/etc/environment`，需要 root 权限（或该文件的写权限）
- `proxytool docker enable/disable` 修改 `/etc/docker/daemon.json`，通常需要 root 权限
- 修改 Docker 代理后需要 `systemctl restart docker` 才能生效
- mihomo 内核首次运行时自动从 GitHub 下载（约 20MB），需要网络连接

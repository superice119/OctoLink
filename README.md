# OctoLink — 章鱼物联

基于 [Oktopus](https://github.com/OktopUSP/oktopus) 二次开发的定制化 USP (TR-369) 设备管理系统。

## 简介

OctoLink 是一个开源的 USP Controller 和 CWMP 兼容的多厂商设备管理平台，专为 CPE 与物联网设备设计。任何遵循 TR-369 或 TR-069 协议的设备均可纳管。

## 目录结构

```
OctoLink/
├── backend/          # 后端微服务 (Go)
├── frontend/         # Web 管理界面 (Next.js)
├── deploy/           # 部署配置 (Docker Compose / Kubernetes)
│   ├── compose/
│   └── kubernetes/
├── docs/             # 项目文档 (见 docs/README.md 规范)
├── .env.example      # 环境变量模板（复制为 .env 后填写真实值）
└── .gitignore
```

## 快速开始

### 前置要求

- Docker & Docker Compose ≥ 2.x
- Go ≥ 1.21（本地开发）
- Node.js ≥ 18（前端本地开发）

### 1. 克隆并配置环境变量

```bash
git clone https://github.com/superice119/OctoLink.git
cd OctoLink
cp .env.example .env
# 编辑 .env，填写数据库密码、JWT_SECRET 等必填项
```

### 2. 使用 Docker Compose 启动

```bash
cd deploy/compose
docker compose up -d
```

### 3. 访问

- Web UI: http://localhost:3000
- API: http://localhost:8080

## 开发指南

- 提交规范：[Conventional Commits](https://www.conventionalcommits.org/)（`feat:`, `fix:`, `docs:`, `chore:` …）
- 代码合入：必须通过 Pull Request，禁止直接 push `main`
- 文档规范：见 [docs/README.md](docs/README.md)

## 安全须知 (Public 仓库)

- **严禁提交任何密钥/证书/口令**：NATS mTLS 证书（certPEM/keyPEM/rootCA）、`.env`、Docker Compose 明文密码、SSH 凭据等一律在 `.gitignore` 中排除。
- 敏感配置使用 `.env.example` 占位，实际值由运维人员在服务器本地设置。
- 数据库迁移及线上部署须经人类总监审批（`Awaiting Approval`）。

## 版权声明

本项目基于 [Oktopus](https://github.com/OktopUSP/oktopus) 进行二次开发，遵循其原始开源协议(见 [LICENSE](LICENSE))。  
上游版权归 OktopUSP 社区所有。品牌 Rebranding 在 WS-7 阶段完成。

> **基线 provenance**：导入自上游 `OktopUSP/oktopus` commit `e1f07d71a93c4169421f2e94ce6605746ece37ad`。  
> 导入时已剔除所有凭证材料(各服务 `.env`、NATS `*.pem` 证书),仅保留 `.env.example` 占位与 `nats_config/CERTS.md` 说明;上游原始 README 见 [README.upstream.md](README.upstream.md)。

## 文档

详见 [docs/](docs/) 目录，包含架构决策、接口说明与部署指引。

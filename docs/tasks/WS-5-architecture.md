# WS-5 (S1) — 云服务架构梳理 / 微服务边界 / API 契约基线

## 1. 背景与目标

- **Issue**: WS-5 (S1)
- **目标**: 完成 OctoLink 云服务的整体架构梳理，明确微服务边界，建立 API 契约基线，为后续 S2–S7 任务提供架构锚点。

## 2. 设计思路与方案选型

### 总体架构

```
终端层                         云服务层 (OctoLink fork of Oktopus)            接入层
┌───────────────┐   USP/TR-369   ┌──────────────────────────────┐   ┌──────────────┐
│ obuspa Agent  │  (MTP: MQTT/   │ 微服务: 设备管理 / USP 代理   │   │ Web UI (换标)│
│ prplOS/iopsys │   WebSocket)   │ 通知流转 / 多租户 RBAC        │◄──┤ 移动 App     │
│ + rtty client │ ─────────────► │ rttys / WS 控制转发           │   │ 微信小程序   │
└───────────────┘                └──────────────────────────────┘   └──────────────┘
        ▲ 开发期用 agent-sim 模拟设备
```

### 微服务边界 (Oktopus 上游结构)

| 服务 | 职责 | 技术栈 |
|---|---|---|
| controller | USP Controller 核心，处理 USP 消息收发与状态机 | Go |
| adapter-mqtt | MQTT MTP 适配器，桥接设备 MQTT 消息到内部 NATS | Go |
| adapter-ws | WebSocket MTP 适配器 | Go |
| nbi | Northbound Interface API，对外暴露 REST / WebSocket | Go |
| frontend | Web 管理界面 | Next.js |
| nats | 消息总线 (内部服务通信) | NATS |
| postgresql | 持久化存储 | PostgreSQL |

### 关键决策 (ADR 摘要)

- **ADR-1 云端可独立开发**: 用 agent-sim 模拟 USP Device，云端功能仅硬依赖 S1 架构基线。
- **ADR-2 终端方案**: prplOS obuspa 已验证；iopsys obuspa 由老板手动验证后由 Protocol_Pro (WS-6) 接手。
- **ADR-3 批量操作降风险**: 原"批量固件升级"改为只读"批量信息检索(MAC 等)"(WS-8)，缓做。
- **ADR-4 移动端双通道控制 (Phase 2)**: 优先本地直连，不可达回退云端 WebSocket 转发（WS-17）。
- **ADR-5 远程访问控制**: rttys (WS-16) 提供运维 Web 终端；与 App 业务控制通道分离。
- **ADR-6 安全基线**: 本地 WS 亦需 TLS + token；远程 shell 必须 RBAC + 会话审计 + 老板审批。

## 3. 接口 / 协议说明

### USP 协议层

| 协议 | 标准 | 用途 |
|---|---|---|
| TR-369 USP | BBF TR-369 | 设备管理消息协议（Get/Set/Operate/Notify/Subscribe） |
| MQTT v5 | OASIS MQTT | USP MTP for MQTT 设备 |
| WebSocket | RFC 6455 | USP MTP for WS 设备；移动 App 双通道控制 |

### Northbound REST API (基线)

| 端点 | 方法 | 说明 |
|---|---|---|
| `/api/v1/devices` | GET | 列出已注册设备 |
| `/api/v1/devices/{id}` | GET | 获取设备详情 |
| `/api/v1/devices/{id}/parameters` | GET/POST | 读取/设置 USP 参数 |
| `/api/v1/devices/{id}/operate` | POST | 触发 USP Operate 命令 |
| `/api/v1/notifications` | GET (WS) | 实时通知订阅 |

> 详细接口规范将在各 S2–S7 子任务文档中逐步完善。

## 4. 部署 / 使用步骤

### 本地开发环境

```bash
# 1. 克隆仓库
git clone https://github.com/superice119/OctoLink.git && cd OctoLink

# 2. 配置环境变量
cp .env.example .env
# 编辑 .env 填写 POSTGRES_PASSWORD、JWT_SECRET 等

# 3. 启动所有服务
cd deploy/compose
docker compose up -d

# 4. 启动设备模拟器 (无真实设备时)
# 参见 https://github.com/OktopUSP/agent-sim
```

### 测试服务器 (39.97.250.156)

- Docker Compose 已部署，由老板手动维护。
- 域名 / 端口见 WS-5 issue 评论（不在此处记录，避免信息腐烂）。

## 5. 测试与验收记录

| 测试项 | 方法 | 状态 |
|---|---|---|
| agent-sim 设备模拟接入 | 手动启动 agent-sim 并连接 NATS | 待测 (WS-12) |
| USP Get 参数 | REST API 调用 + WS 前端验证 | 待测 (WS-9) |
| 前端登录 / 设备列表 | 浏览器手动测试 | 待测 |

## 6. 变更记录 (Changelog)

| 日期 | 变更 |
|---|---|
| 2026-06-16 | 初始架构文档，整合自 WS-5 (S1) 架构梳理报告，作为 docs/ 脚手架种子文件 (WS-18) |

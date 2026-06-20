# WS-12 (S8) — MVP 测试用例清单与自动化测试体系

**日期:** 2026-06-17
**作者:** QA_Sherlock
**依赖:** WS-5 架构基线 / WS-6 USP 协议联调清单 / WS-7、WS-9、WS-10、WS-11 功能交付
**测试基线:** Controller `39.97.250.156`，USP 参数路径以 TR-181 为准

---

## 1. 背景与目标

WS-12 负责 OctoLink MVP 的全周期质量把关。本阶段按“用例先行”原则，在 S3-S7 代码完全交付前先输出测试用例清单，为后续自动化脚本、接口联调、协议抓包与验收放行提供统一基准。

本轮目标：

- 覆盖 S3 Rebranding UI、S5 实时 USP Get/Set、S6 通知推送、S7 多租户 RBAC 的 MVP 验收路径。
- 明确 API、Web UI、USP 协议、安全、并发和 agent-sim 端到端测试边界。
- 标注自动化优先级与准入条件，避免功能交付后再补测试设计。
- 在核心功能未 100% 通过前，不推进“可发布”；核心链路需老板人工确认。

## 2. 设计思路与方案选型

### 2.1 测试分层

| 层级 | 工具 / 方法 | 覆盖目标 | 触发条件 |
|---|---|---|---|
| API 契约测试 | Pytest + requests / Postman-Newman | REST API 状态码、字段、鉴权、错误码 | S5-S7 API 可用后 |
| Web UI 自动化 | Playwright | 登录、设备列表、Rebranding、Get/Set 操作、通知展示、权限隔离 | S3/S5/S6/S7 前端交付后 |
| USP 协议测试 | agent-sim + 抓包 + 协议日志 | TR-369 消息类型、msg_id、EndpointID、TR-181 路径、错误码 | agent-sim 可连接 Controller 后 |
| 并发 / 稳定性 | k6 / locust / pytest-xdist | 多设备、多用户、批量 Get、通知风暴、长连接稳定性 | 核心接口稳定后 |
| 安全测试 | RBAC 用例 + OWASP API 检查 | 越权、租户隔离、Token 失效、参数注入、敏感日志 | S7 权限交付后 |
| 手工验收 | 浏览器 + 老板确认 | 品牌视觉、核心业务路径、真实终端补充验证 | 自动化通过后 |

### 2.2 环境策略

- **首选模拟设备:** 使用 `agent-sim` 模拟 USP Agent，不依赖真实终端即可覆盖云端 E2E。
- **Controller 地址:** 统一使用 `39.97.250.156`，避免测试脚本散落历史地址。
- **真实终端补充:** prplOS / iopsys obuspa 真机抓包作为后续补充，不阻塞本轮用例清单。
- **数据隔离:** 每轮自动化创建独立租户、用户、设备标识与测试 Subscription，结束后清理。

## 3. 接口 / 协议说明

### 3.1 API 基线

| 端点 | 方法 | 主要验证点 |
|---|---|---|
| `/api/v1/devices` | GET | 设备列表、租户隔离、分页/空列表、离线状态 |
| `/api/v1/devices/{id}` | GET | 设备详情字段、EndpointID、未知设备错误 |
| `/api/v1/devices/{id}/parameters` | GET | USP Get、TR-181 路径、通配符、错误路径 |
| `/api/v1/devices/{id}/parameters` | POST | USP Set、只读参数拒绝、类型校验、部分失败回滚 |
| `/api/v1/devices/{id}/operate` | POST | Operate 命令准入、权限校验、超时处理 |
| `/api/v1/notifications` | WS / GET | 通知订阅、断线重连、权限过滤、消息顺序 |

### 3.2 USP / TR-181 参数基线

| 类型 | 参数路径 | 用途 | 预期 |
|---|---|---|---|
| 设备身份 | `Device.LocalAgent.EndpointID` | Agent 路由键 | 返回非空且与设备详情一致 |
| 厂商信息 | `Device.DeviceInfo.Manufacturer` | 设备基础信息 | Get 成功，类型为 string |
| OUI | `Device.DeviceInfo.ManufacturerOUI` | 设备身份 / MAC 关联 | Get 成功，格式为 6 位十六进制 |
| 型号 | `Device.DeviceInfo.ModelName` | 设备详情展示 | Get 成功，UI 与 API 一致 |
| 序列号 | `Device.DeviceInfo.SerialNumber` | 设备唯一标识 | Get 成功，租户内唯一 |
| 软件版本 | `Device.DeviceInfo.SoftwareVersion` | 通知 ValueChange 目标 | Get 成功，可用于订阅验证 |
| 运行时长 | `Device.DeviceInfo.UpTime` | 实时参数验证 | Get 成功，数值非负 |
| 控制器配置 | `Device.LocalAgent.Controller.1.` | Set 权限/回滚验证 | 仅允许授权角色修改 |
| 订阅对象 | `Device.LocalAgent.Subscription.` | S6 通知链路 | Add/Notify/清理符合 TR-369 |

## 4. 部署 / 使用步骤

### 4.1 后续自动化目录建议

```text
tests/
├── api/
│   ├── test_devices.py
│   ├── test_parameters_get_set.py
│   └── test_rbac.py
├── e2e/
│   ├── rebranding.spec.ts
│   ├── parameters.spec.ts
│   └── notifications.spec.ts
├── protocol/
│   ├── test_usp_get_set.py
│   └── test_notify_subscription.py
└── load/
    └── usp_get_stress.js
```

### 4.2 自动化环境变量建议

| 变量 | 示例 | 说明 |
|---|---|---|
| `OCTOLINK_BASE_URL` | `https://39.97.250.156` | Web / API 根地址 |
| `OCTOLINK_API_BASE` | `https://39.97.250.156/api/v1` | REST API 根地址 |
| `OCTOLINK_WS_URL` | `wss://39.97.250.156/api/v1/notifications` | 通知 WebSocket |
| `OCTOLINK_ADMIN_USER` | `qa-admin@example.com` | 自动化管理员账号 |
| `OCTOLINK_ADMIN_PASSWORD` | secret | 自动化管理员密码 |
| `AGENT_SIM_ENDPOINT_ID` | `os::00256D-QAAgent001` | agent-sim EndpointID |
| `TEST_TENANT_A` / `TEST_TENANT_B` | `qa-tenant-a` / `qa-tenant-b` | 租户隔离测试 |

### 4.3 执行顺序

1. 部署 OctoLink 云服务，确认 Controller 指向 `39.97.250.156`。
2. 启动 agent-sim，注册至少 2 台模拟设备，EndpointID 使用不同 SerialNumber。
3. 准备 `admin`、`tenant_admin`、`operator`、`readonly` 四类账号。
4. 先跑 API 契约测试，再跑 Playwright E2E，最后跑并发与安全回归。
5. 任一 P0/P1 用例失败即驳回，不进入发布候选。

## 5. 测试与验收记录

### 5.1 S3 Rebranding UI 用例

| ID | 优先级 | 场景 | 步骤 | 预期结果 | 自动化 |
|---|---|---|---|---|---|
| UI-RB-001 | P0 | 登录页品牌替换 | 打开 Web 首页 / 登录页 | 展示 OctoLink / 章鱼物联品牌，不能出现 Oktopus 默认品牌、旧版权或旧 favicon | Playwright |
| UI-RB-002 | P1 | 管理后台导航品牌一致 | 登录后检查侧边栏、顶部栏、页脚 | Logo、产品名、版权文案一致，无混用旧品牌 | Playwright |
| UI-RB-003 | P1 | 移动端响应式 | 使用 375px、768px、1440px 视口访问核心页面 | 文案不重叠，Logo 不变形，关键按钮可点击 | Playwright |
| UI-RB-004 | P2 | 静态资源缓存 | 强刷 / 无缓存访问 | favicon、manifest、title 更新为 OctoLink | Playwright |

### 5.2 S5 实时 USP Get/Set 用例

| ID | 优先级 | 场景 | 步骤 | 预期结果 | 自动化 |
|---|---|---|---|---|---|
| USP-GS-001 | P0 | 单参数 Get | 对在线设备读取 `Device.DeviceInfo.SoftwareVersion` | API 200，返回值非空；USP `GET_RESP` msg_id 与请求匹配 | Pytest + agent-sim |
| USP-GS-002 | P0 | 多参数 Get | 同时读取 Manufacturer、ModelName、SerialNumber、UpTime | 全部参数返回，字段类型符合 TR-181，UI 展示与 API 一致 | Pytest + Playwright |
| USP-GS-003 | P0 | 无效路径 Get | 读取 `Device.DeviceInfo.NonExistent` | 返回协议错误映射，不能 500；日志包含 request id | Pytest |
| USP-GS-004 | P0 | Set 可写参数 | 修改允许写入的测试参数 | 返回成功，二次 Get 可读到新值，通知链路可观察到变更 | Pytest + agent-sim |
| USP-GS-005 | P0 | Set 只读参数 | 尝试写 `Device.DeviceInfo.SerialNumber` | 被拒绝，原值不变，错误原因可读 | Pytest |
| USP-GS-006 | P1 | Set 类型错误 | 向 boolean / unsignedInt 参数传非法字符串 | 返回 4xx 或协议错误映射，无部分脏写 | Pytest |
| USP-GS-007 | P1 | 设备离线 | 断开 agent-sim 后发 Get/Set | 返回离线 / 超时错误，UI 不长时间转圈 | Pytest + Playwright |
| USP-GS-008 | P1 | 并发 Get | 20 用户并发读取 50 台模拟设备基础信息 | 无 5xx，P95 延迟在验收阈值内，NATS/Controller 无异常退出 | k6 / locust |

### 5.3 S6 通知推送用例

| ID | 优先级 | 场景 | 步骤 | 预期结果 | 自动化 |
|---|---|---|---|---|---|
| USP-NT-001 | P0 | 建立通知订阅 | 为 `Device.DeviceInfo.SoftwareVersion` 创建 ValueChange Subscription | Subscription 创建成功，Agent 返回实例路径 | Pytest + agent-sim |
| USP-NT-002 | P0 | ValueChange 推送 | 触发订阅参数变化 | API / WS 收到通知，包含 deviceId、path、value、timestamp | Pytest + Playwright |
| USP-NT-003 | P0 | 通知权限过滤 | 租户 A 用户监听租户 B 设备通知 | 租户 A 收不到租户 B 通知 | Pytest |
| USP-NT-004 | P1 | WS 断线重连 | 建立通知 WS 后断网 / 重连 | 客户端自动重连，不重复订阅，不丢未确认错误 | Playwright |
| USP-NT-005 | P1 | 通知风暴 | 1 分钟内模拟 1000 条通知 | 服务不崩溃，无明显乱序，背压 / 限流行为可观测 | k6 / custom |
| USP-NT-006 | P1 | NOTIFY_RESP | send_resp=true 时观察 Controller 响应 | Controller 返回 `NOTIFY_RESP`，msg_id 对应 | 协议测试 |

### 5.4 S7 多租户 / RBAC 用例

| ID | 优先级 | 场景 | 步骤 | 预期结果 | 自动化 |
|---|---|---|---|---|---|
| RBAC-001 | P0 | 租户设备列表隔离 | A/B 租户分别登录查看设备 | 只能看到本租户设备 | Pytest + Playwright |
| RBAC-002 | P0 | 越权读取设备详情 | 租户 A 请求租户 B deviceId | 返回 403/404，不能泄漏设备字段 | Pytest |
| RBAC-003 | P0 | 只读角色禁止 Set | readonly 用户执行参数 Set | 返回 403，Agent 不收到 SET | Pytest |
| RBAC-004 | P0 | operator 可执行 Get | operator 用户读取授权设备参数 | Get 成功，但无租户管理权限 | Pytest |
| RBAC-005 | P1 | Token 过期 | 使用过期 / 篡改 Token 请求 API 和 WS | 请求被拒绝，WS 关闭原因明确 | Pytest |
| RBAC-006 | P1 | 审计记录 | admin 执行 Set / Operate | 审计日志记录操作者、租户、设备、路径、结果 | Pytest |

### 5.5 安全与异常用例

| ID | 优先级 | 场景 | 步骤 | 预期结果 | 自动化 |
|---|---|---|---|---|---|
| SEC-001 | P0 | 未认证访问 | 不带 Token 调用 devices / parameters / notifications | 返回 401，无敏感信息 | Pytest |
| SEC-002 | P0 | 参数路径注入 | 路径中插入 SQL / shell / `../` / 超长字符串 | 返回校验错误，无 5xx，无日志注入 | Pytest |
| SEC-003 | P1 | CORS / Origin | 使用非白名单 Origin 建立 WS / API 请求 | 被拒绝或不返回授权头 | Pytest |
| SEC-004 | P1 | 敏感日志 | 触发登录失败、Set 失败、协议失败 | 日志不打印密码、Token、密钥 | 手工 + 日志扫描 |
| SEC-005 | P1 | 速率限制 | 对登录 / 参数接口发起高频请求 | 触发限流或保护策略，服务可恢复 | k6 |

### 5.6 agent-sim 端到端用例

| ID | 优先级 | 场景 | 步骤 | 预期结果 | 自动化 |
|---|---|---|---|---|---|
| E2E-SIM-001 | P0 | 模拟设备注册 | 启动 agent-sim 连接 Controller | 设备出现在 `/api/v1/devices` 和 UI 列表 | Pytest + Playwright |
| E2E-SIM-002 | P0 | 设备详情到参数读取 | 在 UI 打开模拟设备详情并读取 DeviceInfo | UI 展示与 API / USP 响应一致 | Playwright |
| E2E-SIM-003 | P0 | Set 后通知闭环 | UI 修改可写参数，监听通知 | Set 成功，通知到达，最终 Get 值一致 | Playwright + Pytest |
| E2E-SIM-004 | P1 | 多设备并行 | 同时连接 10 台 agent-sim | 设备路由不串线，EndpointID 唯一 | Pytest |
| E2E-SIM-005 | P1 | 3-5 台终端基础接入 | 同时启动 3-5 台 agent-sim，各用不同 SerialNumber / EndpointID 接入 Controller；逐台验证设备出现在 `/api/v1/devices` 与 UI 列表，打开详情读 DeviceInfo，并对每台执行参数 Get/Set | 设备归属租户正确，EndpointID 唯一，列表 / 详情数据一致，参数 Get/Set 成功，S6 通知定向到正确租户房间且不串线 | Pytest + Playwright |

### 5.7 当前执行状态

| 交付项 | 状态 | 说明 |
|---|---|---|
| 测试用例清单 | 已完成 | 本文档即本轮交付 |
| 自动化脚本 | 待执行 | 依赖 S3-S7 功能交付后促活 |
| agent-sim E2E | 待执行 | 可在 S5/S6 接口落地后优先联调 |
| 真机协议抓包 | 待补充 | 配合 Protocol_Pro 在真实 obuspa 设备上补测 |
| 老板人工确认 | 待触发 | 自动化通过后对核心功能与品牌效果确认 |

## 6. 变更记录 (Changelog)

| 日期 | 变更 |
|---|---|
| 2026-06-20 | 新增 E2E-SIM-005，覆盖 3-5 台 agent-sim 基础接入、租户归属、EndpointID 唯一性、Get/Set 与通知房间隔离 |
| 2026-06-17 | 新建 WS-12 MVP 测试用例清单，覆盖 S3/S5/S6/S7、权限安全、并发与 agent-sim E2E 规划 |

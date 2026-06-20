# WS-10 通知推送流转 (USP Notify)

## 1. 背景与目标
- 对应议题：WS-10《S6 通知推送流转 (USP Notify)》。
- 目标：打通 USP Notify 从设备上报、NATS 分发、后端持久化到前端实时展示/历史查询的完整链路。
- 范围覆盖：`mtp/adapter`、`socketio`、`controller`、前端通知中心与文档交付。

## 2. 设计思路与方案选型
### 2.1 端到端链路
1. 设备通过 MQTT / WebSocket 等 MTP 上报 USP Notify。
2. `mtp/adapter` 在 JetStream 消费侧识别 `notify` 类型，解包 USP Record / USP Msg，归一化为 JSON 通知事件并发布到 `notification.v1.{device_sn}`。
3. `socketio` 订阅 `notification.v1.>`，向浏览器广播 `usp_notify`，同时预留 Webhook 推送出口。
4. `controller` 同步订阅 `notification.v1.>`，把通知写入 MongoDB `octolink.notifications`。
5. 前端通过 REST 读取历史，通过 Socket.IO 接收实时通知，在顶部铃铛和通知中心统一展示。

### 2.2 方案选择
- **NATS Subject 归一化**：统一使用 `notification.v1.{sn}`，方便 WebSocket 广播、控制器持久化和后续多端复用。
- **MongoDB 持久化**：补充历史查询、已读状态、清空能力，避免 Notify 只存在于实时通道。
- **Context + Reducer**：前端使用 `NotificationProvider` 统一管理历史数据、实时追加、已读/清空操作，降低页面耦合。
- **Webhook 预留**：`socketio` 增加 `WEBHOOK_URL`，便于 Phase 2 对接移动端或小程序推送。

## 3. 接口 / 协议说明
### 3.1 NATS 主题
- 输入：MTP Adapter JetStream 侧既有 USP 主题（识别末尾 `notify`）
- 输出：`notification.v1.{device_sn}`

### 3.2 通知事件 JSON
```json
{
  "device_sn": "string",
  "subscription_id": "string",
  "type": "event|value_change|obj_creation|obj_deletion|oper_complete|on_board_req|unknown",
  "obj_path": "string",
  "event_name": "string",
  "params": {"key": "value"},
  "param_path": "string",
  "param_value": "string",
  "timestamp": "RFC3339"
}
```

### 3.3 REST API
#### GET `/api/notifications`
- 查询参数：
  - `page`：页码，默认 `1`
  - `page_size`：每页条数，默认 `20`，最大 `100`
  - `device_sn`：可选，按设备过滤
- 返回：
```json
{
  "notifications": [],
  "total": 0,
  "unread": 0
}
```

#### PUT `/api/notifications/read`
- 请求体：
```json
{ "all": true }
```
或
```json
{ "ids": ["mongo_object_id"] }
```
- 作用：全部已读或按 ID 批量已读。

#### DELETE `/api/notifications`
- 查询参数：`device_sn` 可选
- 返回：
```json
{ "deleted": 0 }
```

### 3.4 前端实时事件
- Socket.IO event：`usp_notify`
- Payload：与 NATS JSON 结构一致，前端收到后默认标记为未读。

## 4. 部署 / 使用步骤
### 4.1 SocketIO 环境变量
```env
CORS_ALLOWED_ORIGINS="http://localhost:3000"
NATS_URL="nats://nats:4222"
WEBHOOK_URL=""
```

### 4.2 编译 / 运行
```bash
cd backend/services/mtp/adapter && go build ./...
cd backend/services/controller && go build ./...
cd backend/services/utils/socketio && npm install && npm start
cd frontend && npm install && npm run lint && npm run build
```

### 4.3 使用说明
1. 登录前端后，顶部导航显示通知铃铛与未读数。
2. 打开 `/notifications` 查看历史通知。
3. 设备发送 USP Notify 后：
   - 前端实时收到 `usp_notify`
   - `controller` 自动入库
   - 用户可执行“全部标为已读”或“清空”。

## 5. 测试与验收记录
### 5.1 本次执行的校验
- `backend/services/mtp/adapter`: `go build ./...`
- `backend/services/controller`: `go build ./...`
- `frontend`: `npm run lint`（安装依赖后）
- `frontend`: `npm run build`（安装依赖后）
- `backend/services/utils/socketio`: `npm install` 更新 `nats` 依赖

### 5.2 建议联调场景
1. 使用 agent-sim 或真实终端触发 USP Notify。
2. 观察 `notification.v1.{sn}` 是否有消息。
3. 验证前端铃铛未读数实时增长。
4. 验证 `/api/notifications` 历史查询与已读/清空接口。
5. 如配置 `WEBHOOK_URL`，验证移动端推送出口收到同源 JSON。

## 6. 变更记录 (Changelog)
- 2026-06-18
  - `mtp/adapter` 新增 USP Notify 解析与通知发布。
  - `socketio` 新增 NATS 订阅桥接和 Webhook 预留出口。
  - `controller` 新增 notifications 集合、订阅入库与 REST API。
  - 前端新增通知上下文、通知中心页面、顶部铃铛入口。
  - 新增任务文档 `docs/tasks/WS-10-notifications.md`。
  - S6 QA Round 1: 修复匿名连接（JWT 握手），补齐 SessionContextRecord 解析，发送 NotifyResp 回包。
  - S6 QA Round 2: 同步 package-lock.json，设备级 NATS 验证，NotifyResp 路由修正，cwmp.go int→string 修复。
  - S6 QA Round 3: 移除 tenant:all 全量推送；按 S7 模型路由 tenant:{owner_tenant_id} + device:{sn}；subscribe_device 检查 device.Customer。
- 2026-06-20
  - S6 QA Round 4: 落地 S7 JWT claim 契约（role + tenant_id 进入 controller 签发的 token）。
    - `auth/auth.go`: JWTClaim 新增 Role/TenantID，GenerateJWT 接受 role/tenantID，添加 ValidateTokenFull。
    - `db/user.go`: User 新增 TenantID/Role 字段，EffectiveRole() 方法（兼容旧 Level 字段），内联角色常量与 DefaultTenantID。
    - `api/user.go`: 登录时将 EffectiveRole()+TenantID 编入 JWT。
    - `socketio/server.js`: subscribe_device 加强 — 非 super_admin 时 deviceOwnerTenant 与 tenantId 均须非空且完全匹配。
  - **已知问题（不阻塞本轮合并）**：前端 npm run lint（missingRefs/defaultMeta）、npm audit high 漏洞、socketio npm audit — 均为既存构建链问题。
  - S6 QA Round 5 (post-S7 rebase): S6 branch rebased onto S7 merge commit `e697051`.
    - Conflicts resolved: `api/api.go` (notifs middleware added to S7 auth chain), `db/db.go` (notifications collection added alongside S7 tenants/roles collections), `frontend/.../config.js` (通知 nav item coexists with Roles/Tenants items).
    - Removed duplicate constants from `db/user.go` (now live in `role.go` + `tenant.go` via S7).
    - `api/user.go` login: upgraded to `user.EffectiveTenantID()` — non-super_admin with empty TenantID now falls back to `"default"`, so JWT claim `tenant_id` is never empty for regular users.

## 5. 测试与验收记录

### 5.1 编译验证
- `backend/services/controller`: `go build ./...` ✅
- `backend/services/mtp/adapter`: `go build ./...` ✅
- `backend/services/utils/socketio`: `node --check server.js` ✅

### 5.2 端到端联调场景（待联调后补全日志/截图）

以下为验收清单，S7 合并并部署后执行：

| # | 场景 | 预期结果 |
|---|------|---------|
| 1 | 租户 A 用户登录，JWT 含 `tenant_id=A`，device.Customer=A 触发 Notify | 只有租户 A 的 socket 连接收到 `usp_notify`；租户 B 无收到 |
| 2 | 无 TenantID 旧账户登录 | `EffectiveTenantID()` 返回 `"default"`, 用户加入 `tenant:default`；Customer="default" 的设备通知可达 |
| 3 | 租户 B 用户调用 `subscribe_device` 订阅属于租户 A 的设备 | `subscribe_error: Device not found or access denied` |
| 4 | super_admin 用户连接并调用 `subscribe_device` 订阅任意设备 | 加入 `admin` 房间，所有设备通知均可达；subscribe_device 无需租户匹配 |
| 5 | 匿名客户端（无 token）连接 | `Authentication required: no token provided`，连接拒绝 |

### 5.3 已知问题（不阻塞本轮合并）
- 前端 `npm run lint`：`missingRefs` / `defaultMeta` 警告 — 既存构建链问题
- 前端 `npm audit`：high 漏洞 — 既存依赖问题
- socketio `npm audit` — 既存依赖问题

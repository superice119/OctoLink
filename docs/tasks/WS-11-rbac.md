# WS-11 基础多租户 / 权限管理 (RBAC)

## 背景与目标

OctoLink 需要支持多租户隔离与基于角色的访问控制（RBAC），为商业化多租户场景打基础。本模块在现有的用户认证体系上新增了：

- 租户（Tenant）模型与隔离
- 角色（Role）与权限（Permission）体系
- 鉴权中间件升级：JWT 携带角色与租户信息
- 前端：用户/角色/租户管理界面 + 权限受控菜单

---

## 设计思路与方案选型

### 角色设计（内建角色）

| 角色 | 权限 | 说明 |
|---|---|---|
| `super_admin` | 全部 | 全局管理员，跨租户可见 |
| `tenant_admin` | 设备读写 + 用户读写 | 租户内管理员 |
| `operator` | 设备读写 | 操作员 |
| `viewer` | 设备只读 | 观察者 |

### 权限常量

```
devices:read   devices:write
users:read     users:write
tenants:manage roles:manage
```

### 租户隔离策略

- 每个用户绑定一个 `tenant_id`
- 查询用户/设备时，非 `super_admin` 仅能看到本租户数据
- 默认租户 ID 为 `default`（向后兼容已有用户）

### JWT 升级

JWT Payload 新增 `role` 和 `tenant_id` 字段，中间件解析后注入 request context，无需每次请求查 MongoDB：

```json
{
  "email": "admin@example.com",
  "username": "Admin",
  "role": "super_admin",
  "tenant_id": "default",
  "exp": ...
}
```

### 向后兼容

- 已有用户 `role` 字段为空时，`EffectiveRole()` 回退：`AdminUser(1)` → `super_admin`，`NormalUser(0)` → `operator`
- 已有 `Level` 字段保留，不破坏存量数据

---

## 接口说明

### Tenant API（需 `super_admin`）

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/tenants` | 列出所有租户 |
| POST | `/api/tenants` | 创建租户 |
| PUT | `/api/tenants/{id}` | 更新租户 |
| DELETE | `/api/tenants/{id}` | 删除租户（不可删 `default`）|

创建租户示例：
```json
POST /api/tenants
{ "name": "AcmeCorp", "description": "Acme Corporation tenant" }
```

### Role API（需 `super_admin` 或 `tenant_admin`）

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/roles` | 列出角色（super_admin 见全部；其他见系统角色 + 本租户自定义角色）|
| POST | `/api/roles` | 创建自定义角色 |
| PUT | `/api/roles/{id}` | 更新角色权限（系统角色只读）|
| DELETE | `/api/roles/{id}` | 删除自定义角色 |
| POST | `/api/roles/assign` | 为用户分配角色和租户 |

分配角色示例：
```json
POST /api/roles/assign
{ "email": "user@example.com", "role": "operator", "tenant_id": "acme-corp-uuid" }
```

### User API 变化

- `GET /api/users`：`super_admin` 返回全部用户；其他角色仅返回同租户用户
- `POST /api/auth/register`：支持 `role`/`tenant_id` 字段；`tenant_admin` 创建的用户自动归属自身租户
- JWT login 响应现在包含 `role` 和 `tenant_id`

---

## 数据模型

### Tenant（MongoDB `account-mngr.tenants`）

```go
type Tenant struct {
    ID          string `bson:"_id"`
    Name        string `bson:"name"`
    Description string `bson:"description"`
    CreatedAt   int64  `bson:"created_at"`
}
```

### Role（MongoDB `account-mngr.roles`）

```go
type Role struct {
    ID          string   `bson:"_id"`
    Name        string   `bson:"name"`
    TenantID    string   `bson:"tenant_id"` // 空 = 系统角色
    Permissions []string `bson:"permissions"`
    IsSystem    bool     `bson:"is_system"`
}
```

### User（新增字段）

```go
type User struct {
    ...
    TenantID string `bson:"tenant_id,omitempty"`
    Role     string `bson:"role,omitempty"`
}
```

---

## 部署与使用步骤

### 1. 启动服务

无需额外迁移脚本。服务启动时 `NewDatabase()` 自动 seed：
- 默认租户 `default`
- 4 个内建系统角色

```bash
# 与原来相同，无配置变更
docker compose up -d
```

### 2. 初始化超级管理员

首次注册 Admin（无需 token，同原来）：
```bash
curl -X POST /api/auth/admin/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"secret123"}'
```

Admin 默认角色为 `super_admin`，`tenant_id` 为 `default`。

### 3. 创建租户

```bash
curl -X POST /api/tenants \
  -H "Authorization: <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"TenantA","description":"First tenant"}'
```

### 4. 创建租户管理员

```bash
curl -X POST /api/auth/register \
  -H "Authorization: <super_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{"email":"ta@example.com","password":"pass","role":"tenant_admin","tenant_id":"<tenantA-id>"}'
```

---

## 前端变更

### 新增页面

- `/access-control/roles` — 角色列表；`super_admin`/`tenant_admin` 可创建/删除自定义角色
- `/access-control/tenants` — 租户列表（仅 `super_admin` 可访问）

### 导航栏

新增 **Roles** 和 **Tenants** 菜单项（`config.js`）。

### 用户列表升级

- 新增 **Role** 列（`super_admin`/`tenant_admin` 可下拉修改角色）
- 新增 **Tenant** 列
- 创建用户时支持设置 `role` 字段

### Auth Context

登录后 JWT payload 解析出 `role` 和 `tenantId`，存入 `auth.user` 供组件消费。

---

## 测试与验收记录

### 待 QA_Sherlock 覆盖的安全用例

1. **租户越权**：`tenant_admin` 用户尝试访问其他租户的用户列表 → 应返回 403 或只返回本租户数据
2. **角色提权**：`operator` 用户尝试 POST `/api/tenants` → 应返回 403
3. **系统角色保护**：尝试 DELETE `/api/roles/super_admin` → 应返回 403
4. **JWT 过期**：使用过期 token 访问任意 `/api/device` → 应返回 401
5. **非 super_admin 创建租户**：`tenant_admin` 调用 POST `/api/tenants` → 应返回 403
6. **角色分配越权**：`tenant_admin` 尝试为用户分配 `super_admin` 角色并将其移到其他租户 → 租户 ID 应被强制设为调用者自身的租户

---

## 变更记录 (Changelog)

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-06-18 | 初始 RBAC 实现：租户模型、4 内建角色、JWT 升级、Tenant/Role/User API、前端管理界面 |
| v1.1 | 2026-06-18 | QA 修复：设备租户隔离（全路径）、tenant_admin 跨租户越权修复、RequirePermission 中间件挂载、前端菜单按角色过滤；新增 RBAC 单元测试 |
| v1.2 | 2026-06-18 | QA Round 3 修复：空 tenant_id 中间件拒绝、CWMP/WiFi/FW 租户校验、extractable cross-tenant 守卫重构、handler 级隔离测试 |

### v1.2 修复说明

1. **空 `tenant_id` 中间件拒绝（Issue 1）**
   - `Middleware()` 新增：非 `super_admin` 且 `tenant_id == ""` → 403，在进入任何业务 handler 之前即拒绝。
   - `retrieveDevices` 中移除 `callerTenantID != ""` 冗余保护分支（中间件已保证非空），避免绕过。
   - 新增 `TestMiddleware_EmptyTenantID_NonSuperAdmin_Rejected` / `_SuperAdmin_Allowed` 测试。

2. **CWMP/WiFi/FW per-SN 归属校验（Issue 2）**
   - `cwmp.go` 所有 7 个 handler（`cwmpGenericMsg`、`cwmpGetParameterNamesMsg`、`cwmpGetParameterAttributesMsg`、`cwmpGetParameterValuesMsg`、`cwmpSetParameterValuesMsg`、`cwmpAddObjectMsg`、`cwmpDeleteObjectMsg`）均在第一行追加 `if !a.requireDeviceAccess(w, r, sn) { return }`。
   - `wifi.go` `deviceWifi` 和 `fwupdate.go` `deviceFwUpdate` 同步补齐。
   - 至此全部 per-SN 控制路径（USP 8 条 + CWMP 7 条 + WiFi + FW）均受 `requireDeviceAccess()` 保护。

3. **JWT claim 契约确认（Issue 3）**
   - `user.go:generateToken`（line 356）调用 `auth.GenerateJWT(user.Email, user.Name, user.EffectiveRole(), user.TenantID)` — 4 参形式，`role` + `tenant_id` 已正确签入 JWT。
   - `auth.go` `JWTClaim` 包含 `Role string` + `TenantID string`；`ValidateTokenFull()` 正确解出并填入 `TokenInfo`。
   - **S6 对齐说明**：socketio / WS-16 从 JWT 取 `tenant_id` 用 claim key `"tenant_id"`，取 `role` 用 claim key `"role"`，与本中间件 context key 一致。

4. **Handler 级跨租户用例（Issue 4）**
   - 新增 `internal/api/utils.go: checkUserTenantOwnership(callerRole, callerTenantID, targetUserTenantID string) bool` — 纯函数，无 DB/NATS 依赖，`role.go` 与 `user.go` 改用此函数，逻辑语义不变。
   - 新增 `internal/api/isolation_test.go`，覆盖：
     - `TestCheckDeviceTenantAccess_SuperAdminBypass` — nil NATS 不 panic，super_admin 返回 true
     - `TestCheckUserTenantOwnership_SuperAdmin_CanActCrossTenant` — super_admin 跨租户 OK
     - `TestCheckUserTenantOwnership_TenantAdmin_SameTenant_Allowed` — 同租户 OK
     - `TestCheckUserTenantOwnership_TenantAdmin_CrossTenant_Denied` — 跨租户 403 核心路径
     - `TestCheckUserTenantOwnership_Operator_CrossTenant_Denied` — 普通用户跨租户 403
     - `TestRequireDeviceAccess_SuperAdminBypassViaContext` — 通过 context 注入 super_admin，nil NATS 不出错
     - `TestRequireDeviceAccess_NonSuperAdmin_BlockedWhenNATSUnavailable` — fail-closed：NATS 不可用时非 super_admin 被拒
   - `go test ./...` 全绿（见 CI）。

### v1.1 修复说明

1. **设备租户隔离（Issue 1）**
   - `retrieveDevices` GET 列表：非 `super_admin` 在 NATS filter 中自动追加 `customer: callerTenantID`
   - `retrieveDevices` GET 单台：验证 `device.Customer == callerTenantID`，否则 403
   - `retrieveDevices` DELETE：对每个 SN 做归属校验，跨租户 403
   - 全部 USP per-SN handlers（`deviceGenericMessage` / `deviceGetMsg` / `deviceUpdateMsg` / `deviceCreateMsg` / `deviceDeleteMsg` 等）：新增 `requireDeviceAccess()` 检查
   - 新增 `checkDeviceTenantAccess()` / `Api.requireDeviceAccess()` 辅助函数

2. **assignUserRole 越权（Issue 2）**
   - `tenant_admin` 调用前先 `FindUser(req.Email)`，验证 `targetUser.TenantID == callerTenantID`；不匹配 403

3. **deleteUser 越权（Issue 3）**
   - `tenant_admin` 删除他人前先 `FindUser(target)`，验证租户归属；跨租户 403

4. **RequirePermission 中间件挂载（Issue 4）**
   - `iot` 路由组：叠加 `RequirePermission("devices:read")` + `DeviceWritePermission`（PUT/POST/DELETE 要求 `devices:write`）
   - `dash` 路由组：叠加 `RequirePermission("devices:read")`
   - `users` 路由组：叠加 `RequirePermission("users:read")`
   - 新增 `DeviceWritePermission` 中间件（GET/HEAD 放行，其余方法要求 `devices:write`）

5. **前端菜单可见性（Issue 5）**
   - `side-nav.js` 引入 `ROUTE_ROLE_REQUIREMENTS` 映射，在 items 渲染前按 `auth.user.role` 过滤
   - `Users`/`Roles`/`Tenants` 菜单仅对 `tenant_admin` 以上角色可见

---

## 相关文件

### 后端

- `internal/db/tenant.go` — 租户模型与 CRUD
- `internal/db/role.go` — 角色/权限常量与 CRUD
- `internal/db/user.go` — 新增 `TenantID`, `Role`, `EffectiveRole()`, `FindUsersByTenant()`, `UpdateUserRole()`
- `internal/db/db.go` — 新增 tenants/roles 集合 + 数据 seed
- `internal/api/auth/auth.go` — JWT Claims 新增 `role`/`tenant_id`，新增 `ValidateTokenFull()`, `CheckPermission()`
- `internal/api/middleware/middleware.go` — context 注入 `role`/`tenant_id`，新增 `RequirePermission()`
- `internal/api/tenant.go` — Tenant REST API
- `internal/api/role.go` — Role REST API + assign
- `internal/api/user.go` — 用户接口 RBAC 适配
- `internal/api/api.go` — 路由注册

### 前端

- `src/contexts/auth-context.js` — 登录/初始化解析 JWT role/tenantId
- `src/pages/access-control/roles.js` — 角色管理页
- `src/pages/access-control/tenants.js` — 租户管理页
- `src/pages/access-control/users.js` — 用户管理页（角色分配）
- `src/sections/customer/customers-table.js` — 新增 Role/Tenant 列
- `src/layouts/dashboard/config.js` — 导航菜单新增 Roles/Tenants

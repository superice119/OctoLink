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

# WS-9 — S5 实时 USP 参数检索 (Get/Set)

对应 issue: WS-9 (`4873a4fc-9242-4930-a139-25ea6831140b`)

---

## 1. 背景与目标

OctoLink 作为 TR-369 (USP) 控制器，已具备完整的 USP 协议代理层（Get/Set/Add/Delete/Operate/GetInstances/GetSupportedDM），但缺少：

1. **参数查询缓存**：每次 GET 都穿透到设备，高频操作对设备端造成无谓压力，且对移动/低带宽链路不友好。
2. **前端直接查询面板**：现有 `Parameters` 标签页以树形浏览为主，缺少直接输入 TR-181 路径并快速获取/下发值的交互界面。

本任务目标：
- 新增 `PUT /api/device/{sn}/{mtp}/get/cached` 端点，基于 NATS JetStream KeyValue 缓存 GET 结果（5 min TTL）；SET 成功后自动按设备维度清空缓存。
- 新增前端 **Query** 标签页（`DevicesParamQuery` 组件），支持批量输入 TR-181 路径 → 一键查询（缓存或实时）→ 结果表格展示 → 内联 SET 下发。

---

## 2. 设计思路与方案选型

### 2.1 缓存策略

| 方案 | 选择理由 |
|------|----------|
| 透明缓存（修改现有 `/get`） | ❌ SET 后可能返回旧值，破坏 DevicesDiscovery 的编辑回显语义 |
| 独立端点 `/get/cached` | ✅ 不影响现有调用路径；使用方按需选择缓存/实时 |
| Redis / 外部缓存 | ❌ 引入额外依赖；NATS KV 已在架构内 |

**KV 设计**：
- Bucket：`usp-param-cache`，bucket 级 TTL = 5 min
- Key 格式：`usp.<sanitized_sn>.<sha256_hex(sorted_paths)>`
  - SN 中的非法字符（如 `:` in `os::OUI-SN`）替换为 `-`
  - 路径集合排序后 SHA-256，保证同一组路径幂等命中
- 写入条件：HTTP 200 **且** body 包含 `"req_path_results"`（过滤 USP Error 伪 200）
- 清除策略：SET 成功后异步遍历 KV ListKeys，前缀匹配 `usp.<sanitized_sn>.` 全部 Delete

### 2.2 `paramKv` 初始化位置

在 `api.NewApi()` 内部调用 `js.CreateOrUpdateKeyValue()`，不改动 `nats.StartNatsClient()` 签名也不改动 `main.go`。初始化失败仅 warn log，`paramKv` 置 nil，所有 cache 路径 nil-guard 降级为直通 live GET。

### 2.3 SET 缓存失效

`deviceUpdateMsg` 改为先将 USP 响应写入 `httptest.ResponseRecorder`，成功后 goroutine 异步清除该设备的全部缓存 key，再将结果转发至真实 `ResponseWriter`。

---

## 3. 接口 / 协议说明

### 3.1 新增后端接口

#### `PUT /api/device/{sn}/{mtp}/get/cached`

缓存版参数查询，逻辑：缓存命中 → 返回缓存值（含 `X-Cache: HIT`）；缓存未命中 → 透传 USP GET → 缓存结果 → 返回。

**Request Body**（同 `/get`，protojson 格式）：

```json
{
  "param_paths": ["Device.DeviceInfo.", "Device.LANConfigSecurity."],
  "max_depth": 0
}
```

**Response（成功，HTTP 200）**：

```json
{
  "req_path_results": [
    {
      "requested_path": "Device.DeviceInfo.",
      "resolved_path_results": [
        {
          "resolved_path": "Device.DeviceInfo.",
          "result_params": {
            "Manufacturer": "iopsys",
            "SoftwareVersion": "3.2.1"
          }
        }
      ]
    }
  ]
}
```

**Response Headers（缓存命中时额外）**：
```
X-Cache: HIT
```

**Response（失败）**：同 `/get` 错误格式，`err_code` + `err_msg`。

#### `PUT /api/device/{sn}/{mtp}/set`（行为变更）

原有接口不变，新增副作用：SET 成功后异步清除 `usp-param-cache` 中该设备的全部 key。

### 3.2 TR-181 参数路径约定

参考 WS-6 协议联调清单（`docs/tasks/WS-6-usp-protocol-alignment-checklist.md`），常用路径：

| 路径 | 说明 |
|------|------|
| `Device.DeviceInfo.` | 设备基本信息（Manufacturer、SoftwareVersion 等） |
| `Device.LocalAgent.` | USP Agent 信息（EndpointID、Controller 列表） |
| `Device.IP.Interface.*.` | IP 接口列表 |
| `Device.WiFi.` | Wi-Fi 配置 |
| `Device.DHCP.` | DHCP 配置 |

### 3.3 前端 API 调用

| 操作 | 方法 | 路径 |
|------|------|------|
| 缓存查询 | PUT | `/api/device/{sn}/any/get/cached` |
| 实时刷新 | PUT | `/api/device/{sn}/any/get` |
| SET 参数 | PUT | `/api/device/{sn}/any/set` |

---

## 4. 部署 / 使用步骤

### 4.1 环境依赖

- NATS Server with JetStream enabled（现有 `docker-compose` 已启用）
- Go 1.21+（标准库 `net/http/httptest`、`crypto/sha256`）
- Node.js 18+ / Next.js（现有 `frontend/`）

### 4.2 后端编译

```bash
cd backend/services/controller
go build ./...
```

首次启动时 `usp-param-cache` KV bucket 自动创建（TTL 5 min）。若 NATS 无 JetStream 权限，控制器以 warn log 降级，缓存端点仍可用（实时直通）。

### 4.3 前端

无新依赖，`DevicesParamQuery` 使用已有 MUI + heroicons。

```bash
cd frontend && npm run dev
```

访问任意 USP 设备详情页，点击 **Query** 标签即可使用。

### 4.4 agent-sim 联调

启动模拟器（TR-369 over MQTT）：

```bash
# 参考 agent-sim README，配置 endpoint ID 与控制器地址
docker compose -f docker-compose-dev.yml up agent-sim
```

在 Query 面板输入 `Device.DeviceInfo.` → 点击 **Query (Cached)** → 首次为 live GET（X-Cache 无）；5 min 内再次查询相同路径 → `X-Cache: HIT`。

### 4.5 回滚

- 后端：回滚到上一个 commit，`usp-param-cache` bucket 会在无读写后自然过期；可手动 `nats kv del usp-param-cache` 清除。
- 前端：删除 `devices-params-query.js`，恢复 `[...id].js` 的 import 和 tab。

---

## 5. 测试与验收记录

### 5.1 用例清单

| TC | 操作 | 预期结果 |
|----|------|----------|
| TC-01 | PUT `/get/cached`，首次查询 `Device.DeviceInfo.` | HTTP 200，body 含 `req_path_results`，无 `X-Cache` header |
| TC-02 | 5 min 内重复同路径 cached GET | HTTP 200，`X-Cache: HIT`，body 与首次相同 |
| TC-03 | PUT `/set`（修改某参数），再 cached GET 同路径 | 缓存被清除，返回新值（X-Cache 无） |
| TC-04 | 设备离线时 cached GET | 缓存命中时直接返回；未命中时 503（设备离线） |
| TC-05 | 前端 Query 面板：输入路径 → Query → 结果表格展示 | 渲染行数 = 参数个数，path/value 列正确 |
| TC-06 | Query 面板：点击编辑图标 → 修改值 → Apply | SET 成功后自动 Refresh Live，表格更新 |
| TC-07 | 批量路径（多行）→ cached GET | 所有路径合并为一个 KV key，结果合并展示 |
| TC-08 | NATS KV bucket 不可用（降级场景） | 端点以 live GET 直通，无崩溃，log 打印 warn |

> 注：TC-01 ~ TC-04 待 agent-sim / 真实设备接入后完成验收；TC-05 ~ TC-07 可在本地 Mock 后端完成。

### 5.2 构建验证

```
go build ./...  # ✅ 0 errors, 0 warnings（见本 PR CI 输出）
```

---

## 6. 变更记录 (Changelog)

| 日期 | 改动 |
|------|------|
| 2026-06-18 | 初版实现：后端 `usp-param-cache` KV bucket + `/get/cached` 端点 + SET 后缓存清除；前端 `DevicesParamQuery` 组件 + Query 标签页；本文档 |
| 2026-06-18 | fix(s5): 修复 CORS `ExposedHeaders` 缺失问题 — 在 `backend/services/controller/internal/api/cors/cors.go` 增加 `ExposedHeaders: []string{"X-Cache"}`，使跨域 fetch 下浏览器可读取 `X-Cache` 响应头，缓存命中提示正常显示 |

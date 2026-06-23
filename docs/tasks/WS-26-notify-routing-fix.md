# WS-26 — Notify routing fix: MTP controller messages misclassified as `.info`

## 1. 背景与目标

**Issue**: WS-26 (基于 WS-25 fix 之上)

WS-25 修掉了 adapter 在真机 Notify 时的 nil-panic（PR #16，commit `5af4973`，已上线 `39.97.250.156`）。
但 06-23 QA 在 WS-20 复跑 **S6 真机 Notify 房间定向仍 FAIL**：adapter 不再崩，但
`octolink.notifications=0`，三个 socket.io 租户客户端均为 0。

**目标**：修复 MTP bridge 将所有 controller 频道消息（含 USP Notify）硬编码发布到 `.info` NATS
subject 导致 `HandleNotify` 永远不被调用的路由缺陷，使 S6 真机 Notify 房间定向 PASS。

## 2. 根因分析

### 根因链

1. **`backend/services/mtp/mqtt-adapter/internal/bridge/bridge.go` → `mqttMessageHandler`**：
   MQTT `controller` 频道所有消息无条件发布到 `mqtt.usp.v1.<device>.info`（硬编码 `.info` 后缀）：
   ```go
   case c := <-controller:
       b.Pub(NATS_MQTT_SUBJECT_PREFIX+getDeviceFromTopic(c.Topic)+".info", c.Payload)
   ```
   bridge 把 payload 当作不透明字节，无法区分 USP Notify / GetResp。

2. **`backend/services/mtp/adapter/internal/events/events.go` → `StartEventsListener`**：
   adapter 仅按 NATS subject 末段路由（`.info`→`HandleDeviceInfo`，`.notify`→`HandleNotify`）。

3. **结果**：真机 Notify 落 `.info` → `HandleDeviceInfo`（只处理 `Body_Response`/GetResp，见到
   `Body_Request` 直接 `return`）→ 丢弃。全工作区无任何 adapter 产生过 `.notify` subject（grep
   已确认）→ `HandleNotify` 在 MQTT 路径是死代码。

### 为什么不改 bridge

修改 bridge 需改 mqtt/ws/stomp 三个服务 + 三个二进制构建；而在 adapter 侧按 USP 消息内容分流，
一次修改覆盖所有 MTP 的 controller→.info 汇流，且不需要新构建 bridge。

## 3. 修复方案

### 修改位置

**单文件**：`backend/services/mtp/adapter/internal/events/usp_handler/info.go`

### 核心变更

在 `HandleDeviceInfo` 入口处，**先于 `defer ack()`** 检测 USP 消息类型。若确认为 Notify 请求，
则转交 `HandleNotify`（其内部有自己的 `defer ack()`）并立即 return，避免 double-ack：

```go
func (h *Handler) HandleDeviceInfo(device, subject string, data []byte, mtp string, ack func()) {
    if isUspNotify(data) {
        h.HandleNotify(device, subject, data, mtp, ack)
        return
    }
    defer ack()
    // ... 原 info 逻辑不变
}
```

新增辅助函数 `isUspNotify(data []byte) bool`：解析 Record → Msg，检测 `Body_Request.Notify != nil`。
**严格使用 WS-25 的安全 record-context switch**（`GetNoSessionContext()/GetSessionContext()` 判空），
不裸调 `.Payload`，杜绝在 session-context 记录上 nil-panic：

```go
func isUspNotify(data []byte) bool {
    var record usp_record.Record
    if err := proto.Unmarshal(data, &record); err != nil {
        return false
    }
    var msgPayload []byte
    switch {
    case record.GetNoSessionContext() != nil:
        msgPayload = record.GetNoSessionContext().GetPayload()
    case record.GetSessionContext() != nil:
        for _, chunk := range record.GetSessionContext().GetPayload() {
            msgPayload = append(msgPayload, chunk...)
        }
    default:
        return false
    }
    var message usp_msg.Msg
    if err := proto.Unmarshal(msgPayload, &message); err != nil {
        return false
    }
    if message.Body == nil {
        return false
    }
    req, isRequest := message.Body.MsgBody.(*usp_msg.Body_Request)
    if !isRequest {
        return false
    }
    return req.Request.GetNotify() != nil
}
```

### 不受影响的路径

- GetResp（`Body_Response`）：`isUspNotify` 仅匹配 `Request.Notify`，GetResp 继续由 `HandleDeviceInfo` 原逻辑处理。
- ws/stomp bridge：结构与 mqtt-bridge 相同（controller→.info），adapter 侧分类对三者通用，理论上一并覆盖；ws/stomp 标注为「同形待确认」，非本期 MVP 验收必经项。
- 前置条件：WS-24 设备分配已 PASS，设备已在库且 Customer 已赋值，满足 `HandleNotify` 的租户路由前提。

## 4. 部署 SOP（deploy-ready，上线须经 Archie 复核 + 老板审批）

```bash
# 1. 在 OctoLink 仓库根目录交叉编译 adapter 单二进制
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -o adapter-linux-amd64 \
  ./backend/services/mtp/adapter/cmd/adapter/

# 2. scp 到控制器
scp adapter-linux-amd64 root@39.97.250.156:/opt/octolink/adapter/

# 3. 确认 docker-compose.override.yaml 已 bind-mount
# bind-mount: /opt/octolink/adapter/adapter-linux-amd64:/app/adapter

# 4. 只重启 adapter（禁 server build，不动 socketio override）
docker compose -f docker-compose.yml -f docker-compose.override.yaml \
  up -d --no-build adapter

# 5. 验证
docker compose ps adapter       # 期望: Up
docker compose logs -f adapter  # 观察无 panic；应出现 "Device ... sent USP Notify" + "HandleNotify: published notification"
```

## 5. 验收标准（由 QA_Sherlock 在 WS-20 复跑）

- 真机触发 `Device.Boot!` → adapter 日志出现：
  - `Device <sn> sent USP Notify`
  - `HandleNotify: published notification to notification.v1.<sn>`
- 房间定向不串台：
  - `tenant-a` 客户端只收 200/201 设备的通知
  - `tenant-b` 客户端只收 202 设备的通知
  - `super_admin`/`admin` 可见全部
- adapter 全程 Up，不崩。

## 6. 静态检查结果

```
go build ./...     → 通过（无错误）
go vet ./internal/events/...  → 通过（无警告）
```

注：`go vet ./...` 全模块扫描中，`internal/db`、`internal/usp`、`internal/reqs` 包存在
pre-existing `primitive.E unkeyed fields` / `lock by value` 警告，均为既存问题，非本次引入。

## 7. 变更记录 (Changelog)

| 日期 | 改动 | 作者 |
|---|---|---|
| 2026-06-23 | fix(adapter): peek USP msg type in HandleDeviceInfo and redirect Notify to HandleNotify (WS-26) | Cloud_Guru |

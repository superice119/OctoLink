# WS-25 — Adapter nil-panic fix: `usp_handler/info.go` GetNoSessionContext guard

## 1. 背景与目标

**Issue**: WS-25 (阻塞 WS-20 收口)

QA_Sherlock 在 WS-20 复跑 S6 真机 Notify 房间定向时，adapter 进程在处理真机（obuspa）发来的 USP Record 时崩溃退出，`docker compose ps` 显示 adapter 容器不再 Up。三个 socket.io 客户端（tenant-a/tenant-b/super_admin）的 `usp_notify` 计数均为 0。

**目标**：修复 adapter 在真机 topic / 重连 / 设备信息解析路径上的 nil pointer dereference panic，使 adapter 全程不崩，完成 S6 真机 Notify 闭环。

## 2. 设计思路与方案选型

### 根因分析

崩溃位于 `backend/services/mtp/adapter/internal/events/usp_handler/info.go:51`（修复前）：

```go
err = proto.Unmarshal(record.GetNoSessionContext().Payload, &message)
```

`record.GetNoSessionContext()` 在 USP Record 使用 **session context**（而非 no-session-context）时返回 `nil`，对 `nil` 调用 `.Payload` 即为 **nil pointer dereference panic**。

- **agent-sim / 合成消息**：发 no-session-context 记录，不触发。
- **真机 obuspa**：Notify、重连路径发 session-context 记录，必崩。

这是上游 OktopUSP/oktopus 的潜伏 bug（自 commit `94f3186` 导入以来未改动），被真机首次暴露，与 WS-24 无关。

### 修复方案

参照同目录 `notify.go:42-49` 已有的正确写法，在 `parseDeviceInfoMsg` 解析 payload 前增加 record type 判断：

```go
var msgPayload []byte
switch {
case record.GetNoSessionContext() != nil:
    msgPayload = record.GetNoSessionContext().GetPayload()
case record.GetSessionContext() != nil:
    // SessionContextRecord.Payload 是 [][]byte（分段）；拼接所有分片
    for _, chunk := range record.GetSessionContext().GetPayload() {
        msgPayload = append(msgPayload, chunk...)
    }
default:
    log.Printf("parseDeviceInfoMsg: unsupported record type for device %s, subject %s", sn, subject)
    return db.Device{}
}
```

同时补充后续 `message.Body == nil` 的防护，避免真机发来格式异常消息时换位崩溃。

**不改 adapter 业务逻辑**；topic 不匹配属 provisioning 配置问题（见 §3）。

## 3. 接口 / 协议说明

### USP Record type 对应关系

| Record 类型 | 触发场景 | Payload 字段 |
|---|---|---|
| `NoSessionContextRecord` | agent-sim、简单场景 | `[]byte`（单字段） |
| `SessionContextRecord` | 真机 obuspa Notify/重连 | `[][]byte`（分段，需拼接） |

### topic 不匹配说明（Step 3 评估）

- 真机 obuspa 默认 controller topic：`oktopus/v1/controller`
- adapter 消费的 topic：`oktopus/usp/v1/controller/<sn>`

这是**设备 provisioning / 配置**问题，需在 obuspa 配置中将 controller endpoint topic 改为 `oktopus/usp/v1/controller/<sn>` 格式（或通过 MQTT broker provisioning 对接）。**不属于 adapter 订阅缺陷**，本次修复不改 adapter 代码，由 Protocol_Pro / 真机侧跟进配置对齐。

## 4. 部署 / 使用步骤

### 上线 SOP（免构建，沿用 WS-24 模式）

**严禁未审批直接上线**；本次交付为 PR + deploy-ready，实际上线须经老板审批。

```bash
# 1. 交叉编译 adapter 单二进制（2C/2G 控制器上禁止 docker build）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -o adapter-linux-amd64 \
  ./backend/services/mtp/adapter/cmd/adapter/

# 2. scp 到控制器
scp adapter-linux-amd64 root@39.97.250.156:/opt/octolink/adapter/

# 3. 确认 docker-compose.override.yaml 已绑挂载（扩展现有 socketio 条目，不 clobber）
# bind-mount: /opt/octolink/adapter/adapter-linux-amd64:/app/adapter

# 4. 只重启 adapter（禁 server build）
docker compose -f docker-compose.yml -f docker-compose.override.yaml \
  up -d --no-build adapter
```

### 验证适配器已启动

```bash
docker compose ps adapter          # 期望: Up
docker compose logs -f adapter     # 观察无 panic 日志
```

## 5. 测试与验收记录

### 单测验证点

- adapter 接收 session-context USP Record 时 `parseDeviceInfoMsg` 不 panic，返回 `db.Device{}`（记日志），不崩进程。
- adapter 接收 no-session-context USP Record 时行为与修复前一致。
- `message.Body == nil` 时安全返回，不 panic。

### S6 真机 Notify 闭环验收（待 QA_Sherlock 执行）

环境：
- 控制器：`39.97.250.156`
- agent-sim 主机：`39.105.150.244`
- 真机：`ws20qa-200`（tenant-a）、`ws20qa-201`（tenant-a）、`ws20qa-202`（tenant-b）

验收步骤：
1. 部署修复后的 adapter 二进制，确认 `docker compose ps adapter` = Up。
2. 真机触发 `Device.Boot!` Notify（或等待设备重连）。
3. 观察 adapter 日志：无 `panic`，可见 `Device <sn> info` 和 `HandleNotify: published notification` 日志。
4. socket.io 客户端 tenant-a 收到 `usp_notify`，tenant-b 不收到 tenant-a 设备消息（不串线）。

**MVP 证据**：adapter 容器日志（无 panic）+ socket.io 事件计数截图（各租户只收到自己设备的 Notify）。

### 当前状态

- [x] 代码修复完成（`info.go` nil guard）
- [x] `go vet` 通过
- [ ] 部署到 `39.97.250.156` 待老板审批
- [ ] QA_Sherlock 复跑 WS-20 S6 真机 Notify 验收

## 6. 变更记录 (Changelog)

| 日期 | 改动 | 作者 |
|---|---|---|
| 2026-06-23 | fix(adapter): guard nil no-session-context in parseDeviceInfoMsg；补 message.Body nil 检查 | Cloud_Guru |

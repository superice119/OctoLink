# WS-6 (S2) — USP/TR-369 协议联调接口清单

**日期:** 2026-06-17（v1）/ 2026-06-18（v2 prpl 专项更新）
**作者:** Protocol_Pro
**依赖:** WS-6 obuspa 双方案评估报告(`docs/tasks/WS-6-obuspa-evaluation.md`) / TR-369 规范(`docs/references/tr-x69-specifications.md`)
**对照基准:** BBF TR-369 (USP) + TR-181 Issue-2 Amendment-15

> **v2 选型更新（2026-06-18）**：MVP 终端栈已定 **prpl Ambiorix**（iopsys 保留为商用备选）；统一 Controller 地址更新为 **39.97.250.156**。本文档在 v1 stack-agnostic 基础上补充 §9 prpl 专项 onboarding 指南。

---

## 1. 背景与目标

本文档是 WS-6(S2)的核心交付物：**stack-agnostic 协议联调接口清单**，基于 TR-181 `Device.` 数据模型，涵盖终端侧 obuspa Agent 与云端 OctoLink Controller(Oktopus fork)之间所有消息类型的格式规范、参数路径、数据类型，以及 S1 API 的字段对齐映射。

目标：
- 为 Cloud_Guru(S5 Get/Set、S6 Notify、S4 信息检索)提供端到端字段对齐基准。
- 为 QA_Sherlock 提供 Wireshark 抓包规则与边界值测试用例依据。
- 固化已验证的联调要点，避免换机或重部署时重踩已知坑。

---

## 2. 联调环境基线

| 项 | 值 | 来源 |
|---|---|---|
| **Controller 地址（生产）** | `ws://39.97.250.156:8080/ws/agent` | 选型决策 2026-06-17（统一 Controller） |
| Controller 地址（旧测试，已弃用） | ~~`ws://39.105.150.244:8080/ws/agent`~~ | 已废止，勿用 |
| Controller EndpointID | `oktopusController` | 验证报告 §3.5 坑#1 |
| Agent EndpointID 格式 | `os::<ManufacturerOUI>-<SerialNumber>` | TR-369 §2.3.2 |
| Agent EndpointID 示例(prpl) | `os::00256D-OpenWrtGateway001` | 验证报告 §2.6 |
| Agent EndpointID 示例(iopsys) | `os::00256D-iopsysGateway001` | 验证报告 §3.5 坑#2 |
| MTP | USP over WebSocket(RFC 6455) | TR-369 §8.1 |
| Controller 授权角色 | `assigned_role_name='full_access'` → `AssignedRole=ControllerTrust.Role.1` | 验证报告 §3.5 坑#4 |
| **MVP 终端栈** | **prpl Ambiorix**（obuspa v11.0.2） | 选型决策 2026-06-17 |
| 数据模型版本(prpl) | `RootDataModelVersion=2.17` | 验证报告 §2.5 |
| 数据模型版本(iopsys，备选） | `RootDataModelVersion=2.20` | 验证报告 §3.6 |
| 消息编码 | Protocol Buffers(proto3),封装于 USP Record | TR-369 §5 |
| USP Record 外层类型 | `NoSessionContext` 或 `SessionContext` | TR-369 §5.1 |

> **坑警告**：`ws://` 无 TLS 证书时，Controller 信任等级落到 `Untrusted`(Role.2)，无读权限，Parameters discovery 一直转圈。**必须**在 Oktopus 控制台显式设置 `assigned_role_name='full_access'`。

---

## 3. USP 消息类型与格式规范

### 3.1 消息通用结构

每条 USP 消息由两层构成（TR-369 §5）：

```
USP Record (外层)
  ├── version        = "1.3"
  ├── to_id          = <目标 EndpointID>
  ├── from_id        = <源 EndpointID>
  ├── payload_security = PLAINTEXT (WS 无 TLS 时) | TLS (推荐)
  └── record_type:
        NoSessionContext.payload = <序列化 USP Msg>

USP Msg (内层)
  ├── Header
  │     ├── msg_id   = <UUID 或递增 ID，每条唯一>
  │     └── msg_type = GET | GET_RESP | SET | SET_RESP | ... (见下表)
  └── Body
        └── <对应消息体>
```

| msg_type 枚举值 | 方向 | 说明 |
|---|---|---|
| `GET` | Controller → Agent | 读取一或多个参数/对象 |
| `GET_RESP` | Agent → Controller | GET 的响应 |
| `SET` | Controller → Agent | 写入参数值 |
| `SET_RESP` | Agent → Controller | SET 的响应 |
| `ADD` | Controller → Agent | 创建多实例对象新实例 |
| `ADD_RESP` | Agent → Controller | ADD 的响应 |
| `DELETE` | Controller → Agent | 删除多实例对象实例 |
| `DELETE_RESP` | Agent → Controller | DELETE 的响应 |
| `OPERATE` | Controller → Agent | 触发 USP 命令(RPC) |
| `OPERATE_RESP` | Agent → Controller | OPERATE 同步响应 |
| `GET_INSTANCES` | Controller → Agent | 获取多实例对象的实例列表 |
| `GET_INSTANCES_RESP` | Agent → Controller | GET_INSTANCES 的响应 |
| `GET_SUPPORTED_DM` | Controller → Agent | 查询 Agent 支持的数据模型 |
| `GET_SUPPORTED_DM_RESP` | Agent → Controller | GET_SUPPORTED_DM 的响应 |
| `NOTIFY` | Agent → Controller | 主动上报事件(基于 Subscription) |
| `NOTIFY_RESP` | Controller → Agent | NOTIFY 的确认(若 send_resp=true) |
| `SUBSCRIBE` | Controller → Agent | 订阅事件 |
| `SUBSCRIBE_RESP` | Agent → Controller | SUBSCRIBE 的响应 |
| `UNSUBSCRIBE` | Controller → Agent | 取消订阅 |
| `UNSUBSCRIBE_RESP` | Agent → Controller | UNSUBSCRIBE 的响应 |
| `ERROR` | 双向 | 协议层错误 |

---

### 3.2 GET — 参数读取

**Controller → Agent**

```protobuf
// TR-369 §9.1
Msg {
  Header { msg_id: "get-001", msg_type: GET }
  Body {
    request {
      get {
        param_paths: [
          "Device.DeviceInfo.",           // 读取整个子树（带点后缀）
          "Device.LocalAgent.EndpointID", // 读取单个参数
          "Device.IP.Interface.*.IPv4Address.*.IPAddress"  // 通配符
        ]
        max_depth: 0  // 0 = 完整深度；>0 限制层数
      }
    }
  }
}
```

**Agent → Controller（成功）**

```protobuf
Msg {
  Header { msg_id: "get-001", msg_type: GET_RESP }
  Body {
    response {
      get_resp {
        req_path_results: [
          {
            requested_path: "Device.DeviceInfo."
            resolved_path_results: [
              {
                resolved_path: "Device.DeviceInfo."
                result_params: {
                  "Manufacturer":      "iopsys"
                  "ManufacturerOUI":   "00256D"
                  "ModelName":         "RT-MT7621"
                  "SerialNumber":      "000001"
                  "SoftwareVersion":   "ImmortalWrt-24.10"
                  "HardwareVersion":   "v8.0"
                  "UpTime":            "12345"
                  "ProvisioningCode":  ""
                }
              }
            ]
          }
        ]
      }
    }
  }
}
```

**Agent → Controller（错误）**

```protobuf
// 路径不存在时，result_code = 9003 (Request Denied – Invalid Path)
req_path_results: [{
  requested_path: "Device.DeviceInfo.NonExistent"
  err_code: 9003
  err_msg: "Invalid path"
}]
```

---

### 3.3 SET — 参数写入

**Controller → Agent**

```protobuf
// TR-369 §9.2
Msg {
  Header { msg_id: "set-001", msg_type: SET }
  Body {
    request {
      set {
        allow_partial: false  // false=全部成功或全部回滚；true=部分成功
        update_objs: [
          {
            obj_path: "Device.LocalAgent.Controller.1."
            param_settings: [
              { param: "Alias",        value: "oktopus",           required: true }
              { param: "EndpointID",   value: "oktopusController", required: true }
              { param: "Enable",       value: "true",              required: true }
            ]
          }
        ]
      }
    }
  }
}
```

**Agent → Controller（成功）**

```protobuf
Msg {
  Header { msg_id: "set-001", msg_type: SET_RESP }
  Body {
    response {
      set_resp {
        updated_obj_results: [
          {
            requested_path: "Device.LocalAgent.Controller.1."
            oper_status { oper_success { updated_inst_results: [] } }
          }
        ]
      }
    }
  }
}
```

---

### 3.4 ADD — 创建多实例对象

**Controller → Agent**（示例：新增 Subscription）

```protobuf
// TR-369 §9.3
Msg {
  Header { msg_id: "add-001", msg_type: ADD }
  Body {
    request {
      add {
        allow_partial: false
        create_objs: [
          {
            obj_path: "Device.LocalAgent.Subscription."  // 尾部带点
            param_settings: [
              { param: "Enable",           value: "true" }
              { param: "NotifType",        value: "ValueChange" }
              { param: "ReferenceList",    value: "Device.DeviceInfo.SoftwareVersion" }
              { param: "ID",               value: "sub-swver-001" }
              { param: "Persistent",       value: "true" }
            ]
          }
        ]
      }
    }
  }
}
```

**Agent → Controller（成功）**

```protobuf
Msg {
  Header { msg_id: "add-001", msg_type: ADD_RESP }
  Body {
    response {
      add_resp {
        created_obj_results: [
          {
            requested_path: "Device.LocalAgent.Subscription."
            oper_status {
              oper_success {
                instantiated_path: "Device.LocalAgent.Subscription.3."  // Agent 分配的实例号
                unique_keys: { "ID": "sub-swver-001" }
              }
            }
          }
        ]
      }
    }
  }
}
```

---

### 3.5 DELETE — 删除实例

**Controller → Agent**

```protobuf
// TR-369 §9.4
Msg {
  Header { msg_id: "del-001", msg_type: DELETE }
  Body {
    request {
      delete {
        allow_partial: false
        obj_paths: ["Device.LocalAgent.Subscription.3."]
      }
    }
  }
}
```

**Agent → Controller（成功）**

```protobuf
Msg {
  Header { msg_id: "del-001", msg_type: DELETE_RESP }
  Body {
    response {
      delete_resp {
        deleted_obj_results: [
          {
            requested_path: "Device.LocalAgent.Subscription.3."
            oper_status { oper_success { affected_paths: ["Device.LocalAgent.Subscription.3."] } }
          }
        ]
      }
    }
  }
}
```

---

### 3.6 OPERATE — RPC 命令

**Controller → Agent**（示例：Reboot）

```protobuf
// TR-369 §9.5
Msg {
  Header { msg_id: "op-001", msg_type: OPERATE }
  Body {
    request {
      operate {
        command:    "Device.Reboot()"  // TR-181 §7.1.1
        command_key: "reboot-001"      // 用于 OperationComplete 事件追踪
        send_resp:  true               // 需要同步 OperateResp
        input_args: {}                 // Reboot 无入参
      }
    }
  }
}
```

**Agent → Controller（已接受执行，异步）**

```protobuf
Msg {
  Header { msg_id: "op-001", msg_type: OPERATE_RESP }
  Body {
    response {
      operate_resp {
        operation_results: [
          {
            executed_command: "Device.Reboot()"
            oper_status { req_obj_path_err { err_code: 0 } }  // 0 = 接受
            // 重启结果通过 OperationComplete 事件异步上报（如设备重启后需重连）
          }
        ]
      }
    }
  }
}
```

**常用 Operate 命令清单**

| 命令路径 | 入参 | 异步 | 说明 | TR-181 章节 |
|---|---|---|---|---|
| `Device.Reboot()` | — | 是 | 重启设备 | §7.1.1 |
| `Device.FactoryReset()` | — | 是 | 恢复出厂 | §7.1.1 |
| `Device.IP.Diagnostics.IPPing()` | Host, Count, Timeout, DataBlockSize | 是 | Ping 诊断 | §15.2.1 |
| `Device.IP.Diagnostics.TraceRoute()` | Host, Timeout, MaxHopCount | 是 | 路由追踪 | §15.2.3 |
| `Device.IP.Diagnostics.DownloadDiagnostics()` | DownloadURL, TestFileLength | 是 | 下载诊断 | §15.2.5 |
| `Device.IP.Diagnostics.UploadDiagnostics()` | UploadURL | 是 | 上传诊断 | §15.2.6 |
| `Device.SoftwareModules.DeploymentUnit.*.Update()` | URL, UUID | 是 | 固件更新 | §14 |

---

### 3.7 GET_INSTANCES — 获取实例列表

**Controller → Agent**

```protobuf
// TR-369 §9.6
Msg {
  Header { msg_id: "gi-001", msg_type: GET_INSTANCES }
  Body {
    request {
      get_instances {
        obj_paths:     ["Device.LocalAgent.Subscription.", "Device.IP.Interface."]
        first_level_only: false  // false=递归返回所有层级实例
      }
    }
  }
}
```

**Agent → Controller**

```protobuf
Msg {
  Header { msg_id: "gi-001", msg_type: GET_INSTANCES_RESP }
  Body {
    response {
      get_instances_resp {
        req_path_results: [
          {
            requested_path: "Device.LocalAgent.Subscription."
            curr_insts: [
              { instantiated_obj_path: "Device.LocalAgent.Subscription.1." unique_keys: { "ID": "sub-oc-001" } }
              { instantiated_obj_path: "Device.LocalAgent.Subscription.2." unique_keys: { "ID": "sub-boot-001" } }
            ]
          }
        ]
      }
    }
  }
}
```

---

### 3.8 GET_SUPPORTED_DM — 查询支持的数据模型

**Controller → Agent**（首次接入时用于 discovery）

```protobuf
// TR-369 §9.7
Msg {
  Header { msg_id: "gsdm-001", msg_type: GET_SUPPORTED_DM }
  Body {
    request {
      get_supported_dm {
        obj_paths:          ["Device."]
        first_level_only:   false
        return_commands:    true
        return_events:      true
        return_params:      true
        return_unique_key_sets: true
      }
    }
  }
}
```

返回结构（`GET_SUPPORTED_DM_RESP`）中每个节点包含：
- `supported_obj_path`：对象路径
- `access`：OBJ_READ_ONLY | OBJ_ADD_DELETE | OBJ_ADD_ONLY | OBJ_DELETE_ONLY
- `is_multi_instance`：是否多实例（Table）
- `supported_params`：参数列表（含 param_type、value_change、access）
- `supported_commands`：命令列表（含 input/output 参数）
- `supported_events`：事件列表

---

### 3.9 NOTIFY / SUBSCRIBE — 事件订阅与上报

#### 3.9.1 订阅流程

1. Controller 发 ADD 创建 `Device.LocalAgent.Subscription.N.`（见 §3.4）。
2. 满足条件时，Agent 主动发送 NOTIFY 消息。
3. 若 `send_resp=true`，Controller 需回 NOTIFY_RESP。

#### 3.9.2 通知类型（NotifType）

| NotifType | 触发条件 | 典型 ReferenceList |
|---|---|---|
| `ValueChange` | 参数值发生变化 | `Device.DeviceInfo.SoftwareVersion` |
| `ObjectCreation` | 多实例对象新增实例 | `Device.LocalAgent.Controller.` |
| `ObjectDeletion` | 多实例对象实例删除 | `Device.LocalAgent.Controller.` |
| `OperationComplete` | Operate 命令执行完成 | `Device.Reboot()` / `Device.SoftwareModules.DeploymentUnit.*.Update()` |
| `OnBoardRequest` | 设备首次上线请求入网 | （系统级，无 ReferenceList） |
| `Boot` | 设备重启完成 | `Device.Boot!`（事件） |
| `Event` | 自定义事件(设备主动触发) | 设备自定义事件路径 |

#### 3.9.3 NOTIFY 消息格式

**Agent → Controller**（ValueChange 示例）

```protobuf
// TR-369 §9.9
Msg {
  Header { msg_id: "ntf-001", msg_type: NOTIFY }
  Body {
    request {
      notify {
        subscription_id: "sub-swver-001"  // 对应 Subscription.N.ID
        send_resp: false
        notification {
          value_change {
            param_path:  "Device.DeviceInfo.SoftwareVersion"
            param_value: "ImmortalWrt-24.11"
          }
        }
      }
    }
  }
}
```

**Agent → Controller**（Boot 事件示例）

```protobuf
Msg {
  Header { msg_id: "ntf-002", msg_type: NOTIFY }
  Body {
    request {
      notify {
        subscription_id: "sub-boot-001"
        send_resp: true
        notification {
          event {
            obj_path:   "Device."
            event_name: "Boot!"
            params: {
              "FirmwareUpdated":         "false"
              "ParameterMap":            ""
              "CommandKey":              ""
            }
          }
        }
      }
    }
  }
}
```

**Agent → Controller**（OperationComplete 示例）

```protobuf
Msg {
  Header { msg_id: "ntf-003", msg_type: NOTIFY }
  Body {
    request {
      notify {
        subscription_id: "sub-opcomplete-001"
        send_resp: false
        notification {
          oper_complete {
            obj_path:    "Device."
            command_name: "Reboot()"
            command_key:  "reboot-001"
            oper_status {
              oper_success { output_args: {} }
              // 或 oper_failure { err_code: 9001  err_msg: "..." }
            }
          }
        }
      }
    }
  }
}
```

---

## 4. TR-181 核心参数路径清单

> 说明："支持" 列以 ✅ 标注 obuspa 双方案均验证可读写，⚠️ 表示 iopsys 需补对应 manager，❌ 表示当前不支持。

### 4.1 DeviceInfo（设备身份）

| 参数路径 | 数据类型 | 访问 | 来源/注意 |
|---|---|---|---|
| `Device.DeviceInfo.Manufacturer` | string | R | prpl: 插件内置；iopsys: `/etc/board-db/config/device` → `manufacturer` |
| `Device.DeviceInfo.ManufacturerOUI` | string(6位十六进制) | R | 与 AgentEndpointID `os::<OUI>-<serial>` 中 OUI **必须一致** |
| `Device.DeviceInfo.ModelName` | string | R | iopsys 示例: `RT-MT7621` |
| `Device.DeviceInfo.Description` | string | R | 可选 |
| `Device.DeviceInfo.ProductClass` | string | R | iopsys: `board-db` `product_class` |
| `Device.DeviceInfo.SerialNumber` | string | R | iopsys: `board-db` `serial_number` |
| `Device.DeviceInfo.HardwareVersion` | string | R | — |
| `Device.DeviceInfo.SoftwareVersion` | string | R | iopsys 示例: `ImmortalWrt-24.10` |
| `Device.DeviceInfo.UpTime` | unsignedInt | R | 秒 |
| `Device.DeviceInfo.ProvisioningCode` | string(64) | RW | 可用于 ACS/USP 配网标识 |
| `Device.DeviceInfo.MemoryStatus.Total` | unsignedInt | R | KB；验证读到 `442368`(≈432 MB) |
| `Device.DeviceInfo.MemoryStatus.Free` | unsignedInt | R | KB |
| `Device.DeviceInfo.ProcessStatus.CPUUsage` | unsignedInt | R | % |

### 4.2 LocalAgent（USP Agent 自身管理）

| 参数路径 | 数据类型 | 访问 | 说明 |
|---|---|---|---|
| `Device.LocalAgent.EndpointID` | string | R | Agent 的 USP EndpointID，格式 `os::<OUI>-<serial>` |
| `Device.LocalAgent.SupportedProtocols` | string | R | 支持的 USP 版本，如 `"1.3"` |
| `Device.LocalAgent.RootDataModelVersion` | string | R | prpl: `2.17`；iopsys: `2.20` |
| `Device.LocalAgent.ControllerNumberOfEntries` | unsignedInt | R | 已注册的 Controller 数量 |
| `Device.LocalAgent.Controller.{i}.Alias` | string | RW | 标识符 |
| `Device.LocalAgent.Controller.{i}.Enable` | boolean | RW | 启用/禁用 Controller 连接 |
| `Device.LocalAgent.Controller.{i}.EndpointID` | string | RW | Controller EndpointID，**必须**与 Oktopus 实际端点一致 |
| `Device.LocalAgent.Controller.{i}.AssignedRole` | string | RW | 对应信任等级路径，如 `ControllerTrust.Role.1`（full_access） |
| `Device.LocalAgent.MTP.{i}.Enable` | boolean | RW | 启用 MTP |
| `Device.LocalAgent.MTP.{i}.Protocol` | string | RW | `"WebSocket"` / `"MQTT"` / `"STOMP"` |
| `Device.LocalAgent.MTP.{i}.WebSocket.URL` | string | RW | `ws://39.97.250.156:8080/ws/agent`（生产） |
| `Device.LocalAgent.MTP.{i}.WebSocket.EnableEncryption` | boolean | RW | false（ws://）/ true（wss://） |
| `Device.LocalAgent.Subscription.{i}.Enable` | boolean | RW | 启用订阅 |
| `Device.LocalAgent.Subscription.{i}.ID` | string | RW | 订阅标识符（唯一键） |
| `Device.LocalAgent.Subscription.{i}.NotifType` | string | RW | 见 §3.9.2 |
| `Device.LocalAgent.Subscription.{i}.ReferenceList` | string | RW | 被监视的参数或事件路径 |
| `Device.LocalAgent.Subscription.{i}.Persistent` | boolean | RW | 重启后保留 |
| `Device.LocalAgent.Subscription.{i}.TimeToLive` | int(-1/≥0) | RW | -1=永不过期；>0=存活秒数 |

### 4.3 IP 网络接口

| 参数路径 | 数据类型 | 访问 | 说明 |
|---|---|---|---|
| `Device.IP.InterfaceNumberOfEntries` | unsignedInt | R | IP 接口数量 |
| `Device.IP.Interface.{i}.Name` | string | R | 接口名，如 `br-lan`、`eth0` |
| `Device.IP.Interface.{i}.Status` | string | R | `Up` / `Down` |
| `Device.IP.Interface.{i}.IPv4AddressNumberOfEntries` | unsignedInt | R | |
| `Device.IP.Interface.{i}.IPv4Address.{j}.IPAddress` | string | RW | IPv4 地址 |
| `Device.IP.Interface.{i}.IPv4Address.{j}.SubnetMask` | string | RW | 子网掩码 |
| `Device.IP.Interface.{i}.Stats.BytesSent` | unsignedLong | R | 统计字节数 |
| `Device.IP.Interface.{i}.Stats.BytesReceived` | unsignedLong | R | |

### 4.4 以太网接口

| 参数路径 | 数据类型 | 访问 | 说明 |
|---|---|---|---|
| `Device.Ethernet.InterfaceNumberOfEntries` | unsignedInt | R | |
| `Device.Ethernet.Interface.{i}.Name` | string | R | 如 `eth0` |
| `Device.Ethernet.Interface.{i}.MACAddress` | string | R | 格式 `AA:BB:CC:DD:EE:FF`；**S4 批量信息检索的核心字段** |
| `Device.Ethernet.Interface.{i}.Status` | string | R | `Up` / `Down` |
| `Device.Ethernet.Interface.{i}.MaxBitRate` | int(-1/≥0) | RW | Mbps；-1=自动 |
| `Device.Ethernet.Interface.{i}.DuplexMode` | string | RW | `Half` / `Full` / `Auto` |

### 4.5 WiFi

| 参数路径 | 数据类型 | 访问 | 说明 |
|---|---|---|---|
| `Device.WiFi.RadioNumberOfEntries` | unsignedInt | R | |
| `Device.WiFi.Radio.{i}.Enable` | boolean | RW | |
| `Device.WiFi.Radio.{i}.Status` | string | R | `Up` / `Down` |
| `Device.WiFi.Radio.{i}.Channel` | unsignedInt | RW | 当前信道 |
| `Device.WiFi.Radio.{i}.OperatingFrequencyBand` | string | R | `2.4GHz` / `5GHz` |
| `Device.WiFi.SSID.{i}.SSID` | string | RW | SSID 名称 |
| `Device.WiFi.SSID.{i}.Enable` | boolean | RW | |
| `Device.WiFi.AccessPoint.{i}.Security.ModeEnabled` | string | RW | `WPA2-Personal` / `WPA3-Personal` 等 |
| `Device.WiFi.AccessPoint.{i}.AssociatedDeviceNumberOfEntries` | unsignedInt | R | 已接入客户端数 |
| `Device.WiFi.AccessPoint.{i}.AssociatedDevice.{j}.MACAddress` | string | R | 客户端 MAC |
| `Device.WiFi.AccessPoint.{i}.AssociatedDevice.{j}.SignalStrength` | int | R | dBm |

### 4.6 DHCP

| 参数路径 | 数据类型 | 访问 | 说明 |
|---|---|---|---|
| `Device.DHCPv4.Server.Enable` | boolean | RW | DHCP 服务器开关 |
| `Device.DHCPv4.Server.Pool.{i}.Enable` | boolean | RW | |
| `Device.DHCPv4.Server.Pool.{i}.MinAddress` | string | RW | 地址池起始 |
| `Device.DHCPv4.Server.Pool.{i}.MaxAddress` | string | RW | 地址池结束 |
| `Device.DHCPv4.Server.Pool.{i}.LeaseTime` | int | RW | 租约时长(秒) |
| `Device.DHCPv4.Client.{i}.Enable` | boolean | RW | WAN 侧 DHCP 客户端 |
| `Device.DHCPv4.Client.{i}.IPAddress` | string | R | 已获取的 IP |

### 4.7 Time（NTP）

| 参数路径 | 数据类型 | 访问 | 说明 |
|---|---|---|---|
| `Device.Time.Enable` | boolean | RW | NTP 同步开关 |
| `Device.Time.Status` | string | R | `Synchronized` / `Unsynchronized` / `Error` |
| `Device.Time.NTPServer1` | string | RW | 主 NTP 服务器 |
| `Device.Time.CurrentLocalTime` | dateTime | R | RFC3339 格式 |
| `Device.Time.LocalTimeZone` | string | RW | 如 `Asia/Shanghai` |

### 4.8 软件模块（固件更新相关）

| 参数路径 | 数据类型 | 访问 | 说明 |
|---|---|---|---|
| `Device.SoftwareModules.ExecEnv.{i}.Name` | string | R | 执行环境名 |
| `Device.SoftwareModules.ExecEnv.{i}.Type` | string | R | `Opkg` 等 |
| `Device.SoftwareModules.ExecEnv.{i}.Status` | string | R | `Active` |
| `Device.SoftwareModules.DeploymentUnit.{i}.Name` | string | R | 软件包名 |
| `Device.SoftwareModules.DeploymentUnit.{i}.Version` | string | R | 版本号 |
| `Device.SoftwareModules.DeploymentUnit.{i}.Status` | string | R | `Installed` / `Installing` |
| `Device.SoftwareModules.DeploymentUnit.{i}.URL` | string | RW | 更新来源 URL |

---

## 5. S1 API ↔ USP 消息映射（端到端字段对齐）

> S1 接口基线来自 `docs/tasks/WS-5-architecture.md §3`。实际 Oktopus nbi 路由由 `adapter-ws` 将 WebSocket USP 消息桥接到 NATS，再由 `nbi` 对外暴露 REST。

| S1 REST 端点 | HTTP 方法 | 对应 USP 消息 | 关键字段映射 |
|---|---|---|---|
| `/api/v1/devices` | GET | — | 读 DB，不触发 USP（设备上线时 USP 已建连） |
| `/api/v1/devices/{id}` | GET | GET(`Device.DeviceInfo.*`) | `id` = Agent EndpointID；返回 `Manufacturer`/`ModelName`/`SerialNumber`/`SoftwareVersion` |
| `/api/v1/devices/{id}/parameters` | GET | GET(指定 `param_paths`) | body:`{ "paths": ["Device.WiFi.SSID.1.SSID", ...] }` → USP GET |
| `/api/v1/devices/{id}/parameters` | POST | SET | body:`{ "params": {"Device.WiFi.SSID.1.SSID": "MyNet"} }` → USP SET，`allow_partial=false` |
| `/api/v1/devices/{id}/operate` | POST | OPERATE | body:`{ "command": "Device.Reboot()", "input_args": {} }` → USP OPERATE |
| `/api/v1/notifications` | GET (WS) | NOTIFY | 长连接推送；服务端收到 NOTIFY 后 forward 到此 WS；字段含 `subscription_id`/`event_type`/`params` |

**字段对齐重点（S5/S6 开发必读）：**

1. **设备寻址**：S1 用 `{id}` = Agent `EndpointID`（`os::00256D-xxx`），与 obuspa 配置的 `localagent EndpointID` 必须严格一致，否则 Controller 侧无法路由。
2. **参数路径格式**：S1 body 里的路径必须完整（`Device.WiFi.SSID.1.SSID`），不可省略实例号，否则 USP GET 无法解析到具体值。
3. **SET 原子性**：S1 POST `/parameters` 建议默认 `allow_partial=false`（全成功或全回滚），前端错误提示要区分"参数不可写(9005)"与"参数值非法(9007)"。
4. **NOTIFY 推送**：S6 的通知流转依赖 obuspa Subscription 机制；Controller 需在设备上线时自动创建核心 Subscription（Boot/ValueChange/OperationComplete），避免设备重启后订阅丢失（`Persistent=true` 保证 obuspa 重启后订阅保留，但需数据落盘）。
5. **异步操作追踪**：`Device.Reboot()` / 固件更新等异步命令，S1 `/operate` 仅返回"已接受"，最终结果通过 NOTIFY(OperationComplete) 推送，S6 需将 `command_key` 与 S1 请求关联。

---

## 6. 已验证联调要点（prpl Ambiorix MVP 栈）

以下各项均已在 MT7621 + ImmortalWrt 24.10 上端到端验证（见 `docs/tasks/WS-6-obuspa-evaluation.md`），**换机时按此清单核对即可**：

> **Controller 地址统一使用 `39.97.250.156`（旧测试地址 `39.105.150.244` 已停用）**

| # | 必做操作 | 影响 | 对应配置 |
|---|---|---|---|
| 1 | obuspa 配置 `controller EndpointID 'oktopusController'` | 否则 USP Record from_id 不匹配，所有消息被丢弃 | `obuspa/files/etc/config/obuspa` |
| 2 | obuspa 配置 `localagent EndpointID 'os::<OUI>-<serial>'` | 否则端点 ID 随机，无法路由 | 同上 |
| 3 | obuspa 配置 MTP WebSocket URL 为 `ws://39.97.250.156:8080/ws/agent` | 连接到正确的生产 Controller | `obuspa/files/etc/config/obuspa` |
| 4 | Oktopus 后台对该 EndpointID 设置 `assigned_role_name='full_access'` | 否则 Parameters discovery 无权限，界面转圈 | Oktopus Admin UI 或 API |
| 5 | iopsys 专项：创建 `/etc/board-db/config/device`，写入 ManufacturerOUI/SerialNumber 等 | 否则 DeviceInfo 所有身份字段空，EndpointID 自动推导失败 | `sysmngr/files/etc/board-db/config/device` |
| 6 | iopsys 专项：`disable sysntpd` 后启 timemngr | 否则 timemngr 的 ntpd 绑 :123 失败，被 procd 反复拉起，Time 状态永远 Unsynchronized | `timemngr/files/etc/uci-defaults/97-disable-sysntpd-for-timemngr` |
| 7 | ManufacturerOUI（board-db / prpl DeviceInfo plugin）与 EndpointID 中的 OUI 段必须一致 | 否则身份校验失败（Controller 可能拒接） | 手动核对 |

---

## 7. QA 联调测试用例索引（供 QA_Sherlock 参考）

| # | 测试用例 | 方法 | 期望结果 |
|---|---|---|---|
| T1 | 设备首次上线 | Wireshark 抓 WS 端口 8080；过滤 `websocket`，目标 `39.97.250.156` | 看到 USP Record(NoSessionContext) 的 NOTIFY(OnBoardRequest) |
| T2 | GET DeviceInfo | `/api/v1/devices/{id}` GET | 返回 Manufacturer/SerialNumber/SoftwareVersion 均非空 |
| T3 | GET 不存在路径 | GET `Device.NonExistent.` | 返回 err_code 9003 |
| T4 | SET 只读参数 | POST `/parameters` {`Device.DeviceInfo.Manufacturer`:"xxx"} | 返回 err_code 9008 (Non-Writable Parameter) |
| T5 | SET allow_partial=false，一个参数非法 | POST 含合法+非法参数 | 全部回滚，返回失败的参数路径 |
| T6 | Subscribe ValueChange + 触发变更 | ADD Subscription → SET 触发变更 | NOTIFY(ValueChange) 推到 `/api/v1/notifications` |
| T7 | Subscribe Boot + 重启 | ADD Subscription(Boot!) → OPERATE Reboot() | 重启后 NOTIFY(Boot) 推送到 WS |
| T8 | OperationComplete 追踪 | OPERATE Reboot() + command_key | OperationComplete event 携带相同 command_key |
| T9 | GET_INSTANCES Subscription | GET_INSTANCES `Device.LocalAgent.Subscription.` | 返回所有已存在实例及 unique_key(ID) |
| T10 | 断线重连 | 拔网线 30s 后接回 | obuspa 以指数退避重连，重连后 Controller 侧设备重新 Online |
| T11 | iopsys full_access 缺失 | 不设置 assigned_role_name | discovery 转圈；设置后立即生效 |
| T12 | iopsys board-db 缺失 | 删除 /etc/board-db/config/device | DeviceInfo 全空；补写后重启 sysmngr 即恢复 |

---

## 9. prpl Ambiorix MVP 栈 — 专项 Onboarding 指南

> 本节面向 prpl Ambiorix 终端栈（已选型 MVP），基于 obuspa v11.0.2、TR-181 RootDataModelVersion 2.17、WebSocket MTP。参考：`docs/tasks/WS-6-obuspa-evaluation.md §二`。

### 9.1 prpl Feed 关键版本清单

| 包 | 版本 | 说明 |
|---|---|---|
| `obuspa` | **v11.0.2**（commit `92ecb4c0`） | BroadbandForum 上游；WebSocket MTP 默认开启（prpl feed 将 `OBUSPA_WEBSOCKET_MTP_SUPPORT` 改为 `default y`） |
| `amxrt` | v2.6.1 | Ambiorix runtime |
| `tr181-device` | v0.36.0 | Device. 主树 |
| `deviceinfo-manager` | v2.41.0 | DeviceInfo 子树 |
| `time-manager` | v2.11.0 | Time/NTP 子树 |
| `tr181-security` | v0.14.0 | Security 子树 |
| `tr181-gatewayinfo` | v0.2.0 | GatewayInfo 子树 |

### 9.2 prpl 架构简图

```
Oktopus (39.97.250.156:8080)
    │  WebSocket (RFC 6455)
    ▼
obuspa v11.0.2  ←──(USP Record, Protobuf)──→ TR-369 Controller
    │ amxb (ubus/usp bus)
    ├──► tr181-device          (Device. 主树骨架)
    ├──► deviceinfo-manager    (Device.DeviceInfo.*)
    ├──► time-manager          (Device.Time.*)
    ├──► tr181-security        (Device.Security.*)
    └──► tr181-gatewayinfo     (Device.GatewayInfo.*)

运行时：amxrt 加载各插件 ODL；mod-amxb-ubus/usp 提供 amxb 后端
数据模型版本：Device.LocalAgent.RootDataModelVersion = "2.17"
```

### 9.3 onboarding 配置文件（`/etc/config/obuspa`）

```uci
# /etc/config/obuspa  — prpl Ambiorix 联调基线配置
# 对照 TR-369 §7.2 Agent MTP 配置

config localagent
    option EndpointID  'os::00256D-<SerialNumber>'   # TR-369 §2.3.2；<SerialNumber> 从 DeviceInfo 或 /proc/sys/kernel/hostname 获取
    option WebsocketPort '8080'

config controller
    option EndpointID  'oktopusController'            # 必须与 Oktopus 服务端完全一致
    option WebsocketURL 'ws://39.97.250.156:8080/ws/agent'  # 生产 Controller（2026-06-17 更新）
    option assigned_role_name 'full_access'           # ControllerTrust.Role.1；缺少则 discovery 无权限
```

> ⚠️ **坑**：`ws://` 无 TLS 时 Controller 信任等级落到 `Untrusted`（Role.2），无读权限。**必须**在 Oktopus 控制台对该 EndpointID 显式设置 `assigned_role_name='full_access'`（REST API: `PUT /api/v1/controllers/{id}` 设 `assigned_role_name`）。

### 9.4 prpl 数据模型特性（v2.17）

- **开箱即可读全树**：prpl 默认所有参数均可 GET，无须额外 board-db 配置（对比 iopsys 需创建 `/etc/board-db/config/device`）。
- **身份字段来源**：`deviceinfo-manager` 从内核/OpenWrt UCI 读取 `Manufacturer`/`ModelName`/`SerialNumber`，无外部依赖。
- **已验证可读子树**：`DeviceInfo / Time / LocalAgent / STOMP / MQTT / UnixDomainSockets / USPServices / InterfaceStack / IP / Ethernet / WiFi（若 radio 存在）`。
- **写操作限制**：`obuspa` 自身不直接写系统配置；写操作由对应 `tr181-*` plugin 通过 amxb RPC 落地（各 plugin 负责其子树）。

### 9.5 prpl 启动顺序与依赖

```
1. 启动 Ambiorix 总线后端：mod-amxb-ubus（需 ubusd 已运行）
2. 启动 amxrt + plugins（按 ODL requires 顺序）：
   tr181-device → deviceinfo-manager → time-manager → tr181-security → tr181-gatewayinfo
3. 启动 obuspa（依赖 amxrt/plugin 已就绪）：
   obuspa -p /etc/obuspa.cfg -r /etc/obuspa.db

诊断命令：
   amx-cli /tmp/usp/agent.sock            # 进入 amx CLI，可手动 GET/SET
   obuspa-cli                             # obuspa 内置 CLI（v11.x 支持）
   logread | grep obuspa                  # 查看 WS 连接日志
```

### 9.6 prpl onboarding 成功标志

1. `logread | grep obuspa` 出现 `Connected to Controller` 且无 `Untrusted` 告警。
2. Oktopus 管理界面设备列表显示 EndpointID `os::00256D-*` 状态 **Online**。
3. `GET Device.LocalAgent.RootDataModelVersion` 返回 `"2.17"`。
4. `GET Device.DeviceInfo.Manufacturer` 返回非空字符串。
5. `GET Device.LocalAgent.Controller.1.AssignedRole` 返回 `ControllerTrust.Role.1`（full_access）。

---

## 10. 变更记录

| 日期 | 版本 | 变更 |
|---|---|---|
| 2026-06-17 | v1 | 初始版本，产出 USP/TR-369 全消息类型格式规范、TR-181 核心参数路径清单、S1 API 字段对齐映射、已验证联调要点、QA 测试用例索引。 |
| 2026-06-18 | v2 | 选型确认 prpl Ambiorix 为 MVP；Controller 地址更新为 `39.97.250.156`；新增 §9 prpl 专项 onboarding 指南（obuspa v11.0.2、RootDataModelVersion 2.17、WebSocket 配置、启动顺序与成功标志）；§6 联调要点补充 #3（MTP URL 更新）；T1 测试用例更新目标 IP。 |

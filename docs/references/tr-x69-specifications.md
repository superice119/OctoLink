# TR-x69 标准规范(Broadband Forum)

OctoLink 基于 Broadband Forum 的 CWMP(TR-069)与 USP(TR-369)协议族。本页汇总产品涉及的权威规范,作为各任务(尤其 S2 协议对齐、S5 USP Get/Set、S6 通知、终端 obuspa 二开)的对照依据。

> 实现要求:涉及协议字段/数据模型的代码与文档,必须可回溯到下表对应规范的章节(见 `docs/README.md` 文档规范)。

## 规范清单

| 规范 | 标题 | 链接 | 与 OctoLink 的相关性 |
|---|---|---|---|
| **TR-069** | CPE WAN Management Protocol (CWMP) | [PDF](https://www.broadband-forum.org/technical/download/TR-069_Amendment-6_Corrigendum-1.pdf) | 传统 ACS 管理协议;对应 `acs` 容器(TR-069 ACS)与 CWMP 设备详情/RPC 面板。 |
| **TR-106** | Data Model Template for CWMP Endpoints and USP Agents | [PDF](https://www.broadband-forum.org/technical/download/TR-106_Amendment-10.pdf) | 数据模型模板规范,定义对象/参数命名与结构,是 TR-181 等数据模型的基础。 |
| **TR-142** | Framework for CWMP and USP enabled PON Devices | [PDF](https://www.broadband-forum.org/technical/download/TR-142_Issue-4.pdf) | PON 光网络设备管理框架;纳管 PON 终端时的数据模型扩展依据。 |
| **TR-143** | Enabling Network Throughput Performance Tests and Statistical Monitoring | [PDF](https://www.broadband-forum.org/technical/download/TR-143_Amendment-1_Corrigendum-2.pdf) | 吞吐/网络性能诊断(测速)能力依据,可用于设备诊断类功能。 |
| **TR-157** | Component Objects for CWMP | [PDF](https://www.broadband-forum.org/technical/download/TR-157_Amendment-10_Corrigendum-1.pdf) | CWMP 通用组件对象(如软件模块、固件管理等),固件升级相关对象的参考。 |
| **TR-181** | Device Data Model for TR-069 (Device:2) | [PDF](https://www.broadband-forum.org/technical/download/TR-181_Issue-2_Amendment-15.pdf) | **核心数据模型**:USP/CWMP 的参数路径(如 `Device.DeviceInfo.*`、`Device.WiFi.*`、`Device.Ethernet.Interface.*.MACAddress`)均源于此,S5 Get/Set、S4 批量信息检索的参数依据。 |
| **TR-369** | User Services Platform (USP) | [usp.technology](https://usp.technology/) | **产品核心协议**:obuspa 实现 USP Agent,Oktopus/OctoLink 为 USP Controller;Get/Set/Add/Del/Operate/Notify 消息、MTP(MQTT/WebSocket/STOMP)均依此。 |
| **TR-471** | Maximum IP-Layer Capacity Metric, Related Metrics, and Measurements | [PDF](https://www.broadband-forum.org/technical/download/TR-471_Issue-3.pdf) | IP 层最大容量(测速)度量方法,配合 TR-143 用于性能测量类能力。 |

## 在 OctoLink 中的映射速查
- **协议/MTP**:TR-369(USP)+ TR-069(CWMP) → `controller`、`mtp/*` 适配器、`acs`。
- **数据模型**:TR-106(模板)→ TR-181(Device:2)→ S5 USP Get/Set 的参数路径来源。
- **设备类型扩展**:TR-142(PON)。
- **诊断/性能**:TR-143 + TR-471。
- **组件对象/固件**:TR-157。

## 变更记录
- 2026-06-16:新建,收录 TR-069/106/142/143/157/181/369/471 规范及相关性映射。

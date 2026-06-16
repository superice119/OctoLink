# OctoLink(章鱼物联)— 产品范围与任务依赖树

基于 **Oktopus + obuspa** 的定制化 USP(TR-369)设备管理系统。父任务:Multica WS-4。

## Phase 1 — MVP 核心
| 任务 | 标题 | 负责 | 依赖 |
|---|---|---|---|
| WS-5 (S1) | 云服务架构梳理 / 微服务边界 / API 契约基线 | Cloud_Guru | — (根, 进行中) |
| WS-6 (S2) | obuspa 评估 / USP 协议联调清单 | Protocol_Pro | 老板手动验证 iopsys 后 |
| WS-7 (S3) | Web UI / Logo / 版权 深度换标 | Cloud_Guru | S1 |
| WS-8 (S4) | 设备批量信息检索(MAC 等,只读) | Cloud_Guru | S1(缓做,末期/下阶段) |
| WS-9 (S5) | 实时 USP 参数检索 Get/Set | Cloud_Guru | S1(可用 agent-sim 独立开发) |
| WS-10 (S6) | 通知推送流转 | Cloud_Guru | S1 |
| WS-11 (S7) | 基础多租户 / 权限管理 RBAC | Cloud_Guru | S1 |
| WS-12 (S8) | 测试用例 + 自动化测试 | QA_Sherlock | 用例先行;自动化依赖 S3-S7 |

## Phase 2 — 长期 / 增强
| 任务 | 标题 | 负责 | 依赖 |
|---|---|---|---|
| WS-13 | Android 客户端(双通道控制) | Droid_Master | S1/S5/S6, WS-17 |
| WS-14 | iOS 客户端(双通道控制) | iOS_Evangelist | S1/S5/S6, WS-17 |
| WS-15 | 微信小程序 | Mini_Alchemist | S1/S6 |
| WS-16 | 整合 rttys 远程访问控制(Web 终端) | Cloud_Guru | S1/S7 |
| WS-17 | 移动端控制通道(云端 WS 转发 + 本地 WS) | Cloud_Guru | S1/S7 |

## 开发/测试环境
- 设备模拟:**agent-sim**(github.com/OktopUSP/agent-sim),云端与 QA 可不依赖真实终端独立开发。
- 服务器:已部署 Oktopus/OctoLink Docker 测试环境(访问见 WS-5)。

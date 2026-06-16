# OctoLink 系统架构与关键决策

## 1. 总体架构
```
终端层                         云服务层 (OctoLink fork of Oktopus)            接入层
┌───────────────┐   USP/TR-369   ┌──────────────────────────────┐   ┌──────────────┐
│ obuspa Agent  │  (MTP: MQTT/   │ 微服务: 设备管理 / USP 代理   │   │ Web UI (换标)│
│ prplOS/iopsys │   WebSocket)   │ 通知流转 / 多租户 RBAC        │◄──┤ 移动 App     │
│ + rtty client │ ─────────────► │ rttys / WS 控制转发           │   │ 微信小程序   │
└───────────────┘                └──────────────────────────────┘   └──────────────┘
        ▲ 开发期用 agent-sim 模拟设备
```

## 2. 关键决策(ADR 摘要)
- **ADR-1 云端可独立开发**:用 agent-sim 模拟 USP Device,云端功能(S5/S6/S7)仅硬依赖 S1 架构基线,不阻塞于真实终端;真机字段对齐由 S2 并行补齐。
- **ADR-2 终端方案**:prplOS obuspa 已验证;iopsys obuspa 由老板手动验证(opkg 依赖序安装),结论出后由 Protocol_Pro 接手(S2)。
- **ADR-3 批量操作降风险**:原"批量固件升级"(Operate+重启,高危)改为只读"批量信息检索(MAC 等)"(S4),并缓做至末期/下阶段。
- **ADR-4 移动端双通道控制(Phase 2)**:
  ```
            ┌── 远程 (5G) ──► 云端 ──(WebSocket 转发)──┐
   手机 App ┤                                          ▼
            └── 本地 (Wi-Fi) ── WebSocket 直连 ──► 终端设备 (Agent)
  ```
  客户端优先本地直连(低时延),不可达回退云端转发;底座见 WS-17。
- **ADR-5 远程访问控制**:rttys(WS-16)提供运维 Web 终端/shell;与 WS-17(App 业务控制)通道分离,共用 RBAC + 审计底座。
- **ADR-6 安全基线**:本地 WS 亦需 TLS + token;远程 shell 高危能力必须 RBAC + 会话审计 + 老板审批;设备寻址以 USP Endpoint 为路由键,本地用 mDNS + 证书校验防误连。

## 3. 品牌
- 临时 Logo:章鱼 + 触手末端连接节点(Octo + Link)。主渐变 青 #0EA5A4 → 蓝 #2563EB,辅助灰 #64748B。资产见 `docs/branding/`。

## 4. 合规铁律
- 数据库迁移、线上部署、上架发布等敏感操作须经人类总监(老板)审批(`Awaiting Approval`)。

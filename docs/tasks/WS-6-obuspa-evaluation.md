<!-- 来源:人类总监(王祺)2026-06-16 完成的 obuspa 双方案手动验证产出,纳入 OctoLink 文档库作为 WS-6(S2 协议)评估基线。 -->
<!-- 原始构件(ipk/feed 源码)见工作区仓库:superice119/prpl-usp-feed、superice119/iopsys-usp-feed。 -->

# OpenWrt 网关接入 Oktopus USP Controller — 双方案适配验证报告

**日期:** 2026-06-16
**目标:** 在同一台 OpenWrt 网关上,分别用 **prpl Ambiorix** 与 **iopsys bbfdm** 两套 USP 数据模型栈接入 Oktopus USP Controller,各自做成**自包含 OpenWrt feed**,完成"编译 → 安装 → 连接 → 数据模型可读写"的端到端验证,并横向对比。

## 验证平台

| 项 | 值 |
|---|---|
| 设备 | **Oolite V8.0** |
| SoC / 架构 | MediaTek **MT7621**(`mipsel_24kc`) |
| 标称容量 | **32MB**(Flash) |
| 运行系统 | ImmortalWrt 24.10-SNAPSHOT(内核 6.6) |
| 实测内存 | `Device.DeviceInfo.MemoryStatus.Total ≈ 432 MB`(运行期读出) |
| Controller | Oktopus,`39.105.150.244`,USP over WebSocket(`ws://…:8080/ws/agent`) |
| 编译方式 | GitHub Actions + 官方 OpenWrt SDK(`openwrt/gh-action-sdk`),目标 OpenWrt 24.10.7 |
| 验证架构 | mt7621 `mipsel_24kc`(主)+ rk3308 `aarch64_generic`(并行验证可移植性) |

> 说明:设备 hostname 显示 `ImmortalWrt`;两套方案先后部署在同一硬件上,Oktopus 中分别以不同 EndpointID 区分。

---

## 一、公共背景

- **USP(TR-369)**:Broadband Forum 的设备管理协议。设备侧跑 **USP Agent**,经 MTP(此处为 WebSocket)连到 **USP Controller**(Oktopus)。Agent 暴露一棵 **TR-181 `Device.` 数据模型树**供 Controller 读写。
- **两套栈的本质差异**在于"谁来提供并维护这棵 `Device.` 树":
  - **prpl**:Ambiorix 框架 + 一组 `tr181-*` 插件守护进程,每个插件自带一片子树;USP Agent 用 BBF 上游 obuspa。
  - **iopsys**:`bbfdm` 作为 ubus broker + 微服务;数据模型被拆散到约 50 个 manager 守护进程;USP Agent 用 iopsys 的 obuspa fork(内置 bbfdm 后端)。

两套都落地为**单一 git feed**,可被 OpenWrt SDK 直接 `feeds install` 后整体编译为 `.ipk`。

---

## 二、prpl Ambiorix 方案适配详情

### 2.1 架构

```
                ┌─────────────────────────────────────────┐
   Oktopus ◄───►│ obuspa (BBF upstream, USP Agent + WS MTP) │
   (WS)         └───────────────┬─────────────────────────┘
                                │ amxb (ubus/usp bus)
        ┌───────────────────────┼───────────────────────────┐
        ▼                       ▼                           ▼
  tr181-device           deviceinfo-manager           time-manager  …
  (Device. 主树)         (Device.DeviceInfo)          (Device.Time)
        └──── 全部基于 libamx* + amxrt 运行时,ODL 定义数据模型 ────┘
```

- 运行时:`amxrt`(Ambiorix runtime)加载各插件的 **ODL** 数据模型定义。
- 总线:`mod-amxb-ubus` / `mod-amxb-usp` 提供 amxb 后端;`mod-dmext`、`data-model-mapper` 等做扩展/映射。
- USP Agent:**BroadbandForum/obuspa 上游版**(非 fork),WebSocket MTP。

### 2.2 feed 组成(`prpl-feed/`,共 **37** 个包,4 类)

| 类别 | 数量 | 代表包 |
|---|---|---|
| `libs/` | 19 | libamxc/d/b/o/p/…(Ambiorix 核心库)、libusp、libuspprotobuf、libnetmodel、uriparser |
| `apps/` | 7 | **amxrt**、amx-cli、amxo-cg、**obuspa**、data-model-mapper、usp-discovery、chrony-prpl |
| `mods/` | 6 | mod-amxb-ubus、mod-amxb-usp、mod-dmext、mod-dmproxy、mod-ba-cli、mod-usp-registration |
| `plugins/` | 5 | **tr181-device**、**deviceinfo-manager**、**time-manager**、tr181-security、tr181-gatewayinfo |

### 2.3 关键版本 pin(均经 `git ls-remote` 核实,未臆造)

| 包 | 版本 | 来源 |
|---|---|---|
| obuspa | `v11.0.2`(commit `92ecb4c0…`) | github.com/BroadbandForum/obuspa |
| amxrt | `v2.6.1` | gitlab prpl-foundation/…/amxrt |
| tr181-device | `v0.36.0` | gitlab prpl-foundation/…/tr181-device |
| deviceinfo-manager | `v2.41.0` | gitlab …/tr181-deviceinfo |
| time-manager | `v2.11.0` | gitlab …/tr181-time |
| tr181-security | `v0.14.0` | gitlab …/tr181-security |
| tr181-gatewayinfo | `v0.2.0` | gitlab …/tr181-gatewayinfo |

### 2.4 编译

- 四条编译路径:GitHub CI / 容器(Dockerfile + docker-build.sh)/ 标准 OpenWrt feed 流程 / 本地 build.sh。
- WebSocket 默认开启:`apps/obuspa/Config.in` 把 `OBUSPA_WEBSOCKET_MTP_SUPPORT` 改为 `default y`(上游默认 n;本 feed 就是为 WS onboarding 而生)。
- 整 feed 编译以验证全部 37 个包。

### 2.5 适配/踩坑

- ODL `requires` 依赖链:插件间通过 ODL `requires` 声明依赖,缺包会在加载期失败,需要按依赖把 plugins/mods 一并打齐。
- 数据模型版本:运行后 `Device.RootDataModelVersion = 2.17`。
- 依赖相对"重":Ambiorix 全家桶(19 个 lib)+ amxrt 运行时是硬底座。

### 2.6 验证结果

- Oktopus 显示设备 **Online**,EndpointID `os::00256D-OpenWrtGateway001`。
- Device 树完整:`DeviceInfo / Time / LocalAgent / STOMP / MQTT / UnixDomainSockets / USPServices / InterfaceStack …`,身份字段有值。
- **结论:开箱即得完整树,适配工作量集中在"把依赖打齐"。**

---

## 三、iopsys bbfdm 方案适配详情

### 3.1 架构

```
                 ┌──────────────────────────────────────────────┐
   Oktopus ◄────►│ obuspa (iopsys fork, USP Agent + WS, 内置bbfdm) │
   (WS)          └────────────────────┬─────────────────────────┘
                                       │ ubus
                            ┌──────────▼──────────┐
                            │  bbfdmd  (broker)   │  ubus 对象: bbfdm
                            └──────────┬──────────┘
            路由到各微服务(每个注册自己负责的子树):
        ┌──────────────┬──────────────┬──────────────┬─────────┐
        ▼              ▼              ▼              ▼
 dm-service -m core   obuspa.so     sysmngr        timemngr   … (~50 managers)
 (薄"core"片)        (LocalAgent/   (DeviceInfo)   (Time)
                      MQTT)
```

**关键发现:`bbfdm + obuspa` 本身并不包含 TR-181 数据模型主体。** bbfdmd 只是 broker,`dm-service -m core` 注册的仅是一个**薄片**:`LANConfigSecurity / Schedules / Security / PacketCaptureDiagnostics / SelfTestDiagnostics / {VENDOR}OpenVPN / RootDataModelVersion / Reboot() / FactoryReset()`。`DeviceInfo / Time / IP / Ethernet / WiFi / DHCP …` 分散在约 **50 个独立 manager 守护进程**里(sysmngr、timemngr、netmngr、ethmngr、wifimngr、dhcpmngr、gateway-info…),各自经 `BBFDM_REGISTER_SERVICES` 注册子树。

> 直接证据:`ubus call bbfdm.core get '{"path":"Device.DeviceInfo."}'` 返回 **fault 9005 "Invalid parameter name"**(对象根本不在 core 的 schema 里,而非"值为空")。

### 3.2 feed 组成(`iopsys-feed/`,采用**扁平布局**)

| 包 | 版本 | 作用 | 来源(commit pin 已核实) |
|---|---|---|---|
| **bbfdm**(5 子包) | 1.20.2(`99e70ae4…`) | broker + libbbfdm-api + libbbfdm-ubus + dm-service(core) + bbf_configmngr | dev.iopsys.eu/bbf/bbfdm |
| **obuspa** | 11.0.3.1-r1(`92aabe98…`) | iopsys USP Agent fork(WS MTP,内置 bbfdm 后端) | dev.iopsys.eu/bbf/obuspa |
| **libeasy** | 7.5.1(`b981f7e1…`) | dm-service 运行依赖 | dev.iopsys.eu/hal/libeasy |
| **sysmngr** | 1.3.0(`a806420d…`) | 提供 `Device.DeviceInfo` | dev.iopsys.eu/system/sysmngr |
| **timemngr** | 1.1.15(`d4c2d84c…`) | 提供 `Device.Time`(基于 ntpd) | dev.iopsys.eu/bbf/timemngr |
| **jqhost** | 1.7.1 | 构建辅助:把静态 jq 放到 build host PATH | github.com/jqlang/jq(release) |

> 扁平布局原因:`obuspa/Makefile`、`sysmngr/Makefile`、`timemngr/Makefile` 都 `include ../bbfdm/bbfdm.mk`,故 bbfdm 必须与它们同级。

### 3.3 编译 + 核心踩坑

1. **隐式 host `jq` 依赖(最关键)**:`bbfdm/tools/bbfdm.sh` 在**安装期**(`BBFDM_REGISTER_SERVICES`/`BBFDM_INSTALL_MS_DM` 宏)shell out 调 `jq` 来校验/注册微服务 JSON。OpenWrt SDK 容器(user `buildbot`,无 apt、无 jq)缺 jq → obuspa 与 dm-service 安装步骤失败,在 `IGNORE_ERRORS=1` 下被**静默丢弃**:构建显示绿色,却少了两个最关键的 ipk。
   - **解法**:新增 `jqhost` 包 —— `HostBuild` 把官方静态 `jq-linux-amd64`(1.7.1)装到 `STAGING_DIR_HOST/bin`;bbfdm/obuspa/sysmngr/timemngr 全部 `PKG_BUILD_DEPENDS:=jqhost/host`。两个易错点:① `PKG_SOURCE` 必须正好是 `jq-linux-amd64`(release 资产真名);② 纯 HostBuild 不产生包索引,需加 `BUILDONLY:=1` 的 `Package/` 让 `feeds install -a` 能软链它。
2. **验证方式**:必须按 **ipk 是否存在**判定成败,而不是构建退出码(`IGNORE_ERRORS` 会掩盖丢包)。
3. **结果**:CI 在 mt7621 + rk3308 两架构均绿,**9/9 ipk** 齐全(含 obuspa、dm-service、sysmngr、timemngr)。

### 3.4 数据模型补全:补 sysmngr + timemngr

- 仅 bbfdm+obuspa:设备能 **Online**,但 Parameters 树几乎空(只有薄 core + LocalAgent/MQTT)。
- 加 `sysmngr`(→ DeviceInfo)、`timemngr`(→ Time)后,9005 消失,对象出现。
- 两者依赖很轻(仅 OpenWrt base + bbfdm 库;timemngr 另需 ntpd),**不拖 iopsys HAL**(libwifi/libdsl/libvoice 这类才拖)。

### 3.5 非 iopsys 硬件上的 4 处"平台胶水"(本次适配的核心工作量)

iopsys 这套在自家发行版上默认环境齐备,移到 ImmortalWrt 才暴露这些隐式依赖:

| # | 现象 | 根因(已追到源码) | 解法 |
|---|---|---|---|
| 1 | 连不上,日志 `Ignoring USP record with inconsistent endpoint (from_id=oktopusController)` | controller 的 EndpointID 与 Oktopus 自报端点不一致,USP 记录被丢 | controller `EndpointID='oktopusController'` |
| 2 | agent 身份畸形 | DeviceInfo 空 + `db`/`get_serial_number` 缺,自动推导端点不可靠 | 显式 `localagent EndpointID='os::00256D-iopsysGateway001'` |
| 3 | DeviceInfo 身份字段全空(SerialNumber/SoftwareVersion/Manufacturer…) | `sysmngr/src/deviceinfo.c` 用 `db_get_value_string("device","deviceinfo",…)`,经 `libbbfdm-api` 读 UCI **`/etc/board-db/config/device`**(`ETC_DB_CONFIG`);该路径 ImmortalWrt 不存在 | 创建 `/etc/board-db/config/device`,填 `config deviceinfo 'deviceinfo'` 各字段 |
| 4 | Parameters 一直转圈(discovery 拉不到) | `ws://` 无 TLS 证书,controller 未显式授角色 → 落到 `Untrusted`(Role.2),无读权限 | controller `assigned_role_name='full_access'` → `AssignedRole=ControllerTrust.Role.1` |

> 上述 4 处 + 下述 2 处冲突现已**全部固化进 feed**(commit `ecb55b4`,CI 两架构 9/9 复测通过),换板无需再手敲,详见第五节。

**附带两处运行冲突(非阻断):**
- **timemngr vs 系统 ntpd**:timemngr 设计上独占 ntp(`system.ntp` 存在则拒绝启动,迁移脚本会删它),但系统里已在跑的 `sysntpd` 占着 :123 → timemngr 的 ntpd 绑不上、被 procd 反复拉起。解法:`disable sysntpd` 让 timemngr 独占。
- **sysmngr fwbank 刷屏**:`/etc/sysmngr/fwbank` source 了 `/lib/functions/iopsys-fwbank.sh` 等 iopsys **A/B 双分区**胶水,单镜像设备读空、重试 10 次后**放弃但继续运行**(非崩溃)。可重编关闭 `CONFIG_SYSMNGR_FWBANK_UBUS_SUPPORT`。

### 3.6 验证结果(逐项打通)

| 阶段 | 结果 |
|---|---|
| 两架构编译 | ✅ 9/9 ipk |
| 安装 | ✅ |
| USP WebSocket 连接 | ✅ Online |
| DeviceInfo 身份透传 Oktopus | ✅ MODEL `RT-MT7621` / VENDOR `iopsys` / VERSION `ImmortalWrt-24.10` |
| Device.Time | ✅ `Status: Synchronized` |
| 完整数据模型 discovery | ✅ full_access,树全展开,`RootDataModelVersion 2.20` |

---

## 四、两方案对比

### 4.1 关键维度对比

| 维度 | prpl Ambiorix | iopsys bbfdm |
|---|---|---|
| USP Agent | BBF 上游 obuspa `v11.0.2` | iopsys obuspa fork `11.0.3.1`(内置 bbfdm 后端) |
| 数据模型框架 | Ambiorix + ODL,每插件自带子树 | bbfdmd broker + 微服务,树拆散到 ~50 managers |
| 数据模型版本 | `RootDataModelVersion 2.17` | `RootDataModelVersion 2.20` |
| feed 包数 | **37** | **8 个目录 / ~11 个 ipk** |
| 运行依赖 | 重(19 个 libamx* + amxrt 运行时) | 轻(OpenWrt base + libbbfdm + libeasy);完整树需逐个加 manager |
| "开箱即完整树" | **是**(打齐依赖即可) | **否**(bbfdm+obuspa 只给薄 core,要补 manager) |
| 非原生平台适配点 | 主要是 ODL `requires` 依赖打齐 | 4 处平台胶水(endpoint/board-db/role/ntpd)+ jqhost 构建坑 |
| 构建坑 | ODL 依赖链 | **安装期隐式 host jq**(静默丢包)→ 需 jqhost |
| 身份字段来源 | 插件内置/系统读取,默认有值 | UCI `/etc/board-db/config/device`,非 iopsys 平台需手动建 |
| 角色/权限 | 默认即可读全树 | `ws://` 无证书必须显式 `full_access`,否则 discovery 空 |

### 4.2 适配工作量与"坑"的性质对比

- **prpl**:工作量在**广度**——把 37 个相互 `requires` 的包打齐;一旦齐全,数据模型与权限基本默认可用。坑是"缺包加载失败"。
- **iopsys**:工作量在**深度**——栈本身轻,但数据模型是分布式的,且大量依赖 iopsys 发行版隐式提供的环境(host jq、`db`/`board-db` 后端、ntp 独占、双分区 fwbank、角色授权)。坑是"绿色构建却静默丢包""对象注册了但值为空""连上了但 discovery 没权限"——每一处都需要追到源码才能定位。

### 4.3 选型建议

| 场景 | 倾向 |
|---|---|
| 想最快拿到完整 TR-181 树、接受较重依赖 | **prpl** |
| 想要轻量 broker、只暴露自己需要的子树、可接受逐步集成 manager | **iopsys** |
| 目标是贴近 iopsys 商用栈/复用其 ~50 个 manager 生态 | **iopsys**(但要预算平台胶水移植成本,含 HAL 类 manager) |
| 非 iopsys 硬件、想省事 | **prpl**(iopsys 的隐式平台依赖在异构硬件上代价更高) |

---

## 五、结论

两套方案在 **Oolite V8.0 / MT7621 / ImmortalWrt 24.10** 上**均完整打通**:编译 → 安装 → WebSocket 连接 Oktopus → Online → DeviceInfo/Time 等数据模型有值 → discovery 全树可读。

- **prpl** 以较重的 Ambiorix 依赖换来"开箱即完整树",适配集中在依赖打齐。
- **iopsys** 栈更轻、broker+微服务架构更现代,但数据模型分布式、对 iopsys 发行版环境有多处隐式依赖;在非原生硬件上需补齐 4 处平台胶水并解决 host-jq 构建坑。两者各有取舍,可按"完整度优先(prpl)/ 轻量与生态优先(iopsys)"选型。

### 复现要点(iopsys)— 已固化进 feed(commit `ecb55b4`)

以下 5 项已全部写入 `iopsys-feed/`,并经 CI(run `27604361409`,mt7621 + rk3308 两架构 **9/9 ipk**)复测通过,换板即用:

| # | 固化内容 | 落点 |
|---|---|---|
| 1 | `obuspa` controller 示例段(注释)含 `EndpointID 'oktopusController'` + `assigned_role_name 'full_access'`,并标注两字段的坑 | `obuspa/files/etc/config/obuspa` |
| 2 | `/etc/board-db/config/device` 身份模板(`INSTALL_CONF`,示例值 + 注释,升级不覆盖) | `sysmngr/files/etc/board-db/config/device` + Makefile install |
| 3 | uci-default 安装期 `disable sysntpd`(timemngr 独占 ntp) | `timemngr/files/etc/uci-defaults/97-disable-sysntpd-for-timemngr` |
| 4 | `CONFIG_SYSMNGR_FWBANK_UBUS_SUPPORT` 默认改 `n`(消 fwbank 噪音) | `sysmngr/Config.in` |
| 5 | 所有含 `bbfdm.mk` 的包带 `PKG_BUILD_DEPENDS:=jqhost/host` | `bbfdm`/`obuspa`/`sysmngr`/`timemngr` |

> 注:模板用占位身份值(未写死真实序列号);`ManufacturerOUI` 需 6 位十六进制并与 agent EndpointID `os::<OUI>-<serial>` 一致。

---

*附:所有版本 pin / commit hash / PKG_HASH 均经 `git ls-remote` 或官方 release 核实,未臆造。*

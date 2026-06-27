# WS-31 控制台各菜单右侧内容区中英双语(方案 A:跟随语言切换器)

## 1. 背景与目标

- 对应议题:WS-31《[前端] 控制台各菜单右侧内容区中英双语(方案A: 跟随语言切换器)》,父任务 WS-4。
- 背景:WS-30 已完成 i18n 基建(react-i18next + 语言检测)、顶栏语言切换器、侧边导航 / 顶栏 / 账户菜单 / 各页面 `<head>` 标题双语。但**右侧主内容区**(`frontend/src/sections/` 下的内容组件)仍全部硬编码英文,无一接入 i18n。
- 目标(方案 A):右上角语言切换器切到「中文」→ 各菜单右侧内容整页中文;切到「English」→ 整页英文;与侧边栏行为一致,**无需刷新即时生效**。
- 范围:`frontend/src/sections/` 下 24 个内容组件 + 以 props 透传可见英文文案的相关 `pages/` 页面。不改后端字段与 API。

## 2. 设计与选型

- **沿用既有栈**:react-i18next + `useTranslation()`,catalog 为 `frontend/src/i18n/locales/zh.json` / `en.json`。不引入与现有 MUI + Emotion 体系冲突的方案(Tailwind / shadcn 等)。
- **键命名**:按菜单 / 组件分层组织,复用 WS-30 既有命名空间并扩展:
  - `overview.*`(budget / totalProfit / totalDevices / sinceLastMonth / sales / latestProducts / latestOrders / traffic / mtp)
  - `devices.page.*`、`devices.detail.*`、`devices.usp.{discovery,paramsQuery,rpc}.*`、`devices.cwmp.{rpc,wifi}.*`
  - `credentials.*` / `credentials.table.*`
  - `settings.{colorTheme,notifications,password}.*`
  - `notifications.list.*`
  - `account.{profile,profileDetails}.*`
  - `companies.*`、`customers.*`
  - `accessControl.{roles,tenants,users}.*`(含 `accessControl.users.roles.*` 角色显示标签)
  - `chat.*`
- **展示层 vs 逻辑层分离(核心红线)**:多个组件用英文字面量做逻辑判断,只翻译**渲染显示**,判断常量 / 开关值 / 对象键 / 路由 / 后端字段一律保持原英文:
  - `overview-traffic.js`:`title` prop 仍为英文常量 `'Status'` / `'Vendors'` / `'Devices Type'`,供 `useChartOptions` 选色逻辑使用;新增 `displayTitle` 仅用于 `CardHeader` 显示翻译。
  - `overview-latest-orders.js`:设备状态色仍由数值 `statusMap[order.Status]` 决定;`status()` 改为返回**键名**(`statusOffline/statusAssociating/statusOnline/statusUnknown`),渲染时 `t()` 取译文;协议判断 `usp/cwmp`、`order.SN/Alias/Status` 等后端字段不变。
  - `overview-tasks-progress.js`:`type` prop(`stomp/mqtt/websocket`)保持英文常量;显示标题 `mtp` 由 `pages/index.js` 改为 `t('overview.mtp.*')` 传入。
  - 设备 RPC / WiFi:CWMP RPC 方法名(GetParameterValues、AddObject、Reboot 等)作为发往后端的标识保持原文;USP 路径 / TR-181 / SSID 等专有名词不翻译。
- **专有名词**(中英一致不翻译):OctoLink、USP、CWMP、MQTT、STOMP、WebSocket(s)、TR-069、TR-181、SSID、RPC、WiFi。
- **即时生效**:所有 `t()` 调用置于组件函数体内,不在模块作用域缓存字符串,语言切换触发组件重渲染,无需刷新。

## 3. 接口与协议

- 本任务**仅前端文案外置**,不改任何后端 API 的请求 / 响应结构与字段。
- 设备状态、别名、厂商、型号、版本等仍按既有大写 `order.*` 字段读取;USP / CWMP 参数 Get/Set、RPC 方法名等保持原协议契约。
- 与 Cloud_Guru 的 S1 API 契约保持不变,字段与错误码不受影响。

## 4. 部署与使用

- **使用**:右上角语言切换器选择「中文 / English」,选择持久化于 `localStorage`(key `octolink.lang`),刷新 / 重进不丢失;右侧内容区随切换即时改变语言。
- **部署纪律(强制)**:
  - 前端镜像务必在**离线构建机 `39.105.150.244`** 上 `next build` →`docker build`→`docker save`→ 传至控制器 `docker load`。
  - **严禁在 2G 控制器 `39.97.250.156` 上跑 `next build` / 镜像构建**——会 OOM 导致全站宕机。
  - **上线须老板审批**,本任务不得自行部署。

## 5. 测试与验收

对照 WS-31 验收标准:

1. **无残留硬编码英文**:切到「中文」后,总览 / 设备(USP+CWMP)/ 凭据 / 设置 / 通知 / 账户 / 角色 / 租户 / 用户 / 公司等各菜单右侧内容区无残留英文(专有名词除外);切到 English 全英文。
2. **键集合一致**:`zh.json` 与 `en.json` 深度键集合完全一致(400 = 400,已用脚本校验,无缺键,避免 fallback 露英文)。
3. **即时生效**:切换语言无需刷新;表格表头 / 图表标题与图例 / 弹窗 / 表单 label / 状态标签 / 按钮 / Tooltip 均覆盖。
4. **逻辑无回归**:`overview-traffic` 选色、`overview-latest-orders` 状态色与协议跳转、`overview-tasks-progress` MTP 类型判断、设备 RPC 方法名等均使用原英文常量,逻辑未变。
5. **构建验证**:`next build` 通过(19 条路由全部编译成功)。`next lint` 因工具链预存 ajv 兼容性问题(`NOT SUPPORTED: option missingRefs`)在评估任何源码前即失败,与本次改动无关;`next build` 默认跳过 lint。
6. **回归复审**:交 QA_Sherlock 做双语完整性 + Playwright 中英切换巡检与回归。

## 6. 变更记录

| Commit | 内容 |
|--------|------|
| `95be555` | 新增 overview / devices / credentials / settings / notifications / account / access-control / companies / customers 内容组件的 zh/en catalog 键 |
| `bbb7ea8` | overview 内容组件接入 i18n(卡片 / 图表 / 表格 + index.js 透传 props);新增 `overview.traffic.online/offline` |
| `ce9da2a` | 设备 USP/CWMP 内容组件 + devices.js + usp/cwmp 详情页 |
| `4ff90ed` | access-control(roles/tenants/users)、companies、customers 内容 |
| `59b34ad` | settings、account、notifications、credentials、chat 内容 |

**改动文件**:`frontend/src/sections/{overview,devices,credentials,settings,notifications,account,companies,customer}/**`、相关 `frontend/src/pages/{index,devices,account,notifications,settings,credentials,companies,chat,access-control/*}.js`、`frontend/src/i18n/locales/{zh,en}.json`。

**新增 catalog 键集**:`overview.*`(含 traffic.online/offline、mtp.*)、`devices.{page,detail,usp,cwmp}.*`、`credentials.*`、`settings.*`、`notifications.list.*`、`account.{profile,profileDetails}.*`、`companies.*`、`customers.*`、`accessControl.{roles,tenants,users}.*`、`chat.*`(zh/en 各 400 键,完全一致)。

**PR**:https://github.com/superice119/OctoLink/pull/23

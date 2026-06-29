# WS-36 前端发版门禁修复:build / lint / 依赖审计

> 任务:解除 QA_Sherlock 在 WS-27 发版门禁 SOP 中发现的三项前端阻塞——`npm run build` 失败、`npm run lint` 失败、`npm audit` 5 条 high。第 1 项是 WS-31/33/35 共用前端镜像的前置。

## 1. 背景与目标

QA 在走发版门禁时,在 `frontend/` 依次执行:

1. `npm run build` → 报 `PageNotFoundError: Cannot find module for page: /devices/cwmp/[...id]`
2. `npm run lint` → 报 `NOT SUPPORTED: option missingRefs ... Cannot set properties of undefined (setting 'defaultMeta')`
3. `npm audit --omit=dev --audit-level=high` → 5 条 high

目标:让 production build 成功、lint 通过(exit 0)、并对 5 条 high 给出逐条评估与排期建议(是否本轮处理由老板拍板)。

## 2. 设计选型与根因分析

### 2.1 build 失败 —— 根因是构建上下文污染,非源码 bug

在干净 checkout 上复现失败 **未果**:`rm -rf .next && npm run build` 在 node 18(`node:18-alpine`,即生产镜像基底)与 node 22 上均**成功**,`/devices/cwmp/[...id]` 正常产出。WS-31 对该页的改动(仅文案 i18n,新增 `useTranslation`)不影响构建。

根因定位为**构建上下文里残留的陈旧 `.next`**:

- QA 复现前刚跑过 `npm run test:e2e`(Playwright),会留下 **dev 模式的 `.next/`**;紧接着 `npm run build` 让 production build 在 dev 产物上做增量收集 page data,正是 Next.js `Cannot find module for page` 的已知触发场景(对 node 版本/时序敏感)。
- 生产镜像走 `build/Dockerfile` 的 `COPY . ./`,而 **仓库此前没有 `.dockerignore`**,会把宿主机的 `.next/`(214M)与 `node_modules/`(1G,且是 darwin 架构)整包拷进镜像上下文,既污染构建又拖慢/撑大镜像。
- 控制器仅 2G 内存,在受限机器上 build 还可能因 OOM 让 webpack 静默漏写 page chunk,同样表现为该错误(故既有计划「离线构建机出镜像→传控制器,严禁 2G 控制器 build」)。

因此修复方向是**让构建确定性地从干净状态开始**,而非改页面源码。

### 2.2 lint 失败 —— `ajv` override 把 ajv 8 强加给 eslint 8

`package.json` 的 `overrides` 里有 `"ajv": "^8.18.0"`(自基线导入提交 `94f3186` 起就在)。它把全工程的 ajv 统一锁到 8,而 `next lint` 用的 eslint 8 依赖 ajv **6** 的 API(`missingRefs` 选项、`defaultMeta`)。ajv 8 移除了这些 → eslint 初始化即崩(报 `NOT SUPPORTED: option missingRefs` / `Cannot set properties of undefined (setting 'defaultMeta')`)。

修复:**移除该 override**。eslint 随即解析回自带的 `ajv@6.15.0`(`ajv@6.12.3+` 已修复原型污染 CVE,无 high 漏洞),lint 工具链恢复运行。

工具链修好后,`next lint` 暴露出 **10 条预存代码错误**(均来自上游 Oktopus 基线,非本次引入;`next.config.js` 里 `eslint.ignoreDuringBuilds: true` 使其不阻塞 build,但阻塞门禁的 `npm run lint` exit 0)。逐条修复见下。

## 3. 接口协议 / 变更清单

无对外 API 变更。改动均为构建配置与代码质量修复。

| 文件 | 改动 | 说明 |
|---|---|---|
| `frontend/package.json` | 移除 `overrides.ajv` | 修复 lint 工具链崩溃 |
| `frontend/package.json` | 新增 `prebuild` 脚本 | 每次 `npm run build` 前用 node 跨平台 `rm -rf .next`,确保干净 production build,杜绝陈旧 `.next` 触发的 `PageNotFoundError` |
| `frontend/.dockerignore` | 新增 | 把 `node_modules`/`.next`/`.env*`/`e2e` 等挡在镜像构建上下文外;上下文从 ~1.2G 降到 **564kB** |
| `frontend/package-lock.json` | 随 ajv override 移除重解析 | ajv 回到 6.15.0 |
| `src/pages/access-control/tenants.js` | 修 `react-hooks/rules-of-hooks` | 见下 |
| `src/layouts/dashboard/side-nav.js`、`side-nav-item.js` | `react/no-children-prop` 行内 disable | `children` 此处是子菜单**数据数组**而非 JSX,行内豁免并加注释 |
| `src/pages/devices.js`(Chip + 6×MenuItem)、`sections/devices/{cwmp,usp}/devices-rpc.js` | 补 `key` prop | 修 `react/jsx-key` |

### tenants.js 的 hooks 修复(唯一涉及运行行为的改动)

原代码在 `useState` 之后、`useEffect` 之前对非 `super_admin` 提前 `return null`,导致 hook 调用顺序随渲染变化(React 会抛 "rendered more hooks than during the previous render")。修复保持原有跳转语义:

- 计算 `isSuperAdmin`,去掉提前 return;
- 把 `/403` 跳转与 `fetchTenants()` 收进 `useEffect`(依赖 `[isSuperAdmin]`);
- 在所有 hooks 之后用 `if (!isSuperAdmin) return null` 守卫渲染。

非 super_admin 行为不变(跳 `/403`、不渲染、不拉数据),且消除了潜在的 hooks 顺序崩溃。

## 4. 部署 / 使用

- 本地门禁:`cd frontend && npm install && npm run build && npm run lint`(`prebuild` 会自动清 `.next`)。
- 镜像构建(沿用既有铁律,**离线构建机出镜像,严禁 2G 控制器 build**):
  ```
  cd frontend/build && make build   # docker build -f Dockerfile ../,基底 node:18-alpine
  make push / make release          # 推镜像(按既有流程)
  ```
  新增 `.dockerignore` 后,上下文 564kB、构建干净。该前端镜像即 WS-31 双语 + WS-33 接入方式列/默认英文 + WS-35 窄屏修复的共用产物。
- 生产替换须老板审批。

## 5. 测试验收

均在 `node:18-alpine`(生产基底)与本机 node 22 下验证:

| 门禁项 | 结果 |
|---|---|
| `npm run build`(含 `prebuild` 清理) | ✅ Compiled successfully,19/19 静态页,`/devices/cwmp/[...id]` 正常 |
| `make build`(真实生产路径) | ✅ 上下文 564kB,builder 阶段 `npm run build` 通过,产出 `octolink/frontend:<sha>` |
| `npm run lint` | ✅ exit 0(0 error,148 warning;warning 为既有 `jsx-max-props-per-line` 风格项,不阻塞) |

## 6. 依赖审计评估(item 3 —— 报排期,老板拍板)

实跑 `npm audit --omit=dev`:**5 high + 5 moderate**。说明:QA 报告里提到的 `lodash`/`lodash-es` 在本次环境确实存在(经 formik/simplebar 传递),已纳入下表。

| 包 | 当前 | 漏洞要点 | 可修版本 | 风险/建议 |
|---|---|---|---|---|
| `next` | 15.5.12 | SSRF / 缓存投毒 / DoS / 中间件绕过等(多为 App Router;本项目是 **Pages Router**,部分不适用) | **15.5.19**(`^15` 内 patch,非 major) | **低风险,建议升**——`npm audit fix` 即可,免破坏 |
| `ws` | 8.17.1 | 未初始化内存泄露 / 分片 DoS(经 socket.io-client→engine.io-client 传递) | 8.21.0(非 major) | **低风险,建议升** |
| `socket.io-parser` | 4.2.4 | 二进制附件无上限 | 4.2.6(patch) | **低风险,建议升** |
| `socket.io-client` | 4.7.5 | 依赖上面两者 | 4.8.3(非 major) | **低风险,建议升** |
| `lodash` / `lodash-es` | 4.17.23 | `_.template` 代码注入 / `_.unset`·`_.omit` 原型污染 | **暂无修复版本**(advisory range `<=4.17.23`,上游未发补丁);经 `formik@2.2.9`、`simplebar-core` 传递 | **暂缓**——应用未把不可信输入喂给 `_.template`/`_.unset`/`_.omit`,实际可利用性低;待 lodash 发补丁或 formik 升级后再处理,不应阻塞发版 |

**建议:** 其余 4 条 high(next/ws/socket.io-parser/socket.io-client)可用一条 `npm audit fix --omit=dev` 一次性、非破坏性解决(dry-run 确认全部为非 major,next 仍停在 15.x)。考虑到此前服务器有勒索事件,建议尽快处理这 4 条;`lodash` 系暂缓监控。是否并入本轮还是单独排期,由老板拍板——本 PR 暂未改动依赖,保持门禁修复聚焦。

## 7. 变更记录

- 2026-06-30 Web_Maestro:
  - 移除 `package.json` 的 `ajv` override,修复 `next lint` 工具链崩溃;
  - 新增 `prebuild` 清理脚本 + `.dockerignore`,让 build 确定性从干净状态开始,根治 `PageNotFoundError`(构建上下文污染类),镜像上下文 1.2G→564kB;
  - 修复 10 条预存 lint 错误(rules-of-hooks ×1、no-children-prop ×2、jsx-key ×7);
  - 完成依赖审计评估并给出排期建议(本轮未改依赖)。

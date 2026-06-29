# WS-35 修复 13.3" 窄屏布局 3 处 bug(Docs 遮挡 / 卡片图标重叠 / 断点比例)

> 父任务:WS-4。诊断根因见 WS-4 评论。本任务按老板「继续推进」放行修复。

## 1. 背景与目标

老板在 **ThinkPad S2 13.3"(1920×1080 面板,Windows 150% 缩放 → Chrome CSS 视口约 1280px)** 上发现三处控制台布局缺陷:

1. **Docs 菜单被「Powered by」+ logo 遮挡、点不到** —— 侧栏 footer 浮在菜单之上,矮屏时压住最后一项 Docs 并劫持点击。
2. **概览卡片图标与标题重叠**(TOTAL DEVICES / STOMP / MQTT / WEBSOCKETS)—— 列变窄后标题文字挤到右侧 56×56 图标上。
3. **该屏整体比例错乱** —— 与第 2 点同源:断点设计让 1200–1450px 区间正文只有 ~1000px 却硬塞一行 4 张卡。

**目标**:在 ~1280px 视口下 Docs 可见可点、四卡图标与标题不重叠、整体比例正常;同时**不回归宽屏**(1920px 仍四卡一行)。

## 2. 设计与选型

沿用既有 **MUI 5 + Emotion** 主题与栅格体系,不引入新方案。三处均为最小侵入式的布局修正:

- **Docs 遮挡(根因:绝对定位 footer)**
  `frontend/src/layouts/dashboard/side-nav.js` 的 footer 原先由两个 `position:absolute; bottom; zIndex:9999` 元素拼成,脱离布局流浮在菜单上。
  改法:把 footer 改为**正常布局流的「列底」元素**(容器本就 `display:flex; flexDirection:column; height:100%`,nav 用 `flexGrow:1` 自然把 footer 顶到底部),去掉 absolute / zIndex hack。
  同时给 `<Box component="nav">` 加 `minHeight:0; overflowY:auto` —— 抵消 flex 的 `min-height:auto` 下限,**矮屏时菜单区内部滚动**(Docs 始终可达),footer 恒定钉在底部、永不压住菜单。

- **图标 / 标题重叠(根因:flex 子项不收缩)**
  `overview-tasks-progress.js` 与 `overview-total-customers.js` 同款布局:`Stack direction="row" justifyContent="space-between"`,左侧标题 Stack 默认 `min-width:auto`(不肯收缩),右侧固定 56×56 `Avatar` 无 `flexShrink:0`。列一窄,标题就挤到图标上。
  改法:左侧文本 `Stack` 加 `minWidth:0`(允许收缩 / 换行),`Avatar` 加 `flexShrink:0`(不被压扁)。这是 MUI 该类布局的标准修法。

- **断点比例(根因:lg 段一行塞 4 卡)**
  `layout.js` 在 `lg`(≥1200)给正文让出固定 280px 侧栏,而 `pages/index.js` 四卡又在同一 `lg` 切成一行 4 张(`xs=12 sm=6 lg=3`),于是 1200–1450px 时每张仅 ~220px → 触发第 2 点。
  改法:把「一行 4 张」推到更宽断点 —— `xs=12 sm=6 lg=6 xl=3`。即 **lg 段每行 2 张(1280px 下每张 ~480px),xl(≥1536)恢复一行 4 张**。下方三张 `OverviewTraffic` 图表维持 `lg=4`,不在本次范围。

## 3. 接口 / 协议

纯前端布局修复,**不涉及任何后端接口、数据结构或 API 契约变更**。仅触及 MUI 栅格断点与 `sx` 样式属性。

## 4. 部署与使用

- **不单独发版**:与 [WS-33] 的前端镜像**一起打包**(一张前端镜像同时含 WS-31 双语内容 + WS-33 接入方式列/默认英文 + 本任务三处布局修复)。
- 出镜像走**离线构建机** → 传控制器;**严禁在 2G 控制器上 build**。
- 生产替换**须老板审批**。
- 本 PR 仅合入代码,不触发任何构建 / 部署动作(超出本角色范围)。

## 5. 测试与验收

**自动化回归(Playwright,WS-29 脚手架)**:新增 `frontend/e2e/layout-responsive.spec.js`,3 条用例:

| 用例 | 视口 | 断言 |
| --- | --- | --- |
| Docs 可见可点 | 1280×640 | `getByRole('link',{name:'Docs'})` 可见,`click({trial:true})` 通过遮挡 hit-test(不被 footer 覆盖) |
| 卡片标题不压图标 | 1280×640 | 四卡各自标题文字盒与 `.MuiAvatar-root` 盒**无几何相交** |
| 宽屏不回归 | 1920×1080 | 四卡 `boundingBox().y` 偏差 < 5px(仍同一行) |

运行:
```bash
cd frontend
npm ci
npx playwright install chromium
CI=1 npx playwright test layout-responsive.spec.js
```

**本地验证结果**:
- `npm run build` ✅ 通过(无编译错误)。
- 修复后 3/3 用例通过。
- **复现验证**:回退三处源码改动(保留新测试)后,Docs 用例**失败**(footer 遮挡劫持点击)—— 证明用例确实复现了老板报告的 bug,而非「改完才碰巧通过」。

**Definition of Done**:
- [x] 1280px 视口:Docs 可见且可点击。
- [x] 1280px 视口:四卡图标与标题不重叠(每张 ~480px,余量充足)。
- [x] 整体比例正常(lg 段 2 卡/行)。
- [x] 不回归宽屏:1920px 四卡一行。
- [x] 1280px 布局回归用例已加入 Playwright。
- [ ] QA_Sherlock 回归确认(交接后)。

> 注:`npm run lint` 在本机因 eslint/ajv 环境问题(`option missingRefs` / ajv `defaultMeta`)崩溃,与本次 JSX 样式改动无关;`npm run build` 已替代验证编译正确性。

## 6. 变更记录

| 文件 | 改动 |
| --- | --- |
| `frontend/src/layouts/dashboard/side-nav.js` | footer 由绝对定位改为正常流列底元素(去 absolute/zIndex 9999);nav 区加 `minHeight:0 + overflowY:auto` 实现矮屏内部滚动;保留 "Powered by" 文本与 `alt="OctoLink logo"` 的 `/assets/logo.png` |
| `frontend/src/sections/overview/overview-tasks-progress.js` | 文本 `Stack` 加 `minWidth:0`;`Avatar` 加 `flexShrink:0` |
| `frontend/src/sections/overview/overview-total-customers.js` | 同上 |
| `frontend/src/pages/index.js` | 四张概览卡 `lg={3}` → `lg={6} xl={3}` |
| `frontend/e2e/layout-responsive.spec.js` | 新增 1280px/1920px 布局回归用例(3 条) |

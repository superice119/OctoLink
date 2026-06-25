# WS-27 — Web UI Logo 未换标修复（线上仍显示 Oktopus）

## 1. 背景与目标

WS-7（S3 换标）虽已标记 done，但老板在线上 `39.97.250.156` Web UI 截图中发现左上角与左下角的 Logo 仍是上游 **OKTOPUS** 品牌字标,未换成 OctoLink。本任务负责定位根因并把全站 Logo / favicon 资产统一为 OctoLink 品牌。

**目标**:线上 Web UI 的左上 / 左下 Logo、favicon、产品名、版权信息全部为 OctoLink,无任何 Oktopus 残留。

## 2. 设计与选型

### 2.1 线上运行层核查结论(根因定位)

通过 SSH 到 `39.97.250.156` 执行 `docker ps` 核查,**线上前端运行的是 `octolink/frontend:local` 镜像**(非上游 oktopus 前端)。即 WS-7 的品牌**文本**换标(产品名 / 版权 / 主题色)已经上线生效。

> 即问题属于「较小范围」一类:整套换标已部署,**仅 Logo 图片资产从未被替换**,而非「整个换标从未上线」。

进一步核查两处 Logo 来源(均经 nginx 由 `http://39.97.250.156/` 对外提供):

| 引用路径 | 来源 | 用途 | 现状 |
|---|---|---|---|
| `/assets/logo.png` | 打包进 `octolink/frontend:local` 镜像的 `frontend/public/assets/logo.png` | side-nav 左下、登录页右下角小标 | Oktopus 字标 358×120 |
| `/images/logo.png` | bind-mount `./images:/app/images`(compose 第 277 行),源文件 `deploy/compose/images/logo.png` | side-nav 左上、登录页中央大图 | Oktopus 字标 358×120 |

两个文件 `md5` 完全相同(`6309c58d…`),都是上游 Oktopus 原始 358×120 字标。`git log` 显示 `frontend/public/assets/logo.png` 最后一次改动来自上游导入提交 `94f3186`,从未被换标提交触及。

### 2.2 资产生成

OctoLink 正式 Logo 设计稿在父任务早期已产出(`octolink-logo.svg` 横版锁版,viewBox 420×128;`octolink-icon.svg` 方形图标)。本任务由 SVG 渲染为位图:

- **横版 Logo**:`octolink-logo.svg` → `840×256` PNG(2× 高分屏清晰),用于两处 `logo.png`。
- **favicon / 图标**:`octolink-icon.svg` → 16/32/180 PNG + 多尺寸 `.ico`。

沿用既有 MUI + Emotion 主题体系,品牌主色 `#2563EB` / 青 `#0EA5A4` 与 `src/theme/brand.js`、`manifest.json` 一致。`src/components/logo.js`(内联通用八爪鱼 SVG,非品牌字标,无文本)无需更改。

## 3. 接口/资产协议

替换 / 新增的静态资产(均为 OctoLink 品牌):

| 文件 | 尺寸 | 说明 |
|---|---|---|
| `frontend/public/assets/logo.png` | 840×256 | 镜像内横版 Logo(`/assets/logo.png`) |
| `deploy/compose/images/logo.png` | 840×256 | bind-mount 横版 Logo(`/images/logo.png`) |
| `frontend/public/favicon.ico` | 16/32/48/64 多尺寸 | 浏览器标签图标 |
| `frontend/public/favicon-32x32.png` | 32×32 | 同上 |
| `frontend/public/favicon-16x16.png` | 16×16 | 同上 |
| `frontend/public/apple-touch-icon.png` | 180×180 | `_document.js` 已引用,原缺失,本次补齐 |
| `frontend/public/assets/brand/octolink-logo.svg` | — | 品牌 SVG 源,便于后续再生成 |
| `frontend/public/assets/brand/octolink-icon.svg` | — | 图标 SVG 源 |

`_document.js`、`manifest.json`、`side-nav.js`、`auth/layout.js` 的引用路径均未改动(沿用原有路径,仅替换图片字节)。

## 4. 部署与使用(待 Archie 复核 + 老板审批)

服务器仅 2G 内存,**禁止在控制器上重型 build**。两处 Logo 部署路径不同:

- **`/images/logo.png`(bind-mount,无需 build)**:直接用本 PR 的 `deploy/compose/images/logo.png` 覆盖服务器 `<compose 目录>/images/logo.png` 即可,nginx/容器即时生效,**无需重启、无需构建**。这是最低风险、可立即上线的修复。
- **`/assets/logo.png`(打包进前端镜像)**:有两种方案,二选一,由 Archie/老板定夺:
  - **方案 A(免构建热替换,推荐先行)**:`docker cp` 新 `logo.png` 到运行中的 `frontend` 容器内 Next.js 静态资源目录,即时生效。优点:零构建、零停机;缺点:容器重建后失效,需配合方案 B 固化。
  - **方案 B(重建镜像,固化)**:在**本地/CI 而非 2G 控制器**上 `docker build` 出新的 `octolink/frontend:local`,推送/导入服务器后 `docker compose up -d frontend` 滚动替换。作为长期固化手段。

建议:先执行 `/images` 覆盖 + `/assets` 方案 A 让线上即时纠正,再排期方案 B 在轻量环境重建镜像固化。所有上线动作经 Archie 复核 + 老板审批后进行。

## 5. 测试与验收

### 已完成(repo 侧)
- 渲染产物经目视核对:横版 Logo 为「八爪鱼图标 + OctoLink + 章鱼物联」蓝青渐变锁版;favicon 为八爪鱼图标。
- 文件类型/尺寸校验:`logo.png` 840×256 RGBA;`favicon.ico` 多尺寸合法。
- `grep -ri oktopus frontend/src frontend/public` 仍无文本命中;本次清除最后的 Logo 位图残留。

### 上线后待验收(HTTP 证据)
```
curl -s http://39.97.250.156/assets/logo.png  | file -   # 应为 840×256 OctoLink
curl -s http://39.97.250.156/images/logo.png  | file -   # 应为 840×256 OctoLink
```
- [ ] side-nav 左上 / 左下 Logo 为 OctoLink
- [ ] 登录页中央大图 + 右下角小标为 OctoLink
- [ ] 浏览器标签 favicon 为 OctoLink 八爪鱼
- [ ] 产品名 / 版权信息为 OctoLink(WS-7 已生效,复核无回归)

## 6. 变更记录

- 2026-06-25:核查线上前端镜像(`octolink/frontend:local`,换标文本已上线);定位 Logo 资产从未替换;由品牌 SVG 渲染并替换 `/assets/logo.png`、`/images/logo.png`、favicon 全套,补齐 `apple-touch-icon.png`,归档品牌 SVG 源。文档 + PR,部署待 Archie 复核 + 老板审批。

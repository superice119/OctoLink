# WS-7 S3 Web UI / Logo / 产品名 / 版权 深度换标 (Rebranding)

## 1. 背景与目标

**Issue**: WS-7 (父 Issue: WS-4/WS-5 Rebranding 改造点清单)

将 OctoLink fork 自 OktopUSP/oktopus 的 Web UI 完整去除上游 Oktopus 品牌，替换为 OctoLink 自有品牌。目标：

- 所有用户可见的 UI 文字、页面标题、footer 版权链接 去 Oktopus 化
- 配色方案由橙棕调（`#c05521`）换为 OctoLink 品牌蓝（`#2563EB`），sidebar 渐变由青色 → 蓝色（`#0EA5A4` → `#2563EB`）
- 品牌配置抽为环境变量，便于后续多租户/白标维护

---

## 2. 设计思路与方案选型

### 2.1 方案选型

| 方案 | 说明 | 决策 |
|---|---|---|
| 全量字符串替换 | 直接 grep 替换所有文件中的 "Oktopus" | **采用**，范围可控，风险低 |
| 主题 JSON + 环境变量 | 品牌色、名称、URL 集中到配置文件 | **采用**，便于后续维护 |
| 重新设计 Logo 组件 | 替换 `<Logo>` SVG 组件 | 暂缓，等正式品牌 SVG 资产到位（WS-4 附件） |

### 2.2 关键设计

1. **品牌常量文件** `frontend/src/theme/brand.js`：导出 `brand` 对象，所有字段优先读取 `NEXT_PUBLIC_BRAND_*` 环境变量，保留硬编码默认值作为 fallback。

2. **配色环境变量**：`create-palette.js` 中六个颜色值全部读取 `NEXT_PUBLIC_COLOR_*` 环境变量，无需修改代码即可换肤。

3. **docker-compose 镜像名** (`oktopusp/*`)：属于上游 Docker Hub 镜像引用，不在 S3 UI 换标范围内，留待 CI/CD 阶段（后续 WS）自建镜像时处理。

---

## 3. 接口 / 协议说明

无新增 API。新增环境变量清单（详见 `docs/branding/brand.json`）：

| 变量名 | 默认值 | 说明 |
|---|---|---|
| `NEXT_PUBLIC_BRAND_NAME` | `OctoLink` | 产品名 |
| `NEXT_PUBLIC_BRAND_TAGLINE` | `物联控制器` | 副标题 |
| `NEXT_PUBLIC_BRAND_WEBSITE` | GitHub URL | 官网/仓库链接 |
| `NEXT_PUBLIC_BRAND_DOCS_URL` | GitHub Wiki | 文档链接 |
| `NEXT_PUBLIC_COLOR_PRIMARY` | `#2563EB` | 主色（按钮、高亮） |
| `NEXT_PUBLIC_COLOR_SIDEBAR_START` | `#0D3D3D` | Sidebar 渐变起点（青） |
| `NEXT_PUBLIC_COLOR_SIDEBAR_END` | `#1D4ED8` | Sidebar 渐变终点（蓝） |
| `NEXT_PUBLIC_COLOR_ACCENT` | `#0EA5A4` | 连接状态指示色（MTP） |
| `NEXT_PUBLIC_COLOR_TABLE_HEADER` | `#1E3A5F` | 表头背景色 |
| `NEXT_PUBLIC_COLOR_TEXT` | `#1E293B` | 文字主色 |

---

## 4. 部署 / 使用步骤

### 4.1 本地开发

```bash
cd frontend
cp .env.example .env.local
# .env.local 已更新为 localhost 默认值，无需改动即可启动
npm install
npm run dev
```

### 4.2 换肤（多租户 / 白标）

在 `.env.local` 或容器环境变量中覆盖 `NEXT_PUBLIC_*` 变量即可，无需重新构建源码（Next.js 静态替换在 build 时注入）：

```env
NEXT_PUBLIC_BRAND_NAME=MyBrand
NEXT_PUBLIC_COLOR_PRIMARY=#FF5733
```

然后 `npm run build && npm start`。

### 4.3 Logo 资产替换（待 WS-4 完成后执行）

- 将正式品牌 SVG 放入 `frontend/public/assets/logo.png`（或修改 `<Logo>` 组件引用路径）
- 替换 `frontend/public/favicon*.png` 和 `favicon.ico`

---

## 5. 修改文件清单

### 5.1 前端 UI（`frontend/src/`）

| 文件 | 改动内容 |
|---|---|
| `pages/_app.js` | 全局 title: `Oktopus \| Controller` → `OctoLink \| 物联控制器` |
| `pages/auth/login.js` | title、社区链接（GitHub/Slack）、demo host 检测 |
| `pages/404.js` | title |
| `pages/403.js` | title |
| `pages/settings.js` | title |
| `pages/credentials.js` | title |
| `pages/index.js` | title |
| `pages/devices.js` | title |
| `pages/access-control/users.js` | title |
| `pages/devices/cwmp/[...id].js` | title |
| `pages/devices/usp/[...id].js` | title |
| `pages/chat/room.js` | console.log 品牌文字 |
| `layouts/dashboard/side-nav.js` | footer 链接 URL、alt 文字 |
| `layouts/auth/layout.js` | footer 链接 URL、alt 文字 |
| `layouts/dashboard/config.js` | Docs 外链 URL |
| `layouts/dashboard/top-nav.js` | 升级链接 URL |
| `sections/account/account-profile-details.js` | demo email |
| `theme/create-palette.js` | 六项配色换为 OctoLink 品牌色，支持环境变量覆盖 |
| `.env.example` | demo 域名更新 |

### 5.2 新增文件

| 文件 | 说明 |
|---|---|
| `frontend/src/theme/brand.js` | 品牌常量（支持 env 覆盖） |
| `docs/branding/brand.json` | 品牌配置 + 环境变量参考手册 |
| `docs/tasks/WS-7-rebranding.md` | 本文档 |

### 5.3 部署配置

| 文件 | 改动内容 |
|---|---|
| `deploy/compose/nginx.conf` | `/companylink` 重定向 → OctoLink GitHub |
| `frontend/public/manifest.json` | PWA name/short_name、theme_color |

---

## 6. 测试与验收记录

### 6.1 自检项

- [x] 所有页面 `<title>` 不含 "Oktopus" 字样
- [x] Sidebar footer 链接指向 OctoLink GitHub
- [x] Login 页社区链接更新
- [x] `manifest.json` 名称已更新
- [x] nginx `/companylink` 重定向更新
- [x] 主题色方案环境变量化
- [x] `grep -rn "oktopus" frontend/src/` 返回空（OktopUSP 协议名除外）

### 6.2 待 QA_Sherlock 验收

- [ ] UI 截图回归：Sidebar 渐变色（青→蓝）正常
- [ ] 所有页面标题浏览器 tab 显示 OctoLink
- [ ] Login 页 logo 区域、footer 链接点击正确
- [ ] PWA 安装时 app 名称为 "OctoLink 物联控制器"

---

## 7. 变更记录 (Changelog)

| 日期 | 改动摘要 |
|---|---|
| 2026-06-18 | S3 全量换标：UI 文字、配色、manifest、nginx 重定向；新增品牌配置文件 `brand.js` + `brand.json` |

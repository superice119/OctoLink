# WS-33 接入方式列 + 默认英文 收口（含 WS-31 双语前端打包部署）

## 1. 背景与目标

前端有三块**已开发但尚未上线**的改动需要在当前 `main` 上收口并一次性打包部署：

1. **WS-31 右侧内容双语**：PR #23 已合并进 `main`，但尚未构建/部署。
2. **接入方式列**（老板 06-27 第 2 点）：设备列表需展示「接入方式」（MQTT / WebSocket / STOMP / CWMP）。逻辑原型在旧 PR #22（commit `caa1e85`），但 #22 的 `devices.js` / locale 与已合并的 WS-31(#23) 冲突且过时，**不直接合 #22**，在当前 `main` 上重做。
3. **默认英文**（老板 06-26「默认英文，上」决策）：旧 PR #22 把 `DEFAULT_LANGUAGE='en'` 但未合并，当前 `main` 仍是 `'zh'`。若直接从 `main` 重建镜像会把默认语言退回中文、推翻老板决策，因此必须先把默认英文落到 `main`。

目标：在当前 `main` 基础上出一份干净不冲突的 PR，补齐「接入方式列 + 默认英文」，并基于合并后的 `main` 离线构建前端镜像、临时端口验证、待老板审批后替换正式容器。

## 2. 设计与选型

- 沿用现有 MUI + react-i18next（zh/en catalog）体系，不引入新栈。
- **接入方式列**：参考 #22 `caa1e85` 的 `accessMethod(order)` 逻辑——后端已在每个设备上序列化各 MTP 的连接状态（`order.Mqtt / Websockets / Stomp / Cwmp`，0 离线 / 1 关联中 / 2 在线），`>= 1` 即视为该协议在用，拼接协议名展示。**协议名（MQTT/WebSocket/STOMP/CWMP）不翻译**；无任何在用协议时显示 `—`。后端无需改动。
- **列显隐**：`access` 列纳入 `columns` 配置，默认 `true`，并入列设置菜单（可勾选显隐）。对已有用户的 `localStorage.columns` 做 `{ access: true, ...saved }` 合并——默认开启，但用户显式关掉后仍会被尊重。
- **列头文案双语**：新增 `devices.page.access` 键（en `Access Method` / zh `接入方式`），与现有 `devices.page.*` 命名空间一致。
- **默认英文**：`DEFAULT_LANGUAGE = 'en'`（`fallbackLng` 随之），并从语言探测顺序中**移除 `navigator`**（`order: ['localStorage','cookie']`），确保首次进站无论浏览器语言都确定为英文；用户经顶部切换器保存的选择（localStorage/cookie）仍然优先且持久化。同步 `_document.js` 的 `<html lang="en">`。

## 3. 接口 / 数据协议

- 不新增、不修改任何后端接口。
- 设备列表沿用现有 `GET /api/device` 响应；接入方式列只消费响应中已有的 `Mqtt / Websockets / Stomp / Cwmp` 整型状态字段，不改变请求/响应结构。
- i18n 仅新增展示层文案键，业务判断常量保持原英文，不影响逻辑。

## 4. 部署 / 使用

部署纪律（见 WS-4 / WS-30 / WS-31）：

- **离线构建机 `39.105.150.244`**：`next build` → `docker build` → `docker save`。
- **严禁在 2G 控制器 `39.97.250.156` 上 `next build` / 构建镜像**（会 OOM 致全站宕机）。
- 镜像 `docker save` → 传控制器 → `docker load`。
- 先在控制器**临时端口**验证（语言切换器 / 双语内容 / 接入方式列 / 默认英文 / API 通），无误后再替换正式前端容器。
- **生产容器替换=改动线上，须老板审批后再 swap**。

## 5. 测试与验收（Definition of Done）

- 默认进站为英文；切换器切中文→每个菜单右侧内容整页中文，切英文整页英文（WS-31）。
- 设备列表出现「接入方式」列且取值正确（MQTT / WebSocket / STOMP / CWMP）；列可在列设置中显隐。
- zh/en 键集合一致（防 fallback 露英文）。本次新增 `devices.page.access` 后校验通过：`PARITY OK: 403 keys`。
- 旧 PR #22 前端部分由本 PR 取代后在 #22 标注；ws-adapter 后端部分已由 PR #24 拆出（WS-32）。
- 由 QA_Sherlock 回归复审。

## 6. 变更记录

| 文件 | 改动 |
|------|------|
| `frontend/src/pages/devices.js` | 新增 `accessMethod(order)`；`columns` 默认含 `access: true`；`getColumns` 对历史 localStorage 做 `{access:true, ...saved}` 合并；列设置菜单 + 表头 + 表体新增「接入方式」列（`t('devices.page.access')`） |
| `frontend/src/i18n/locales/en.json` | 新增 `devices.page.access = "Access Method"` |
| `frontend/src/i18n/locales/zh.json` | 新增 `devices.page.access = "接入方式"` |
| `frontend/src/i18n/index.js` | `DEFAULT_LANGUAGE 'zh' → 'en'`；探测顺序移除 `navigator`（`['localStorage','cookie']`） |
| `frontend/src/pages/_document.js` | `<html lang>` `zh → en` |

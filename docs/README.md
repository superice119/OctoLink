# OctoLink 文档规范(Documentation SOP)

> 铁律:**所有任务交付物必须文档齐全(Markdown),并提交到本仓库 `docs/` 下输出到 GitHub。** 文档未齐,不得推进至"待测试/验收/可发布"。

## 1. 目录结构
```
docs/
├── README.md                # 本规范(文档之文档)
├── 00-project-overview.md   # 产品范围与任务依赖树
├── 01-architecture.md       # 系统架构与关键决策(ADR 汇总)
├── branding/                # 品牌资产(logo svg、配色)
└── tasks/
    └── WS-<n>-<slug>.md      # 每个任务一份交付文档
```

## 2. 每个任务文档必含章节
1. **背景与目标**:对应 issue (WS-n)、要解决的问题。
2. **设计思路与方案选型**:架构/技术选型、权衡与取舍。
3. **接口 / 协议说明**:API、USP 参数路径、消息格式、字段表。
4. **部署 / 使用步骤**:环境变量、Docker/编译、运行与回滚。
5. **测试与验收记录**:用例、结果、QA 结论。
6. **变更记录 (Changelog)**:日期 + 改动摘要。

## 3. 提交流程
- 提交规范:Conventional Commits,文档用 `docs(<scope>): ...`(如 `docs(s5): USP Get/Set 接口说明`)。
- 通过 **Pull Request** 合入,禁止直接 push 主分支。
- 完成后在对应 Multica issue **回贴文档/PR 链接**,作为推进"待测试"的前置条件。

## 4. Definition of Done(文档维度)
- [ ] 任务文档置于 `docs/tasks/WS-<n>-*.md` 且六大章节齐全。
- [ ] 涉及接口/协议的,字段表完整、可被对端直接联调。
- [ ] 已通过 PR 合入,issue 内已贴链接。

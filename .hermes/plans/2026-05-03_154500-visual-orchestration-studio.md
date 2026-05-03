# 可视化编排工作室 (Visual Orchestration Studio) — 实施计划

**日期**: 2026-05-03
**状态**: 草稿
**范围**: Phase 3 核心功能

---

## 1. 目标

为 Korus Web Console 构建**表单驱动的 Agent 编排工作室**，让用户无需手写 YAML 即可完成 Agent 的创建和编辑。

**不是**画布式拖拽编辑器（那是 Phase B），而是**结构化表单 + 实时预览**。

## 2. 现状分析

### 后端差距

Manager API 的 `AgentRecord` 目前只存储 13 个基础字段：

```go
type AgentRecord struct {
    ID, TenantID, WorkspaceID, Slug, DisplayName, Description, Status string
    Pattern, RuntimeEngine, RunnerClass, ModelProvider, ModelName string
    LatestRevision string
}
```

而 CRD 的 `AgentSpec` 有完整的编排定义：
- `models` — 多模型配置（provider, model, baseURL, temperature, maxTokens...）
- `pattern` — 模式配置（type, version, modelRef, toolRefs, knowledgeRefs, maxIterations, routes...）
- `promptRefs` — 系统提示引用
- `knowledgeRefs` — 知识库绑定（name, ref, retrieval config）
- `toolRefs` — 工具引用列表
- `skillRefs` — Skill 绑定
- `subAgentRefs` — SubAgent 引用
- `mcpRefs` — MCP Server 引用
- `graph` — 自定义图定义（nodes + edges，用于 workflow 模式）
- `interfaces` — 输入/输出 schema
- `runtime` — 运行时配置（engine, runnerClass, mode...）
- `policyRef` — 策略引用
- `identity` — 身份信息（displayName, role, description）
- `observability` — 可观测性配置

**关键差距**: Manager 不存储完整 spec，编排工作室无法读写编排数据。

### 前端现状

- Agent 列表页 ✅
- Agent 详情页 ✅（只读展示基础字段）
- 编排编辑 ❌

## 3. 设计决策

### 3.1 表单驱动 vs 画布驱动

| 方案 | 优点 | 缺点 |
|------|------|------|
| **表单驱动**（选定） | 实现简单、表单验证天然、符合 CRD 结构化本质、无需额外依赖 | 不够"酷"、复杂图不易可视化 |
| 画布驱动 | 直观、适合复杂图 | 需要 react-flow 等库、交互复杂、v1 范围过大 |

**决策**: v1 采用表单驱动，但包含一个**只读图预览**组件展示编译后的图结构。画布编辑作为 Phase B。

### 3.2 数据流

```
Web Console  →  Manager API  →  Manager DB (source of truth)
                    ↓ (CRD sync)
               K8s Agent CRD  →  Controller  →  Worker
```

- Manager DB 是 Agent spec 的产品级存储
- CRD sync 层将 DB 状态推送到 K8s
- 编排工作室通过 Manager API 读写

### 3.3 模式 (Pattern) 中心化

6 种模式各有不同配置需求：

| 模式 | 关键配置 |
|------|---------|
| react | modelRef, toolRefs, knowledgeRefs, maxIterations, stopWhen |
| router | modelRef, routes[] (label + agentRef/modelRef + default) |
| reflection | modelRef, maxIterations |
| tool_calling | modelRef, toolRefs |
| plan_execute | plannerModelRef, executorModelRef, planSteps |
| workflow | graph.nodes[], graph.edges[] (自由定义) |

**策略**: 根据用户选择的 pattern 动态渲染不同的配置表单区域。

## 4. 实施计划

### Slice 1: 后端 — 扩展 AgentRecord 存储完整 spec

**目标**: Manager API 支持读写完整的 AgentSpec。

**改动**:

1. **`internal/manager/store.go`** — 扩展 `AgentRecord`：
   ```go
   type AgentRecord struct {
       // 现有字段保持不变
       ID, TenantID, WorkspaceID, Slug, DisplayName, Description, Status string
       Pattern, RuntimeEngine, RunnerClass, ModelProvider, ModelName string
       LatestRevision string

       // 新增：完整 spec 以 JSON blob 存储
       Spec *AgentSpecData `json:"spec,omitempty"`
   }

   // AgentSpecData 是 CRD AgentSpec 的 Manager 侧镜像
   type AgentSpecData struct {
       Runtime       RuntimeConfig           `json:"runtime,omitempty"`
       Models        map[string]ModelConfig  `json:"models,omitempty"`
       Identity      IdentityConfig          `json:"identity,omitempty"`
       Pattern       *PatternConfig          `json:"pattern,omitempty"`
       PromptRefs    PromptRefsConfig        `json:"promptRefs,omitempty"`
       KnowledgeRefs []KnowledgeBinding      `json:"knowledgeRefs,omitempty"`
       ToolRefs      []string                `json:"toolRefs,omitempty"`
       SkillRefs     []SkillBinding          `json:"skillRefs,omitempty"`
       SubAgentRefs  []SubAgentBinding       `json:"subAgentRefs,omitempty"`
       MCPRefs       []string                `json:"mcpRefs,omitempty"`
       PolicyRef     string                  `json:"policyRef,omitempty"`
       Interfaces    InterfaceConfig         `json:"interfaces,omitempty"`
       Graph         *GraphConfig            `json:"graph,omitempty"`
   }
   ```

2. **`internal/manager/store.go`** — SQL migration 添加 `spec JSONB` 列
3. **`internal/manager/server.go`** — Agent 响应中包含完整 spec
4. **`internal/manager/dev_stores.go`** — dev store 支持 spec 字段
5. **`internal/manager/syncer.go`** — CRD sync 使用 spec 字段构建完整 AgentSpec

**文件**:
- `internal/manager/store.go` (AgentRecord + AgentSpecData types, SQL migration, CRUD)
- `internal/manager/dev_stores.go` (dev store update)
- `internal/manager/server.go` (request/response types, handlers)
- `internal/manager/server_test.go` (test stubs)

### Slice 2: 后端 — Agent 资产查询 API

**目标**: 编排工作室需要知道当前租户/工作区下可用的 tools、knowledge、skills、sub-agents。

**改动**:

1. **`internal/manager/server.go`** — 新增查询端点：
   - `GET /api/v1/agents/{id}/available-assets` — 返回可用的 tools, knowledge, skills, sub-agents

   或者更简单的方式：复用已有的 list 端点，在前端按需查询。

**决策**: v1 复用现有 list 端点（`/api/v1/providers/`, `/api/v1/agents/`），不新增端点。
工具和知识库列表在前端通过已有 API 获取。

### Slice 3: 前端 — 编排工作室布局

**目标**: 在 Agent 详情页中添加"编辑"按钮，进入编排工作室视图。

**改动**:

1. **`web/src/pages/AgentStudioPage.tsx`** — 新页面，编排工作室主布局：
   - 左侧面板：模式选择 + 模式配置
   - 中间面板：模型配置、工具/知识/Skill 绑定、系统提示
   - 右侧面板：实时预览（编译后的图结构、输入/输出 schema）
   - 底部：保存 / 取消 / 发布按钮

2. **`web/src/routes/index.tsx`** — 新增路由：
   - `/tenants/:tenantId/agents/:agentId/studio` → AgentStudioPage

3. **`web/src/components/studio/`** — 新目录，编排工作室组件：
   - `PatternSelector.tsx` — 模式选择器（6 种模式卡片）
   - `PatternConfigForm.tsx` — 模式特定配置（动态渲染）
   - `ModelConfigForm.tsx` — 模型配置表单
   - `ToolBindingPanel.tsx` — 工具绑定面板
   - `KnowledgeBindingPanel.tsx` — 知识库绑定面板
   - `SkillBindingPanel.tsx` — Skill 绑定面板
   - `SubAgentBindingPanel.tsx` — SubAgent 绑定面板
   - `SystemPromptEditor.tsx` — 系统提示编辑器
   - `GraphPreview.tsx` — 只读图预览
   - `InterfacePreview.tsx` — 输入/输出 schema 预览
   - `StudioToolbar.tsx` — 保存/取消/发布工具栏

4. **`web/src/api/agents.ts`** — 新增 mutation hooks：
   - `useUpdateAgent()` — 更新 agent spec
   - `usePublishAgent()` — 发布 agent

5. **i18n keys** — 编排工作室相关翻译

### Slice 4: 前端 — 模式特定配置

**目标**: 根据选择的模式动态渲染配置表单。

**6 种模式的配置 UI**:

| 模式 | 配置 UI |
|------|---------|
| react | 模型选择下拉、maxIterations 滑块、stopWhen 文本框 |
| router | 路由表编辑器（动态行：label + agentRef/modelRef + default 复选框） |
| reflection | 模型选择下拉、maxIterations 滑块 |
| tool_calling | 模型选择下拉 |
| plan_execute | 规划器模型下拉、执行器模型下拉 |
| workflow | 节点编辑器（动态行：name + kind + refs）+ 边编辑器（from + to + when） |

### Slice 5: 前端 — 图预览

**目标**: 只读可视化展示编译后的 agent 图。

**方案**: 使用简单的 SVG/HTML 渲染，不引入 react-flow 等外部依赖。

**展示内容**:
- 节点（按 kind 着色：model=蓝, tool=绿, agent=紫, knowledge=橙）
- 边（带 when 条件标注）
- START/END 虚拟节点

### Slice 6: 集成测试 + 文档

**目标**: 端到端验证编排工作室。

**验证**:
1. 创建新 agent → 选择 react 模式 → 配置模型和工具 → 保存
2. 打开已有 agent → 切换到 router 模式 → 配置路由 → 保存
3. 验证保存后的 agent spec 可以被 controller 编译
4. `make ci` 通过
5. `tsc --noEmit` 通过

**文档更新**:
- README.md / README.zh-CN.md — 更新 Phase 3 进度
- web/README.md — 更新 Console 当前状态

## 5. 实施顺序

```
Slice 1 (后端 spec 扩展)     ← 基础，阻塞所有前端工作
    ↓
Slice 2 (资产查询)           ← 可选，v1 复用已有 API
    ↓
Slice 3 (工作室布局)         ← 前端骨架
    ↓
Slice 4 (模式配置表单)       ← 核心交互
    ↓
Slice 5 (图预览)             ← 增强体验
    ↓
Slice 6 (集成 + 文档)        ← 收尾
```

**估算**: Slice 1 最复杂（后端 schema 迁移 + API 扩展），Slice 3-4 工作量最大（前端组件多）。

## 6. 风险与权衡

| 风险 | 缓解 |
|------|------|
| JSONB spec 字段过大 | v1 直接存储完整 JSON，未来可拆分 |
| 模式切换时表单状态丢失 | 切换前确认对话框 |
| CRD sync 字段映射不完整 | 逐字段映射，先覆盖核心字段 |
| 图预览渲染复杂 | v1 用简单 SVG，不做交互式画布 |
| 前端组件过多 | 组合模式，每个面板独立 |

## 7. 开放问题

- [ ] Agent spec 的 Manager 存储方案：JSONB 列 vs 拆分表？（建议 JSONB，v1 简单）
- [ ] 编排工作室是否需要独立页面，还是嵌入 Agent 详情页的 tab？
- [ ] workflow 模式的节点/边编辑器复杂度是否需要降级为 YAML 编辑器？
- [ ] 保存时是否自动触发编译验证？

---

**下一步**: 确认方案后从 Slice 1 开始实施。

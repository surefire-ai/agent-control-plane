# Agent Control Plane

<p align="center">
  <a href="./README.md">English</a> | 中文
</p>

Agent Control Plane 是一个 Kubernetes 原生控制平面，用于声明、发布、运行、治理和评估 AI Agent。

源码仓库是 `github.com/surefire-ai/agent-control-plane`，Kubernetes API Group 使用
`windosx.com/v1alpha1`。

当前实现以 `config/samples/ehs` 中的 EHS 危害识别样例为起点。

## 项目定位

Agent Control Plane 应该被理解为一个 **面向企业级、多租户场景的 Agent 编排、评测与发布平台**，
底座建立在 Kubernetes 之上，而不是一个独立 SDK、一个薄薄的集群管理台，也不是只跑一次的工作流脚本。

- **operator 层** 负责持续 reconcile `Agent`、`AgentRun`、
  `PromptTemplate`、`ToolProvider`、`KnowledgeBase`、`AgentPolicy`
  等 CRD。
- **控制面 controller** 负责把期望状态编译成确定性的 runtime artifact，
  发布 status，执行平台契约，并把运行请求分发给执行 backend。
- **执行层** 位于 worker 和 runtime backend 后面，负责模型调用、tool
  调用、retrieval、checkpoint，以及未来的图执行。
- **console 层** 是一等产品入口，负责承载可视化的 Agent 编排、Evaluation
  查看、revision 对比和发布体验，而不是仅仅展示 Kubernetes 资源。

换句话说，这个项目并不是要把所有 Agent 逻辑都塞进 operator 进程里。
operator 负责声明、reconcile、调度和治理；worker 负责执行；console 负责团队每天
真正会使用的编排、评测、发布和协作体验。项目方向也明确是企业产品优先：多租户、
evaluation、provider 广度、治理和日常使用体验，和 runtime 能力本身同等重要。

## 项目能力

- `Agent` 声明 runtime、模型、prompt、知识库、工具、MCP Server、策略、图结构、接口、记忆和可观测性配置。
- `AgentRun` 记录一次不可变的执行请求及其执行状态。
- `Tenant`、`Workspace`、`PromptTemplate`、`KnowledgeBase`、`ToolProvider`、`Dataset`、`MCPServer`、`AgentPolicy` 和 `AgentEvaluation` 提供控制平面所需的配套资源。
- controller-manager 负责编译 `Agent` 资源、发布确定性的状态，并将 `AgentRun` 分发到 runtime backend。
- worker runtime backend 可以将每次运行分发为 Kubernetes Job。当前 worker 仍是占位实现，用于支撑 Eino runtime 成熟前的控制平面验证。

## 使用场景

Agent Control Plane 面向需要把 AI Agent 当作生产平台资源来运营的团队，而不是把
Agent 做成一次性脚本或隐藏在业务应用里的内部逻辑。

- **企业级 Agent 发布**：平台团队可以用 Kubernetes 原生 spec、status、revision 和
  RBAC 边界来定义、评审、发布和回滚 Agent。
- **可视化 Agent 编排**：应用团队可以直接在 Web Console 里组合 prompt、tool、
  knowledge、skill 和 graph flow，并把结果发布为统一的控制平面资源，而不是手写全部 YAML。
- **业务流程自动化**：产品团队可以暴露可重复运行的 Agent 工作流，例如文档审核、工单分诊、事件响应、巡检分析和知识辅助决策。
- **受监管、可审计的 AI 运行**：风控、合规和运营团队可以为每次 Agent 调用关联策略、trace reference、评估计划和不可变运行记录。
- **垂直领域 Agent 系统**：领域团队可以打包 EHS 危害识别、质量巡检、维修计划、客户支持、财务运营等知识密集型场景的专用 Agent。
- **多租户 Agent 平台**：组织可以将团队或租户映射到 namespace，执行 runtime 边界，并集中观测同一集群中的多个 Agent。
- **Agent Marketplace 与复用**：共享 prompts、tools、knowledge bases、MCP servers、policies 和 evaluations 可以沉淀为未来 Agent package 的可复用构件。

## 架构方向

- 本仓库正在朝 **Agent Control Plane Operator for Kubernetes** 的方向演进。
- 产品方向是 **Enterprise Multi-Tenant Agent Control Plane**，并将
  evaluation 作为一等能力，而不是附属功能。
- Go 承载 Kubernetes API 类型、CRD controller、compiler、admission check、runtime dispatch，以及未来的 gateway。
- Go 预计承载基于 Eino 的 runtime worker。
- 默认 runner 方向是 `runtime.engine: eino` 与 `runtime.runnerClass: adk`；LangGraph 保留为未来兼容 adapter。
- PostgreSQL、pgvector、S3 兼容存储和队列预计用于状态、检索、产物和异步执行。
- TypeScript 应承载未来的企业级控制台、可视化编排工作台、Marketplace UI 和生成式 SDK。

### 控制面边界

- `controller-manager` 是 operator 控制面，负责监听 CRD、reconcile 期望状态、编译 artifact，并管理 run lifecycle。
- `worker` 是执行侧 runtime 入口，负责消费 compiled artifact 和 run input，并执行模型调用以及未来的 tool/retrieval 工作。
- CRD 仍然是平台用户与 operator 之间的声明式 API 边界。

### 产品优先级

项目后续设计应明确围绕以下主线展开：

- **企业级与多租户默认前提**：tenancy、隔离、RBAC、quota、审计、workspace
  边界都应被视为产品基本要求，而不是后期补丁。
- **Evaluation 作为核心卖点**：评测集管理、revision 对比、阈值门禁、
  上线前评估、线上回归监控和多模型横向比较，应成为产品辨识度的一部分。
- **模型厂商支持作为平台能力矩阵**：模型支持应从“有一条 OpenAI-compatible
  路径”升级为 provider capability matrix，兼顾国际厂商与中国本土模型提供商。
- **UX-first Web Console**：Phase 3 的目标不是做一个薄管理台，而是做一个用户
  会高频使用的产品界面。它应该成为可视化 Agent 编排、Evaluation、发布和治理的主入口。

### Build / Buy / Integrate 原则

本仓库不应该尝试把整个 Agent 技术栈从头重写一遍。这个项目最有价值的地方，
是在保持 Kubernetes-native 控制面模型自主性的同时，务实地复用成熟底层能力。

- **应该自研的**：CRD API 设计、compiler 行为、确定性 artifact、run lifecycle、
  policy 挂载、Kubernetes runtime dispatch，以及带有项目取向的 `Skill` 和
  `Pattern` 模型。
- **应该借鉴的**：tenancy、多租户产品边界、package / marketplace 设计、
  SubAgent 边界，以及 A2A 兼容的资源建模方式。
- **应该集成而不是重写的**：模型 provider SDK、图执行引擎、向量检索底座、
  对象存储、队列、tracing、metrics，以及其他并非本项目核心差异的基础设施层。

一句话说，Agent Control Plane 应该掌握的是 **API、compiler 和 runtime
contract**，而不是把 contract 之下的所有执行和平台基础设施都重做一遍。

## 当前进度

状态日期：2026-04-20。

| 模块 | 状态 | 证据 |
| --- | --- | --- |
| YAML Agent Spec | 进行中 | `api/v1alpha1` 和 `config/crd/bases` 下已有 Go API 类型和 CRD；`config/samples/ehs` 下已有 EHS YAML 样例。 |
| 编译成 Eino | 部分完成 | `internal/compiler` 已能校验跨资源引用、产出面向 runtime 的 compiled artifact，并生成确定性 revision；尚未产出可执行的 Eino runner artifact。 |
| 发布 endpoint | Bootstrap | Agent controller 已发布 `Agent.status.endpoint.invoke`；invoke gateway 可接收 POST 请求并创建 `AgentRun` 资源。 |
| trace | 部分完成 | 已有 `AgentRun.status.traceRef`，mock/worker backend 会写入该字段；完整分布式 tracing 和 trace 存储尚未实现。 |
| version | 部分完成 | 已有 `Agent.status.compiledRevision` 与 `AgentRun.status.agentRevision`；语义化版本、发布通道和 revision history 仍待实现。 |
| runtime execution | Bootstrap | `mock` runtime 可确定性完成运行；`worker` runtime 可创建 Kubernetes Job、接收 compiled artifact，并返回占位输出。 |
| Policy | 仅有 Spec | 已有 `AgentPolicy` CRD 和 `Agent.spec.policyRef`；runtime dispatch 前的策略执行仍待实现。 |
| Evaluation | 部分完成 | `AgentEvaluation` 现在已有 typed dataset、baseline、evaluator、threshold gate 和 reporting 字段；`Dataset` CRD 可提供带 `expected` 的可复用评测样本，controller 会解析引用并创建一条或多条受管 `AgentRun`，再把聚合后的运行状态、基础规则型指标，以及 `risk_level_match`、`hazard_coverage` 这类早期 structured metric 回写到 status，同时给出 current/baseline 的对比分数差值；丰富结果上报仍待实现。 |

## 里程碑

### Phase 1：Kubernetes-Native MVP

目标：让一个通过 Kubernetes 声明的 Agent 能够完成编译、发布状态、通过
Kubernetes Job 运行，并端到端记录 output、trace reference 和 revision identity。

| 里程碑 | 当前状态 | 下一步 |
| --- | --- | --- |
| YAML Agent Spec | 已有初始 CRD 和 EHS YAML 样例。 | 强化 schema 校验、默认值、必填字段和 admission check。 |
| Agent compiler | 已有静态引用 compiler，可写入 `Agent.status.compiledArtifact`、基于 artifact 生成 revision，并将 artifact 传递给 worker。 | 逐步演进为兼容 Eino 的 runner artifact。 |
| AgentRun lifecycle | 已实现 `Pending`、`Running`、`Succeeded` 和 `Failed` 状态流转。 | 增加取消、超时、重试和幂等语义。 |
| Kubernetes Job runtime | `worker` backend 已能创建 Job，并在完成后更新 `AgentRun` 状态。 | 持久化更丰富的 worker output，并暴露 Job/Pod 失败详情。 |
| Invoke gateway | `Agent.status.endpoint.invoke` 已发布调用路径，gateway 可通过 POST 请求创建 `AgentRun` 资源。 | 增加认证、鉴权、限流和幂等控制。 |
| Packaging and deployment | 已有 Dockerfile、RBAC、`config/default` 部署清单、CI、GHCR 镜像发布 workflow、release tag 说明和 Helm chart skeleton。 | v0.1.0 前将 chart 从 dev/E2E 安装路径提升为正式安装 artifact。 |

Phase 1 退出标准：

- 应用 EHS 样例资源后可以得到 Ready 状态的 `Agent`。
- 通过 gateway 调用 Agent 后可以创建 `AgentRun`。
- 运行通过 Kubernetes Job runtime backend 执行。
- 运行结果记录 output、trace reference 和准确的 agent revision。
- controller-manager 和 worker 镜像可构建、可部署、可发布。

详细 release checklist 见 `docs/releases/v0.1.0-readiness.md`。

Release notes 见 `docs/releases/v0.1.0.md`。

当前阶段已知限制：

- runtime execution 仍是结构化占位实现，还不是真实 Eino 执行。
- gateway 认证、鉴权、限流和幂等尚未实现。
- AgentRun 取消、超时、重试和幂等语义尚未实现。
- durable run artifacts 和 trace storage 尚未实现。
- Helm chart 仍是开发和 E2E 安装路径。

### Phase 2：真实 Agent Runtime

目标：用真正基于 Eino 的 runtime 替换占位 worker，同时保持 Kubernetes
原生控制平面契约不变。

Phase 2 runtime 设计见 `docs/phase2/eino-runtime-design.md`。
Agent pattern、SubAgent 和 A2A TODO 见
`docs/phase2/agent-patterns-and-a2a-todo.md`。
Phase 3 console 规划见
`docs/phase3/console-information-architecture.md`。
Tenancy 与 workspace 设计说明见
`docs/phase3/tenancy-workspace-model.md`。

| 里程碑 | 当前状态 | 下一步 |
| --- | --- | --- |
| Eino compile artifact | 已有静态引用 compiler、typed compiled artifact decoder 和 v1 runner artifact 输出。 | 继续把 prompt/tool/knowledge 内容解析进 runner artifact。 |
| Eino runtime worker | Go placeholder worker 已能校验注入的运行上下文和 compiled artifact 元数据。 | 使用 Eino 执行已编译 artifact，并返回结构化结果。 |
| Model credentials | 进行中。 | Sample Agent 已可通过同 namespace 的 Kubernetes Secret 引用模型凭据，worker Job 会注入密钥但不会把明文写入 status 或 artifacts。 |
| Tenancy 与 workspace 模型 | Early foundation | 已加入 `Tenant` 与 `Workspace` CRD skeleton，作为企业级作用域的第一层控制面接口；轻量 controller 已能解析 `tenantRef`、发布 workspace console scope，并统计 tenant 下的 workspace 数量；`Agent` / `AgentEvaluation` 现在也可显式声明 `workspaceRef`，并由 controller 执行基础校验。 | 继续把这层模型扩展到 runtime 隔离、RBAC、quota、workspace 绑定与未来 UI 语义中。 |
| Model provider strategy | 已有早期基础。 | compiler 现在会按 provider catalog 校验 `ModelSpec.provider`，并把 provider family 元数据写入 compiled artifact。OpenAI-compatible 一族目前已可统一覆盖 OpenAI、Azure OpenAI、DeepSeek、Qwen、Moonshot、Doubao、GLM、Baichuan、MiniMax、SiliconFlow。 | 继续扩展 capability matrix，并在需要时增加 provider-specific runtime adapter，同时为未来 UI 和 policy 暴露这份 catalog。 |
| Runtime contract | `AgentRun` 已携带 input、output、trace reference 和 revision。 | 定义 artifacts、logs、errors、取消和重试行为。 |
| Policy checks | 已有 `AgentPolicy` CRD 和 `Agent.spec.policyRef`。 | 在 dispatch 前执行模型/工具预算、guardrails 和审批门禁。 |
| Agent patterns | 部分完成。 | 已支持 `spec.pattern`，compiler 会保留 pattern 元数据，并在 `spec.graph` 为空时把 `react` 展开成会消费 Agent 已选 tools 与 knowledge 的 runner graph；更多 runtime 语义仍待实现。 |
| Durable run records | 当前状态存储在 `AgentRun` 上。 | 增加持久化 trace、artifact 和 result storage。 |
| Evaluation | `AgentEvaluation` 已具备 typed dataset、baseline、evaluator、threshold gate 和 reporting 字段；`Dataset` CRD 可提供带 `expected` 的可复用评测样本，controller 会解析 readiness、分别为 current/baseline 创建受管 `AgentRun`，并把 baseline revision、聚合后的 run state、基础规则型指标、早期 structured metric、gate 结果和 comparison delta 写入 status。 | 在此基础上继续扩展 richer result reporting、revision 对比和发布门禁行为。 |

Phase 2 退出标准：

- EHS AgentRun 通过真实 Eino worker 执行。
- Policy 可以在不安全运行开始前阻断或要求审批。
- worker Pod 消失后仍可查看 run artifacts 和 traces。
- Evaluation 资源可以针对某个 agent revision 执行并发布结果。

### Phase 3：产品界面与治理

目标：让平台不仅能被集群 operator 使用，也能被团队直接使用。

| 里程碑 | 当前状态 | 下一步 |
| --- | --- | --- |
| UX-first Web Console | 本仓库尚未开始。 | 构建围绕 tenant/workspace 导航、可视化 Agent 编排、Agent 构建与发布、Run 调试、Evaluation 对比、provider 管理、协作与发布体验的控制台。 |
| Marketplace | 尚未开始。 | 定义可复用 agents/tools 的包元数据、发布流程、信任信号和安装流程。 |
| SubAgent composition | 尚未开始。 | 增加一等公民 `subAgentRefs`、graph `kind: agent`、revision pinning 和父子 trace 关联。 |
| Tenant 与 workspace 体验 | 尚未开始。 | 增加租户模型、workspace 映射、RBAC 边界、quota、审计轨迹和用户可感知的隔离体验。 |
| Evaluation-first workflows | 尚未开始。 | 将评测集管理、revision 对比、阈值门禁、上线检查和回归视图纳入产品主界面。 |
| Provider management UX | 尚未开始。 | 提供 provider 选择、能力差异、credential 引用和模型切换的用户工作流。 |
| Governance workflows | 已有 Policy CRD。 | 增加 review、approval、human-in-the-loop 和 exception 工作流。 |

Phase 3 退出标准：

- 用户可以在 UI 中可视化编排、发布、查看、调用、评测和调试 agents。
- Marketplace package 可以被列出、安装、版本化和审查。
- API、runtime、storage 和 observability 中的租户隔离都有明确边界。
- 治理工作流可审计、可执行。

### Phase 4：分布式 Agent Fabric

目标：从单 Agent 执行扩展到多 runtime、多 Agent 的协作网络。

| 里程碑 | 当前状态 | 下一步 |
| --- | --- | --- |
| Multi-runtime | runtime interface 已支持在 `mock` 和 `worker` backend 之间选择。 | 增加 Eino、LangGraph 兼容层和远程 runtime adapter。 |
| Agent autoscaling | 尚未开始。 | 增加基于队列深度、延迟和成本的 runtime worker 扩缩容信号。 |
| Agent mesh | 尚未开始。 | 定义 Agent 间发现、调用、策略传播、身份、trace 关联和 A2A 协议互操作。 |

Phase 4 退出标准：

- 多个 runtime backend 可以运行兼容的 agent revision。
- Agent 可以基于需求和策略限制自动扩缩容。
- Agent 到 Agent 的调用能保留 identity、policy、version 和 trace context。
- A2A-compatible endpoint 可以暴露 Agent Card，并将 task、message 和 artifact 映射到 AgentRun 记录。

## 本地开发

运行 Go 测试套件：

```bash
go test ./...
```

生成 deepcopy 代码：

```bash
make generate
```

生成 CRD manifests：

```bash
make manifests
```

本地运行 controller manager：

```bash
make run
```

构建 controller-manager 和 worker 二进制：

```bash
make build
```

构建容器镜像：

```bash
make docker-build
```

CI 和镜像发布：

- `.github/workflows/ci.yml` 会在 pull request 和 `main` 上运行格式检查、测试、二进制构建和 Docker 镜像构建。
- `.github/workflows/publish-images.yml` 会在 `main`、`v*` tag 和手动触发时发布 controller-manager 与 worker 镜像到 GHCR。

Release tag：

- 使用 `v0.1.0` 这样的语义化版本 tag。
- 推送 `v*` tag 时会发布两个镜像，并同时生成同名版本 tag 和 `sha-*` 可追踪 tag。
- 准备 release branch 或 release archive 时，应将 Kubernetes manifests 固定到对应 release 镜像 tag。

将 CRDs、RBAC 和 controller-manager 部署到当前 Kubernetes context：

```bash
make deploy
```

使用开发版 Helm chart 安装：

```bash
helm upgrade --install agent-control-plane charts/agent-control-plane \
  --namespace agent-control-plane-system \
  --create-namespace
```

本地检查 chart：

```bash
make helm-lint
make helm-template
```

本地镜像测试时，可以覆盖 controller-manager 和 worker 镜像 tag：

```bash
helm upgrade --install agent-control-plane charts/agent-control-plane \
  --namespace agent-control-plane-system \
  --create-namespace \
  --set controllerManager.image.tag=latest \
  --set controllerManager.worker.image.tag=latest
```

移除已部署的控制平面：

```bash
make undeploy
```

用于本地 OrbStack 验证时，可以构建本地 controller 和 worker 镜像：

```bash
make docker-build-controller-local
make docker-build-worker-local
```

如果你的本地环境需要自定义的 kubectl 包装命令或 context helper，也可以
通过 `KUBECTL` 传入，例如 `make KUBECTL="kubectl"`。

## EHS 模型执行验证

EHS 样例现在已经可以验证 Phase 2 的第一条文本执行路径，并对接
OpenAI-compatible endpoint。

1. 应用 EHS 样例资源：

```bash
kubectl create namespace ehs --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -k config/samples/ehs
```

2. 基于示例清单创建模型凭据 Secret：

```bash
cp config/samples/ehs/openai-credentials.example.yaml /tmp/openai-credentials.yaml
# 编辑 /tmp/openai-credentials.yaml，将 REPLACE_WITH_REAL_API_KEY 替换为真实值
kubectl apply -f /tmp/openai-credentials.yaml
```

在示例 Secret 模板存在的情况下，不要再用 `kubectl apply -f config/samples/ehs`；
请改用 `-k`，这样默认样例安装不会把占位凭据清单一起 apply 进去。

3. 以 `--runtime-backend=worker` 运行 controller-manager，然后调用样例
   Agent 或直接 apply 样例 `AgentRun`。

4. 查看结构化输出：

```bash
kubectl -n ehs get agentrun ehs-hazard-run-20260416-0001 -o jsonpath='{.status.output}'
```

当前行为说明：

- 当声明多个模型槽位时，worker 当前优先选择 `planner` 模型。
- 第一条文本执行路径要求目标 endpoint 兼容 OpenAI `/chat/completions`
  协议，并返回满足 `spec.interfaces.output.schema` 的结构化 JSON。
- `status.output.result` 保留 worker 原始 payload，而 `summary`、
  `overallRiskLevel` 等顶层字段会提升到 `AgentRun.status.output`，
  便于直接消费。

如果只是想在本地快速验证控制面到 worker 的完整闭环，而不依赖真实
OpenAI 凭据，可以直接使用 OrbStack smoke overlay：

```bash
make k8s-smoke-ehs
```

这个目标会：

- 确保 `ehs` namespace 存在；
- apply `config/samples/ehs-orbstack-smoke`；
- 注入一个 dummy `openai-credentials` Secret；
- 部署 `mock-openai` 服务，并把 sample Agent 的 `baseURL` 改写到它；
- 重建固定样例 `AgentRun`；
- 打印最终的 `AgentRun.status.output`。

对应 overlay 位于 `config/samples/ehs-orbstack-smoke`。

## Runtime Backends

controller manager 接受 `--runtime-backend` 参数。

- `mock`：默认 backend。它会确定性地完成 `AgentRun` 对象，用于控制平面验证。
- `worker`：在 `AgentRun` 所在 namespace 中创建 Kubernetes Job。它通过 `--worker-job-image` 和 `--worker-job-command` 指向 worker 镜像和命令。Job 会从 `Agent.status.compiledArtifact` 接收 `AGENT_COMPILED_ARTIFACT`，校验后在 worker 结果中输出 artifact 摘要。Job 完成后，controller 会读取 worker Pod 日志、解析结构化 worker 结果，并将结果摘要写回 `AgentRun.status.output`。

本仓库包含两个镜像入口：

- `cmd/controller-manager`：协调控制平面资源。
- `cmd/worker`：校验注入的运行环境和 compiled artifact 元数据，并输出结构化占位结果。

Worker result contract v0：

规范 Go 类型和解析器位于 `internal/contract`。

成功：

```json
{
  "status": "succeeded",
  "message": "agent control plane worker placeholder completed",
  "compiledArtifact": {
    "apiVersion": "windosx.com/v1alpha1",
    "kind": "AgentCompiledArtifact",
    "runtimeEngine": "eino",
    "runnerClass": "adk",
    "policyRef": "ehs-default-safety-policy"
  }
}
```

失败：

```json
{
  "status": "failed",
  "reason": "WorkerFailed",
  "message": "AGENT_COMPILED_ARTIFACT kind is required"
}
```

发生结构化失败时，controller 会将 `AgentRun` 标记为 `Failed`，并在 status 中保留 worker summary 和 trace reference。

当前 placeholder worker 只接受默认 Eino runtime identity：`runtime.engine=eino`
和 `runtime.runnerClass=adk`。缺省值会按默认值处理；显式填写不支持的值会让本次运行以结构化 worker failure 失败。

worker 内部已经通过 runner 边界分发执行。第一版实现是
`EinoADKPlaceholderRunner`，它会保持当前占位行为，同时为后续接入真实 Eino
runner 留出明确的集成点。

## Invoke Gateway

controller-manager 会在 `--gateway-bind-address` 上启动 invoke gateway，默认地址为
`:8082`。它接受：

```text
POST /apis/windosx.com/v1alpha1/namespaces/{namespace}/agents/{agent}:invoke
```

请求体：

```json
{
  "input": {
    "task": "identify_hazard",
    "payload": {
      "text": "inspection text"
    }
  },
  "execution": {
    "mode": "sync"
  }
}
```

本地部署控制平面后，可以 port-forward gateway service 并调用 EHS 样例 Agent：

```bash
kubectl -n agent-control-plane-system port-forward svc/agent-control-plane-gateway 8082:8082
curl -sS -X POST http://127.0.0.1:8082/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-hazard-identification-agent:invoke \
  -H 'Content-Type: application/json' \
  -d '{"input":{"task":"identify_hazard","payload":{"text":"巡检发现配电箱门打开，现场地面有积水。"}},"execution":{"mode":"sync"}}'
```

gateway 会返回已接受的 `AgentRun` 名称，随后 `AgentRun` controller 会通过当前配置的
runtime backend 分发执行。

## 仓库结构

```text
api/v1alpha1/                 Kubernetes API types
cmd/controller-manager/        controller-manager entrypoint
cmd/worker/                    worker entrypoint
config/crd/                    generated CRD manifests
config/default/                installable Kustomize entrypoint
config/manager/                controller-manager and gateway service manifests
config/samples/ehs/            sample custom resources
internal/compiler/             Agent compiler and reference validation
internal/controller/           Agent and AgentRun reconcilers
internal/gateway/              invoke gateway
internal/runtime/              runtime backend abstraction and implementations
internal/worker/               placeholder worker implementation
```

## 开源协议

Agent Control Plane 使用 Apache License, Version 2.0 授权。详见 `LICENSE`。

本项目依赖的第三方 Go modules 使用各自的开源协议。当前直接 runtime 依赖为 Kubernetes 和 controller-runtime 相关模块，协议为 Apache-2.0。传递依赖包含 Apache-2.0、BSD-style、MIT-style 和 ISC 等宽松开源协议。

分发源码包、二进制或容器镜像前，请保留项目 `LICENSE`，保留 `NOTICE`，并按照 `THIRD_PARTY_NOTICES.md` 中的说明包含第三方依赖许可证声明。

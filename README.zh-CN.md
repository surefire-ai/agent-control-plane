# Agent Control Plane

<p align="center">
  <a href="./README.md">English</a> | 中文
</p>

Agent Control Plane 是一个 Kubernetes 原生控制平面，用于声明、发布、运行、治理和评估 AI Agent。

本项目由 [windosx](https://github.com/windosx) 维护。源码仓库是
`github.com/windosx/agent-control-plane`，Kubernetes API Group 使用
`windosx.com/v1alpha1`。

当前实现以 `examples/ehs` 和 `config/samples/ehs` 中的 EHS 危害识别样例为起点。

## 项目能力

- `Agent` 声明 runtime、模型、prompt、知识库、工具、MCP Server、策略、图结构、接口、记忆和可观测性配置。
- `AgentRun` 记录一次不可变的执行请求及其执行状态。
- `PromptTemplate`、`KnowledgeBase`、`ToolProvider`、`MCPServer`、`AgentPolicy` 和 `AgentEvaluation` 提供控制平面所需的配套资源。
- controller-manager 负责编译 `Agent` 资源、发布确定性的状态，并将 `AgentRun` 分发到 runtime backend。
- worker runtime backend 可以将每次运行分发为 Kubernetes Job。当前 worker 仍是占位实现，用于支撑 LangGraph runtime 成熟前的控制平面验证。

## 架构方向

- Go 承载 Kubernetes API 类型、CRD controller、compiler、admission check、runtime dispatch，以及未来的 gateway。
- Python 预计承载兼容 LangGraph 的 runtime worker。
- PostgreSQL、pgvector、S3 兼容存储和队列预计用于状态、检索、产物和异步执行。
- TypeScript 可用于未来的控制台、Marketplace UI 和生成式 SDK。

## 当前进度

状态日期：2026-04-17。

| 模块 | 状态 | 证据 |
| --- | --- | --- |
| YAML Agent Spec | 进行中 | `api/v1alpha1` 和 `config/crd/bases` 下已有 Go API 类型和 CRD；`examples/ehs` 与 `config/samples/ehs` 下已有 EHS YAML 样例。 |
| 编译成 LangGraph | 部分完成 | `internal/compiler` 已能校验跨资源引用并生成确定性 revision；尚未产出可执行的 LangGraph graph。 |
| 发布 endpoint | 部分完成 | Agent controller 已发布 `Agent.status.endpoint.invoke`，形式为稳定的 Kubernetes 风格 invoke 路径；真实 gateway/handler 尚未实现。 |
| trace | 部分完成 | 已有 `AgentRun.status.traceRef`，mock/worker backend 会写入该字段；完整分布式 tracing 和 trace 存储尚未实现。 |
| version | 部分完成 | 已有 `Agent.status.compiledRevision` 与 `AgentRun.status.agentRevision`；语义化版本、发布通道和 revision history 仍待实现。 |
| runtime execution | Bootstrap | `mock` runtime 可确定性完成运行；`worker` runtime 可创建 Kubernetes Job 并返回占位输出。 |
| Policy | 仅有 Spec | 已有 `AgentPolicy` CRD 和 `Agent.spec.policyRef`；runtime dispatch 前的策略执行仍待实现。 |
| Evaluation | 仅有 Spec | 已有 `AgentEvaluation` CRD；评估 reconciler 和结果上报仍待实现。 |

## 里程碑

### Phase 1：核心 Agent 控制平面

目标：让一个通过 Kubernetes 声明的 Agent 能够端到端完成编译、发布、运行、trace 和版本标识。

| 里程碑 | 当前状态 | 下一步 |
| --- | --- | --- |
| YAML Agent Spec | 已有初始 CRD 和 YAML 样例。 | 强化 schema 校验、默认值、必填字段和 admission check。 |
| 编译成 LangGraph | 已有静态引用 compiler。 | 产出兼容 LangGraph 的中间表示，并持久化或传递给 runtime worker。 |
| 发布 endpoint | 状态中已发布计划的 `:invoke` 路径。 | 增加 gateway/API handler，用于接收 invoke 请求并创建 `AgentRun` 资源。 |
| trace | `TraceRef` 已贯穿 `AgentRun` 状态。 | 集成 OpenTelemetry 或 runtime 原生 tracing，并统一存储 trace ID。 |
| version | 已有 compiled agent 的 revision hash。 | 增加 revision history、兼容性规则、发布标签和回滚语义。 |

Phase 1 退出标准：

- 应用 EHS 样例资源后可以得到 Ready 状态的 `Agent`。
- 调用已发布 endpoint 后可以创建 `AgentRun`。
- 运行由真实 LangGraph worker 执行，而不是 mock backend。
- 运行结果记录 output、trace reference 和准确的 agent revision。
- controller-manager 和 worker 镜像可构建、可部署。

### Phase 2：产品界面与治理

目标：让平台不仅能被集群 operator 使用，也能被团队直接使用。

| 里程碑 | 当前状态 | 下一步 |
| --- | --- | --- |
| UI | 本仓库尚未开始。 | 构建用于 agents、runs、traces、evaluations 和发布流程的控制台。 |
| Marketplace | 尚未开始。 | 定义可复用 agents/tools 的包元数据、发布流程、信任信号和安装流程。 |
| Policy | 已有 CRD 结构。 | 执行模型/工具预算、guardrails、审批门禁、安全边界和 runtime 约束。 |
| Tenant | 尚未开始。 | 增加租户模型、namespace 映射、RBAC 边界、quota 和审计轨迹。 |

Phase 2 退出标准：

- 用户可以在 UI 中发布、查看、调用和调试 agents。
- Marketplace package 可以被列出、安装、版本化和审查。
- 策略决策可以在不安全运行开始前阻断或要求审批。
- API、runtime、storage 和 observability 中的租户隔离都有明确边界。

### Phase 3：分布式 Agent Runtime

目标：从单 Agent 执行扩展到多 runtime、多 Agent 的协作网络。

| 里程碑 | 当前状态 | 下一步 |
| --- | --- | --- |
| Multi-runtime | runtime interface 已支持在 `mock` 和 `worker` backend 之间选择。 | 增加 LangGraph、远程 runtime 以及未来非 Python runtime 的真实 adapter。 |
| Agent Autoscaling | 尚未开始。 | 增加基于队列深度、延迟和成本的 runtime worker 扩缩容信号。 |
| Agent Mesh | 尚未开始。 | 定义 Agent 间发现、调用、策略传播、身份和 trace 关联。 |

Phase 3 退出标准：

- 多个 runtime backend 可以运行兼容的 agent revision。
- Agent 可以基于需求和策略限制自动扩缩容。
- Agent 到 Agent 的调用能保留 identity、policy、version 和 trace context。

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

将 CRDs、RBAC 和 controller-manager 部署到当前 Kubernetes context：

```bash
make deploy
```

移除已部署的控制平面：

```bash
make undeploy
```

用于本地 OrbStack 验证时，可以构建本地 worker 镜像：

```bash
make docker-build-worker-local
```

## Runtime Backends

controller manager 接受 `--runtime-backend` 参数。

- `mock`：默认 backend。它会确定性地完成 `AgentRun` 对象，用于控制平面验证。
- `worker`：在 `AgentRun` 所在 namespace 中创建 Kubernetes Job。它通过 `--worker-job-image` 和 `--worker-job-command` 指向 worker 镜像和命令。

本仓库包含两个镜像入口：

- `cmd/controller-manager`：协调控制平面资源。
- `cmd/worker`：校验注入的运行环境，并输出结构化占位结果。

## 仓库结构

```text
api/v1alpha1/                 Kubernetes API types
cmd/controller-manager/        controller-manager entrypoint
cmd/worker/                    worker entrypoint
config/crd/                    generated CRD manifests
config/samples/ehs/            sample custom resources
examples/ehs/                  source sample resources
internal/compiler/             Agent compiler and reference validation
internal/controller/           Agent and AgentRun reconcilers
internal/runtime/              runtime backend abstraction and implementations
internal/worker/               placeholder worker implementation
```

## 开源协议

Agent Control Plane 使用 Apache License, Version 2.0 授权。详见 `LICENSE`。

本项目依赖的第三方 Go modules 使用各自的开源协议。当前直接 runtime 依赖为 Kubernetes 和 controller-runtime 相关模块，协议为 Apache-2.0。传递依赖包含 Apache-2.0、BSD-style、MIT-style 和 ISC 等宽松开源协议。

分发源码包、二进制或容器镜像前，请保留项目 `LICENSE`，保留 `NOTICE`，并按照 `THIRD_PARTY_NOTICES.md` 中的说明包含第三方依赖许可证声明。

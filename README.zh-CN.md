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
- worker runtime backend 可以将每次运行分发为 Kubernetes Job。当前 worker 仍是占位实现，用于支撑 Eino runtime 成熟前的控制平面验证。

## 使用场景

Agent Control Plane 面向需要把 AI Agent 当作生产平台资源来运营的团队，而不是把
Agent 做成一次性脚本或隐藏在业务应用里的内部逻辑。

- **企业级 Agent 发布**：平台团队可以用 Kubernetes 原生 spec、status、revision 和
  RBAC 边界来定义、评审、发布和回滚 Agent。
- **业务流程自动化**：产品团队可以暴露可重复运行的 Agent 工作流，例如文档审核、工单分诊、事件响应、巡检分析和知识辅助决策。
- **受监管、可审计的 AI 运行**：风控、合规和运营团队可以为每次 Agent 调用关联策略、trace reference、评估计划和不可变运行记录。
- **垂直领域 Agent 系统**：领域团队可以打包 EHS 危害识别、质量巡检、维修计划、客户支持、财务运营等知识密集型场景的专用 Agent。
- **多租户 Agent 平台**：组织可以将团队或租户映射到 namespace，执行 runtime 边界，并集中观测同一集群中的多个 Agent。
- **Agent Marketplace 与复用**：共享 prompts、tools、knowledge bases、MCP servers、policies 和 evaluations 可以沉淀为未来 Agent package 的可复用构件。

## 架构方向

- Go 承载 Kubernetes API 类型、CRD controller、compiler、admission check、runtime dispatch，以及未来的 gateway。
- Go 预计承载基于 Eino 的 runtime worker。
- 默认 runner 方向是 `runtime.engine: eino` 与 `runtime.runnerClass: adk`；LangGraph 保留为未来兼容 adapter。
- PostgreSQL、pgvector、S3 兼容存储和队列预计用于状态、检索、产物和异步执行。
- TypeScript 可用于未来的控制台、Marketplace UI 和生成式 SDK。

## 当前进度

状态日期：2026-04-20。

| 模块 | 状态 | 证据 |
| --- | --- | --- |
| YAML Agent Spec | 进行中 | `api/v1alpha1` 和 `config/crd/bases` 下已有 Go API 类型和 CRD；`examples/ehs` 与 `config/samples/ehs` 下已有 EHS YAML 样例。 |
| 编译成 Eino | 部分完成 | `internal/compiler` 已能校验跨资源引用、产出面向 runtime 的 compiled artifact，并生成确定性 revision；尚未产出可执行的 Eino runner artifact。 |
| 发布 endpoint | Bootstrap | Agent controller 已发布 `Agent.status.endpoint.invoke`；invoke gateway 可接收 POST 请求并创建 `AgentRun` 资源。 |
| trace | 部分完成 | 已有 `AgentRun.status.traceRef`，mock/worker backend 会写入该字段；完整分布式 tracing 和 trace 存储尚未实现。 |
| version | 部分完成 | 已有 `Agent.status.compiledRevision` 与 `AgentRun.status.agentRevision`；语义化版本、发布通道和 revision history 仍待实现。 |
| runtime execution | Bootstrap | `mock` runtime 可确定性完成运行；`worker` runtime 可创建 Kubernetes Job、接收 compiled artifact，并返回占位输出。 |
| Policy | 仅有 Spec | 已有 `AgentPolicy` CRD 和 `Agent.spec.policyRef`；runtime dispatch 前的策略执行仍待实现。 |
| Evaluation | 仅有 Spec | 已有 `AgentEvaluation` CRD；评估 reconciler 和结果上报仍待实现。 |

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
| Packaging and deployment | 已有 Dockerfile、RBAC 和 `config/default` 部署清单。 | 增加 CI、镜像发布、release tag 和 Helm Chart。先补 chart skeleton 用于 dev/E2E 安装，v0.1.0 前再提升为正式安装 artifact。 |

Phase 1 退出标准：

- 应用 EHS 样例资源后可以得到 Ready 状态的 `Agent`。
- 通过 gateway 调用 Agent 后可以创建 `AgentRun`。
- 运行通过 Kubernetes Job runtime backend 执行。
- 运行结果记录 output、trace reference 和准确的 agent revision。
- controller-manager 和 worker 镜像可构建、可部署、可发布。

### Phase 2：真实 Agent Runtime

目标：用真正基于 Eino 的 runtime 替换占位 worker，同时保持 Kubernetes
原生控制平面契约不变。

| 里程碑 | 当前状态 | 下一步 |
| --- | --- | --- |
| Eino compile artifact | 已有静态引用 compiler。 | 产出兼容 Eino 的 runner artifact。 |
| Eino runtime worker | Go placeholder worker 已能校验注入的运行上下文和 compiled artifact 元数据。 | 使用 Eino 执行已编译 artifact，并返回结构化结果。 |
| Runtime contract | `AgentRun` 已携带 input、output、trace reference 和 revision。 | 定义 artifacts、logs、errors、取消和重试行为。 |
| Policy checks | 已有 `AgentPolicy` CRD 和 `Agent.spec.policyRef`。 | 在 dispatch 前执行模型/工具预算、guardrails 和审批门禁。 |
| Durable run records | 当前状态存储在 `AgentRun` 上。 | 增加持久化 trace、artifact 和 result storage。 |
| Evaluation | 已有 `AgentEvaluation` CRD。 | 增加 evaluation reconciler 和结果上报。 |

Phase 2 退出标准：

- EHS AgentRun 通过真实 Eino worker 执行。
- Policy 可以在不安全运行开始前阻断或要求审批。
- worker Pod 消失后仍可查看 run artifacts 和 traces。
- Evaluation 资源可以针对某个 agent revision 执行并发布结果。

### Phase 3：产品界面与治理

目标：让平台不仅能被集群 operator 使用，也能被团队直接使用。

| 里程碑 | 当前状态 | 下一步 |
| --- | --- | --- |
| UI | 本仓库尚未开始。 | 构建用于 agents、runs、traces、evaluations 和发布流程的控制台。 |
| Marketplace | 尚未开始。 | 定义可复用 agents/tools 的包元数据、发布流程、信任信号和安装流程。 |
| Tenant | 尚未开始。 | 增加租户模型、namespace 映射、RBAC 边界、quota 和审计轨迹。 |
| Governance workflows | 已有 Policy CRD。 | 增加 review、approval、human-in-the-loop 和 exception 工作流。 |

Phase 3 退出标准：

- 用户可以在 UI 中发布、查看、调用和调试 agents。
- Marketplace package 可以被列出、安装、版本化和审查。
- API、runtime、storage 和 observability 中的租户隔离都有明确边界。
- 治理工作流可审计、可执行。

### Phase 4：分布式 Agent Fabric

目标：从单 Agent 执行扩展到多 runtime、多 Agent 的协作网络。

| 里程碑 | 当前状态 | 下一步 |
| --- | --- | --- |
| Multi-runtime | runtime interface 已支持在 `mock` 和 `worker` backend 之间选择。 | 增加 Eino、LangGraph 兼容层和远程 runtime adapter。 |
| Agent autoscaling | 尚未开始。 | 增加基于队列深度、延迟和成本的 runtime worker 扩缩容信号。 |
| Agent mesh | 尚未开始。 | 定义 Agent 间发现、调用、策略传播、身份和 trace 关联。 |

Phase 4 退出标准：

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
examples/ehs/                  source sample resources
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

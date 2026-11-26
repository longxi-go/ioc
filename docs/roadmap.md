# ioc roadmap / ioc 路线图

The stages describe capability bands and their usable outcomes. Existing support
is documented in the [manual](design/docs/index.md); future directions below are
not current APIs or delivery promises. Web integration remains outside the core container.

各阶段按能力组组织，并说明可用结果。现有支持以[手册](design/docs/index.zh.md)
为准；下列后续方向不是当前 API 或交付承诺。Web 集成独立于核心容器。

| Stage / 阶段 | Capability band / 能力组 | Usable outcome / 可用结果 | Scope / 范围 |
| --- | --- | --- | --- |
| I1 | Definition catalog, graph binding, scopes, lifecycle, cycle/concurrency checks / 定义目录、图绑定、作用域、生命周期、循环与并发检查 | Explicit managed ownership through runtime APIs / 可通过运行时 API 使用明确所有权管理 | Supported / 已支持 |
| I2 | Annotation compiler, library publication, entry composition, typed queries / 注解编译器、库发布、入口组合、类型化查询 | Libraries and applications compose through generated ordinary Go / 库与应用通过生成的普通 Go 代码组合 | Supported within documented publication compatibility / 在文档发布兼容边界内已支持 |
| I3 | Runnable examples, bilingual reference, reproducible quality gates / 可运行示例、双语参考、可复现质量门禁 | Developers can build, test and operate a generated application / 开发者可构建、测试和管理生成应用 | Supported; validation must pass per revision / 已支持，各修订仍需通过验证 |
| I4 | Typed events, instance ordering, publication ownership, lifecycle barriers / 类型化事件、实例排序、投递所有权、生命周期屏障 | Synchronous in-process events share container ownership / 进程内同步事件共享容器所有权 | Supported / 已支持 |
| I4.1 | Finite injection grammar, explicit Bean conditions and selection diagnostics / 有限注入语法、显式 Bean 条件与选择诊断 | Six validated dependency forms and inspectable metadata-only selection / 六种经过校验的依赖形态与可查询的元数据选择结果 | Supported / 已支持 |
| I4.2 | Named initialization prerequisites and condition-order separation / 具名初始化前置依赖与条件求值解耦 | Lazy prerequisites complete Init before dependent construction, with explicit scope and cycle rules / lazy 前置依赖在依赖方构造前完成 Init，作用域和依赖环有明确约束 | Supported / 已支持 |
| I4.3 | Structured annotation attributes, Environment bootstrap and profile conditions / 结构化注解属性、Environment 启动与 Profile 条件 | A generated application reads one configuration Environment and fixes topology at startup / 生成应用通过统一 Environment 读取配置，并在启动时固定拓扑 | Supported / 已支持 |
| I5 | HTTP/MVC and configuration-center adapters / HTTP/MVC 与配置中心适配器 | Web applications reuse explicit scopes and cleanup rules / Web 应用复用显式作用域与清理规则 | Planned; separate repositories and design / 后续方向，独立仓库与设计 |
| I6 | Independent adapters and compatibility evidence / 独立适配器与兼容性证据 | Contracts remain usable across implementations and releases / 契约可跨实现与版本使用 | Planned / 后续方向 |

Current mechanisms: [architecture](design/docs/architecture.md),
[annotations](design/docs/annotations.md), [events](design/docs/events.md).
Existing APIs do not imply automatic rebinding, dynamic graph refresh, arbitrary
interception or general multi-target library bundles. Compatibility and supported
build environments must be validated for the relevant release.

当前机制：[架构](design/docs/architecture.zh.md)、[注解](design/docs/annotations.zh.md)、
[事件](design/docs/events.zh.md)。现有 API 不代表支持自动重绑定、动态图刷新、任意
拦截或通用多目标库打包。兼容性与支持的构建环境需对相应发布进行验证。

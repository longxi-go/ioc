# 开发与验证

## 前提条件

使用 Go 1.27.0 或更高版本。Quality 工作流测试最低版本以及配置的完整验证工具链。
依赖与 golayout 工具固定在 go.mod/go.sum 中，独立检出可解析公开模块，无需兄弟
仓库。请安装 [Quality 工作流](https://github.com/longxi-go/ioc/blob/main/.github/workflows/quality.yml)
指定的 golangci-lint 版本。手册使用 Python 3.12 和 uv 构建。

```sh
make setup
make format
make check-format
make check
```

`setup` 验证共享 golayout 工具，不安装本地格式化器、不改模块文件。`format`
重写本仓库维护的 Go 源码及完整 `testdata/**/*.go.tmpl` fixture；`check-format` 只读。
普通负例 fixture、忽略文件、缓存和 vendor 不参与。生成器使用同一固定版本的
`golayout/pkg/format` 实现。单独 gofmt/gofumpt 不执行完整项目布局策略。

`make check` 执行许可头与格式检查、格式化回归、模块整洁度、vet、覆盖率测试、
随机顺序 race 测试、golangci-lint 和示例生成产物一致性检查。本仓库维护的每个包都应包含单元
测试和可运行示例。生成器修改需要编译/运行集成验证，包括库发布和消费方组合，
不能只比较输出字符串。

Go 源码声明以 `codequality/HEADER` 为准。`make license-headers` 更新本仓库维护的源码；
`make check-license-headers` 只读检查。应用产物使用其自身可选声明模板，而非 ioc
的版权所有者。

## 本地生态工作区 {#local-workspace}

同时修改多个生态仓库时，可在父工作区选择本地模块。从共同父目录执行，前提是尚无
go.work：

```sh
go work init ./golayout ./gosr ./gosr/tools ./nemo ./ioc
```

已有工作区使用 `go work use` 扩展。`gosr/tools` 是独立嵌套模块，必须显式选择。
go.work/go.work.sum 保持本地使用，生产 go.mod 不应包含兄弟目录 replace。
在 ioc 目录运行工作区生成与检查：

```sh
make generate WORKSPACE=../go.work
make check WORKSPACE=../go.work
```

`WORKSPACE` 验证 Go 选中了指定工作区。隔离的消费方测试程序通过
`IOC_TEST_WORKSPACE` 使用其中选定的模块；replace 仅写入临时 go.mod，生产模块
依赖保持不变。省略 `WORKSPACE` 时，`make check` 使用 `GOWORK=off` 和固定的
公开依赖。工作区检查与独立模块解析是两条验证路径。

## 文档

```sh
make docs-setup
make docs-serve
make docs-build
```

这些命令使用 `docs/design` 下的锁定依赖。严格构建生成英文和简体中文。预览地址
为 `http://127.0.0.1:8000/ioc/`，中文位于 `/ioc/zh/`。同时添加 `page.md` 与
`page.zh.md`，更新两套导航，保持命令、默认值、失败和所有权语义一致。仓库相对
链接必须指向随版本交付的文件。生成的 HTML 和虚拟环境不应提交到版本控制。

Mermaid 代码块使用 Material 渲染器，通过 `overrides/main.html` 加载固定版本
Mermaid 脚本；渲染需要访问该脚本 CDN。双语图表应保持含义一致，并在浏览器检查
生成 SVG，确认无语法错误、标签裁切，明暗配色均可读。Markdown 构建成功本身
不能验证图表渲染。

Gin 与 Coffee 站点页通过本地 Markdown 片段包含示例正文。分别修改
`examples/gin/README[.zh_CN].md` 和
`examples/coffee/docs/integration[.zh_CN].md` 中的 `*-manual` 区域；仓库文件与
站点包装页各自定义相对链接，正文无需复制。包含路径相对于 MkDocs 配置定位，
文件或区域缺失时构建失败。新增引用时同步维护两种阅读位置的链接定义。

## 发布与在线文档

发布工具在正式发布前验证版本/tag、依赖、质量结果和已发布源码。Release 工作流在
必要质量门禁后发布稳定 GitHub Release，再为发布源码调用 Docs Pages。Pages
工作流确认所选稳定发布已公开，且部署前仍为 Latest。手动重试 Pages 需要选择
发布 tag，不会发布任意分支文档。

文档站点为 `https://longxi-go.github.io/ioc/`，中文位于 `/ioc/zh/`。仓库 Pages
来源选择 GitHub Actions，并在 `github-pages` 环境中添加 `v*` 标签规则。
仅允许 `main` 分支不能授权标签触发的部署。`DOCS_SITE_URL` 控制本地/自定义基路径构建，必须以 `/` 结尾；在线构建从
Pages 配置获取基地址。使用自定义域名还需设置 Pages 域名。在线手册对应最新稳定
发布。

操作参数与检查见
[发布工具文档](https://github.com/longxi-go/ioc/tree/main/docs/releases) 和
[工作流](https://github.com/longxi-go/ioc/tree/main/.github/workflows)。已发布版本标签
不可变；发布或 Pages 部署失败时，通过重试工作流恢复。

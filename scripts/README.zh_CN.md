# 仓库脚本

[English](README.md) | [简体中文](README.zh_CN.md)

使用 Python 3.12 和 `go.mod` 要求的 Go 版本，在仓库根目录运行 Make 目标。
格式化还需要 Git 和 Bash；只读 GitHub 预检查需要已认证的 GitHub CLI。

| 脚本 | 职责 | 目标 |
| --- | --- | --- |
| `golayout.py` | 解析声明的 golayout 模块，将格式化、文件选择和文件头处理委托给其共享脚本 | `setup`、`format`、`check-format`、`test-format`、`license-headers`、`check-license-headers` |
| `go_environment.py` | 拒绝进程或持久 `GOFLAGS` 替换待检查的依赖图 | `verify-environment` |
| `go_module.py` | 使用 Go 解析模块声明，不解析依赖 | 被其他脚本导入 |
| `release.py` | 验证根模块稳定标签、模块身份、已发布依赖候选版本和维护的说明；按需读取 Pages 设置及上游 Release | `validate-release`、`preflight-release` |
| `publish_release.py` | 核对标签提交，创建或复用内容一致的 Release，防止 Pages 使用移动的标签或被取代的版本 | Release 和 Docs Pages 工作流 |

`make check` 禁用 workspace 和自动工具链切换。所有 Go 门禁拒绝非空 `GOFLAGS`，
包括持久配置中的标志。`GO` 选择 Go 可执行文件；代理和校验服务配置保持有效。
格式化语言版本来自本模块的 Go 指令。适配器先通过 `-mod=readonly` 检查工具依赖图，
缺少校验记录时失败，不改写 `go.mod` 或 `go.sum`。不需要兄弟仓库。

`make test-scripts` 运行隔离单元测试，包括使用真实 Go 的模块解析、缺失校验记录和
持久标志回归。受控子进程/API 响应覆盖发布恢复、HTTP 错误、Pages 策略、说明冲突和
移动的标签，不发布 Release 或修改 Pages 设置。
`make test-format` 验证实际的共享格式化器。

## 发布准备与恢复

1. 在 `docs/releases/vX.Y.Z.md` 维护英文说明，标题必须为 `# ioc vX.Y.Z`。
2. 运行 `make check`、`make docs-build` 和 `make validate-release TAG=vX.Y.Z`。
3. 运行 `make preflight-release TAG=vX.Y.Z`。它读取上游稳定 Release，检查模块整洁性、
   校验记录、Pages Actions 模式，以及 `github-pages` 是否允许该标签部署。
   只有 main 分支的策略会返回缺少标签规则的明确错误。该命令不修改远端设置。
4. 在目标提交创建不可变 `vX.Y.Z` 标签。Release 工作流验证该提交，
   并对它运行可复用 Quality 后才创建仅含源码的 GitHub Release。Quality 也严格构建
   两种文档语言。不提供预编译 CLI 二进制；用户使用
   `go get github.com/longxi-go/ioc@vX.Y.Z` 和
   `go install github.com/longxi-go/ioc/cmd/ioctl@vX.Y.Z`。

恢复时，从包含所需工具的工作流引用手动触发 `release.yml`，传入
`release_tag=vX.Y.Z`。例如工具修复已进入 main：

```sh
gh workflow run release.yml --repo longxi-go/ioc --ref main -f release_tag=v0.1.0
```

工作流重新解析并验证现有标签，绝不移动或重新创建它。已发布说明和元数据一致时直接复用；
内容冲突会报错。发布查询出现临时 404 时进行三次有上限的读取，才认定资源不存在。
权限错误和写入结果不确定的失败保持失败；POST 不会自动重试。
重新运行时先读取 Release，避免响应不确定后重复创建。
补发旧版本会保留更新的 Latest。Latest 不可用或等待后仍未收敛会明确失败，
不会静默跳过部署。

Pages 从当前已发布稳定版本构建文档，在部署前再次核对 Latest 和构建产物对应的提交。
重新构建时，传入当前版本标签手动触发 `docs-pages.yml`：

```sh
gh workflow run docs-pages.yml --repo longxi-go/ioc --ref v0.1.0 -f release_tag=v0.1.0
```

环境保护检查工作流的引用，而非 checkout 目录。所选工作流引用也必须被 `github-pages`
允许；在 main 手动恢复时，除了预检查要求的标签规则，还需要允许 main 的规则。
已被新版本取代的 Release 不能覆盖当前站点。这些工作流不会修改 Pages 设置或
移动现有标签，Pages 保护规则需要单独配置。

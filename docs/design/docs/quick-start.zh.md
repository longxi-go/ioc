# 快速开始

使用 Go 1.27.0 或更高版本，CLI 与应用采用同一 ioc 版本。在空目录中执行：

```sh
go mod init example.com/hello
go get github.com/longxi-go/ioc@latest
go install github.com/longxi-go/ioc/cmd/ioctl@$(go list -m -f '{{.Version}}' github.com/longxi-go/ioc)
```

`@latest` 在初次设置时选择已发布版本，之后保留 go.mod 中选定的版本。
将 `$(go env GOPATH)/bin` 加入 PATH，或使用 `ioctl` 的完整路径。使用 ioc 源码时，
应从同一修订构建 CLI。

## 编写并生成应用

将以下完整程序保存为 `main.go`：

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"
    "time"

    "github.com/longxi-go/ioc"
)

// +ioc:component
type Greeter struct{}

func (*Greeter) Message() string {
    return "hello from generated IoC"
}

func run(ctx context.Context) (result error) {
    app, err := ioc.NewApplicationContext(ctx)
    if err != nil {
        return err
    }
    defer func() {
        cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
        defer cancel()
        result = errors.Join(result, app.Close(cleanup))
    }()
    if err := app.Start(ctx); err != nil {
        return err
    }
    greeter, err := app.GetBean[*Greeter](ctx)
    if err != nil {
        return err
    }
    fmt.Println(greeter.Message())
    return nil
}

func main() {
    if err := run(context.Background()); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

```sh
ioctl generate --entry . --roots .
go mod tidy
go run .
ioctl check --entry . --roots .
```

输出为 `hello from generated IoC`。生成器创建普通 Go 源码，包含同包构造代码和
应用查询计划。生成文件应随源码保存；`go build` 不会执行生成。`check` 只比较
预期产物，不重写文件。注解、依赖、入口根或构建约束变化后应重新生成。

延迟执行的 Close 也会在启动失败后运行，并将清理错误与原错误合并。超时只限制
调用方等待，容器会继续清理；再次调用 Close 可以等待同一次清理的结果。构造函数、
Init 和 Start 接收操作 Context，Stop 和 Destroy 接收容器的清理 Context。不得
保存回调 Context，也不能向字段注入 Context。

## 构造函数与字段注入

在构造函数使用 `+ioc:bean`，在结构体使用 `+ioc:component`。参数与被注入字段
通过 Go 类型声明依赖契约。用 `+ioc:expose` 显式暴露接口，仅实现接口不会自动
注册。名称、限定符和主候选用于消除多个候选的歧义。
工厂接收注入参数，返回值不会再经过独立字段注入；组件直接分配实例，只注入标注
字段。详见[注入模型](architecture.md#injection-model)和[完整注解参考](annotations.md)。

以下作用域示例中的 `gs` 指契约包：

```go
import gs "github.com/longxi-go/gosr/pkg/ioc"
```

使用 `gs.Provider[T]` 按每次调用方的上下文解析，避免单例捕获请求对象。通过
`requestCtx, scope, err := app.OpenScope(ctx, gs.Request)` 打开请求作用域，将
`requestCtx` 传给查询/Provider，并在每条退出路径上关闭 `scope`。资源原型使用
`app.AcquireBean[T](ctx)`，并关闭返回的 Lease。详见[作用域所有权](architecture.md#scope-ownership)。

## 运行仓库示例

在 ioc 仓库中执行：

```sh
make build
make generate
go run ./examples/hello
go run ./examples/events
make check-generated
```

CLI 位于 `.tmp/bin/ioctl`。hello 示例输出 `hello from static IoC (request-1)`，
涵盖接口/私有字段注入、资源和作用域。事件示例依次输出 `audit order-42` 和
`notify order-42`。二者都包含可执行测试，并显式执行启动与清理。源码见
[hello](https://github.com/longxi-go/ioc/tree/main/examples/hello) 与
[events](https://github.com/longxi-go/ioc/tree/main/examples/events)。

## 检查组合结果

```sh
ioctl generate --entry ./cmd/server --roots ./app/...
ioctl check --entry ./cmd/server --roots ./app/...
ioctl explain --entry ./cmd/server --roots ./app/...
ioctl graph --entry ./cmd/server --roots ./app/...
```

这些命令使用 `cmd/server` 作为入口，`app/...` 作为生成根。
`explain` 输出候选定义 JSON；
`graph` 输出潜在依赖和排序边的 JSON，包括条件分支。两者都不写源码。

`codegen.Edge.Kind` 的类型为 `codegen.EdgeKind`：

| 常量 | JSON 值 | 关系 |
| --- | --- | --- |
| `codegen.DependencyEdge` | `dependency` | 注入依赖；`Access` 区分直接解析与延迟解析 |
| `codegen.PrerequisiteEdge` | `depends-on` | 按名称声明的初始化前置依赖 |

`--dir` 选择目标模块目录，默认为 `.`；`--roots` 接受逗号分隔的本模块源码模式。
`--tags a,b` 设置显式 Go 构建标签，并覆盖 `GOFLAGS` 中的 `-tags`；目标平台设置
及已启用的检测选项仍然生效。库作者使用 `ioctl generate --library`，
详见[库发布](architecture.md#library-publication)。

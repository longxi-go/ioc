# Quick start

Use Go 1.27.0 or later and the same ioc version for the CLI and application.
In an empty directory:

```sh
go mod init example.com/hello
go get github.com/longxi-go/ioc@latest
go install github.com/longxi-go/ioc/cmd/ioctl@$(go list -m -f '{{.Version}}' github.com/longxi-go/ioc)
```

`@latest` selects a published version for the initial setup; retain the selected
version in go.mod. Put `$(go env GOPATH)/bin` on PATH, or invoke `ioctl` by its
full path. When using an ioc source checkout, build the CLI from that same revision.

## Write and generate the application

Save this complete program as `main.go`:

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

Output: `hello from generated IoC`. Generation creates ordinary Go source, including
same-package construction and an application query plan. Keep generated files
with your source; `go build` does not run generation. `check` compares expected
output without rewriting files. Rerun generation when annotations, dependencies,
entry roots or build constraints change.

The deferred Close also runs after failed startup and joins cleanup errors with
the original error. Its timeout limits the caller's wait; the container continues
cleanup, and another Close call can wait for the same result. Constructors, Init
and Start receive operation contexts; Stop and Destroy receive the container's
cleanup Context. Do not retain callback contexts or inject Context fields.

## Constructor and field injection

Use `+ioc:bean` on a constructor or `+ioc:component` on a struct. Parameters and
injected fields declare dependency contracts through Go types. Explicitly expose
interfaces with `+ioc:expose`; implementing an interface does not register it
automatically. Names, qualifiers and primary resolve multiple candidates.
A factory receives injected parameters; its result does not receive a separate
field-injection pass. A component is allocated directly and receives only its
annotated fields. See the [injection model](architecture.md#injection-model) and
[complete annotation reference](annotations.md).

In the scope examples below, `gs` names the contract package:

```go
import gs "github.com/longxi-go/gosr/pkg/ioc"
```

Use `gs.Provider[T]` to resolve with each caller's context rather than capturing a
request object in a singleton. Open a request with
`requestCtx, scope, err := app.OpenScope(ctx, gs.Request)`, pass `requestCtx` to
lookups/providers and close `scope` on every exit path. Resource prototypes use
`app.AcquireBean[T](ctx)` and the returned Lease's Close. See
[scope ownership](architecture.md#scope-ownership).

## Run the repository examples

From an ioc checkout:

```sh
make build
make generate
go run ./examples/hello
go run ./examples/events
make check-generated
```

The CLI is `.tmp/bin/ioctl`. The hello example prints
`hello from static IoC (request-1)` and covers interface/private-field injection,
resources and scopes. The event example prints `audit order-42`, then
`notify order-42`. Both include executable tests and full lifecycle handling.
Read [hello](https://github.com/longxi-go/ioc/tree/main/examples/hello) and
[events](https://github.com/longxi-go/ioc/tree/main/examples/events).

## Inspect composition

```sh
ioctl generate --entry ./cmd/server --roots ./app/...
ioctl check --entry ./cmd/server --roots ./app/...
ioctl explain --entry ./cmd/server --roots ./app/...
ioctl graph --entry ./cmd/server --roots ./app/...
```

These paths assume your project contains `cmd/server` and `app`. `explain` outputs
candidate definitions as JSON; `graph` outputs potential dependency and ordering
edges as JSON, including conditional alternatives. Neither writes source.

`codegen.Edge.Kind` uses `codegen.EdgeKind`:

| Constant | JSON value | Relationship |
| --- | --- | --- |
| `codegen.DependencyEdge` | `dependency` | An injection dependency; `Access` distinguishes direct and deferred resolution |
| `codegen.PrerequisiteEdge` | `depends-on` | A named initialization prerequisite |

`--dir` selects the target module directory (default `.`); `--roots` accepts
comma-separated source patterns within that module. `--tags a,b` sets explicit Go
build tags, overriding `GOFLAGS`'s `-tags` value. Target settings and enabled
instrumentation flags still apply.
Library authors use `ioctl generate --library`; see
[library publication](architecture.md#library-publication).

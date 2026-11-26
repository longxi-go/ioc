# Coffee MVC 应用

在仓库的 `examples/coffee` 下启动服务：

```sh
make run
# http://127.0.0.1:8082/
```

订单 CRUD、制作和取餐同时提供 HTML 页面与 REST API，数据持久化到本地
`coffee.db`。[示例 README][coffee-readme] 列出状态规则、路由、请求示例和配置项。

--8<-- "examples/coffee/docs/integration.zh_CN.md:coffee-manual"

[coffee-examples-coffee-internal-configuration-settings-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/configuration/settings.go
[coffee-examples-coffee-application-yml]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/application.yml
[coffee-examples-coffee-internal-configuration-zz-nemo-gen-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/configuration/zz_nemo.gen.go
[coffee-examples-coffee-cmd-coffee-main-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/cmd/coffee/main.go
[coffee-examples-coffee-internal-repository-sqlite-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/repository/sqlite.go
[coffee-examples-coffee-internal-infrastructure-database-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/infrastructure/database.go
[coffee-examples-coffee-internal-infrastructure-migration-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/infrastructure/migration.go
[coffee-examples-coffee-internal-service-service-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/service/service.go
[coffee-examples-coffee-internal-observation-logging-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/observation/logging.go
[coffee-examples-coffee-internal-observation-audit-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/observation/audit.go
[coffee-examples-coffee-internal-controller-api-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/controller/api.go
[coffee-examples-coffee-internal-controller-pages-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/controller/pages.go
[coffee-examples-coffee-makefile]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/Makefile
[coffee-examples-coffee-cmd-coffee-zz-ioc-gen-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/cmd/coffee/zz_ioc.gen.go
[coffee-examples-coffee-internal-application-application-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/application/application.go
[coffee-examples-coffee-internal-infrastructure-request-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/infrastructure/request.go
[coffee-examples-coffee-internal-application-scope-go]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/internal/application/scope.go
[coffee-docs-design-docs-annotations-zh-md]: ../annotations.md
[coffee-readme]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/README.zh_CN.md

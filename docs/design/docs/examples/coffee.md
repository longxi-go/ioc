# Coffee MVC application

Start the server from `examples/coffee` in a checkout:

```sh
make run
# http://127.0.0.1:8082/
```

HTML forms and a REST API provide order CRUD, preparation and collection. SQLite
persists data in local `coffee.db`. The [example README][coffee-readme] lists
workflow rules, routes, requests and configuration.

--8<-- "examples/coffee/docs/integration.md:coffee-manual"

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
[coffee-docs-design-docs-annotations-md]: ../annotations.md
[coffee-readme]: https://github.com/longxi-go/ioc/blob/main/examples/coffee/README.md

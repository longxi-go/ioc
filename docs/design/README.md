# ioc documentation site / ioc 文档站点

The English and Simplified Chinese manual uses MkDocs Material with locked
Python 3.12 dependencies. From the repository root:

英文与简体中文手册使用 MkDocs Material 和锁定的 Python 3.12 依赖。在仓库根目录
执行：

```sh
make docs-setup
make docs-serve
make docs-build
```

| | English | 简体中文 |
| --- | --- | --- |
| Preview / 预览 | `http://127.0.0.1:8000/ioc/` | `http://127.0.0.1:8000/ioc/zh/` |
| Hosted manual / 在线手册 | [ioc Guide](https://longxi-go.github.io/ioc/) | [ioc 使用手册](https://longxi-go.github.io/ioc/zh/) |

From this directory / 在本目录执行：

```sh
uv sync --locked
uv run --locked mkdocs build --strict
```

Add `page.md` and `page.zh.md` together, with corresponding navigation entries in
`mkdocs.yml`. Keep commands, API behavior and diagrams equivalent in both languages.
Generated `site/` and `.venv/` stay out of version control. `DOCS_SITE_URL` overrides
the site base URL and must include a trailing slash.

Mermaid uses Material's renderer and a pinned CDN script. Inspect the rendered
diagrams in a browser as well as running the strict build. Pages deploys the
latest stable published release. Workspace, validation and release requirements
are in the [development guide](docs/development.md).

同时添加 `page.md` 和 `page.zh.md`，并在 `mkdocs.yml` 中配置对应导航。双语命令、
API 行为和图表应保持一致。生成的 `site/` 和 `.venv/` 不纳入版本控制。
`DOCS_SITE_URL` 可覆盖站点基地址，需保留末尾 `/`。

Mermaid 使用 Material 渲染器和固定版本的 CDN 脚本。除严格构建外，还应在浏览器
检查实际图表。Pages 部署最新稳定发布版本，工作区、验证和发布要求见
[开发指南](docs/development.zh.md)。

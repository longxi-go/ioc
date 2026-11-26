# Development and verification

## Prerequisites

Use Go 1.27.0 or later; the Quality workflow tests the minimum version and the
configured full-validation toolchain. Dependencies and the golayout tool are
pinned in go.mod/go.sum. A standalone checkout resolves published modules without
sibling repositories. Install the golangci-lint version in the
[Quality workflow](https://github.com/longxi-go/ioc/blob/main/.github/workflows/quality.yml).
Python 3.12 and uv build the manual.

```sh
make setup
make format
make check-format
make check
```

`setup` verifies shared golayout tooling without installing a local formatter or
changing module files. `format` rewrites Go source maintained in this repository and complete
`testdata/**/*.go.tmpl` fixtures; `check-format` is read-only. Ordinary negative
fixtures, ignored files, caches and vendor are excluded. Generators use the same
pinned `golayout/pkg/format` implementation. Direct gofmt/gofumpt alone does not
enforce the complete project layout policy.

`make check` runs license-header and formatting checks, formatter regressions,
module tidiness, vet, coverage tests, shuffled race tests, golangci-lint and
freshness checks for generated examples. Each maintained package must have unit tests and
runnable examples. Generator changes need compile/run integration, including
library publication and consumer composition, not just output string assertions.

The canonical Go notice is `codequality/HEADER`. `make license-headers` refreshes
source maintained in this repository; `make check-license-headers` checks without writes. Application
output uses its own optional header template rather than ioc's copyright owner.

## Local ecosystem workspace {#local-workspace}

For simultaneous changes across ecosystem repositories, select local modules in
a parent workspace. From their common parent, if no go.work exists:

```sh
go work init ./golayout ./gosr ./gosr/tools ./nemo ./ioc
```

Use `go work use` to extend an existing workspace. `gosr/tools` is a separate nested
module and must be selected explicitly. Keep go.work/go.work.sum local; production
go.mod files must not contain sibling-directory replacements. Run workspace
generation and checks from the ioc directory:

```sh
make generate WORKSPACE=../go.work
make check WORKSPACE=../go.work
```

`WORKSPACE` verifies that Go selected the specified workspace. Isolated consumer
fixtures use its selected modules through `IOC_TEST_WORKSPACE`; replacements are
written only to their temporary go.mod files. Production module requirements remain
unchanged. Without `WORKSPACE`, `make check` uses `GOWORK=off` and the pinned
published dependencies. Workspace checks and standalone module resolution are
separate verification paths.

## Documentation

```sh
make docs-setup
make docs-serve
make docs-build
```

These use locked dependencies under `docs/design`. The strict build produces both
English and Simplified Chinese. Local preview is `http://127.0.0.1:8000/ioc/`, with
Chinese under `/ioc/zh/`. Add `page.md` and `page.zh.md` together, update both nav
trees and keep commands, defaults, failures and ownership semantics equivalent.
Repository-relative links must resolve to files included in the release. Keep
generated HTML and virtual environments out of version control.

Mermaid fences use Material's renderer and a version-pinned Mermaid script loaded
by `overrides/main.html`; rendering requires access to that script CDN. Keep diagrams
meaning-equivalent in both languages, inspect the rendered SVG in a browser, and
check for syntax errors, clipped labels and readable light/dark colors. A successful
Markdown build alone does not validate diagram rendering.

The Gin and Coffee manual pages include local Markdown sections from
`examples/gin/README[.zh_CN].md` and
`examples/coffee/docs/integration[.zh_CN].md`. Edit their `*-manual` regions;
repository files and site wrappers supply their own link definitions. The include
base is relative to the MkDocs config, and missing files or regions fail the
build. New references require definitions for both reading contexts.

## Release and hosted documentation

Release tooling validates the version/tag, dependencies, quality result and
published source before publication. The Release workflow publishes a stable GitHub
Release after its required quality gates, and invokes Docs Pages for the released
source. The Pages workflow checks that the requested stable release is published
and remains Latest before deployment. A manual Pages retry selects the release
tag; it does not publish arbitrary branch documentation.

The documentation site is `https://longxi-go.github.io/ioc/`, with Chinese under
`/ioc/zh/`. Choose GitHub Actions as the repository's Pages source and configure the
`github-pages` environment to allow release tags through a `v*` tag rule.
A rule for the `main` branch alone does not authorize tag-triggered deployments.
`DOCS_SITE_URL` controls local/custom-base builds and must end with `/`; hosted
builds obtain their base URL from Pages configuration. Set a custom domain in
Pages settings as well. The live manual describes the latest stable release.

See the
[release tooling](https://github.com/longxi-go/ioc/tree/main/docs/releases) and
[workflows](https://github.com/longxi-go/ioc/tree/main/.github/workflows) for exact
operator inputs and checks. Published release tags are immutable; use workflow
retries to recover a failed release or Pages deployment.

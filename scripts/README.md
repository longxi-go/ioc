# Repository scripts

[English](README.md) | [简体中文](README.zh_CN.md)

Run Make targets from the repository root with Python 3.12 and the Go version
required by `go.mod`. Formatting also requires Git and Bash. Read-only GitHub
preflight requires an authenticated GitHub CLI.

| Script | Responsibility | Targets |
| --- | --- | --- |
| `golayout.py` | Resolve the declared golayout module and delegate formatting, file selection and source notices to its shared scripts | `setup`, `format`, `check-format`, `test-format`, `license-headers`, `check-license-headers` |
| `check_modules.py` | Check nested Gin and application modules in an isolated checkout copy, using temporary local replacements without changing source metadata | `check-adapters` |
| `go_environment.py` | Reject process and persistent `GOFLAGS` that can substitute the checked dependency graph | `verify-environment` |
| `go_module.py` | Parse module declarations using Go without resolving dependencies | Imported helper |
| `release.py` | Validate stable root/adapter tags, module identity, published dependency candidates and maintained notes; optionally read Pages settings and upstream Releases | `validate-release`, `preflight-release` |
| `publish_release.py` | Verify the tag commit, create or reuse matching release content, and guard Pages against moved tags and superseded releases | Release and Docs Pages workflows |

`make check` disables workspaces and automatic toolchain switching. All Go gates
reject nonempty `GOFLAGS`, including persistent flags. `GO` selects the Go
executable; proxy and checksum configuration are retained. The formatter language
comes from this module's Go directive. The formatter adapter checks its dependency
graph with `-mod=readonly` before delegation; missing checksums fail without
rewriting `go.mod` or `go.sum`. There is no sibling repository requirement.

`make test-scripts` runs isolated unit tests, including real Go module parsing,
missing-checksum and persistent-flag regressions. Controlled subprocess/API
responses exercise release recovery, HTTP errors, Pages policies, conflicting
notes and moved tags without publishing Releases or changing Pages settings.
`make test-format` exercises the actual shared formatter.

## Release preparation and recovery

1. Maintain English notes in `docs/releases/vX.Y.Z.md` with a matching
   `# ioc vX.Y.Z` title.
2. Run `make check`, `make docs-build` and `make validate-release TAG=vX.Y.Z`.
3. Run `make preflight-release TAG=vX.Y.Z`. It reads upstream stable Releases,
   checks module tidiness and checksums, and requires Pages Actions mode plus a
   `github-pages` policy allowing the tag. A main-only branch policy fails with
   the exact missing tag rule. This command never changes remote settings.
4. Create the immutable `vX.Y.Z` tag on the
   intended commit. The Release workflow validates and runs reusable Quality on
   that exact commit before creating a source-only GitHub Release. Quality also
   builds both documentation languages strictly. No prebuilt CLI binaries are
   attached; consumers use `go get github.com/longxi-go/ioc@vX.Y.Z` and
   `go install github.com/longxi-go/ioc/cmd/ioctl@vX.Y.Z`.

For recovery, dispatch `release.yml` with `release_tag=vX.Y.Z` from a workflow ref
that contains the required tooling. For example, after a tooling fix on main:

```sh
gh workflow run release.yml --repo longxi-go/ioc --ref main -f release_tag=v0.1.0
```

The existing tag is resolved and checked again; it is never moved or recreated.
Matching published notes and metadata are reused. Conflicting release content is
an error. Temporary release-lookup 404s receive three bounded reads before absence
is accepted. Permission errors and uncertain write failures remain failures; POST
requests are never retried automatically. Rerunning the workflow reads the release
first, preventing duplicate creation after an uncertain response. Backfilling an
older version preserves a newer Latest release. Unavailable or nonconverging Latest
metadata fails after a bounded wait instead of silently skipping deployment.

Pages builds documentation from the current published stable release and rechecks
Latest and the built commit immediately before deployment. To rebuild it, dispatch
`docs-pages.yml` with the current release tag:

```sh
gh workflow run docs-pages.yml --repo longxi-go/ioc --ref v0.1.0 -f release_tag=v0.1.0
```

Environment protection evaluates the workflow's ref, not its checkout directory.
The selected workflow ref must also be allowed by `github-pages`; a recovery
workflow dispatched on main needs an allowed main rule in addition to the tag
policy required by preflight. A superseded release cannot replace the current
site. These workflows do not change Pages settings or move existing tags;
configure Pages protection separately.

## Gin adapter releases

The Gin module uses `adaptors/gin/vX.Y.Z` tags and maintained notes in
`adaptors/gin/docs/releases/vX.Y.Z.md`, headed `# ioc Gin adapter vX.Y.Z`.
Run `make validate-release MODULE=gin TAG=adaptors/gin/vX.Y.Z` and, after the
required root version is published,
`make preflight-release MODULE=gin TAG=adaptors/gin/vX.Y.Z`.

The Release workflow checks out and verifies the exact tagged source for either
module. Adapter publication requires its declared stable root Release, never
claims Latest, and does not trigger Pages. Root release tags retain Latest and
Pages ownership. Recovery uses the same `release_tag` dispatch input with the
module's complete tag.

# Contributing to flow

flow is deliberately small. Read the [Philosophy](README.md#philosophy):

> Does this help someone understand their network in under one second?
> If no — cut it.

PRs adding CPU panels, ping, packet counts, multi-pane layouts, or tabs
will be declined regardless of code quality. Open an issue first.

UI changes should preserve the current split between data collection,
state updates, and rendering. Keep motion spring-driven, pulses brief,
and drawing cheap. Avoid border-heavy layouts or anything that
reintroduces dashboard clutter.

## Setup

Requires Go 1.22+ and git.

```
git clone https://github.com/programmersd21/flow
cd flow
make build
./bin/flow --version
```

Install golangci-lint (https://golangci-lint.run/):

```
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
  | sh -s -- -b $(go env GOPATH)/bin v1.59.1
```

## Check suite

Run before every PR:

```
make check
```

Runs fmt-check, vet, lint, test in sequence. CI runs the same command.

| Command          | What it does                           |
|------------------|----------------------------------------|
| make build       | go build with ldflags from VERSION     |
| make fmt         | gofmt -l -w .                          |
| make vet         | go vet ./...                           |
| make lint        | golangci-lint run ./...                |
| make test        | go test ./... -race -cover             |
| make check       | fmt-check, vet, lint, test in sequence |
| make clean       | removes bin/ and dist/                 |
| make release-dry | goreleaser snapshot, no publish        |

## Commits

Conventional Commits recommended, not enforced. Subject under 72 chars.
Use "Closes #N" in the body when applicable.

```
feat: add --no-header flag
fix: prevent sparkline crash on width < 10
docs: clarify tmux integration
refactor: extract formatBytes helper
```

## Branch protection

main requires the build-and-test status check before merge. Enable in
Settings > Branches > Branch protection rules and add the check named
build-and-test (the job name in ci.yml).

## Versioning and releases

The VERSION file is the single source of truth. Never hardcode a version
string elsewhere. The build injects it via `-ldflags "-X main.version=$(cat VERSION)"`.

Releases are cut via the Release workflow (workflow_dispatch):

1. Update CHANGELOG.md with the new version section.
2. Bump VERSION.
3. Merge the PR.
4. Run the Release workflow from GitHub Actions.

## Code of Conduct

This project follows the Contributor Covenant v2.1. See CODE_OF_CONDUCT.md.

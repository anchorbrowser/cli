# AnchorBrowser CLI

AnchorBrowser CLI with `agent-browser` parity UX, backed by AnchorBrowser sessions over CDP.

## Install

### Homebrew (recommended)

```bash
brew tap anchorbrowser/homebrew-tap
brew install anchorbrowser
```

### npm

```bash
npm install -g @anchor-browser/cli
anchorbrowser --help
```

### Build from source

```bash
git clone https://github.com/anchorbrowser/cli.git
cd cli
make build
./bin/anchorbrowser version
```

## Command model

`agent-browser` parity commands are available through the `proxy` command (`anchorbrowser proxy ...`) and are executed through pinned backend (`v0.20.13`).

Anchor API-specific commands are namespaced under `anchor`:

```bash
anchorbrowser anchor session ...
anchorbrowser anchor identity ...
anchorbrowser anchor task ...
```

Operational commands remain top-level:

```bash
anchorbrowser auth ...
anchorbrowser proxy ...
anchorbrowser update
anchorbrowser version
```

Breaking change:

- old top-level `session`, `identity`, `task` commands moved under `anchor`.

## Authentication

API key precedence:

1. `--api-key`
2. `--key <name>`
3. `ANCHORBROWSER_API_KEY`
4. active key stored in OS keychain

```bash
anchorbrowser auth login --name default
anchorbrowser auth keys list
anchorbrowser auth current
```

## Parity usage

Examples (mirroring `agent-browser` style):

```bash
anchorbrowser proxy open https://example.com
anchorbrowser proxy snapshot -i
anchorbrowser proxy click @e1
anchorbrowser proxy fill @e2 "hello"
anchorbrowser proxy screenshot page.png
anchorbrowser proxy close
```

Session bridge behavior for parity commands:

1. hidden `--session-id` if provided,
2. cached latest session,
3. otherwise auto-create a new session.

Auto-created sessions enable recommended anti-bot defaults:

- `session.proxy.active=true` with `type=anchor_proxy`
- `browser.extra_stealth.active=true`
- `browser.captcha_solver.active=true`

For authenticated browsing, pre-create an authenticated Anchor session and then run parity commands:

```bash
anchorbrowser anchor session create --interactive
anchorbrowser proxy open https://your-app.example
```

Power-user flags for parity commands (intentionally hidden):

- `--session-id`
- `--new-session`
- `--no-cache`

## Proxy bootstrap

Backend bootstrap is strict at install time:

- Homebrew install runs `anchorbrowser proxy --help`
- npm postinstall runs `anchorbrowser proxy --help`
- install fails if backend bootstrap fails

Runtime auto-install/self-heal still applies if users manually remove/corrupt backend binaries.

## Anchor namespace commands

```bash
anchorbrowser anchor session create --interactive
anchorbrowser anchor session list
anchorbrowser anchor identity list --application-url https://example.com
anchorbrowser anchor task run <task-id> --input "key=value"
```

## Global flags

```text
--api-key string
--key string
--base-url string
--timeout duration
--output json|yaml
--compact
--dry-run
--verbose
```

## Development

```bash
make generate
make fmt
make lint
make test
make test-race
make vulncheck
make build
make release-check
```

## Release + Homebrew automation

Tagging `v*` triggers GoReleaser (`.github/workflows/release.yml`) to:

- build and publish binaries/checksums
- create/update GitHub release in `anchorbrowser/cli`
- commit formula updates to `anchorbrowser/homebrew-tap`
- publish `@anchor-browser/cli` to npm

Required repo secrets in `anchorbrowser/cli`:

- `HOMEBREW_TAP_GITHUB_TOKEN`
- `NPM_TOKEN`

## Version-driven releases

When `package.json` version changes and merges to `main`, workflow
`.github/workflows/tag-release-from-package.yml` creates and pushes tag `v<version>`.
That tag triggers the release workflow.

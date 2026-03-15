# AnchorBrowser CLI

Go-based CLI for AnchorBrowser sessions, agent tasks, tasks v2, and identities.

## Install

### Homebrew (recommended)

```bash
brew tap anchorbrowser/homebrew-tap
brew install anchorbrowser
```

Shell completions are installed automatically for bash/zsh/fish with Homebrew.

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

## Authentication

The CLI supports API-key auth only.

Credential precedence for every API command:

1. `--api-key`
2. `--key <name>`
3. `ANCHORBROWSER_API_KEY`
4. active key stored in OS keychain

Store a key securely:

```bash
anchorbrowser auth login --name default
```

Manage keys:

```bash
anchorbrowser auth keys list
anchorbrowser auth keys use default
anchorbrowser auth keys rename default prod
anchorbrowser auth keys remove prod
anchorbrowser auth current
```

## Core commands

### Sessions

```bash
anchorbrowser session create --initial-url https://example.com --tag prod
anchorbrowser session create --interactive

anchorbrowser session list --page 1 --limit 20
anchorbrowser session get
anchorbrowser session end
anchorbrowser session end-all
anchorbrowser session pages
anchorbrowser session history --from-date 2026-01-01T00:00:00Z
anchorbrowser session status-all --tags prod
anchorbrowser session downloads
anchorbrowser session recordings
anchorbrowser session recording fetch-primary --out recording.mp4
```

`session create` caches the created session ID by default. Session actions use:
`--session-id` flag > cached latest session.
Set `--no-cache` to force explicit session selection.
When a command targets a cached/selected session, the CLI prints `Using session: <id>`.
Interactive mode (`--interactive`) is exclusive with create payload flags (`--body`, `--initial-url`, proxy/browser/profile flags, identities/integrations).
In a real terminal, interactive mode uses keyboard-driven TUI selection (arrows + type-to-search). In non-TTY contexts, it falls back to plain prompts.

### Session controls (flat)

```bash
anchorbrowser session screenshot --out shot.png
anchorbrowser session click --selector "button.submit"
anchorbrowser session click --x 120 --y 220 --button left
anchorbrowser session double-click --x 100 --y 100
anchorbrowser session mouse-down --x 100 --y 100
anchorbrowser session mouse-up --x 100 --y 100
anchorbrowser session move --x 200 --y 300
anchorbrowser session drag-drop --start-x 10 --start-y 10 --end-x 200 --end-y 200
anchorbrowser session scroll --x 100 --y 100 --delta-y 600
anchorbrowser session type --text "hello"
anchorbrowser session shortcut --keys ctrl,v
anchorbrowser session clipboard get
anchorbrowser session clipboard set --text "copied"
anchorbrowser session copy
anchorbrowser session paste --text "paste me"
anchorbrowser session goto https://anchorbrowser.io
anchorbrowser session goto --url https://anchorbrowser.io
anchorbrowser session upload --file ./document.pdf
```

### Agent run

```bash
anchorbrowser session run-agent --prompt "extract the pricing table" --url https://example.com
anchorbrowser session run-agent --session-id <session-id> --prompt "fill this form" --async
anchorbrowser session run-agent status <workflow-id>
```

### Tasks v2

```bash
anchorbrowser task run <task-id> --input "File Name=invoice.pdf" --input "Operation=extract_text"
anchorbrowser task status <run-id>
```

### Identities

```bash
anchorbrowser identity list --application-url https://example.com
anchorbrowser identity create --source https://example.com/login --username user@example.com --password secret
anchorbrowser identity get <identity-id>
anchorbrowser identity update <identity-id> --name "Updated name"
anchorbrowser identity delete <identity-id>
anchorbrowser identity credentials <identity-id>
anchorbrowser identity credentials <identity-id> --reveal-secrets
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

JSON is default output.

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

Required repo secret in `anchorbrowser/cli`:

- `HOMEBREW_TAP_GITHUB_TOKEN` (write access to `anchorbrowser/homebrew-tap`)
- `NPM_TOKEN` (npm automation token with publish access for `@anchor-browser/cli`)

## Version-driven releases

When `package.json` version changes and is merged to `main`, workflow
`.github/workflows/tag-release-from-package.yml` creates and pushes tag `v<version>`.
That tag triggers the release workflow automatically.

Example:

1. Change `package.json` `"version"` from `0.1.0` to `0.2.0`
2. Merge to `main`
3. Workflow creates tag `v0.2.0`
4. Release workflow publishes GitHub binaries, Homebrew update, and npm package

## npm token setup notes

For npm granular access tokens, ensure token permissions include:

- packages and scopes: read and write
- organization access: `Read and write` for `anchor-browser`

If organization permission is `No access`, npm publish to `@anchor-browser/cli` will fail.

## API references

- [OpenAPI spec](https://docs.anchorbrowser.io/openapi.yaml)
- [Start Browser Session](https://docs.anchorbrowser.io/api-reference/browser-sessions/start-browser-session)
- [Perform Web Task](https://docs.anchorbrowser.io/api-reference/ai-tools/perform-web-task)
- [Run a Task](https://docs.anchorbrowser.io/api-reference/tasks/run-a-task)
- [Identities](https://docs.anchorbrowser.io/api-reference/identities)

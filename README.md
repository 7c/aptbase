# aptbase

A human-friendly command-line client for the [aptly](https://www.aptly.info/)
REST API. `aptbase` drives one or more **remote** aptly servers over HTTP so you
can manage Debian/Ubuntu package repositories from your terminal — upload and
publish packages, inspect what is live, and run day-to-day operations — with
colored output, aligned tables, live task progress, and a config file so you are
not retyping URLs and credentials.

It is a friendly adapter, not a reimplementation: every command maps onto aptly's
own API. The flagship command, `aptbase repo deploy`, takes a `.deb` from your
disk to live-and-verified in a single step.

---

## Installation

### With `go install` (recommended)

Requires Go 1.25+.

```bash
go install github.com/7c/aptbase@latest
```

This builds and installs the `aptbase` binary into your Go bin directory. Make
sure it is on your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"   # add to ~/.zshrc or ~/.bashrc
```

Verify:

```bash
aptbase --version
```

To install a specific release:

```bash
go install github.com/7c/aptbase@v1.0.0
```

### From source

```bash
git clone https://github.com/7c/aptbase.git
cd aptbase
make build        # produces ./bin/aptbase
# or:
make install      # installs into $GOBIN via `go install` with version metadata
```

`make build` injects the version, git commit, and build date via `-ldflags`.
`go install` builds without the Makefile but still reports the module version and
VCS commit embedded by Go.

---

## Quick start

```bash
# 1. Point aptbase at your aptly API and check connectivity
aptbase --api http://localhost:8080 ping

# 2. Scaffold a config file so you stop retyping flags
aptbase config new --path ~/.config/aptbase/config.ini
$EDITOR ~/.config/aptbase/config.ini

# 3. See what aptbase resolved (and where each value came from)
aptbase config list

# 4. Release a package end to end
aptbase repo deploy app-stable ./app_1.2.3_amd64.deb -d noble
```

---

## Configuration

`aptbase` reads settings from layered sources. **Later layers override earlier
ones (last value wins):**

```
built-in defaults
  → /etc/aptbase/config.ini
  → ~/.config/aptbase/config.ini
  → APTBASE_* environment variables
  → command-line flags
```

When `--config <path>` (or `APTBASE_CONFIG`) names an explicit file, only that
file is read instead of the two well-known locations.

Inspect the fully resolved configuration at any time, including the source of
each value and its built-in default:

```bash
aptbase config list
```

```
SETTING        VALUE                          SOURCE                   DEFAULT
─────────────  ─────────────────────────────  ───────────────────────  ───────────────
api            http://localhost:8080          ~/.config/aptbase/...    (none)
distributions  noble jammy focal              ~/.config/aptbase/...    (none)
prefix         .                              default                  .
timeout        1m0s                           default                  1m0s
...
```

### The config file

Generate an annotated starter file:

```bash
aptbase config new                                   # writes /etc/aptbase/config.ini (needs sudo)
aptbase config new --path ./aptbase.ini              # write anywhere
aptbase config new --path ~/.config/aptbase/config.ini --force
```

Example `config.ini`:

```ini
[default]
# One or more aptly API base URLs. Commands fan out to all of them unless
# narrowed with --api or --server. Whitespace- or comma-separated.
api           = http://prod:8080 http://replica:8080

# Basic-auth username and password (both optional — see Authentication).
user          = deploy
# password    = s3cr3t        ; if set here, chmod 600 this file

# Default distributions used when -d/--distribution is omitted.
distributions = noble jammy focal

# Default local repos used when the <repo> argument is omitted.
repos         = app-stable

# Default publish prefix, TLS, timeout, output.
prefix        = .
insecure      = false
timeout       = 60s
json          = false
no-color      = false

# A named server, selected with: aptbase --server staging ...
[server:staging]
api           = http://staging:8080
distributions = noble
```

### Multiple servers (fan-out)

List several URLs under `api` (or pass `--api` repeatedly) and commands run
against **all** of them, with a per-server status line and an aggregate exit
code:

```bash
aptbase --api http://prod:8080 --api http://replica:8080 repo deploy app-stable ./app.deb -d noble
```

Use `[server:NAME]` sections plus `--server NAME` to switch between named
environments.

### Environment variables

Every setting has an `APTBASE_*` counterpart:

| Variable | Setting |
|----------|---------|
| `APTBASE_API` | `api` (space/comma list) |
| `APTBASE_USER` | `user` |
| `APTBASE_PASSWORD` | `password` |
| `APTBASE_DISTRIBUTIONS` | `distributions` |
| `APTBASE_REPOS` | `repos` |
| `APTBASE_PREFIX` | `prefix` |
| `APTBASE_INSECURE` | `insecure` |
| `APTBASE_TIMEOUT` | `timeout` |
| `APTBASE_JSON` | `json` |
| `APTBASE_NO_COLOR` | `no-color` |
| `APTBASE_YES` | `yes` |
| `APTBASE_CONFIG` | explicit config file path |

### Global flags

| Flag | Description |
|------|-------------|
| `--api` (repeatable) | aptly API base URL; fans out to all |
| `--server` | named `[server:NAME]` config section |
| `--user` | HTTP Basic auth username |
| `--password` | HTTP Basic auth password (prompted on 401 if omitted) |
| `--config` | explicit config file path |
| `-d, --distributions` (repeatable) | target distribution |
| `--prefix` | publish prefix (default `.`) |
| `--json` | machine-readable JSON output |
| `--no-color` | disable colored output |
| `--insecure` | skip TLS certificate verification |
| `--timeout` | per-request timeout (default `1m0s`) |
| `--yes` | assume yes; skip destructive-action confirmations |

---

## Authentication

HTTP Basic auth is **optional and server-triggered**:

- If your aptly API needs no auth (trusted network / localhost), do nothing.
- If it sits behind Basic auth and you have **not** supplied a password,
  aptbase prompts for it (hidden input) the first time the server returns `401`.
- To skip the prompt — for automation/CI — supply the password ahead of time via
  any of: `password` in `config.ini`, `APTBASE_PASSWORD`, or `--password`.

```bash
# Interactive: prompts for the password only if the server challenges
aptbase --api https://apt.example.com --user deploy ping

# Non-interactive: credentials supplied up front
aptbase --api https://apt.example.com --user deploy --password "$APT_PW" ping
```

> If you store the password in a config file, restrict it: `chmod 600`. On a
> shared machine, prefer `APTBASE_PASSWORD` or the interactive prompt.

---

## Commands

Run `aptbase <command> --help` for full, example-rich help on any command.

### Connectivity

```bash
aptbase ping             # reach every configured server, print its aptly version
aptbase version          # local build info + remote aptly versions
aptbase --version        # just the version string
```

### Config

```bash
aptbase config new [--path FILE] [--force]   # scaffold an annotated config.ini
aptbase config print                         # resolved config as config.ini to stdout (pipeable)
aptbase config list [--json]                 # resolved settings, sources, defaults

# Capture an ad-hoc invocation as a system config file:
aptbase --api http://aptbase:8080 -d noble config print | sudo tee /etc/aptbase/config.ini
```

### Repositories

```bash
aptbase repo list                            # all local repos (per server)
aptbase repo show app-stable                 # repo details
aptbase repo create app-stable --distribution noble --component main
aptbase repo edit app-stable --comment "stable channel"
aptbase repo drop app-edge [--force] [--yes] # delete (asks to confirm unless --yes)
aptbase repo packages app-stable -q 'nginx (>= 1.20)'
```

### Add packages

Upload one or more `.deb` files and add them to a repo. Runs as an async aptly
task whose progress is streamed live; the temporary upload directory is cleaned
up afterward.

```bash
aptbase repo add app-stable ./app_1.2.3_amd64.deb
aptbase repo add ./a_1_amd64.deb ./b_1_amd64.deb     # repo(s) from config defaults
```

### Deploy (flagship)

`repo deploy` is the one-command release workflow. For each target server it:

1. uploads the `.deb` file(s),
2. adds them to the target repo(s),
3. re-publishes each target distribution so the package goes live, and
4. verifies the package is present in the repo.

Repos default to the configured `repos`; distributions default to the configured
`distributions` (override with `-d`). It fans out across every configured server.

```bash
# Single repo, two distributions
aptbase repo deploy app-stable ./app_1.2.3_amd64.deb -d noble -d jammy

# Use config defaults for repo / distributions / servers
aptbase repo deploy ./app_1.2.3_amd64.deb

# Signed publish
aptbase repo deploy app-stable ./app.deb -d noble --gpg-key DEADBEEF --batch

# Unsigned (lab) publish, no confirmation prompts
aptbase repo deploy app-stable ./app.deb -d noble --skip-signing --yes
```

Deploy and publish accept signing flags: `--gpg-key`, `--keyring`,
`--passphrase`, `--batch`, `--skip-signing`, and `--force-overwrite`. Deploy also
takes `--no-verify` to skip the post-publish check.

### Publish

```bash
aptbase publish list                         # all publications (per server)
aptbase publish update noble                 # re-publish a distribution
aptbase publish update                       # re-publish configured distributions
aptbase publish update noble jammy --gpg-key DEADBEEF --batch
```

### Packages

```bash
aptbase package search 'nginx'               # search configured repos
aptbase package search 'nginx (>= 1.20)' --repo app-stable
aptbase package show 'Pamd64 nginx 1.20.1-1 abc123'
```

aptly has no global package index, so `search` runs against each target repo
(those given with `--repo`, else the configured `repos`). Queries use aptly's
[query syntax](https://www.aptly.info/doc/feature/query/).

### Tasks

aptbase uses aptly's asynchronous task API for long operations and streams output
live. You can also inspect tasks directly:

```bash
aptbase task list                            # tasks on a server
aptbase task show 7                          # a task's state
aptbase task wait 7                          # block until done, streaming output
aptbase task output 7                        # print accumulated output
```

Task IDs are server-local; with multiple servers configured, task subcommands act
on the first one (narrow with `--api`/`--server`).

---

## Output

- Human-readable colored tables by default.
- `--json` on any command emits machine-readable JSON for scripting.
- Color auto-disables when output is piped or `NO_COLOR` is set; force it off with
  `--no-color`.

```bash
aptbase --json publish list | jq '.[].Distribution'
```

---

## Exit codes

`aptbase` exits non-zero when an operation fails. With multiple servers, it
attempts all of them and exits non-zero if **any** failed, after printing a
per-server result.

---

## Development

```bash
make build      # compile to ./bin/aptbase (with version metadata)
make test       # run unit tests
make vet        # go vet
make fmt        # gofmt
make version    # print the current version
make bump       # bump version.txt (PART=patch|minor|major, default patch)
make help       # list all targets
```

### Versioning & releasing

`version.txt` at the repo root is the **single source of truth** for the build
version (a [semver](https://semver.org/) string). `make build` reads it and
injects the version, git commit, and build date into the binary via `-ldflags`,
so `aptbase version` is self-describing on any host. A plain `go build` (without
the Makefile) shows the placeholder `dev`, which is a useful signal that the
binary was not built through the release path.

Release flow:

```bash
make bump PART=minor          # e.g. 0.1.0 -> 0.2.0 (or PART=patch|major)
git commit -am "release v$(cat version.txt)"
git tag "v$(cat version.txt)" && git push --tags
```

Tagging the commit to match `version.txt` keeps `go install
github.com/7c/aptbase@latest` reporting the same version: a `go install` build
cannot read `version.txt`, so it falls back to the version and commit Go embeds
from the module's VCS tag.

The codebase is organized by concern:

```
version.txt         single source of truth for the build version (semver)
cmd/                cobra commands (thin, human-facing)
internal/config/    layered config resolver
internal/client/    typed aptly API client (+ async tasks, 401 auth)
internal/target/    multi-server fan-out
internal/ui/        colored output and tables
internal/render/    human vs JSON rendering
tools/              dev tools (e.g. increaseversion.go); not part of the binary
```

---

## License

[MIT](LICENSE) © 7c

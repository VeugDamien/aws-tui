# aws-tui

A TUI (terminal interface) to manage your AWS connection profiles, list EC2
instances and start SSM sessions (shell or port-forwarding) without leaving the
terminal.

Written in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).
Read-only AWS access (STS, EC2) goes through the AWS SDK for Go v2; login flows and
interactive SSM sessions are delegated to the AWS CLI v2.

## Features

- Lists profiles from `~/.aws/config`, with authentication type detection
  (`sso`, `login`, `cred-process`, `static`).
- Automatic login based on the profile:
  - `aws sso login --sso-session <session>` for SSO profiles,
  - `aws login --profile <profile>` for `login_session` profiles,
  - transparent refresh for `credential_process`.
- SSH session detection (`SSH_TTY`/`SSH_CONNECTION`): adds `--remote` to login
  commands.
- Logout (`aws sso logout` / `aws logout`).
- Region selector (default `eu-west-3`).
- Filterable EC2 instance list (Name, ID, state, type, private IP, AZ), with
  horizontal scrolling for narrow terminals.
- Per-instance detail page (`enter`): general information, security groups
  (inbound/outbound rules), CloudWatch metrics (CPU, network) with an adjustable
  window (1h / 3h / 12h / 24h, `m` key).
- Auto Scaling Groups: list (min/desired/max capacities, healthy instances) and
  detail page (configuration, instances with health, target groups, recent
  activities).
- Load Balancers (ALB / NLB / Gateway): list (type, scheme, state, DNS) and detail
  page (listeners, target groups with target health).
- Interactive SSM shell session.
- SSM port-forwarding, to an instance port or to a remote host through the instance
  (bastion).
- Copy to clipboard (`y`): profile name, instance ID; copy menu (`Y`) on the detail
  page (private/public IP, ARN, VPC, subnet, AMI…).

## Requirements

- **macOS, Linux or Windows (x64)** with a real terminal (TTY).
- **AWS CLI v2** in `PATH` (`aws sso login`, `aws login`, `aws logout`,
  `aws ssm start-session`).
- **session-manager-plugin** for SSM features
  (https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html).
- **Go ≥ 1.26** to build.
- An AWS configuration file (`~/.aws/config`, or `%USERPROFILE%\.aws\config` on
  Windows) with SSO and/or `login` profiles.

## Installation

### Install script (recommended)

Installs the latest version from GitHub Releases (automatic OS and architecture
detection, SHA-256 checksum verification).

**macOS / Linux:**

```sh
curl -fsSL https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.sh | sh
```

Options: `VERSION=v1.2.3` for a specific version, `BINDIR=~/.local/bin` to choose the
install directory. Example:

```sh
curl -fsSL https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.sh | VERSION=v1.2.3 BINDIR="$HOME/.local/bin" sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.ps1 | iex
```

Or with options:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.ps1))) -Version v1.2.3 -InstallDir "C:\Tools\aws-tui"
```

### Manual download

Grab the archive for your platform from the
[Releases page](https://github.com/VeugDamien/aws-tui/releases), extract it and place
the binary in your `PATH`:

```sh
tar -xzf aws-tui_<version>_<os>_<arch>.tar.gz
sudo install -m 0755 aws-tui_<version>_<os>_<arch>/aws-tui /usr/local/bin/
```

On Linux, `.deb` and `.rpm` packages are also available on the Releases page:

```sh
sudo dpkg -i aws-tui_<version>_linux_amd64.deb    # Debian/Ubuntu
sudo rpm -i  aws-tui_<version>_linux_amd64.rpm    # RHEL/Fedora
```

### Build from source

Requires **Go ≥ 1.26**.

```sh
git clone https://github.com/VeugDamien/aws-tui.git
cd aws-tui
make install                 # build and install into /usr/local/bin
# or, for a user prefix:
make install PREFIX="$HOME/.local"
```

Useful `Makefile` targets:

| Command | Effect |
|----------|-------|
| `make build` | build the binary into `./bin` |
| `make install` | install into `$(PREFIX)/bin` (default `/usr/local`) |
| `make uninstall` | remove the installed binary |
| `make cross` | build for all platforms into `./dist` |
| `make check` | `go vet` + tests |
| `make clean` | remove `bin/` and `dist/` |

To produce cross-platform distribution archives (with checksums) without GoReleaser:

```sh
scripts/build.sh              # artifacts in ./dist
VERSION=v1.2.3 scripts/build.sh
```

### Uninstall

```sh
scripts/uninstall.sh                          # macOS / Linux
# or: make uninstall
```

On Windows, remove the install folder (`%LOCALAPPDATA%\aws-tui`) and remove it from
the user `PATH`.

### Manual cross-compilation (one-off)

```sh
GOOS=windows GOARCH=amd64 go build -o aws-tui.exe .
```

On Windows, `aws` and `session-manager-plugin` must be in `PATH`
(`aws.exe`, `session-manager-plugin.exe`). Stopping a tunnel terminates the full
process tree (CLI + plugin) via `taskkill /T /F`.

## Usage

```sh
./aws-tui
```

### Keyboard shortcuts

| Screen | Keys |
|-------|---------|
| Global | `?` help · `q` / `Ctrl+C` quit |
| Config (from any action screen) | `p` change profile · `g` change region · `L` logout |
| Profiles | `↑/↓` navigate · `/` filter · `y` copy name · `enter` select · `esc` cancel |
| Actions | `↑/↓` navigate · `enter` confirm · (`e` EC2 · `a` ASG · `b` Load Balancers · `t` tunnels) |
| Regions | `↑/↓` · `/` filter · `enter` select · `esc` cancel |
| EC2 | `↑/↓` · `←/→` scroll horizontally · `enter` details · `/` filter · `y` copy ID · `r` refresh · `s` SSM shell · `f` port-forward · `t` tunnels · `esc` back |
| EC2 detail | `↑/↓` scroll · `y` copy ID · `Y` copy menu (IP, ARN, VPC…) · `m` metrics window (1h/3h/12h/24h) · `r` refresh · `esc` back |
| Auto Scaling Groups | `↑/↓` navigate · `enter` details · `/` filter · `y` copy name · `r` refresh · `esc` back |
| ASG detail | `↑/↓` scroll · `y` copy name · `r` refresh · `esc` back |
| Load Balancers | `↑/↓` · `←/→` scroll · `enter` details · `/` filter · `y` copy DNS · `r` refresh · `esc` back |
| LB detail | `↑/↓` scroll · `y` copy DNS · `r` refresh · `esc` back |
| Port-forward (form) | `↑/↓` field · `tab` toggle remote host · `enter` start · `esc` cancel |
| Tunnels | `↑/↓` navigate · `x` stop · `X` stop all · `r` restart · `c` clear stopped · `esc` back |

The interface separates **configuration** (persistent top bar: profile, region,
account — editable via `p`/`g`/`L`) from **functional actions** (EC2, tunnels).

### Typical flow

1. Select a profile. If the token is expired, the browser opens for login (SSO or
   `aws login`), then the identity is verified via STS.
2. You land on the **Actions** screen. The config bar at the top always shows the
   active profile / region / account.
3. Switch profile (`p`) or region (`g`) at any time, from any action screen.
4. List EC2 instances (`e`), Auto Scaling Groups (`a`) or Load Balancers (`b`);
   filter (`/`).
5. On an instance:
   - `s` opens an SSM shell (the TUI hands the terminal over, then resumes);
   - `f` opens the port-forward form (the tunnel then runs in the background, visible
     via `t`).

## Architecture

```
main.go                  entry point (AWS CLI pre-flight + Bubble Tea launch)
internal/awsconfig       reads and classifies ~/.aws/config profiles
internal/auth            GetCallerIdentity + building login/logout commands
internal/awsclient       SDK config loading, EC2/ASG/ELB listing, regions
internal/clipboard       copy to clipboard (pbcopy/xclip/clip…)
internal/ssm             building "aws ssm start-session" commands
internal/ui              Bubble Tea model (model/update/view) + screens
scripts/                 build.sh, install.sh, install.ps1, uninstall.sh
.goreleaser.yaml         release config (archives, .deb/.rpm, checksums)
.github/workflows        CI (build/vet/test) and release (GoReleaser on tag)
specs/aws-tui            specifications (requirements, design, tasks)
```

Details are described in `specs/aws-tui/` (spec-driven development:
requirements → design → tasks, with a traceability matrix).

## Tests

```sh
go test ./...
go vet ./...
```

Covered by unit tests: profile classification, profile → login/logout command
mapping (and the `--remote` addition over SSH), and building SSM arguments.

## Publish a release (maintainer)

Binaries and packages are produced by [GoReleaser](https://goreleaser.com) via GitHub
Actions whenever a `vX.Y.Z` tag is pushed:

```sh
git tag v1.0.0
git push origin v1.0.0
```

The `.github/workflows/release.yml` workflow builds all platforms, generates the
archives, checksums and `.deb`/`.rpm` packages, then creates the GitHub Release.
Local test without publishing:

```sh
make release-check           # goreleaser check + snapshot build
# or: goreleaser release --snapshot --clean --skip=publish
```

To enable distribution via **Homebrew** (tap) and **Scoop** (bucket):

1. Create the `VeugDamien/homebrew-tap` and `VeugDamien/scoop-bucket` repositories.
2. Create a GitHub PAT with write access to those repositories and add it to the
   `GORELEASER_TOKEN` secret of the `aws-tui` repository.
3. Uncomment the `brews:` / `scoops:` sections in `.goreleaser.yaml` and the
   `GORELEASER_TOKEN` line in `release.yml`.

## IAM permissions

Depending on the features used, the role/profile must allow:

- `sts:GetCallerIdentity` — identity verification (always).
- `ec2:DescribeInstances` — instance list.
- `iam:ListAccountAliases` — account alias shown in the bar (else the number).
- `ec2:DescribeSecurityGroups` — "Security / network" block of the detail page.
- `cloudwatch:GetMetricData` — "Metrics" block of the detail page.
- `autoscaling:DescribeAutoScalingGroups`, `autoscaling:DescribeScalingActivities` —
  Auto Scaling Groups list and detail page.
- `elasticloadbalancing:DescribeLoadBalancers`, `DescribeListeners`,
  `DescribeTargetGroups`, `DescribeTargetHealth` — Load Balancers list and detail page
  (including target health).
- `ssm:StartSession` (+ SSM document) — shell and port-forward sessions.

Missing permissions for the detail-page blocks are handled gracefully: the affected
block shows an error, the rest stays usable.

## Security

`aws-tui` writes no secret to disk: it reuses the caches managed by the AWS CLI
(`~/.aws/sso/cache`, `~/.aws/login/cache`). Identities are only shown on screen.

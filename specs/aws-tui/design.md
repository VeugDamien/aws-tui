# Design — AWS TUI

Architecture technique répondant à `requirements.md`.

## 1. Vue d'ensemble

`aws-tui` suit l'architecture **Elm** imposée par Bubble Tea : état immuable
(`Model`), messages (`Msg`), `Update(msg) -> (Model, Cmd)`, `View() -> string`.
Les effets de bord sont encapsulés dans des `tea.Cmd` asynchrones.

Séparation des responsabilités :

- **Données AWS** : SDK for Go v2 (STS, EC2, AutoScaling, ELBv2, IAM, CloudWatch).
- **Authentification interactive** : exec de l'AWS CLI (`aws sso login`, etc.).
- **Sessions SSM** : exec de `aws ssm start-session` (délègue au plugin).
- **Self-update** : API GitHub Releases + vérification SHA-256 + remplacement atomique.
- **UI** : Bubble Tea / Bubbles / Lipgloss, découplée de la logique métier.

## 2. Arborescence des paquets

```
main.go                        Point d'entrée, flags (--version, --help, upgrade,
                               check-update), variables ldflags (version/commit/date).

internal/auth/                 Authentification.
  auth.go                        Whoami(), LoginCommand(), LogoutCommand(), NeedsLogin().

internal/awsclient/            Accès AWS via SDK v2.
  factory.go                     LoadConfig(profile, region).
  ec2.go                         ListInstances(), Instance model.
  securitygroups.go              DescribeSecurityGroups(), SGRule model.
  metrics.go                     GetInstanceMetrics(), InstanceMetrics model.
  asg.go                         ListAutoScalingGroups(), DescribeScalingActivities(),
                                 DescribeTargetGroupsByARNs(). Models: AutoScalingGroup,
                                 ASGInstance, ScalingActivity, TargetGroup, TargetHealth.
  elb.go                         ListLoadBalancers(), DescribeListeners(),
                                 DescribeTargetGroupsForLB(). Models: LoadBalancer, Listener.
  iam.go                         AccountAlias().
  regions.go                     Regions() (liste statique).

internal/awsconfig/            Lecture et classification des profils.
  profiles.go                    Profile, AuthKind, LoadProfiles().
  groups.go                      Groupement des profils par source.

internal/clipboard/            Copie presse-papiers (pbcopy/xclip/clip).
  clipboard.go                   Copy(text).

internal/selfupdate/           Mise à jour automatique.
  selfupdate.go                  Upgrade(), CheckUpdate(), managedBy().
  github.go                      LatestRelease(), Release/Asset models.
  download.go                    downloadToTemp(), verifyChecksum(), fileSHA256().
  extract.go                     extractBinary() (tar.gz + zip, protection taille).
  replace_unix.go                replaceExecutable() (rename atomique).
  replace_windows.go             replaceExecutable() (rename + .old).
  version.go                     semver parsing, IsNewer().

internal/ssm/                  Construction des commandes SSM.
  session.go                     ShellCommand(), PortForwardCommand(),
                                 PortForwardRemoteHostCommand(), PluginAvailable().

internal/state/                Persistance utilisateur.
  state.go                       Load(), Save() (JSON atomique, 0600).

internal/ui/                   Couche présentation.
  model.go                       AppModel, Screen enum, New().
  update.go                      Update() : routing complet.
  view.go                        View() : rendu par écran + footer hints.
  cmd.go                         tea.Cmd wrappers (whoami, list, detail, tunnel…).
  msg.go                         Types de messages internes.
  keys.go                        Keymap + help integration.
  styles.go                      Styles Lipgloss.
  buffer.go                      safeBuffer (goroutine-safe).
  tunnels.go                     tunnelManager, Tunnel lifecycle.
  process_unix.go                setProcessGroup, killProcessGroup (Unix).
  process_windows.go             setProcessGroup, killProcessGroup (Windows).
  screens/
    profiles.go                  Liste des profils (nom, auth, région).
    ec2list.go                   Table EC2 filtrable.
    ec2detail.go                 Fiche EC2 (infos, SG, métriques, tags).
    asglist.go                   Table ASG filtrable.
    asgdetail.go                 Fiche ASG (capacités, instances, TGs, activités).
    elblist.go                   Table ELB filtrable + scroll horizontal.
    elbdetail.go                 Fiche ELB (listeners, TGs, santé cibles).
    regions.go                   Sélecteur de région.
    portforward.go               Formulaire port-forward.
    format.go                    Helpers de formatage.
```

## 3. Dépendances externes

| Package | Usage |
|---------|-------|
| `github.com/charmbracelet/bubbletea` | Moteur TUI |
| `github.com/charmbracelet/bubbles` | Composants (list, spinner, help, textinput) |
| `github.com/charmbracelet/lipgloss` | Styles |
| `github.com/aws/aws-sdk-go-v2/*` | config, sts, ec2, autoscaling, elbv2, iam, cloudwatch |
| `gopkg.in/ini.v1` | Parsing `~/.aws/config` |
| `github.com/atotto/clipboard` | (indirect, via bubbles) |

## 4. Modèles de données principaux

### Profile (`internal/awsconfig`)
```go
type Profile struct {
    Name              string
    AuthKind          AuthKind  // SSO, Login, CredentialProcess, Static, Unknown
    Region            string
    SSOSession        string
    SourceProfile     string
    CredentialProcess string
}
```

### Instance (`internal/awsclient`)
```go
type Instance struct {
    Name, InstanceID, State, Type, PrivateIP, PublicIP string
    AZ, Platform, VPCID, SubnetID, AMI, KeyName, IAMRole string
    LaunchTime string
    SGIDs, SGNames []string
    Tags map[string]string
}
```

### AutoScalingGroup
```go
type AutoScalingGroup struct {
    Name string
    MinSize, MaxSize, DesiredCapacity int32
    HealthCheckType, LaunchType, LaunchName string
    AZs, Subnets, TargetGroupARNs []string
    Instances []ASGInstance
    CreatedTime string
}
```

### LoadBalancer
```go
type LoadBalancer struct {
    Name, ARN, Type, Scheme, State, DNSName, VPCID string
    AZs []string
    CreatedTime string
}
```

## 5. Navigation UI

```
ScreenProfiles ──select──> Whoami ──ok──> ScreenActions
                                  └──auth requise──> ExecProcess(login) ──> retry

ScreenActions:
  e → ScreenEC2            (liste instances)
  a → ScreenASG            (liste ASG)
  b → ScreenELB            (liste Load Balancers)
  t → ScreenTunnels        (tunnels actifs)
  p → ScreenProfiles       (changer de profil)
  g → ScreenRegions        (changer de région)
  L → logout

ScreenEC2 ──enter──> ScreenEC2Detail
ScreenEC2 ──s──> ExecProcess(shell) ──> retour EC2
ScreenEC2 ──f──> ScreenPortForwardForm ──submit──> tunnel started

ScreenASG ──enter──> ScreenASGDetail
ScreenELB ──enter──> ScreenELBDetail

Partout: q/Ctrl+C → quit (arrêt tunnels)
```

## 6. Tunnels (port-forward en arrière-plan)

- Chaque tunnel = `exec.CommandContext` avec process group dédié.
- `tunnelManager` : registre thread-safe (mutex + slice de `*Tunnel`).
- Arrêt : `killProcessGroup` envoie SIGTERM+SIGKILL (Unix) ou `taskkill /T /F` (Win).
- Écran `ScreenTunnels` : liste avec état, sortie capturée, arrêt individuel.

## 7. Self-update

1. `LatestRelease()` → API GitHub `/releases/latest`.
2. `IsNewer(current, latest)` → comparaison semver.
3. `downloadToTemp()` → archive (LimitReader 200 MiB).
4. `verifyChecksum()` → télécharge `checksums.txt`, compare SHA-256.
5. `extractBinary()` → tar.gz ou zip, recherche par basename (protection zip-slip),
   limite 100 MiB.
6. `replaceExecutable()` → rename atomique (Unix) ou rename+old (Windows).
7. `managedBy()` → refuse si chemin Homebrew/apt/Scoop.

## 8. Sécurité

- Aucun secret stocké (REQ-39, NFR-5).
- Commandes externes via `exec.Command` (tableau d'args, pas de shell).
- Self-update : SHA-256 vérifié, taille bornée, recherche par basename (pas de path
  traversal). Signature crypto à ajouter (backlog S1).
- État persisté en 0600, écriture atomique (tmp+rename).

## 9. Build & distribution

- `Makefile` : build, cross, test, vet, install, clean.
- `scripts/build.sh` : cross-compilation + archives + checksums.
- `scripts/install.sh` : installation Unix (`curl | sh`), checksum SHA-256.
- `scripts/install.ps1` : installation Windows (PowerShell).
- `.goreleaser.yaml` : GoReleaser v2, archives, checksums, .deb/.rpm (nfpm),
  Homebrew/Scoop préparés (commentés).
- `.github/workflows/ci.yml` : build + vet + test + cross-compile.
- `.github/workflows/release.yml` : GoReleaser sur tag `v*`.

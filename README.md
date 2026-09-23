# aws-tui

TUI (interface terminal) pour gérer ses profils de connexion AWS, lister les
instances EC2 et lancer des sessions SSM (shell ou port-forwarding) sans quitter le
terminal.

Écrit en Go avec [Bubble Tea](https://github.com/charmbracelet/bubbletea). L'accès
AWS en lecture (STS, EC2) passe par l'AWS SDK for Go v2 ; les flux de connexion et les
sessions SSM interactives sont délégués à l'AWS CLI v2.

## Fonctionnalités

- Liste des profils depuis `~/.aws/config`, avec détection du type d'authentification
  (`sso`, `login`, `cred-process`, `static`).
- Connexion automatique selon le profil :
  - `aws sso login --sso-session <session>` pour les profils SSO,
  - `aws login --profile <profil>` pour les profils `login_session`,
  - refresh transparent pour `credential_process`.
- Détection de session SSH (`SSH_TTY`/`SSH_CONNECTION`) : ajoute `--remote` aux
  commandes de login.
- Déconnexion (`aws sso logout` / `aws logout`).
- Sélecteur de région (par défaut `eu-west-3`).
- Liste des instances EC2 filtrable (Name, ID, état, type, IP privée, AZ), avec
  défilement horizontal pour les terminaux étroits.
- Fiche détaillée par instance (`enter`) : informations générales, security groups
  (règles entrantes/sortantes), métriques CloudWatch (CPU, réseau) avec fenêtre
  réglable (1h / 3h / 12h / 24h, touche `m`).
- Auto Scaling Groups : liste (capacités min/désirée/max, instances saines) et fiche
  détaillée (configuration, instances avec santé, target groups, activités récentes).
- Load Balancers (ALB / NLB / Gateway) : liste (type, schéma, état, DNS) et fiche
  détaillée (listeners, target groups avec santé des cibles).
- Session shell SSM interactive.
- Port-forwarding SSM, vers un port de l'instance ou vers un hôte distant via
  l'instance (bastion).
- Copie vers le presse-papiers (`y`) : nom de profil, ID d'instance ; menu de copie
  (`Y`) sur la fiche détaillée (IP privée/publique, ARN, VPC, subnet, AMI…).

## Pré-requis

- **macOS, Linux ou Windows (x64)** avec un vrai terminal (TTY).
- **AWS CLI v2** dans le `PATH` (`aws sso login`, `aws login`, `aws logout`,
  `aws ssm start-session`).
- **session-manager-plugin** pour les fonctions SSM
  (https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html).
- **Go ≥ 1.26** pour compiler.
- Un fichier de configuration AWS (`~/.aws/config`, ou `%USERPROFILE%\.aws\config`
  sous Windows) avec des profils SSO et/ou `login`.

## Installation

### Script d'installation (recommandé)

Installe la dernière version depuis les Releases GitHub (détection automatique de
l'OS et de l'architecture, vérification du checksum SHA-256).

**macOS / Linux :**

```sh
curl -fsSL https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.sh | sh
```

Options : `VERSION=v1.2.3` pour une version précise, `BINDIR=~/.local/bin` pour
choisir le répertoire d'installation. Exemple :

```sh
curl -fsSL https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.sh | VERSION=v1.2.3 BINDIR="$HOME/.local/bin" sh
```

**Windows (PowerShell) :**

```powershell
irm https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.ps1 | iex
```

Ou avec des options :

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/VeugDamien/aws-tui/main/scripts/install.ps1))) -Version v1.2.3 -InstallDir "C:\Tools\aws-tui"
```

### Téléchargement manuel

Récupérez l'archive de votre plateforme depuis la
[page des Releases](https://github.com/VeugDamien/aws-tui/releases), décompressez-la
et placez le binaire dans votre `PATH` :

```sh
tar -xzf aws-tui_<version>_<os>_<arch>.tar.gz
sudo install -m 0755 aws-tui_<version>_<os>_<arch>/aws-tui /usr/local/bin/
```

Sous Linux, des paquets `.deb` et `.rpm` sont aussi disponibles sur la page des
Releases :

```sh
sudo dpkg -i aws-tui_<version>_linux_amd64.deb    # Debian/Ubuntu
sudo rpm -i  aws-tui_<version>_linux_amd64.rpm    # RHEL/Fedora
```

### Compilation depuis les sources

Nécessite **Go ≥ 1.26**.

```sh
git clone https://github.com/VeugDamien/aws-tui.git
cd aws-tui
make install                 # compile et installe dans /usr/local/bin
# ou, pour un préfixe utilisateur :
make install PREFIX="$HOME/.local"
```

Cibles utiles du `Makefile` :

| Commande | Effet |
|----------|-------|
| `make build` | compile le binaire dans `./bin` |
| `make install` | installe dans `$(PREFIX)/bin` (défaut `/usr/local`) |
| `make uninstall` | retire le binaire installé |
| `make cross` | compile pour toutes les plateformes dans `./dist` |
| `make check` | `go vet` + tests |
| `make clean` | supprime `bin/` et `dist/` |

Pour produire les archives de distribution multi-plateforme (avec checksums), sans
GoReleaser :

```sh
scripts/build.sh              # artefacts dans ./dist
VERSION=v1.2.3 scripts/build.sh
```

### Désinstallation

```sh
scripts/uninstall.sh                          # macOS / Linux
# ou : make uninstall
```

Sous Windows, supprimez le dossier d'installation (`%LOCALAPPDATA%\aws-tui`) et
retirez-le du `PATH` utilisateur.

### Compilation croisée manuelle (ponctuelle)

```sh
GOOS=windows GOARCH=amd64 go build -o aws-tui.exe .
```

Sous Windows, `aws` et `session-manager-plugin` doivent être dans le `PATH`
(`aws.exe`, `session-manager-plugin.exe`). L'arrêt d'un tunnel termine l'arbre de
processus complet (CLI + plugin) via `taskkill /T /F`.

## Utilisation

```sh
./aws-tui
```

### Raccourcis clavier

| Écran | Touches |
|-------|---------|
| Global | `?` aide · `q` / `Ctrl+C` quitter |
| Config (depuis tout écran d'action) | `p` changer de profil · `g` changer de région · `L` logout |
| Profils | `↑/↓` naviguer · `/` filtrer · `y` copier le nom · `enter` sélectionner · `esc` annuler |
| Actions | `↑/↓` naviguer · `enter` valider · (`e` EC2 · `a` ASG · `b` Load Balancers · `t` tunnels) |
| Régions | `↑/↓` · `/` filtrer · `enter` choisir · `esc` annuler |
| EC2 | `↑/↓` · `←/→` défiler horizontalement · `enter` détails · `/` filtrer · `y` copier l'ID · `r` rafraîchir · `s` shell SSM · `f` port-forward · `t` tunnels · `esc` retour |
| Fiche EC2 | `↑/↓` défiler · `y` copier l'ID · `Y` menu de copie (IP, ARN, VPC…) · `m` fenêtre métriques (1h/3h/12h/24h) · `r` rafraîchir · `esc` retour |
| Auto Scaling Groups | `↑/↓` naviguer · `enter` détails · `/` filtrer · `y` copier le nom · `r` rafraîchir · `esc` retour |
| Fiche ASG | `↑/↓` défiler · `y` copier le nom · `r` rafraîchir · `esc` retour |
| Load Balancers | `↑/↓` · `←/→` défiler · `enter` détails · `/` filtrer · `y` copier le DNS · `r` rafraîchir · `esc` retour |
| Fiche LB | `↑/↓` défiler · `y` copier le DNS · `r` rafraîchir · `esc` retour |
| Port-forward (form) | `↑/↓` champ · `tab` basculer hôte distant · `enter` lancer · `esc` annuler |
| Tunnels | `↑/↓` naviguer · `x` arrêter · `X` tout arrêter · `esc` retour |

L'interface sépare la **configuration** (barre persistante en haut : profil, région,
compte — modifiable via `p`/`g`/`L`) des **actions fonctionnelles** (EC2, tunnels).

### Déroulé typique

1. Sélectionnez un profil. Si le token est expiré, le navigateur s'ouvre pour la
   connexion (SSO ou `aws login`), puis l'identité est vérifiée via STS.
2. Vous arrivez sur l'écran **Actions**. La barre de config en haut montre toujours
   le profil / la région / le compte actifs.
3. Changez de profil (`p`) ou de région (`g`) à tout moment, depuis n'importe quel
   écran d'action.
4. Listez les instances EC2 (`e`), les Auto Scaling Groups (`a`) ou les Load
   Balancers (`b`) ; filtrez (`/`).
5. Sur une instance :
   - `s` ouvre un shell SSM (le TUI rend la main au shell, puis reprend) ;
   - `f` ouvre le formulaire de port-forward (le tunnel tourne ensuite en arrière-plan,
     visible via `t`).

## Architecture

```
main.go                  point d'entrée (pré-vol AWS CLI + lancement Bubble Tea)
internal/awsconfig       lecture et classification des profils ~/.aws/config
internal/auth            GetCallerIdentity + construction des commandes login/logout
internal/awsclient       chargement de config SDK, listing EC2/ASG/ELB, régions
internal/clipboard       copie vers le presse-papiers (pbcopy/xclip/clip…)
internal/ssm             construction des commandes "aws ssm start-session"
internal/ui              modèle Bubble Tea (model/update/view) + écrans
scripts/                 build.sh, install.sh, install.ps1, uninstall.sh
.goreleaser.yaml         configuration de release (archives, .deb/.rpm, checksums)
.github/workflows        CI (build/vet/test) et release (GoReleaser sur tag)
specs/aws-tui            spécifications (requirements, design, tasks)
```

Les détails sont décrits dans `specs/aws-tui/` (développement piloté par la
spécification : exigences → design → tâches, avec matrice de traçabilité).

## Tests

```sh
go test ./...
go vet ./...
```

Sont couverts par des tests unitaires : la classification des profils, le mapping
profil → commande de login/logout (et l'ajout de `--remote` en SSH), et la
construction des arguments SSM.

## Publier une release (mainteneur)

Les binaires et paquets sont produits par [GoReleaser](https://goreleaser.com) via
GitHub Actions dès qu'un tag `vX.Y.Z` est poussé :

```sh
git tag v1.0.0
git push origin v1.0.0
```

Le workflow `.github/workflows/release.yml` compile toutes les plateformes, génère
les archives, les checksums et les paquets `.deb`/`.rpm`, puis crée la Release
GitHub. Test local sans publier :

```sh
make release-check           # goreleaser check + build snapshot
# ou : goreleaser release --snapshot --clean --skip=publish
```

Pour activer la distribution via **Homebrew** (tap) et **Scoop** (bucket) :

1. Créez les dépôts `VeugDamien/homebrew-tap` et `VeugDamien/scoop-bucket`.
2. Créez un PAT GitHub avec accès en écriture à ces dépôts et ajoutez-le au secret
   `GORELEASER_TOKEN` du dépôt `aws-tui`.
3. Décommentez les sections `brews:` / `scoops:` dans `.goreleaser.yaml` et la ligne
   `GORELEASER_TOKEN` dans `release.yml`.

## Permissions IAM

Selon les fonctionnalités utilisées, le rôle/profil doit autoriser :

- `sts:GetCallerIdentity` — vérification d'identité (toujours).
- `ec2:DescribeInstances` — liste des instances.
- `iam:ListAccountAliases` — alias de compte affiché dans la barre (sinon le numéro).
- `ec2:DescribeSecurityGroups` — bloc « Sécurité / réseau » de la fiche détaillée.
- `cloudwatch:GetMetricData` — bloc « Métriques » de la fiche détaillée.
- `autoscaling:DescribeAutoScalingGroups`, `autoscaling:DescribeScalingActivities` —
  liste et fiche des Auto Scaling Groups.
- `elasticloadbalancing:DescribeLoadBalancers`, `DescribeListeners`,
  `DescribeTargetGroups`, `DescribeTargetHealth` — liste et fiche des Load Balancers
  (dont la santé des cibles).
- `ssm:StartSession` (+ document SSM) — sessions shell et port-forward.

Les permissions manquantes pour les blocs de la fiche détaillée sont gérées
gracieusement : le bloc concerné affiche une erreur, le reste reste utilisable.

## Sécurité

`aws-tui` n'écrit aucun secret sur le disque : il réutilise les caches gérés par
l'AWS CLI (`~/.aws/sso/cache`, `~/.aws/login/cache`). Les identités ne sont affichées
qu'à l'écran.

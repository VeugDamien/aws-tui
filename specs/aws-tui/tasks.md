# Tasks — AWS TUI

Backlog tracé. Chaque tâche référence les exigences (`REQ-*`) de `requirements.md`.

Statut : `[x]` done · `[ ]` to do

---

## Lot 1 — Profils & authentification

- [x] **T1.1** `awsconfig/profiles.go` : LoadProfiles() + classification AuthKind + SourceProfile. — REQ-1
- [x] **T1.2** Tests unitaires classification (fixtures ini). — REQ-1
- [x] **T1.3** `awsconfig/groups.go` : groupement par source profile. — REQ-1
- [x] **T1.4** `ui/screens/profiles.go` : liste profils (nom, type, région). — REQ-1
- [x] **T1.5** Sélection profil → profil actif + transition écran. — REQ-2
- [x] **T1.6** Gestion config absente/illisible (erreur explicite). — REQ-3
- [x] **T1.7** `state/state.go` : persistance dernier profil (JSON, 0600, atomique). — REQ-4
- [x] **T1.8** Restauration du dernier profil au démarrage. — REQ-4
- [x] **T1.9** `awsclient/factory.go` : LoadConfig(profile, region). — REQ-5
- [x] **T1.10** `auth/auth.go` : Whoami() + timeout 8s. — REQ-5
- [x] **T1.11** `auth.LoginCommand()` / `LogoutCommand()` + détection SSH. — REQ-6, REQ-8, REQ-9
- [x] **T1.12** Tests de table LoginCommand/LogoutCommand. — REQ-6, REQ-9
- [x] **T1.13** `cmd.go` : whoamiCmd (distingue authRequired vs erreur). — REQ-6, REQ-7
- [x] **T1.14** loginCmd via tea.ExecProcess + retry Whoami. — REQ-6, REQ-7
- [x] **T1.15** logoutCmd + retour écran profils. — REQ-8
- [x] **T1.16** accountAliasCmd (IAM ListAccountAliases). — REQ-10

## Lot 2 — Région

- [x] **T2.1** `awsclient/regions.go` : liste statique. — REQ-11
- [x] **T2.2** `ui/screens/regions.go` : sélecteur avec filtre. — REQ-11
- [x] **T2.3** Invalidation cache (instances, asgs, lbs) au changement. — REQ-12

## Lot 3 — EC2

- [x] **T3.1** `awsclient/ec2.go` : ListInstances paginé → []Instance enrichi. — REQ-13
- [x] **T3.2** `ui/screens/ec2list.go` : table filtrable (Name, ID, State, Type, IP, AZ). — REQ-13, REQ-14
- [x] **T3.3** Filtre texte interactif (`/`). — REQ-14
- [x] **T3.4** Rafraîchissement (`r`) + spinner async. — REQ-15
- [x] **T3.5** `ui/screens/ec2detail.go` : fiche 2 colonnes (infos, SGs, métriques, tags). — REQ-16
- [x] **T3.6** `awsclient/securitygroups.go` : DescribeSecurityGroups + flatten rules. — REQ-16
- [x] **T3.7** `awsclient/metrics.go` : GetInstanceMetrics (CloudWatch). — REQ-16
- [x] **T3.8** Fenêtre métriques réglable (touche `m`). — REQ-16
- [x] **T3.9** Blocs async indépendants (SG + métriques). — REQ-17
- [x] **T3.10** Tests unitaires securitygroups. — REQ-16

## Lot 4 — Auto Scaling Groups

- [x] **T4.1** `awsclient/asg.go` : ListAutoScalingGroups + modèles. — REQ-18
- [x] **T4.2** `awsclient/asg.go` : DescribeScalingActivities. — REQ-20
- [x] **T4.3** `awsclient/asg.go` : DescribeTargetGroupsByARNs (avec TargetHealth). — REQ-20
- [x] **T4.4** `ui/screens/asglist.go` : table filtrable. — REQ-18, REQ-19
- [x] **T4.5** `ui/screens/asgdetail.go` : fiche (capacités, instances, TGs, activités). — REQ-20
- [x] **T4.6** Blocs async (TGs + activités). — REQ-21
- [x] **T4.7** Tests unitaires comptages ASG + TargetGroup. — REQ-18

## Lot 5 — Load Balancers

- [x] **T5.1** `awsclient/elb.go` : ListLoadBalancers + modèles. — REQ-22
- [x] **T5.2** `awsclient/elb.go` : DescribeListeners, DescribeTargetGroupsForLB. — REQ-24
- [x] **T5.3** `ui/screens/elblist.go` : table filtrable + scroll horizontal. — REQ-22, REQ-23
- [x] **T5.4** `ui/screens/elbdetail.go` : fiche (listeners, TGs, santé cibles). — REQ-24
- [x] **T5.5** Blocs async (listeners + targets). — REQ-25

## Lot 6 — SSM

- [x] **T6.1** `ssm/session.go` : PluginAvailable(), CLIAvailable(). — REQ-29
- [x] **T6.2** ShellCommand() + ShellArgs(). — REQ-26
- [x] **T6.3** PortForwardCommand() + PortForwardRemoteHostCommand(). — REQ-27
- [x] **T6.4** shellCmd via tea.ExecProcess + reprise TUI. — REQ-26
- [x] **T6.5** `ui/screens/portforward.go` : formulaire (ports, host, validation). — REQ-27
- [x] **T6.6** startTunnelCmd + waitTunnelCmd (arrière-plan). — REQ-28
- [x] **T6.7** `ui/tunnels.go` : tunnelManager (add, stop, stopAll, list). — REQ-28
- [x] **T6.8** process_unix.go + process_windows.go (process group). — REQ-28
- [x] **T6.9** Écran ScreenTunnels (liste, arrêt). — REQ-28
- [x] **T6.10** Pré-vol plugin absent → erreur explicite. — REQ-29
- [x] **T6.11** Tests SSM argv. — REQ-27

## Lot 7 — Clipboard

- [x] **T7.1** `clipboard/clipboard.go` : Copy() multi-outil (pbcopy/xclip/clip). — REQ-30
- [x] **T7.2** copyCmd + clipboardMsg + confirmation status bar. — REQ-30
- [x] **T7.3** Menu copie multi-champ sur EC2 detail (`Y`). — REQ-31
- [x] **T7.4** Tests clipboard. — REQ-30

## Lot 8 — Self-update

- [x] **T8.1** `selfupdate/version.go` : semver parsing + IsNewer(). — REQ-32
- [x] **T8.2** `selfupdate/github.go` : LatestRelease() + findAsset(). — REQ-32
- [x] **T8.3** `selfupdate/download.go` : downloadToTemp() + verifyChecksum(). — REQ-33
- [x] **T8.4** `selfupdate/extract.go` : extractBinary() (tar.gz + zip). — REQ-33
- [x] **T8.5** `selfupdate/replace_unix.go` + `replace_windows.go`. — REQ-33
- [x] **T8.6** `selfupdate/selfupdate.go` : Upgrade() + CheckUpdate() + managedBy(). — REQ-33, REQ-34
- [x] **T8.7** `main.go` : sous-commandes upgrade, check-update, --version. — REQ-32, REQ-35

## Lot 9 — Distribution & build

- [x] **T9.1** `main.go` : injection version/commit/date via ldflags. — REQ-35
- [x] **T9.2** `Makefile` : build, cross, test, vet, install, clean. — NFR-3
- [x] **T9.3** `scripts/build.sh` : cross-compilation + archives + checksums. — NFR-3
- [x] **T9.4** `scripts/install.sh` : install Unix (curl|sh, checksum SHA-256). — NFR-3
- [x] **T9.5** `scripts/install.ps1` : install Windows (PowerShell). — NFR-3
- [x] **T9.6** `scripts/uninstall.sh`. — NFR-3
- [x] **T9.7** `.goreleaser.yaml` : builds, archives, checksums, nfpm, brew/scoop commentés. — NFR-3
- [x] **T9.8** `.github/workflows/ci.yml` : build + vet + test + cross. — NFR-3
- [x] **T9.9** `.github/workflows/release.yml` : GoReleaser sur tag. — NFR-3

## Lot 10 — UI transverse

- [x] **T10.1** Écran Actions (menu principal post-auth). — REQ-36
- [x] **T10.2** Footer hints contextuels par écran. — REQ-36
- [x] **T10.3** Quit propre (stopAll tunnels). — REQ-37
- [x] **T10.4** Affichage erreurs non-bloquant (errMsg → status bar). — REQ-38
- [x] **T10.5** Invalidation cache au changement de profil (instances, asgs, lbs). — REQ-12
- [x] **T10.6** Spinner + loading message pendant les appels async. — NFR-1

## Lot 11 — À faire (backlog)

- [ ] **T11.1** Signer les releases (cosign/GPG) + vérification dans selfupdate. — backlog S1
- [ ] **T11.2** Valider le champ host du formulaire port-forward (regex). — backlog S2
- [ ] **T11.3** Pinner GitHub Actions par SHA. — backlog S3
- [ ] **T11.4** Ajouter govulncheck à la CI. — backlog S4
- [ ] **T11.5** Contrôle de redirection dans selfupdate (vérifier domaine). — backlog S5
- [ ] **T11.6** Notification update dispo au lancement (async, non-bloquant). — backlog F1
- [ ] **T11.7** Raccourci copie multi-champ pour ASG/ELB detail. — backlog F7
- [ ] **T11.8** Indicateur tunnels actifs dans la status bar. — backlog F8
- [ ] **T11.9** golangci-lint dans la CI. — backlog Q3
- [ ] **T11.10** Dependabot (.github/dependabot.yml). — backlog Q5
- [ ] **T11.11** Refactor update.go (extraire par screen). — backlog Q6
- [ ] **T11.12** Tests d'intégration avec mocks SDK (interfaces). — backlog Q1
- [ ] **T11.13** RDS : liste instances/clusters + fiche. — backlog F3
- [ ] **T11.14** Export données (JSON/CSV). — backlog F5

---

## Matrice de couverture (REQ → tâches)

| REQ | Tâches |
|-----|--------|
| REQ-1 | T1.1–T1.4, T1.6 |
| REQ-2 | T1.5 |
| REQ-3 | T1.6 |
| REQ-4 | T1.7, T1.8 |
| REQ-5 | T1.9, T1.10 |
| REQ-6 | T1.11–T1.14 |
| REQ-7 | T1.13, T1.14 |
| REQ-8 | T1.11, T1.15 |
| REQ-9 | T1.11, T1.12 |
| REQ-10 | T1.16 |
| REQ-11 | T2.1, T2.2 |
| REQ-12 | T2.3, T10.5 |
| REQ-13 | T3.1, T3.2 |
| REQ-14 | T3.2, T3.3 |
| REQ-15 | T3.4 |
| REQ-16 | T3.5–T3.8, T3.10 |
| REQ-17 | T3.9 |
| REQ-18 | T4.1, T4.4, T4.7 |
| REQ-19 | T4.4 |
| REQ-20 | T4.2, T4.3, T4.5 |
| REQ-21 | T4.6 |
| REQ-22 | T5.1, T5.3 |
| REQ-23 | T5.3 |
| REQ-24 | T5.2, T5.4 |
| REQ-25 | T5.5 |
| REQ-26 | T6.2, T6.4 |
| REQ-27 | T6.3, T6.5, T6.11 |
| REQ-28 | T6.6–T6.9 |
| REQ-29 | T6.1, T6.10 |
| REQ-30 | T7.1, T7.2, T7.4 |
| REQ-31 | T7.3 |
| REQ-32 | T8.1, T8.2, T8.7 |
| REQ-33 | T8.3–T8.6 |
| REQ-34 | T8.6 |
| REQ-35 | T8.7, T9.1 |
| REQ-36 | T10.1, T10.2 |
| REQ-37 | T10.3 |
| REQ-38 | T10.4 |
| REQ-39 | (by design — pas de tâche dédiée, couvert transversalement) |
| NFR-1 | T10.6 |
| NFR-2 | T9.2, T9.3 |
| NFR-3 | T9.1–T9.9 |
| NFR-4 | T1.6, T6.10, T10.4 |
| NFR-5 | T8.3 |
| NFR-6 | (architecture — couvert par la structure des paquets) |

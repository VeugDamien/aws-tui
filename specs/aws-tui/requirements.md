# Requirements — AWS TUI

## 1. Contexte et objectif

`aws-tui` est une application terminal (TUI) mono-binaire écrite en Go, destinée à
faciliter la gestion quotidienne de plusieurs profils AWS et l'accès aux ressources
d'infrastructure (EC2, Auto Scaling Groups, Load Balancers) via une interface
clavier unifiée, avec sessions SSM intégrées.

L'utilisateur cible gère plusieurs comptes AWS via AWS IAM Identity Center (SSO),
`aws login`, ou des credential processes. Il souhaite, sans quitter le terminal :

- voir et basculer entre ses profils,
- s'authentifier avec la bonne méthode selon le type de profil,
- changer de région,
- lister et inspecter les instances EC2, ASG et Load Balancers,
- ouvrir des sessions shell SSM ou des port-forwards,
- maintenir l'outil à jour via un mécanisme de self-update intégré.

## 2. Périmètre

### Dans le périmètre (v2 — état actuel)

- Lecture et classification des profils depuis `~/.aws/config`.
- Login/logout délégués à l'AWS CLI (`aws sso login`, `aws login`).
- Vérification d'identité via STS `GetCallerIdentity`.
- Résolution de l'alias de compte IAM.
- Sélecteur de région.
- Listing EC2 avec filtre, copie, et fiche détaillée (security groups, métriques
  CloudWatch, tags).
- Listing Auto Scaling Groups avec fiche détaillée (capacités, instances, target
  groups, activités de scaling).
- Listing Load Balancers (ALB/NLB/GWLB) avec fiche détaillée (listeners, target
  groups, santé des cibles).
- SSM : session shell interactive + port-forward (instance et remote host).
- Gestion de tunnels en arrière-plan (liste, arrêt).
- Clipboard (copie d'ID, nom, DNS, IP, ARN).
- Self-update depuis GitHub Releases (check + upgrade avec vérification SHA-256).
- Distribution multi-plateforme (macOS, Linux, Windows) via GoReleaser + scripts.
- Persistance de l'état utilisateur (dernier profil utilisé).

### Hors périmètre

- Édition / création / suppression de profils dans `~/.aws/config`.
- Autres services AWS (RDS, ECS, CloudFront, Route 53…) — backlog futur.
- Réimplémentation native du protocole Session Manager.
- Multi-sélection d'instances / actions de masse.
- Signature cryptographique des releases (backlog sécurité S1).

## 3. Acteurs et pré-requis

- **Utilisateur** : opérateur DevOps disposant d'un terminal.
- **Pré-requis runtime** :
  - AWS CLI v2 disponible dans le `PATH`.
  - `session-manager-plugin` installé (pour SSM).
  - Fichier `~/.aws/config` présent et lisible.

## 4. Définitions

- **AuthKind** : catégorie d'authentification déduite d'un profil :
  - `SSO` — profil avec clé `sso_session`.
  - `Login` — profil avec clé `login_session`.
  - `CredentialProcess` — profil avec `credential_process`.
  - `Static` — profil avec `aws_access_key_id`.
  - `Unknown` — aucun marqueur reconnu.
- **Profil actif** : profil sélectionné, utilisé pour tous les appels AWS.
- **Région active** : région des appels régionaux (défaut : profil, sinon `eu-west-3`).

## 5. Exigences fonctionnelles

### Profils

- **REQ-1** — Au démarrage, le système DOIT lire `~/.aws/config` et présenter la
  liste des profils avec nom, AuthKind, et région.
- **REQ-2** — L'utilisateur DOIT pouvoir sélectionner un profil (profil actif).
- **REQ-3** — Si `~/.aws/config` est absent/illisible, erreur explicite sans crash.
- **REQ-4** — Le système DOIT persister le dernier profil utilisé et le restaurer
  au prochain lancement.

### Authentification

- **REQ-5** — À la sélection d'un profil, vérifier l'identité via `GetCallerIdentity`.
- **REQ-6** — Si credentials absents/expirés et profil SSO/Login, exécuter la
  commande de login appropriée.
- **REQ-7** — Après login réussi, ré-exécuter `GetCallerIdentity` automatiquement.
- **REQ-8** — Permettre le logout (SSO/Login).
- **REQ-9** — Détecter SSH (`$SSH_TTY`) et ajouter `--remote` aux commandes login.
- **REQ-10** — Résoudre et afficher l'alias de compte IAM dans la barre d'état.

### Région

- **REQ-11** — Sélecteur de région avec filtre.
- **REQ-12** — Invalider les caches (EC2, ASG, ELB) au changement de région.

### EC2

- **REQ-13** — Lister les instances EC2 (Name, ID, State, Type, IP, AZ) avec
  pagination SDK.
- **REQ-14** — Filtre texte sur la liste.
- **REQ-15** — Rafraîchissement à la demande sans bloquer l'UI.
- **REQ-16** — Fiche détaillée : infos générales, security groups (règles in/out),
  métriques CloudWatch (fenêtre 1h/3h/12h/24h), tags.
- **REQ-17** — Blocs async indépendants (une erreur de permission n'empêche pas
  l'affichage des autres blocs).

### Auto Scaling Groups

- **REQ-18** — Lister les ASG (nom, capacités min/désirée/max, instances saines,
  lancement, statut).
- **REQ-19** — Filtre texte + copie du nom.
- **REQ-20** — Fiche détaillée : configuration, instances avec indicateur de santé,
  target groups (résumé sain/total), activités récentes de scaling.
- **REQ-21** — Blocs async indépendants (target groups + activités).

### Load Balancers

- **REQ-22** — Lister les Load Balancers ELBv2 (nom, type, schéma, état, DNS).
- **REQ-23** — Scroll horizontal + filtre texte + copie DNS.
- **REQ-24** — Fiche détaillée : infos générales, listeners, target groups avec
  santé des cibles (ID, port, état, raison).
- **REQ-25** — Blocs async indépendants (listeners + targets).

### SSM

- **REQ-26** — Session shell interactive via `aws ssm start-session`, avec restitution
  du terminal après.
- **REQ-27** — Port-forward (instance) et port-forward remote-host (bastion).
- **REQ-28** — Gestion des tunnels en arrière-plan : liste, arrêt individuel,
  arrêt global au quit.
- **REQ-29** — Pré-vol : vérifier `session-manager-plugin` avant toute action SSM.

### Clipboard

- **REQ-30** — Copie vers le presse-papiers système (pbcopy/xclip/clip) avec
  confirmation dans la status bar.
- **REQ-31** — Menu de copie multi-champ sur la fiche EC2 détaillée (`Y`).

### Self-update

- **REQ-32** — Commande `aws-tui check-update` : vérifier si une nouvelle version
  existe sur GitHub Releases.
- **REQ-33** — Commande `aws-tui upgrade` : télécharger, vérifier SHA-256, extraire
  et remplacer le binaire atomiquement.
- **REQ-34** — Refuser l'auto-update si le binaire est managé par un gestionnaire
  de paquets (Homebrew, apt, Scoop).
- **REQ-35** — `--version` affiche version, commit, date de build.

### Transverses

- **REQ-36** — Aide contextuelle (hints par écran dans le footer).
- **REQ-37** — Quitter proprement à tout moment (`q`/Ctrl+C), en arrêtant les tunnels.
- **REQ-38** — Toute erreur AWS affichée lisiblement sans crash.
- **REQ-39** — Aucun secret écrit sur disque ; réutilisation des caches AWS CLI.

## 6. Exigences non fonctionnelles

- **NFR-1 (Réactivité)** — Appels réseau et exec externes asynchrones, jamais
  bloquants pour la boucle UI.
- **NFR-2 (Portabilité)** — macOS (arm64+amd64), Linux (arm64+amd64), Windows
  (amd64). Code OS-spécifique isolé via build tags.
- **NFR-3 (Distribution)** — Binaire unique. Archives tar.gz/zip, checksums SHA-256,
  paquets .deb/.rpm via GoReleaser. Scripts d'installation pour Unix et Windows.
- **NFR-4 (Robustesse)** — Pré-requis manquant = message clair, jamais un panic.
- **NFR-5 (Sécurité)** — Pas de credentials stockés. Self-update vérifie SHA-256.
- **NFR-6 (Maintenabilité)** — Séparation claire : awsconfig, auth, awsclient, ssm,
  selfupdate, clipboard, state, ui.

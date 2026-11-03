# Patate 💣

Jeu multijoueur dans le navigateur (2 à 10 joueurs, bots compris). Une bombe passe d'un joueur à l'autre quand ils se touchent. Celui qui la tient au moment où elle explose est éliminé. Le dernier survivant gagne. Le temps restant avant l'explosion est affiché.

**Modes de jeu** (choisis à la création de la partie) :
- 🔥 **Classique** : mèche de 8 à 20 s.
- 🔄 **Patate chaude** : mèche courte (6 s), mais chaque passe à un *nouveau* joueur la recharge de 1,5 s (8 s maximum). Renvoyer la bombe à celui qui vient de te la donner ne recharge rien.
- 👥 **Équipes** : Rouges contre Bleus, tirés au hasard à chaque manche. Impossible de passer la bombe à un coéquipier. Les survivants de l'équipe gagnante prennent +2.

Dans tous les modes, **une 2e bombe** entre en jeu tant qu'il reste au moins 6 joueurs en vie.

**Parties à code PIN** : un joueur clique sur « Créer une partie » et reçoit un code à 4 chiffres. Les autres le tapent pour rejoindre (ou ouvrent le lien `?pin=1234`). Plusieurs parties peuvent tourner en même temps, et une partie vide est supprimée.

**Commandes** :

| Touche | Action |
|---|---|
| Flèches, ZQSD (AZERTY) ou WASD (QWERTY) | se déplacer |
| 1 2 3 4 | emotes 😂 😡 👋 💀 au-dessus de ton perso |
| Tab (maintenu) | classement |
| B / N | ajouter / retirer un bot (les bots partent quand il ne reste plus aucun humain) |
| M | couper / remettre le son |

**Ambiance** :
- **Sons** synthétisés par le navigateur, sans fichier audio : tic-tac qui accélère sous 3 s, passe, explosion, bonus, victoire.
- **Effets** : explosion en particules, tremblement de l'écran, onde à chaque passe.
- **Fil d'actions** en haut à gauche de l'arène.
- **Records de la manche** à la fin : le plus de passes, le plus longtemps avec la bombe, le chasseur de bonus et la victime préférée.
- **Mouvements fluides** : le navigateur lisse les positions entre deux états reçus du serveur (environ 33 ms de retard), ce qui évite les saccades en jouant via internet.

**Scores** : les points s'accumulent sur toutes les manches de la partie. Survivre à une explosion rapporte +1 et gagner la manche +2. Le classement s'affiche à la fin de chaque manche, ou à tout moment en maintenant **Tab**.

**Arènes** : l'arène change à chaque manche (Plaine, Piliers, Croix, Couloirs, Labyrinthe). Les murs bloquent les joueurs et la bombe ne passe pas à travers. Pour en ajouter une, complète la liste `arenas` dans `main.go` (le test `TestArenasPlayable` vérifie qu'aucune zone n'est fermée).

**Bonus** (jusqu'à 3 en même temps dans l'arène, effet de 4 s) :
- ⚡ **Vitesse** : tu vas 1,5 fois plus vite.
- 🛡️ **Bouclier** : impossible de te passer la bombe.
- 🧊 **Gel** : tous les autres joueurs sont ralentis de moitié.

## Interface

L'interface utilise **Tailwind CSS v4** et les icônes **Heroicons** (de l'équipe Tailwind, licence MIT, intégrées dans `index.html`). Le CSS est généré une seule fois dans `style.css` puis embarqué dans le binaire : rien n'est chargé depuis internet pendant le jeu.

Si tu modifies des classes dans `index.html`, régénère le CSS avec le [binaire autonome Tailwind](https://github.com/tailwindlabs/tailwindcss/releases) (sans Node) :

```sh
tailwindcss -i app.css -o style.css --minify
```

## Lancer en local

```sh
go run .            # puis http://localhost:8080
go test ./...
```

## Déployer sur Proxmox

Le serveur utilise environ 15 Mo de RAM avec une partie pleine.

**1. Créer le conteneur (une seule fois, sur l'hôte Proxmox)** : un LXC Alpine avec 64 Mo de RAM suffit.

```sh
pveam update && pveam available | grep alpine          # repère le nom exact du modèle
pveam download local alpine-3.22-default_XXXX_amd64.tar.xz
pct create 200 local:vztmpl/alpine-3.22-default_XXXX_amd64.tar.xz \
  --hostname patate --cores 1 --memory 64 --swap 0 --rootfs local-lvm:1 \
  --net0 name=eth0,bridge=vmbr0,ip=dhcp --unprivileged 1 --start 1
# accès SSH par clé pour le script de déploiement
pct exec 200 -- sh -c 'apk add openssh && rc-update add sshd && rc-service sshd start && mkdir -p /root/.ssh'
pct push 200 ~/.ssh/authorized_keys /root/.ssh/authorized_keys   # ou colle ta clé publique dedans
```

**2. Déployer / mettre à jour (depuis ton PC)** :

```sh
./deploy.sh root@IP_DU_CONTENEUR            # port 8080
PORT=9000 ./deploy.sh root@IP_DU_CONTENEUR  # autre port
```

Le script lance les tests, compile un binaire statique et l'envoie. Il crée le service au premier déploiement (OpenRC sur Alpine, systemd sur Debian/Ubuntu) puis redémarre le jeu, qui tourne sous l'utilisateur `nobody`. Les logs sont dans `/var/log/patate.log` (Alpine) ou `journalctl -u patate` (Debian).

**3. Jouer** : tes amis ouvrent `http://IP_DU_CONTENEUR:8080`. Hors de ton réseau, tu as deux options :
- ouvrir/rediriger le port sur ta box ;
- le placer derrière ton reverse proxy (Caddy, Nginx Proxy Manager…), en pensant à activer le support WebSocket. Avec HTTPS, le bouton « copier le lien » marche aussi.

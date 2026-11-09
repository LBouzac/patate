# Patate 💣 & Ricochet ⚡

Deux jeux multijoueurs dans le navigateur, servis par un seul petit serveur Go (environ 15 Mo de RAM) :

L'adresse du site (`/`) ouvre le **menu de tous les jeux**. Les anciens liens d'invitation `/?pin=1234` mènent toujours à la partie de Patate.

**Jouer en groupe** (`/salon.html`) : on se retrouve dans un salon avec un seul code, comme en mode télé. Le chef (le premier arrivé, couronne) choisit le jeu et tout le monde y part directement, sans code ni nouvelle invitation. En partie, le bouton **Salon** ramène tout le groupe au salon pour choisir un autre jeu. Si le jeu a moins de places que le groupe (Foot, Billard…), les suivants ouvrent un deuxième salon ensemble.

- **Patate** (`/patate.html`) : de 2 à 10 joueurs, une bombe à se passer avant qu'elle n'explose.
- **Ricochet** (`/ricochet.html`) : un duel de raquettes en 1 contre 1, ou contre un bot.
- **Dessine** (`/dessin.html`) : de 3 à 10 joueurs. Chacun son tour envoie une photo, un joueur tiré au sort la dessine en 30 s et les autres devinent.
- **Foot** (`/foot.html`) : foot de pions à 2 joueurs (ou contre un bot), chacun son tour, premier à 3 buts (réglable de 1 à 10).
- **Pétanque** (`/petanque.html`) : de 2 à 6 joueurs (bots possibles), chacun ses 3 boules, premier à 7.
- **Billard** (`/billard.html`) : jaunes contre rouges à 2 (ou contre un bot), la noire pour finir. En visant, une bille fantôme montre où la blanche touchera et où partira la bille touchée.
- **Artillerie** (`/artillerie.html`) : de 2 à 4 chars (bots possibles), vent et cratères, le dernier debout gagne.
- **Motos** (`/motos.html`) : de 2 à 8 motos en temps réel (bots possibles) qui laissent un mur, premier à 5 manches.
- **Golf** (`/golf.html`) : mini-golf de 1 à 8 joueurs, 9 trous générés au hasard à chaque partie.
- **Lombrics** (`/lombrics.html`) : de 2 à 4 équipes de 3 vers (bots possibles) sur une île qu'on creuse : bazooka, mitrailleuse, lance-flammes, batte, dynamite, frappe aérienne, hélico… La dernière équipe en vie gagne.

Et un **mode télé** (`/tele.html` + `/manette.html`) : le jeu s'affiche sur la télé, les téléphones servent de manettes. Cinq jeux faits pour ça : **Sumo**, **Course**, **Quiz rapide**, **Imposteur** et **Dessin en chaîne** (détails plus bas).


Jeu multijoueur dans le navigateur (2 à 10 joueurs, bots compris). Une bombe passe d'un joueur à l'autre quand ils se touchent. Celui qui la tient au moment où elle explose est éliminé. Le dernier survivant gagne. Le temps restant avant l'explosion est affiché.

## Installation sur Proxmox en une ligne

Dans le shell de l'hôte Proxmox VE (en root) :

```sh
bash -c "$(curl -fsSL https://raw.githubusercontent.com/LBouzac/patate/main/install.sh)"
```

Le script télécharge le jeu (et vérifie sa somme de contrôle), prend le dernier modèle Alpine et crée un conteneur LXC non privilégié (1 cœur, 64 Mo de RAM, 1 Go de disque, DHCP sur `vmbr0`, démarrage automatique). Il installe ensuite le service et affiche l'adresse à donner à tes amis, ainsi que le **mot de passe admin** pour les mises à jour.

- **Mettre à jour depuis le site** : ouvre `http://IP:PORT/admin.html`, tape le mot de passe admin puis clique sur « Installer la dernière version ». Le serveur télécharge la release `latest`, vérifie sa somme de contrôle, remplace son binaire et redémarre tout seul (les parties en cours sont coupées quelques secondes). La page affiche la version installée.
- **Ancien conteneur** (installé avant la mise à jour depuis le site) : relance une seule fois `CTID=200 bash -c "$(curl -fsSL https://raw.githubusercontent.com/LBouzac/patate/main/install.sh)"`. Le script garde le port, déplace le binaire dans `/opt/patate`, crée le mot de passe et l'affiche.
- **Mot de passe oublié** : `pct exec 200 -- sed -n 's/^export ADMIN_PASSWORD=//p' /etc/init.d/patate`.
- **Personnaliser** avec des variables placées avant la commande : `CTID`, `CT_HOSTNAME`, `PORT`, `MEMORY`, `DISK`, `BRIDGE`, `STORAGE`, `TEMPLATE_STORAGE`. Exemple : `PORT=9000 CTID=150 bash -c "$(curl …)"`.

Le binaire est compilé par GitHub Actions à chaque push sur `main`, puis publié dans la release [`latest`](https://github.com/LBouzac/patate/releases/tag/latest).

## Mode télé

Le jeu s'affiche sur un grand écran, chacun joue avec son téléphone. Jusqu'à 10 joueurs.

1. Sur la télé (son navigateur, ou l'ordinateur branché dessus), ouvre `http://IP:PORT/tele.html`. Un salon s'ouvre avec un **QR code** et un **code à 4 chiffres**. Appuie sur OK (ou clique) une fois pour activer le son et le plein écran.
2. Chacun scanne le QR code avec son téléphone (ou ouvre `/manette.html` et tape le code). Le premier arrivé est le **chef** (couronne) : il choisit le jeu sur son téléphone, puis peut rejouer ou changer de jeu à tout moment (bouton « Jeux »). Tout le monde suit d'un jeu à l'autre sans rescanner.
3. Un petit QR code reste dans un coin de la télé pendant les jeux : on peut arriver en cours de route. Si la télé recharge sa page, elle retrouve son salon (le code est dans l'adresse).

Les jeux :

- **Sumo** (1 à 10, bots possibles) : joystick à gauche du téléphone, bouton **dash** à droite (gros élan, et plus lourd un court instant ; recharge 2 s). On pousse les autres hors d'une arène ronde qui rétrécit après 8 s. Le dernier debout marque un point, premier à 3. Le chef ajoute ou retire des bots. Le téléphone vibre à chaque choc (Android).
- **Course** (1 à 10, bots possibles) : petites voitures vues de dessus, 3 tours. La voiture accélère toute seule ; on tourne avec les deux gros boutons du téléphone (ou en l'inclinant, si le site est en https : bouton « Incliner pour tourner », puis « Recentrer »). Les boîtes « ? » donnent un objet, utilisé avec le bouton du milieu : **turbo**, ou **flaque d'huile** lâchée derrière soi (tête-à-queue pour qui roule dessus). Couper par l'herbe ne sert à rien : loin de la piste, on n'avance plus et on est ramené où on l'a quittée. Après la première arrivée, les autres ont 20 s. Trois circuits à la suite (l'ovale, le haricot, le serpentin) ; les points s'additionnent (10, 8, 6, 5…), et le leader part en fond de grille.
- **Quiz rapide** (1 à 10) : 10 questions tirées parmi plus de 150, affichées sur la télé ; on répond A, B, C ou D sur le téléphone en 15 s. Bonne réponse : de 500 à 1000 points selon la rapidité. La télé montre qui a répondu, puis qui a choisi quoi.
  - **Ses propres questions** : en touchant Quiz rapide, le chef peut choisir « Mes questions » et les écrire sur son téléphone (la question, la bonne réponse, 3 mauvaises), ou coller une liste, une question par ligne : `Capitale du Gers ? ; Auch ; Agen ; Pau ; Tarbes` (un copier-coller depuis un tableur marche aussi). 50 questions au plus. Elles restent tant que le salon est ouvert et ne sont **jamais enregistrées** : télé fermée, elles disparaissent.
- **Imposteur** (3 à 10) : chacun reçoit le même mot secret sur son téléphone, sauf l'imposteur qui ne connaît que le thème. Chacun son tour dit un mot à voix haute (la télé indique qui parle), puis tout le monde vote sur son téléphone. Démasqué, l'imposteur peut encore retrouver le mot parmi 12. Points : imposteur jamais démasqué +2, démasqué mais trouve le mot +1, sinon +1 pour tous les autres. Pour revoir son mot, on maintient un bouton : il ne reste pas affiché sous les yeux du voisin.
- **Dessin en chaîne** (3 à 10) : chacun écrit une phrase (ou prend une idée au hasard), le suivant la dessine sur son téléphone, le suivant décrit ce dessin, et ainsi de suite (8 étapes au plus, chaque chaîne passe une fois par chacun). Sans réponse à la fin du chrono, le téléphone envoie ce qu'il a. À la fin, la télé révèle chaque chaîne maillon par maillon : le chef fait défiler avec son téléphone. Rien n'est enregistré.

La télé reçoit l'état public du jeu, chaque téléphone seulement ce qu'il doit voir (le mot secret, le dessin à deviner). On n'envoie que ce qui a changé ; un dessin à deviner part une seule fois, à part du reste.

## Ricochet

Duel de raquettes à 2 joueurs, premier à **7 points**. On crée un duel et on partage le code, comme pour Patate, ou on joue contre un **bot** (touche **B**). Si un ami rejoint ensuite, il prend la place du bot.

- **Commandes** : ↑ ↓ ou Z / S, **B** / **N** pour ajouter ou retirer le bot, **M** pour couper le son. Au doigt (ou souris maintenue) : garde le doigt sur le terrain, ta raquette le suit.
- La balle **accélère à chaque renvoi**, jusqu'à 950 px/s. Elle reste toujours renvoyée correctement, même à pleine vitesse, sans traverser la raquette.
- L'**angle** de renvoi dépend de l'endroit où la balle touche la raquette : tout droit au centre, jusqu'à 60° aux extrémités.
- Après chaque point, le service part vers celui qui vient de le perdre. Une revanche démarre automatiquement en fin de partie.
- Le serveur calcule 60 images par seconde et le navigateur lisse l'affichage. Ambiance néon : traînée de balle, étincelles, tremblement de l'écran à chaque point, sons synthétisés.

## Dessine

De 3 à 10 joueurs. On crée une partie et on partage le code, puis n'importe quel joueur clique sur « Commencer ».

- **Chacun son tour**, un joueur envoie une photo (téléphone ou ordinateur). Elle est réduite dans le navigateur, garde moins de 300 Ko et n'est jamais écrite sur le disque.
- Un autre joueur, **tiré au sort**, est le seul à voir la photo. Il a **30 s** pour la dessiner, et le dessin s'affiche en direct chez tout le monde.
- Les autres **proposent** autant de réponses qu'ils veulent. L'auteur de la photo valide chaque proposition (✓ / ✗). Il a encore 5 s après le chrono pour juger les dernières.
- **Points** : de 100 points pour une proposition envoyée tout de suite à 10 points à la fin du chrono (le compte se fait au moment de l'envoi). Le dessinateur gagne +25 par joueur qui trouve.
- Les joueurs qui devinent ne voient pas les bonnes réponses des autres avant la révélation.
- Si l'auteur de la photo ou le dessinateur quitte la partie, la manche est annulée. Le classement s'affiche quand tout le monde a envoyé sa photo.

## Foot

À 2 joueurs, ou seul contre un **bot** (bouton « Jouer contre un bot »). Si un ami rejoint ensuite, il prend la place du bot.

- Chaque équipe a **6 pions** (orange contre cyan) et il y a un ballon. Avant chaque coup d'envoi, chacun choisit sa **disposition** : Défense (1 gardien, 4 derrière, 1 devant), Équilibrée (3-2-1) ou Attaque (2 derrière, 4 devant). Sans choix au bout de 10 s, c'est Équilibrée.
- **Chacun son tour** : attrape un de tes pions, tire en arrière comme au golf puis relâche. Le pion pousse le ballon et cogne les autres pions. **Un seul coup par tour**, et 15 s pour le jouer.
- Les pions sont plus lourds que le ballon et ne peuvent pas entrer dans les buts. Les poteaux renvoient le ballon.
- **But** quand le ballon franchit la ligne entre les poteaux. On se replace, l'équipe qui a encaissé engage. **Premier à 3 buts**, puis revanche automatique.

## Pétanque

De 2 à 6 joueurs, chacun pour soi, ou avec des **bots** (« Ajouter un bot »). Le cochonnet est posé au hasard au début de chaque mène. Le bot pointe au plus près du cochonnet et tire sur une boule adverse qui le colle quand il n'a pas le point.

- Chacun a **3 boules par mène**. À ton tour, tire en arrière n'importe où sur le terrain puis relâche : la boule part du cercle. On peut **tirer** pour dégommer une boule adverse.
- **Celui qui n'a pas le point rejoue** (le plus mal placé d'abord). Quand les autres n'ont plus de boules, celui qui a le point finit les siennes.
- Le terrain est **entouré de planches** : boules et cochonnet rebondissent dessus, rien ne sort. 15 s par lancer, sinon la boule est perdue.
- À la fin de la mène, celui qui a le point marque **une boule par boule plus proche que la meilleure des autres**, et commence la mène suivante. **Premier à 7 points.**

## Billard

À 2, ou contre un **bot**. Le premier joueur a les **jaunes**, le second les **rouges**.

- Tire en arrière puis relâche : la queue recule avec la puissance.
- **Rentrer une de ses billes fait rejouer.** Faute (la main passe) : blanche rentrée (elle revient au point de départ), aucune bille touchée, ou d'abord une bille adverse.
- **Rentrer la noire après toutes ses billes fait gagner**, la rentrer trop tôt ou sur une faute fait perdre. 25 s par coup.

## Artillerie

De 2 à 4 chars, joueurs ou **bots** (« Ajouter un bot »). Chacun son tour, 20 s pour tirer.

- Tire en arrière puis relâche pour régler l'angle et la puissance. Le **vent** (affiché en haut) change à chaque tour.
- Les obus **creusent le terrain**, et les chars retombent au fond des cratères. Un tir direct enlève 70 PV, moins si l'obus tombe à côté.
- Le dernier char debout gagne, puis un nouveau terrain est généré.

## Lombrics

De 2 à 4 équipes de **3 vers** (chacun son petit nom : Gaston, Ginette, Bébert…), joueurs ou **bots**. Chacun son tour, 45 s : un ver de ton équipe joue (ils jouent à tour de rôle).

- **Grande île** (1600 × 800, deux fois plus large et haute que l'écran) avec des îlots flottants et des grottes. La **vue suit l'action** : le ver qui joue, puis le projectile en vol. **Molette** pour zoomer, bouton 🗺️ ou touche **V** pour voir toute l'île (pratique pour viser la frappe aérienne), et quand ce n'est pas ton tour, **glisse** sur l'île pour regarder ailleurs.

- **Marcher** : ← → (ou Q/A et D), ou les boutons ‹ ›. Le ver monte les petites marches, pas les murs. **Sauter** : Espace (ou ↑, Z, W), ou le bouton « Saut ».
- **Arme** : touches 1 à 9, 0 et R, ou les boutons (sur téléphone, icônes seules), puis tire en arrière et relâche (la force compte pour le bazooka et la grenade, seule la direction pour les autres).
  - 🚀 **Bazooka** : explose au contact, poussé par le **vent** (change à chaque tour).
  - 💣 **Grenade** : rebondit et explose au bout de 3 s.
  - 🔫 **Fusil** : deux tirs droits et instantanés (on ne bouge pas entre les deux).
  - 💥 **Mitrailleuse** : une rafale de 15 balles.
  - 🔥 **Lance-flammes** : un jet de flammes à courte portée, poussé par le vent.
  - 🏏 **Batte** : envoie valser le ver juste devant (30 PV).
  - 🧨 **Dynamite** (2 par partie) : posée à tes pieds en touchant l'île (ou Entrée), explose au bout de 5 s. File !
  - 🪤 **Mine** (2 par partie) : posée à tes pieds, s'arme en 3 s puis saute quand un ver passe.
  - ✈️ **Frappe aérienne** (1 par partie) : touche la cible, 5 missiles tombent dessus.
  - 🚁 **Hélico** (2 par partie) : touche l'île (ou Entrée), puis 5 s de vol : tiens Saut (ou ▲) pour monter, ← → pour avancer. Ne termine pas le tour, on peut tirer après.
  - 🪢 **Corde ninja** : vise un bout de terre (260 px au plus), le ver s'y accroche. ← → pour se balancer, ↑ ↓ (ou ▲ ▼) pour raccourcir ou allonger, Espace (ou Saut) pour lâcher avec l'élan. Ne termine pas le tour. Si la terre où elle tient saute, on tombe.
- Après un tir, **3 s de repli** pour se mettre à l'abri. Un ver qui se blesse pendant son tour (chute, sa propre explosion) le termine aussitôt.
- Le terrain est une île percée de grottes : **chaque explosion la creuse**. Les explosions **projettent** les vers, les grosses chutes font mal et **l'eau noie**.
- L'île cache **8 mines** (un ver qui passe à côté la déclenche : 1,5 s pour s'éloigner) et **5 barils de pétrole** qui explosent quand on les touche (tir, explosion, flammes) en répandant des flammes. Les explosions font sauter les mines et les barils voisins : réactions en chaîne.
- Des **caisses** tombent parfois en parachute : marche dessus pour les prendre. **Soin** (croix rouge, +25 PV) ou **armes** (caisse en bois : une munition de plus de dynamite, mine, frappe aérienne ou hélico).
- **Mort subite** : après 8 tours chacun, l'eau monte à chaque tour.
- La dernière équipe avec un ver vivant gagne, puis une nouvelle île est générée. Le bot tire au bazooka ou à la grenade : il essaie toutes les directions (vent et terrain compris) et évite de toucher ses propres vers. Collé à un adversaire, il sort la batte.

## Motos

De 2 à 8 motos en temps réel, joueurs ou **bots**. Chaque moto avance toute seule et laisse un mur derrière elle.

- **Flèches** ou ZQSD (WASD) pour tourner. Au doigt : boutons ↺ ↻, ou toucher la moitié gauche ou droite de la piste.
- Toucher un bord, un mur (même le sien) ou une autre moto élimine. Deux motos qui visent la même case sont éliminées toutes les deux.
- Le dernier en piste marque un point. **Premier à 5 manches.**

## Golf

Mini-golf de 1 à 8 joueurs. On crée un parcours et on partage le code. Tout le monde joue le même trou **en même temps**, chacun avec sa balle. Les balles **se cognent** : tu peux pousser celle d'un adversaire (si elle finit dans l'eau, elle revient à sa place sans pénalité).

- **Chaque trou est généré au hasard, et difficile** : départ et drapeau aux deux bouts, 4 à 8 murs dont la moitié en travers du chemin, du sable et au moins un étang sur la trajectoire, un bunker ou un mur qui garde le drapeau. Jamais de coup direct possible. Le serveur vérifie qu'un chemin existe avant de le proposer et calcule le **par** (2 à 5) d'après la longueur du chemin.
- **Frapper** : clique n'importe où sur le terrain, tire en arrière comme un lance-pierre, puis relâche. Plus tu tires, plus c'est fort. Ça marche aussi au doigt.
- Le **sable** freine fort, l'**eau** te ramène à ton point de frappe avec un coup de pénalité. Trop vite, la balle saute par-dessus le trou.
- 90 s par trou et 10 coups au maximum. Une balle pas rentrée compte 10.
- **Classement** par rapport au par (`-2`, `E`, `+3`…). La carte de score s'affiche après chaque trou. Après 9 trous, le gagnant est annoncé et un nouveau parcours démarre.
- Un joueur qui arrive en cours de partie part du départ du trou en cours (ou attend le suivant si le trou est déjà fini). Les trous manqués ne comptent pas.

## Patate

**Modes de jeu** (choisis à la création de la partie) :
- 🔥 **Classique** : mèche de 8 à 20 s.
- 🔄 **Patate chaude** : mèche courte (6 s), mais chaque passe à un *nouveau* joueur la recharge de 1,5 s (8 s maximum). Renvoyer la bombe à celui qui vient de te la donner ne recharge rien.
- 👥 **Équipes** : Rouges contre Bleus, tirés au hasard à chaque manche. Impossible de passer la bombe à un coéquipier. Les survivants de l'équipe gagnante prennent +2.

Dans tous les modes, **une 2e bombe** entre en jeu tant qu'il reste au moins 6 joueurs en vie.

**Parties à code PIN** : un joueur clique sur « Créer une partie » et reçoit un code à 4 chiffres. Les autres le tapent pour rejoindre (ou ouvrent le lien `?pin=1234`). Plusieurs parties peuvent tourner en même temps, et une partie vide est supprimée.

**Commandes** :

| Touche | Action |
|---|---|
| Flèches, ZQSD (AZERTY) ou WASD (QWERTY) | se déplacer (au doigt : pose le pouce sur le terrain et glisse) |
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

Le style reprend celui de PVE Hub (listes de réglages façon iPhone). Le **menu** liste tous les jeux. Chaque jeu a le même accueil : lien « ‹ Les jeux », grand titre avec l'icône du jeu, pseudo (gardé d'un jeu à l'autre), gros bouton pour créer une partie dans la couleur du jeu (Patate orange, Ricochet cyan, Golf vert, Dessine rose, Foot vert citron, Pétanque jaune, Billard bleu ciel, Artillerie rouge, Motos violet, Lombrics rose) et une ligne « code + Rejoindre ». Une **barre d'onglets en bas** permet de changer de jeu. En partie, « ‹ Quitter » ramène à l'accueil. La liste des jeux, la barre et ce bouton viennent de `common.js`.

**Sur ordinateur**, le menu et le salon passent en grille de cartes. Le salon s'affiche sur deux colonnes : à gauche l'invitation (gros code et QR code du lien, à scanner avec un téléphone) et la liste des présents avec leur initiale, à droite les jeux. Chaque jeu indique s'il faut plusieurs parties pour tout le groupe (« 3 parties, 2 places chacune ») ou s'il manque du monde. En partie, le terrain s'agrandit pour remplir l'écran (jusqu'à 1600 px de large).

**Sur téléphone**, chaque jeu se joue au doigt :
- le terrain prend toute la place disponible, et un message propose de tourner le téléphone quand le terrain est large ;
- téléphone couché (ou fenêtre basse), le terrain occupe toute la hauteur à gauche et le reste (scores, boutons) passe dans une colonne à droite ;
- un bouton **plein écran** apparaît quand le navigateur le permet (Android, iPad) et bascule le téléphone en paysage ;
- les aides clavier sont remplacées par les gestes : joystick sous le pouce et boutons d'emotes et de classement dans Patate, raquette qui suit le doigt dans Ricochet, boutons ↺ ↻ dans Motos. Les autres jeux se jouaient déjà au doigt (tirer en arrière puis relâcher).

**Trois styles au choix**, avec le bouton **Style** en haut à droite du menu et de chaque accueil. Le choix est gardé d'une page à l'autre, mode télé compris :
- **Classique** : le style ci-dessus, sombre et sobre.
- **Y2K** : fond bleu électrique quadrillé, titres chromés en italique, panneaux à cadre métallique, boutons « gel » brillants et icônes en orbes.
- **Liquid Glass** : verre flou et translucide sur un fond coloré, boutons en capsule, barre des jeux qui flotte en bas. Avec le réglage système « Réduire la transparence », le verre devient presque opaque.

L'interface utilise **Tailwind CSS v4** et les icônes **Heroicons** (de l'équipe Tailwind, licence MIT, intégrées dans `common.js`). Le CSS est généré une seule fois dans `style.css` puis embarqué dans le binaire : rien n'est chargé depuis internet pendant le jeu.

Si tu modifies des classes dans une page, régénère le CSS avec le [binaire autonome Tailwind](https://github.com/tailwindlabs/tailwindcss/releases) (sans Node) :

```sh
tailwindcss -i app.css -o style.css --minify
```

Quelques mises en page sont écrites en CSS classique à la fin d'`app.css` (grille de cartes sur grand écran, salon sur deux colonnes, terrain couché, joystick, styles Y2K et Liquid Glass) : Tailwind les recopie telles quelles dans `style.css`.

## Lancer en local

```sh
go run .            # puis http://localhost:8080
go test ./...
```

## Déployer depuis ton PC (sans l'installeur)

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

Le script lance les tests, compile un binaire statique et l'envoie. Il crée le service au premier déploiement (OpenRC sur Alpine, systemd sur Debian/Ubuntu) puis redémarre le jeu, qui tourne sous l'utilisateur `nobody`. Il garde le mot de passe admin existant (ou en crée un) et l'affiche. Les logs sont dans `/var/log/patate.log` (Alpine) ou `journalctl -u patate` (Debian).

**3. Jouer** : tes amis ouvrent `http://IP_DU_CONTENEUR:8080`. Hors de ton réseau, tu as deux options :
- ouvrir/rediriger le port sur ta box ;
- le placer derrière ton reverse proxy (Caddy, Nginx Proxy Manager…), en pensant à activer le support WebSocket. Avec HTTPS, le bouton « copier le lien » marche aussi.

## Sécurité

- **Mise à jour** : protégée par le mot de passe `ADMIN_PASSWORD`. Sans lui, elle est désactivée. Après 5 mauvais mots de passe, elle est bloquée 15 minutes. Le binaire téléchargé est vérifié par SHA-256 avant d'être installé.
- **HTTPS conseillé** si le jeu est ouvert sur internet : en HTTP simple, le mot de passe admin passe en clair sur le réseau.
- **En-têtes** : politique de sécurité du contenu (pas de script externe, pas d'affichage dans un cadre d'un autre site), `nosniff`, aucun référent envoyé.
- **WebSocket** : refusées si elles viennent d'un autre site. 300 connexions simultanées au maximum, tous jeux confondus, et des délais d'attente contre les connexions qui traînent.
- **Dessine** : 50 parties en même temps au maximum. Photo de 300 Ko au plus, uniquement en JPEG, gardée en mémoire seulement. Taille du dessin limitée.
- **Débit** : les messages WebSocket sont compressés, les jeux envoient leur état 30 fois par seconde (la physique tourne à 60) et le golf n'envoie le trou qu'une fois par trou. Par joueur, ça fait de 4 à 28 Ko/s avant compression, environ deux fois moins sur le réseau.
- **Chat de la partie** (bouton 💬 en bas à droite, en partie) : le serveur relaie chaque message en direct aux joueurs du même salon et **n'enregistre rien** (pas d'historique, un nouveau venu ne voit pas les anciens messages). 200 caractères au plus, un message par demi-seconde. Dans Dessine, le chat est en pause pendant le dessin (on y donnerait la réponse).
- **Reconnexion automatique** : si la connexion lâche (réseau, écran verrouillé, passage wifi → 4G), la place du joueur reste réservée 20 s et la page se reconnecte toute seule (bandeau « Reconnexion… ») : même joueur, même score, même tour. Une connexion figée (plus aucun message depuis 4 s, par exemple au réveil d'un téléphone) est remplacée aussitôt. Quitter (bouton ou onglet fermé) libère la place tout de suite.
- **Déconnexions** : chacune est notée dans le journal du service avec sa raison (`/var/log/patate.log` sur Alpine, `journalctl -u patate` sur Debian).
- **Limites connues** : les codes de partie font 4 chiffres et peuvent se deviner. Le service peut réécrire son propre binaire, c'est le prix de la mise à jour depuis le site.

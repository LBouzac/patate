#!/usr/bin/env bash
# Patate — installation en une ligne sur Proxmox VE, à lancer dans le shell de l'hôte :
#
#   bash -c "$(curl -fsSL https://raw.githubusercontent.com/LBouzac/patate/main/install.sh)"
#
# Mise à jour d'un conteneur existant (garde ses réglages) :
#
#   CTID=200 bash -c "$(curl -fsSL https://raw.githubusercontent.com/LBouzac/patate/main/install.sh)"
#
# Réglages facultatifs (variables d'environnement) :
#   CTID (prochain libre)  CT_HOSTNAME (patate)  PORT (8080)  MEMORY en Mo (64)  DISK en Go (1)
#   BRIDGE (vmbr0)  STORAGE (disque du conteneur, auto)  TEMPLATE_STORAGE (modèles, auto)
set -euo pipefail

URL="https://github.com/LBouzac/patate/releases/download/latest"
PORT="${PORT:-8080}" MEMORY="${MEMORY:-64}" DISK="${DISK:-1}" BRIDGE="${BRIDGE:-vmbr0}" CT_HOSTNAME="${CT_HOSTNAME:-patate}"

step() { printf '\e[36m→\e[0m %s\n' "$*"; }
ok() { printf '\e[32m✓\e[0m %s\n' "$*"; }
die() { printf '\e[31m✗ %s\e[0m\n' "$*" >&2; exit 1; }

# --- vérifications ---
[ "$(id -u)" = 0 ] || die "À lancer en root sur l'hôte Proxmox"
command -v pct >/dev/null || die "pct introuvable : ce script se lance dans le shell de l'hôte Proxmox VE"
[ "$(uname -m)" = x86_64 ] || die "Seul Proxmox x86_64 est géré"
for v in PORT MEMORY DISK; do [[ "${!v}" =~ ^[0-9]+$ ]] || die "$v doit être un nombre (reçu : ${!v})"; done
[[ "${CTID:-0}" =~ ^[0-9]+$ ]] || die "CTID doit être un nombre"
[[ "$CT_HOSTNAME" =~ ^[a-zA-Z0-9-]+$ ]] || die "CT_HOSTNAME invalide"
[[ "$BRIDGE" =~ ^[a-zA-Z0-9._-]+$ ]] || die "BRIDGE invalide"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# --- binaire, vérifié par somme de contrôle ---
step "Téléchargement de Patate"
curl -fsSL -o "$TMP/patate" "$URL/patate-linux-amd64"
curl -fsSL -o "$TMP/SHA256SUMS" "$URL/SHA256SUMS"
(cd "$TMP" && sed 's/patate-linux-amd64$/patate/' SHA256SUMS | sha256sum -c --quiet -) || die "Somme de contrôle invalide : téléchargement corrompu"
ok "Binaire téléchargé et vérifié"

# --- mise à jour d'un conteneur existant ---
if [ -n "${CTID:-}" ] && pct status "$CTID" >/dev/null 2>&1; then
  step "Mise à jour du conteneur $CTID"
  pct exec "$CTID" -- test -f /etc/init.d/patate || die "Le conteneur $CTID n'a pas été installé par ce script"
  # envoi à côté puis renommage : écrire directement sur un binaire en cours d'exécution échoue (« Text file busy »)
  pct push "$CTID" "$TMP/patate" /usr/local/bin/patate.new --perms 755
  pct exec "$CTID" -- sh -c 'mv /usr/local/bin/patate.new /usr/local/bin/patate && rc-service patate restart >/dev/null'
  ok "Patate mis à jour dans le conteneur $CTID"
  exit 0
fi

# --- création du conteneur ---
CTID="${CTID:-$(pvesh get /cluster/nextid)}"
first_storage() { pvesm status -content "$1" | awk 'NR > 1 && $3 == "active" { print $1 }' | grep -m1 -x "$2" || pvesm status -content "$1" | awk 'NR > 1 && $3 == "active" { print $1; exit }'; }
STORAGE="${STORAGE:-$(first_storage rootdir local-lvm)}"
TEMPLATE_STORAGE="${TEMPLATE_STORAGE:-$(first_storage vztmpl local)}"
[ -n "$STORAGE" ] || die "Aucun stockage actif pour les conteneurs (précise STORAGE=...)"
[ -n "$TEMPLATE_STORAGE" ] || die "Aucun stockage actif pour les modèles (précise TEMPLATE_STORAGE=...)"

step "Recherche du dernier modèle Alpine"
pveam update >/dev/null
TEMPLATE="$(pveam available --section system | awk '{ print $2 }' | grep -E '^alpine-3\.[0-9]+-default_[0-9]+_amd64\.tar\.(xz|gz|zst)$' | sort -V | tail -1 || true)"
[ -n "$TEMPLATE" ] || die "Aucun modèle Alpine trouvé (pveam available)"
if ! pveam list "$TEMPLATE_STORAGE" | grep -q "$TEMPLATE"; then
  step "Téléchargement du modèle $TEMPLATE"
  pveam download "$TEMPLATE_STORAGE" "$TEMPLATE" >/dev/null
fi
ok "Modèle : $TEMPLATE"

step "Création du conteneur $CTID ($CT_HOSTNAME, ${MEMORY} Mo, ${DISK} Go sur $STORAGE)"
pct create "$CTID" "$TEMPLATE_STORAGE:vztmpl/$TEMPLATE" \
  --hostname "$CT_HOSTNAME" --ostype alpine --unprivileged 1 --onboot 1 \
  --cores 1 --memory "$MEMORY" --swap 0 --rootfs "$STORAGE:$DISK" \
  --net0 "name=eth0,bridge=$BRIDGE,ip=dhcp" \
  --description "Patate — jeu multijoueur (https://github.com/LBouzac/patate)" >/dev/null
pct start "$CTID"

step "Attente du réseau"
IP=""
for _ in $(seq 1 30); do
  IP="$(pct exec "$CTID" -- ip -4 -o addr show eth0 2>/dev/null | awk '{ split($4, a, "/"); print a[1] }' || true)"
  [ -n "$IP" ] && break
  sleep 1
done
[ -n "$IP" ] || die "Le conteneur $CTID n'a pas obtenu d'adresse IP (DHCP sur $BRIDGE ?)"

step "Installation du service"
pct push "$CTID" "$TMP/patate" /usr/local/bin/patate --perms 755
cat > "$TMP/service" <<EOF
#!/sbin/openrc-run
description="Patate, jeu multijoueur"
command=/usr/local/bin/patate
command_background=true
command_user=nobody
pidfile=/run/patate.pid
output_log=/var/log/patate.log
error_log=/var/log/patate.log
export PORT=$PORT
EOF
pct push "$CTID" "$TMP/service" /etc/init.d/patate --perms 755
pct exec "$CTID" -- sh -c 'touch /var/log/patate.log && chown nobody /var/log/patate.log && rc-update add patate default >/dev/null && rc-service patate start >/dev/null'

up=0
for _ in $(seq 1 10); do
  if pct exec "$CTID" -- wget -qO /dev/null "http://127.0.0.1:$PORT/" 2>/dev/null; then up=1 && break; fi
  sleep 1
done
[ "$up" = 1 ] || die "Le jeu ne répond pas (logs : pct exec $CTID -- cat /var/log/patate.log)"

printf '\n'
ok "En ligne : http://$IP:$PORT  (Patate)  ·  http://$IP:$PORT/ricochet.html  (Ricochet)"
printf '  Conteneur : %s (%s)\n' "$CTID" "$CT_HOSTNAME"
printf '  Mise à jour : CTID=%s bash -c "$(curl -fsSL https://raw.githubusercontent.com/LBouzac/patate/main/install.sh)"\n' "$CTID"

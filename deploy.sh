#!/usr/bin/env bash
# Compile Patate, l'envoie dans le conteneur (LXC Alpine ou Debian) et (re)démarre le service.
# Usage : ./deploy.sh root@192.168.1.50        (port du jeu : PORT=8080 par défaut)
set -euo pipefail

HOST="${1:-${DEPLOY_HOST:-}}"
[ -n "$HOST" ] || { echo "Usage : $0 root@IP_DU_CONTENEUR" >&2; exit 1; }
PORT="${PORT:-8080}"
[[ "$PORT" =~ ^[0-9]+$ ]] || { echo "PORT invalide : $PORT" >&2; exit 1; }
cd "$(dirname "$0")"
command -v go >/dev/null || { echo "Go n'est pas installé (sudo pacman -S go)" >&2; exit 1; }

BIN="$(mktemp)"
trap 'rm -f "$BIN"' EXIT

echo "→ Tests"
go test ./...
echo "→ Compilation (linux/${GOARCH:-amd64})"
CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH:-amd64}" go build -ldflags="-s -w" -o "$BIN" .

echo "→ Envoi vers $HOST"
ssh "$HOST" 'mkdir -p /opt/patate && chown nobody /opt/patate && chmod 755 /opt/patate'
scp -q "$BIN" "$HOST:/opt/patate/patate.new"

# Côté conteneur : remplace le binaire, (ré)écrit le service en gardant le mot de passe admin, redémarre.
# Le binaire appartient à nobody pour pouvoir se mettre à jour lui-même depuis /admin.html.
ssh "$HOST" "PORT=$PORT sh -s" <<'REMOTE'
set -e
mv /opt/patate/patate.new /opt/patate/patate
chmod 755 /opt/patate/patate && chown nobody /opt/patate/patate
rm -f /usr/local/bin/patate
PASS="$(sed -n 's/^export ADMIN_PASSWORD=//p; s/^Environment=ADMIN_PASSWORD=//p' /etc/init.d/patate /etc/systemd/system/patate.service 2>/dev/null | head -1)"
echo "$PASS" | grep -Eq '^[A-Za-z0-9]{12,}$' || PASS="$(tr -dc 'A-Za-z0-9' </dev/urandom | head -c 20 || true)"
if command -v rc-service >/dev/null; then # Alpine (OpenRC)
  cat > /etc/init.d/patate <<EOS
#!/sbin/openrc-run
description="Patate, jeu multijoueur"
command=/opt/patate/patate
command_background=true
command_user=nobody
pidfile=/run/patate.pid
output_log=/var/log/patate.log
error_log=/var/log/patate.log
export PORT=$PORT
export ADMIN_PASSWORD=$PASS
EOS
  chmod 700 /etc/init.d/patate # contient le mot de passe : root seulement
  touch /var/log/patate.log && chown nobody /var/log/patate.log
  rc-update add patate default >/dev/null 2>&1 || true
  rc-service patate restart
else # Debian/Ubuntu (systemd)
  cat > /etc/systemd/system/patate.service <<EOS
[Unit]
Description=Patate, jeu multijoueur
After=network.target

[Service]
ExecStart=/opt/patate/patate
Environment=PORT=$PORT
Environment=ADMIN_PASSWORD=$PASS
User=nobody
Restart=on-failure
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOS
  chmod 600 /etc/systemd/system/patate.service
  systemctl daemon-reload
  systemctl enable --now patate >/dev/null 2>&1
  systemctl restart patate
fi
echo "  Mises à jour depuis le site : /admin.html, mot de passe : $PASS"
REMOTE

echo "✓ En ligne : http://${HOST#*@}:$PORT"

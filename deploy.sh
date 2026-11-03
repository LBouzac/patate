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
scp -q "$BIN" "$HOST:/usr/local/bin/patate.new"

# Côté conteneur : remplace le binaire, crée le service au premier déploiement, redémarre.
ssh "$HOST" "PORT=$PORT sh -s" <<'REMOTE'
set -e
mv /usr/local/bin/patate.new /usr/local/bin/patate
chmod 755 /usr/local/bin/patate
if command -v rc-service >/dev/null; then # Alpine (OpenRC)
  cat > /etc/init.d/patate <<EOS
#!/sbin/openrc-run
description="Patate, jeu multijoueur"
command=/usr/local/bin/patate
command_background=true
command_user=nobody
pidfile=/run/patate.pid
output_log=/var/log/patate.log
error_log=/var/log/patate.log
export PORT=$PORT
EOS
  chmod +x /etc/init.d/patate
  touch /var/log/patate.log && chown nobody /var/log/patate.log
  rc-update add patate default >/dev/null 2>&1 || true
  rc-service patate restart
else # Debian/Ubuntu (systemd)
  cat > /etc/systemd/system/patate.service <<EOS
[Unit]
Description=Patate, jeu multijoueur
After=network.target

[Service]
ExecStart=/usr/local/bin/patate
Environment=PORT=$PORT
User=nobody
Restart=on-failure
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOS
  systemctl daemon-reload
  systemctl enable --now patate >/dev/null 2>&1
  systemctl restart patate
fi
REMOTE

echo "✓ En ligne : http://${HOST#*@}:$PORT"

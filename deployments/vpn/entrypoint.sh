#!/bin/sh
# One userspace WG tunnel per line of keys.txt ("<private_key> <address>"),
# each exposed as SOCKS5 on 1080+i. Every (re)start picks a random Mullvad relay = new IP.
set -eu

KEYS=/etc/wireguard/keys.txt
RELAYS=/tmp/relays
LIST=/run/proxies/list

curl -fsS https://api.mullvad.net/public/relays/wireguard/v2 \
  | jq -r '.wireguard.relays[] | select(.active) | "\(.ipv4_addr_in) \(.public_key)"' > "$RELAYS"
[ -s "$RELAYS" ] || { echo "no relays from mullvad api" >&2; exit 1; }

tunnel() {
  i=$1 key=$2 addr=$3
  while :; do
    read -r ip pub <<EOF
$(shuf -n1 "$RELAYS")
EOF
    cat > "/tmp/wg$i.conf" <<EOF
[Interface]
PrivateKey = $key
Address = $addr
DNS = 10.64.0.1

[Peer]
PublicKey = $pub
Endpoint = $ip:51820
AllowedIPs = 0.0.0.0/0

[Socks5]
BindAddress = 0.0.0.0:$((1080 + i))
EOF
    echo "tunnel $i -> $ip"
    wireproxy -c "/tmp/wg$i.conf" || true
    sleep 0.2
  done
}

: > "$LIST"
i=0
while read -r key addr; do
  case "$key" in ''|\#*) continue ;; esac
  echo "socks5://vpn:$((1080 + i))" >> "$LIST"
  tunnel "$i" "$key" "$addr" &
  i=$((i + 1))
done < "$KEYS"

busybox-extras httpd -f -p 8080 -h /www &  # rotate API: GET /cgi-bin/rotate?<i>

wait

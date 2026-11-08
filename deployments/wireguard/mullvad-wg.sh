#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0
#
# Copyright (C) 2016-2018 Jason A. Donenfeld <Jason@zx2c4.com>. All Rights Reserved.
#
# Writes /etc/wireguard/<relay>.conf for every Mullvad relay, for use with `wg-quick up <relay>`.

die() { echo "[-] Error: $1" >&2; exit 1; }

[[ $UID == 0 ]] || exec sudo -p "[?] ${0##*/} must be run as root. Please enter the password for %u to continue: " -- "$BASH" -- "$(realpath "$0")" "$@"

type curl >/dev/null || die "Please install curl and then try again."
type jq >/dev/null || die "Please install jq and then try again."
set -e

read -p "[?] Please enter your Mullvad account number: " -r ACCOUNT

echo "[+] Contacting Mullvad API for server locations."
RESPONSE="$(curl -LsS https://api.mullvad.net/public/relays/wireguard/v1/)" || die "Unable to connect to Mullvad API."
# one line per relay: <code> TAB <city, country> TAB <public key> TAB <ipv4>
RELAYS="$(jq -r '.countries[] as $c | $c.cities[] as $city | $city.relays[]
  | [(.hostname | sub("-wireguard$"; "")), "\($city.name), \($c.name)", .public_key, .ipv4_addr_in] | @tsv' <<<"$RESPONSE")" || die "Unable to parse response."

PRIVATE_KEY=
while IFS=$'\t' read -r code _; do
	[[ -f /etc/wireguard/$code.conf ]] || continue
	PRIVATE_KEY="$(sed -nE 's/^\s*PrivateKey\s*=\s*([a-zA-Z0-9+/]{43}=)\s*$/\1/Ip' "/etc/wireguard/$code.conf" | head -n1)"
	if [[ -n $PRIVATE_KEY ]]; then echo "[+] Using existing private key."; break; fi
done <<<"$RELAYS"

if [[ -z $PRIVATE_KEY ]]; then
	echo "[+] Generating new private key."
	PRIVATE_KEY="$(wg genkey)"
fi

echo "[+] Contacting Mullvad API."
ADDRESS="$(curl -sSL https://api.mullvad.net/wg -d account="$ACCOUNT" --data-urlencode pubkey="$(wg pubkey <<<"$PRIVATE_KEY")")" || die "Could not talk to Mullvad API."
[[ $ADDRESS =~ ^[0-9a-f:/.,]+$ ]] || die "$ADDRESS"

echo "[+] Writing WireGuard configuration files."
umask 077
mkdir -p /etc/wireguard/
while IFS=$'\t' read -r code _ pubkey ip; do
	f=/etc/wireguard/$code.conf
	rm -f "$f.tmp"
	cat > "$f.tmp" <<-_EOF
		[Interface]
		PrivateKey = $PRIVATE_KEY
		Address = $ADDRESS
		DNS = 10.64.0.1

		[Peer]
		PublicKey = $pubkey
		Endpoint = $ip:51820
		AllowedIPs = 0.0.0.0/0, ::/0
	_EOF
	mv "$f.tmp" "$f"
done <<<"$RELAYS"

echo "[+] Success. The following commands may be run for connecting to Mullvad:"
while IFS=$'\t' read -r code location _; do
	printf -- '- %s:\n  $ wg-quick up %s\n' "$location" "$code"
done <<<"$RELAYS"

echo "Please wait up to 60 seconds for your public key to be added to the servers."

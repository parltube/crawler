#!/usr/bin/env bash
# setup_tunnel.sh <mullvad_account> [n=5]
# Registers n WG keys on the account and appends them to deployments/wireguard/keys.txt.
set -euo pipefail

account="${1:?mullvad account number}"; n="${2:-5}"
out="$(dirname "$0")/../wireguard/keys.txt"

for _ in $(seq "$n"); do
  key="$(wg genkey)"
  addr="$(curl -fsS https://api.mullvad.net/wg -d account="$account" \
    --data-urlencode pubkey="$(wg pubkey <<<"$key")")"
  echo "$key ${addr%%,*}" >> "$out"
done

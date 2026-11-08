#!/bin/sh
# GET http://vpn:8080/cgi-bin/rotate?<i> - reconnect tunnel i to another random relay (new IP).
# Ignored if the tunnel is younger than 5s, so a burst of 429s from many workers rotates once.
i=${QUERY_STRING##*=}
case "$i" in ''|*[!0-9]*) printf 'Status: 400\r\n\r\n'; exit ;; esac
f=/tmp/wg$i.conf
[ -f "$f" ] && [ $(( $(date +%s) - $(stat -c %Y "$f") )) -ge 5 ] && pkill -f "$f"
printf 'Content-Type: text/plain\r\n\r\nok\n'

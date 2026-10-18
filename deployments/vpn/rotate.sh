#!/bin/sh
# rotate.sh <i> - reconnect tunnel i to another random relay (new IP).
# Ignored if the tunnel is younger than 5s, so a burst of 429s from many workers rotates once.
f=/tmp/wg$1.conf
[ -f "$f" ] && [ $(( $(date +%s) - $(stat -c %Y "$f") )) -ge 5 ] && pkill -f "$f"
exit 0

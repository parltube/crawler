#!/bin/sh
# GET http://vpn:8080/cgi-bin/rotate?<i>
i=${QUERY_STRING##*=}
case "$i" in ''|*[!0-9]*) printf 'Status: 400\r\n\r\n'; exit ;; esac
rotate.sh "$i"
printf 'Content-Type: text/plain\r\n\r\nok\n'

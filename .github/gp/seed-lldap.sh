#!/bin/sh
# Seed an LLDAP server with two ordinary users, alice and bob, for the ldap_integration
# test. Run inside the lldap container, which has busybox wget (no curl) and
# lldap_set_password.
#   ADMIN_PASSWORD  the server's LLDAP_LDAP_USER_PASS
#   USER_PASSWORD   the password both users get
set -eu
base=http://127.0.0.1:17170

i=0
until wget -q -O /dev/null "$base/health"; do
  i=$((i + 1)); [ "$i" -lt 60 ] || { echo "lldap did not come up" >&2; exit 1; }
  sleep 1
done

token=$(wget -q -O - --header 'Content-Type: application/json' \
  --post-data "{\"username\":\"admin\",\"password\":\"$ADMIN_PASSWORD\"}" \
  "$base/auth/simple/login" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
[ -n "$token" ] || { echo "admin login failed" >&2; exit 1; }

for user in alice bob; do
  wget -q -O - --header 'Content-Type: application/json' --header "Authorization: Bearer $token" \
    --post-data "{\"query\":\"mutation{createUser(user:{id:\\\"$user\\\",email:\\\"$user@example.com\\\"}){id}}\"}" \
    "$base/api/graphql"
  echo
  /app/lldap_set_password --base-url "$base" --admin-username admin \
    --admin-password "$ADMIN_PASSWORD" --username "$user" --password "$USER_PASSWORD"
done

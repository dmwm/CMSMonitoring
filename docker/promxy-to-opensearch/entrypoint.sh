#!/bin/bash
set -euo pipefail

KEYTAB_PATH="${KRB5_CLIENT_KTNAME:-/etc/krb5.keytab}"

# Extract principal from keytab (last one listed)
principal=$(klist -k "$KEYTAB_PATH" | tail -1 | awk '{print $2}')

echo "[INFO] Using principal: $principal"

# Authenticate with Kerberos
if ! kinit "$principal" -k -t "$KEYTAB_PATH" >/dev/null; then
    echo "[ERROR] Kerberos authentication failed using keytab: $KEYTAB_PATH"
    exit 1
fi

# Strip @REALM to get short name (optional)
SHORT_NAME=$(echo "$principal" | grep -o '^[^@]*')
echo "[INFO] Kerberos auth succeeded. Principal short name: $SHORT_NAME"

python3 "/app/promxy_to_opensearch.py"


#!/usr/bin/env bash

set -euo pipefail

# Verify a file is a valid minisign public key (not a secret key).

FILE="${1:?'usage: verify-minisign-public-key.sh <file>'}"

if [ ! -f "$FILE" ]; then
	echo "error: file not found: $FILE" >&2
	exit 1
fi

# minisign public keys start with "untrusted comment:" and contain base64-encoded key
if ! head -1 "$FILE" | grep -q "^untrusted comment:"; then
	echo "error: $FILE does not appear to be a minisign public key" >&2
	exit 1
fi

# Check it's not a secret key (those contain "SECRET KEY")
if grep -q "SECRET KEY" "$FILE"; then
	echo "error: $FILE appears to be a SECRET key, not a public key!" >&2
	exit 1
fi

echo "✅ $FILE is a valid minisign public key"

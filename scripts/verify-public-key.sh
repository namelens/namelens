#!/usr/bin/env bash

set -euo pipefail

# Verify a file is a valid PGP public key (not a secret key).

FILE="${1:?'usage: verify-public-key.sh <file>'}"

if [ ! -f "$FILE" ]; then
	echo "error: file not found: $FILE" >&2
	exit 1
fi

# Check it's an ASCII-armored public key
if ! grep -q "BEGIN PGP PUBLIC KEY BLOCK" "$FILE"; then
	echo "error: $FILE does not appear to be a PGP public key" >&2
	exit 1
fi

# Check it's not a private key
if grep -q "PRIVATE KEY" "$FILE"; then
	echo "error: $FILE appears to be a PRIVATE key, not a public key!" >&2
	exit 1
fi

echo "✅ $FILE is a valid PGP public key"

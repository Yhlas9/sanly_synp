#!/bin/bash
# Ätiýaçlyk nusgadan maglumatlary dikeltmek.
set -e
cd "$(dirname "$0")"
export PATH="/opt/homebrew/opt/postgresql@16/bin:$PATH"

FILE="${1:-$(ls -t backups/*.sql 2>/dev/null | head -1)}"
if [ -z "$FILE" ] || [ ! -f "$FILE" ]; then
  echo "Ulanylyşy: ./restore.sh backups/sanly-synp-....sql"
  exit 1
fi

echo "Dikeldilýär: $FILE"
psql -q -d sanly_synp -f "$FILE"

# Şol bir wagtda alnan faýllar bar bolsa, olar hem dikeldilýär.
STAMP=$(basename "$FILE" .sql | sed 's/^sanly-synp-//')
ARCHIVE="backups/uploads-$STAMP.tar.gz"
if [ -f "$ARCHIVE" ]; then
  tar -xzf "$ARCHIVE"
  echo "Faýllar dikeldildi: $ARCHIVE"
fi

echo "Maglumatlar dikeldildi."

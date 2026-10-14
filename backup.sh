#!/bin/bash
# Maglumatlaryň ätiýaçlyk nusgasyny döretmek (ТЗ, 10-njy bölüm).
set -e
cd "$(dirname "$0")"
export PATH="/opt/homebrew/opt/postgresql@16/bin:$PATH"

mkdir -p backups
STAMP=$(date +%Y%m%d-%H%M%S)
OUT="backups/sanly-synp-$STAMP.sql"
pg_dump --clean --if-exists sanly_synp > "$OUT"

# Amaly işleriň faýllary maglumat binýadynda däl, diskde saklanýar —
# şonuň üçin olar hem nusga goşulýar.
if [ -d uploads ] && [ -n "$(ls -A uploads 2>/dev/null)" ]; then
  tar -czf "backups/uploads-$STAMP.tar.gz" uploads
  echo "Faýllar: backups/uploads-$STAMP.tar.gz ($(du -h "backups/uploads-$STAMP.tar.gz" | cut -f1))"
fi

echo "Ätiýaçlyk nusga taýýar: $OUT ($(du -h "$OUT" | cut -f1))"

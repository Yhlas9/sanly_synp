#!/bin/bash
# Sanly Synp — mugallymyň kompýuterinde işe girizmek.
# Internet gerek däl: diňe ýerli PostgreSQL we ýygnalan programma.
set -e
cd "$(dirname "$0")"

export PATH="/opt/homebrew/opt/postgresql@16/bin:$PATH"
export DATABASE_URL="${DATABASE_URL:-postgres://localhost:5432/sanly_synp?sslmode=disable}"
export ADDR="${ADDR:-:8080}"

if ! pg_isready -q 2>/dev/null; then
  echo "PostgreSQL işlemeýär. Işe girizilýär..."
  brew services start postgresql@16
  sleep 3
fi

createdb sanly_synp 2>/dev/null || true

if [ ! -x ./sanly-synp ]; then
  echo "Programma ýygnalýar..."
  go build -o sanly-synp ./cmd/server
fi

exec ./sanly-synp

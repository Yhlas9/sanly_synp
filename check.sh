#!/bin/bash
# Görkezmeden öň barlag. Eminleriň barlajak zatlaryny bir buýruk bilen geçirýär.
# Ulanylyşy:  ./check.sh
cd "$(dirname "$0")"
export PATH="/opt/homebrew/opt/postgresql@16/bin:$PATH"

B="${ADDR_CHECK:-localhost:8080}"
PASS=0; WARN=0; FAIL=0

ok()   { printf "  \033[32m✓\033[0m %s\n" "$1"; PASS=$((PASS+1)); }
warn() { printf "  \033[33m!\033[0m %s\n" "$1"; WARN=$((WARN+1)); }
bad()  { printf "  \033[31m✗\033[0m %s\n" "$1"; FAIL=$((FAIL+1)); }

echo
echo "  SANLY SYNP — görkezmeden öňki barlag"
echo "  ════════════════════════════════════"
echo
echo "  Gurşaw"

pg_isready -q 2>/dev/null && ok "PostgreSQL işleýär" || bad "PostgreSQL işlemeýär → brew services start postgresql@16"
command -v pg_dump >/dev/null && ok "pg_dump elýeterli (ätiýaçlyk nusga üçin)" || bad "pg_dump PATH-da ýok"

if GOFLAGS=-mod=vendor GOPROXY=off go build -o /tmp/_check_build ./cmd/server 2>/dev/null; then
  ok "Programma internetsiz ýygnalýar"
  rm -f /tmp/_check_build
else
  bad "Internetsiz ýygnalmaýar → go mod vendor"
fi

echo
echo "  Serwer"

if curl -sf -o /dev/null "http://$B/api/health"; then
  ok "Serwer jogap berýär ($B)"
else
  bad "Serwer jogap bermeýär → ./start.sh"
  echo; echo "  Serwer işlemese galan barlaglar geçirilmeýär."; echo; exit 1
fi

for p in / /okuwcy /materiallar /dolandyrys /hasabat; do
  code=$(curl -s -o /dev/null -w '%{http_code}' "http://$B$p")
  [ "$code" = "200" ] && ok "Sahypa $p" || bad "Sahypa $p → $code"
done

echo
echo "  Oflaýn işlemek"

ext=$(grep -rhoE 'https?://[a-zA-Z0-9.-]+' web/*.html web/css web/js 2>/dev/null \
      | grep -v 'w3.org' | grep -v 'localhost' | sort -u)
[ -z "$ext" ] && ok "web/ içinde daşarky salgy ýok" \
  || { bad "Daşarky salgy tapyldy:"; echo "$ext" | sed 's/^/      /'; }

[ -f web/assets/vue.global.prod.js ] && ok "Vue ýerli faýlda" || bad "Vue faýly ýok"
[ -f web/assets/qrcode.js ] && ok "QR kitaphanasy ýerli faýlda" || bad "QR faýly ýok"
grep -q -- "--sans:.*-apple-system" web/css/app.css && ok "Şriftler ulgamyňky (ýüklenmeýär)" \
  || warn "Şrift sazlamasyny barlaň"

echo
echo "  Ýerli tor"

IPS=$(curl -s "http://$B/api/net" | python3 -c "import sys,json;print(' '.join(json.load(sys.stdin)['student_urls']))" 2>/dev/null)
if [ -n "$IPS" ]; then
  ok "Okuwçylar üçin salgy bar:"
  for u in $IPS; do echo "      http://$u"; done
else
  bad "Ýerli tor salgysy tapylmady → Wi-Fi ýa-da hotspot açyň"
fi

echo
echo "  Mazmun"

CK=$(mktemp)
curl -s -X POST "http://$B/api/login" -d '{"login":"mugallym","password":"mugallym"}' -c "$CK" -o /dev/null
if curl -sf "http://$B/api/subjects" -b "$CK" -o /dev/null; then
  warn "mugallym hasabynyň açar sözi başlangyç ýagdaýda — ÇALŞYŇ"
else
  ok "mugallym hasabynyň açar sözi çalşylan"
fi
curl -s -X POST "http://$B/api/login" -d '{"login":"admin","password":"admin"}' -c "$CK" -o /dev/null
if curl -sf "http://$B/api/admin/stats" -b "$CK" -o /dev/null; then
  warn "admin hasabynyň açar sözi başlangyç ýagdaýda — ÇALŞYŇ"
else
  ok "admin hasabynyň açar sözi çalşylan"
fi

CNT=$(psql -qtA -d sanly_synp -c "select count(*) from lessons where demo_kind <> ''" 2>/dev/null)
[ "${CNT:-0}" -ge 5 ] && ok "$CNT sapak interaktiw görkezme bilen" \
  || bad "Interaktiw görkezmeli sapak az: ${CNT:-0} (5 bolmaly)"

QN=$(psql -qtA -d sanly_synp -c "select count(*) from questions" 2>/dev/null)
[ "${QN:-0}" -ge 50 ] && ok "$QN sorag taýýar" || warn "Sorag az: ${QN:-0}"

KINDS=$(psql -qtA -d sanly_synp -c "select count(distinct kind) from questions" 2>/dev/null)
[ "${KINDS:-0}" -ge 7 ] && ok "Ýedi görnüşli ýumuş hem bar" || bad "Ýumuş görnüşi ${KINDS:-0}/7"

echo
echo "  Ätiýaçlyk nusga"

LAST=$(ls -t backups/*.sql 2>/dev/null | head -1)
if [ -n "$LAST" ]; then
  AGE=$(( ($(date +%s) - $(stat -f %m "$LAST")) / 60 ))
  [ "$AGE" -lt 180 ] && ok "Soňky nusga $AGE minut öň: $(basename "$LAST")" \
    || warn "Soňky nusga $AGE minut öň — täzesini dörediň (./backup.sh)"
else
  bad "Ätiýaçlyk nusga ýok → ./backup.sh"
fi

rm -f "$CK"

echo
echo "  ════════════════════════════════════"
printf "  Geçdi: %d · Duýduryş: %d · Ýykyldy: %d\n" "$PASS" "$WARN" "$FAIL"
echo
if [ "$FAIL" -gt 0 ]; then
  echo "  Çykyşa taýýar däl — ýykylan barlaglary düzediň."
  echo
  exit 1
fi
cat <<'TXT'
  El bilen barlamaly (skript barlap bilmeýär):
    1. Mugallymyň paneli localhost arkaly açylýar          → ekran görkezmek işlemeli
    2. Telefonda ýerli salgy açylýar we sapaga goşulýar     → 1 minutda
    3. Interneti öçüreniňde platforma işlemegini dowam edýär
    4. Tory kesip-dikeldeniňde jogap ýitmeýär
    5. Proýektorda tekst okalýar

TXT

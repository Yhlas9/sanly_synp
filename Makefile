# Sanly Synp — ýygnamak, işe girizmek we hyzmat buýruklary.
#
# Doly sanaw:  make help
#
# ── Nähili işe girýär ────────────────────────────────────────────────
#
#   make run
#
# Bu bir buýruk şu ädimleri yzygiderli ýerine ýetirýär:
#
#   1. PostgreSQL işlemeýän bolsa — işe girizilýär (brew services);
#   2. `sanly_synp` binýady ýok bolsa — döredilýär (createdb);
#   3. programma `vendor/` arkaly ýygnalýar — internet gerek däl;
#   4. serwer göterilýär we ekranda salgylary görkezýär.
#
# Migrasiýalar (`migrations/*.sql`), başlangyç hasaplar we nusgalyk
# mazmun (`seed/content.sql`) serweriň özi tarapyndan her gezek
# awtomatik ulanylýar — aýratyn buýruk gerek däl.
#
# Serwer göterilenden soň:
#
#   Mugallym .... http://localhost:8080/        (mugallym / mugallym)
#   Dolandyryjy . http://localhost:8080/dolandyrys   (admin / admin)
#   Okuwçylar ... make urls  →  ýerli tordaky salgylar
#
# Duruzmak: Ctrl+C, ýa-da başga penjireden — make stop
#
# ── Sazlamalar ───────────────────────────────────────────────────────
#
# Üýtgeýjiler buýruk setirinden berilýär:
#
#   make run ADDR=:9000
#   make run DATABASE_URL='postgres://localhost:5432/test?sslmode=disable' DB=test
#
# ─────────────────────────────────────────────────────────────────────

DATABASE_URL ?= postgres://localhost:5432/sanly_synp?sslmode=disable
ADDR         ?= :8080
DB           ?= sanly_synp
BIN          ?= sanly-synp
PKG          ?= ./cmd/server
PGBIN        ?= /opt/homebrew/opt/postgresql@16/bin
PGSERVICE    ?= postgresql@16

export DATABASE_URL
export ADDR
export PATH := $(PGBIN):$(PATH)

# ADDR ":8080" görnüşinde, port diňe san hökmünde hem gerek (lsof, curl).
PORT := $(patsubst :%,%,$(ADDR))

# Internetsiz ýygnamak: baglylyklar vendor/ bukjasyndan alynýar.
GO_OFFLINE := GOFLAGS=-mod=vendor GOPROXY=off

.DEFAULT_GOAL := help
.PHONY: help run dev build db psql urls stop check backup restore reset-demo \
        fmt vet test vendor tidy clean db-reset

# ── Esasy ────────────────────────────────────────────────────────────

help: ## Şu sanawy görkezmek
	@echo ""
	@echo "  Sanly Synp — buýruklar"
	@echo "  ──────────────────────"
	@grep -hE '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) \
	  | sed 's/:[^#]*## /|/' \
	  | awk -F'|' '{ printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }'
	@echo ""
	@echo "  Işe girizmek:  make run        Salgylar:  make urls"
	@echo "  Jikme-jik düşündiriş Makefile-yň başynda we README.md-de."
	@echo ""

run: build db ## Işe girizmek (binýat + ýygnamak + serwer)
	@echo "Serwer göterilýär: $(ADDR)  (duruzmak üçin Ctrl+C)"
	./$(BIN)

dev: db ## Ýygnamazdan işe girizmek (go run — işläp düzmek üçin)
	$(GO_OFFLINE) go run $(PKG)

build: ## Programmany ýygnamak (internetsiz, vendor arkaly)
	$(GO_OFFLINE) go build -o $(BIN) $(PKG)
	@echo "Ýygnaldy: ./$(BIN)"

stop: ## Serweri duruzmak (porty boşatmak)
	@pid=$$(lsof -ti tcp:$(PORT)); \
	if [ -n "$$pid" ]; then kill $$pid && echo "Duruzyldy (pid $$pid)"; \
	else echo "$(PORT) portunda hiç zat işlemeýär."; fi

urls: ## Okuwçylar üçin ýerli tor salgylaryny görkezmek
	@curl -sf "http://localhost:$(PORT)/api/net" \
	  | python3 -c 'import sys,json; [print("  http://"+u) for u in json.load(sys.stdin)["student_urls"]]' \
	  || echo "Serwer jogap bermeýär — ilki: make run"

# ── Maglumat binýady ─────────────────────────────────────────────────

db: ## PostgreSQL işe girizmek we binýady döretmek
	@pg_isready -q 2>/dev/null || { \
	  echo "PostgreSQL işlemeýär, işe girizilýär..."; \
	  brew services start $(PGSERVICE); \
	  sleep 3; \
	}
	@createdb $(DB) 2>/dev/null && echo "Binýat döredildi: $(DB)" || true

psql: ## Maglumat binýadyna birikmek
	@psql -d $(DB)

db-reset: ## Binýady doly pozup, täzeden döretmek (CONFIRM=yes gerek)
	@test "$(CONFIRM)" = "yes" || { \
	  echo "Bu ÄHLI maglumatlary pozýar. Ynanýan bolsaň:"; \
	  echo "  make db-reset CONFIRM=yes"; \
	  exit 1; \
	}
	dropdb --if-exists $(DB)
	createdb $(DB)
	@echo "Binýat arassa. Shema we nusgalyk mazmun 'make run' bilen dolýar."

# ── Hyzmat skriptleri ────────────────────────────────────────────────

check: ## Görkezmeden öňki barlag (serwer işläp durmaly)
	./check.sh

backup: ## Ätiýaçlyk nusga döretmek (backups/ bukjasyna)
	./backup.sh

restore: ## Nusgadan dikeltmek (FILE=backups/....sql, boş bolsa — soňkusy)
	./restore.sh $(FILE)

reset-demo: ## Synag maglumatlaryny arassalamak (nusgalyk mazmun galýar)
	./reset-demo.sh

# ── Kod ──────────────────────────────────────────────────────────────

fmt: ## Kody formatlamak
	go fmt $(shell go list ./... 2>/dev/null | grep -v /vendor/)

vet: ## Kody barlamak (go vet)
	$(GO_OFFLINE) go vet $(PKG) ./internal/...

test: ## Testleri geçirmek
	$(GO_OFFLINE) go test ./internal/... $(PKG)

vendor: ## Baglylyklary vendor/ bukjasyna täzeden ýazmak (internet gerek)
	go mod vendor
	@echo "vendor/ täzelendi — indi internetsiz ýygnalýar."

tidy: ## go.mod / go.sum tertibe salmak (internet gerek)
	go mod tidy

clean: ## Ýygnalan faýly pozmak
	rm -f $(BIN)
	@echo "Pozuldy: $(BIN)"


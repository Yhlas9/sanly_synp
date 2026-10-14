// Sanly Synp — оффлайн учебная платформа. Запускается на компьютере учителя
// и работает в локальной сети без интернета.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sanlysynp/internal/api"
	"sanlysynp/internal/hub"
	"sanlysynp/internal/store"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	var (
		dsn        = env("DATABASE_URL", "postgres://localhost:5432/sanly_synp?sslmode=disable")
		addr       = env("ADDR", ":8080")
		webDir     = env("WEB_DIR", "web")
		migrateDir = env("MIGRATIONS_DIR", "migrations")
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	st, err := store.Open(ctx, dsn)
	if err != nil {
		log.Fatalf("maglumat binýady: %v", err)
	}
	defer st.Close()

	if err := st.Migrate(ctx, migrateDir); err != nil {
		log.Fatalf("migrasiýa: %v", err)
	}
	if err := st.EnsureSeed(ctx); err != nil {
		log.Fatalf("başlangyç maglumatlar: %v", err)
	}
	if err := st.SeedContent(ctx, env("SEED_FILE", "seed/content.sql")); err != nil {
		log.Fatalf("nusgalyk mazmun: %v", err)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(st, hub.New(), webDir, dsn).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		// Демонстрация экрана и долгие WebSocket держатся без таймаута записи.
		IdleTimeout: 120 * time.Second,
	}

	go func() {
		printBanner(srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("ýapylýar...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
}

// printBanner показывает адреса, которые учитель диктует ученикам.
// Требование приёмки: новый ученик подключается к уроку за одну минуту.
func printBanner(addr string) {
	port := addr
	if _, p, err := net.SplitHostPort(addr); err == nil && p != "" {
		port = p
	}

	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════════╗")
	fmt.Println("  ║   SANLY SYNP — oflaýn okuw platformasy        ║")
	fmt.Println("  ╚══════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("  Mugallym:  http://localhost:%s/\n", port)
	fmt.Println()
	fmt.Println("  Okuwçylar üçin salgylar (şol bir Wi-Fi ýa-da hotspot):")

	for _, ip := range localIPs() {
		fmt.Printf("     http://%s:%s/okuwcy\n", ip, port)
	}

	fmt.Println()
	fmt.Println("  Giriş:  mugallym / mugallym")
	fmt.Println()
}

func localIPs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	var out []string
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipnet.IP.To4(); ip4 != nil {
			out = append(out, ip4.String())
		}
	}
	return out
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sb2sub/internal/buildinfo"
	"sb2sub/internal/config"
	"sb2sub/internal/db"
	"sb2sub/internal/project"
	"sb2sub/internal/render"
	"sb2sub/internal/server"
	"sb2sub/internal/service"
)

func main() {
	baseDir := flag.String("base-dir", "/opt/sb2sub", "runtime base directory")
	listenAddr := flag.String("listen", "127.0.0.1:18080", "daemon listen address")
	showVersion := flag.Bool("version", false, "print version and exit")
	mode := flag.String("mode", "print-layout", "daemon mode: print-layout or serve")
	flag.Parse()

	if *showVersion {
		info := buildinfo.Info()
		fmt.Printf("sb2subd %s (%s) built %s\n", info.Version, info.Commit, info.BuiltAt)
		return
	}

	layout := project.DefaultLayout(*baseDir)
	cfg, err := config.Load(layout.ConfigFile)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if *mode == "render-singbox" {
		store, err := db.Open(layout.DatabaseFile)
		if err != nil {
			log.Fatalf("open database: %v", err)
		}
		defer store.Close()

		if err := store.Migrate(); err != nil {
			log.Fatalf("migrate database: %v", err)
		}

		svc := service.New(store)
		users, err := svc.ListUsers()
		if err != nil {
			log.Fatalf("list users: %v", err)
		}

		doc, err := render.RenderSingBox(cfg, render.RuntimeUsersFromModel(users))
		if err != nil {
			log.Fatalf("render sing-box: %v", err)
		}
		fmt.Fprintln(os.Stdout, string(doc))
		return
	}

	if *mode != "serve" {
		fmt.Fprintf(os.Stdout, "sb2subd bootstrap: config=%s db=%s\n", layout.ConfigFile, layout.DatabaseFile)
		return
	}

	store, err := db.Open(layout.DatabaseFile)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	handler := server.NewHandler(cfg, service.New(store))
	httpServer := &http.Server{
		Addr:              *listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("sb2subd listening on %s", *listenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve http: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown http server: %v", err)
	}
}

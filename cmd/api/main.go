package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	chicors "github.com/go-chi/cors"

	"github.com/test-task-front/wms/internal/auth"
	"github.com/test-task-front/wms/internal/config"
	"github.com/test-task-front/wms/internal/core/repo"
	"github.com/test-task-front/wms/internal/core/services"
	"github.com/test-task-front/wms/internal/db"
	httpHandlers "github.com/test-task-front/wms/internal/http/handlers"
	httpMiddleware "github.com/test-task-front/wms/internal/http/middleware"
	"github.com/test-task-front/wms/seed"
	"github.com/test-task-front/wms/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	database, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(ctx, database); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	if err := seed.Run(ctx, database); err != nil {
		log.Fatalf("seed data: %v", err)
	}

	userRepo := repo.NewUserRepository(database)
	itemRepo := repo.NewItemRepository(database)
	refreshRepo := repo.NewRefreshTokenRepository(database)

	jwtManager := auth.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	authService := services.NewAuthService(userRepo, refreshRepo, jwtManager, cfg.RefreshTokenTTL)
	itemService := services.NewItemService(itemRepo)

	authHandler := httpHandlers.NewAuthHandler(authService)
	itemsHandler := httpHandlers.NewItemsHandler(itemService, cfg.ArtificialDelay)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Logger)
	router.Use(chicors.New(chicors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigin,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{},
		MaxAge:           300,
		AllowCredentials: false,
	}).Handler)

	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	router.Post("/auth/login", authHandler.Login)
	router.Post("/auth/refresh", authHandler.Refresh)

	router.Group(func(r chi.Router) {
		r.Use(httpMiddleware.WithAuth(jwtManager))
		r.Get("/items", itemsHandler.List)
		r.Get("/items/{id}", itemsHandler.GetByID)
		r.Patch("/items/{id}", itemsHandler.PatchQty)
	})

	if swaggerFS, err := web.SwaggerFS(); err == nil {
		fileServer := http.FileServer(http.FS(swaggerFS))
		router.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
			data, err := fsReadFile(swaggerFS, "openapi.yaml")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/yaml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		})

		router.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/swagger/", http.StatusTemporaryRedirect)
		})
		router.Get("/swagger/", func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix("/swagger", fileServer).ServeHTTP(w, r)
		})
		router.Handle("/swagger/*", http.StripPrefix("/swagger", fileServer))
	} else {
		log.Printf("swagger assets unavailable: %v", err)
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.AppPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
}

func fsReadFile(fsys interface {
	Open(name string) (fs.File, error)
}, name string) ([]byte, error) {
	file, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

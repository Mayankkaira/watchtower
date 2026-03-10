package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"saas/auth"
	"saas/handlers"
	"saas/middleware"
	"saas/store"
)

func main() {
	// Config (use env vars in production)
	port := envOr("PORT", "8080")
	jwtSecret := envOr("JWT_SECRET", "change-me-to-a-real-secret-in-production")
	dbPath := envOr("DB_PATH", "saas.db")
	allowedOrigins := envOr("ALLOWED_ORIGINS", "*")

	// Security warnings
	if jwtSecret == "change-me-to-a-real-secret-in-production" {
		log.Println("WARNING: Using default JWT secret. Set JWT_SECRET env var in production.")
	}
	if allowedOrigins == "*" {
		log.Println("WARNING: CORS allows all origins. Set ALLOWED_ORIGINS env var in production.")
	}

	// Initialize store
	db, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize auth service
	authSvc := auth.NewService(jwtSecret, 24*time.Hour)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authSvc, db)
	apiHandler := handlers.NewAPIHandler(db)

	// Auth middleware
	requireAuth := middleware.Auth(authSvc, db)
	requirePro := middleware.RequirePlan("pro", "enterprise")

	// Setup routes
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /health", handlers.HealthCheck)
	mux.HandleFunc("POST /api/v1/auth/signup", authHandler.Signup)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)

	// Protected routes (any authenticated user)
	mux.Handle("GET /api/v1/me", middleware.Chain(
		http.HandlerFunc(apiHandler.GetProfile),
		requireAuth,
	))
	mux.Handle("PUT /api/v1/me", middleware.Chain(
		http.HandlerFunc(apiHandler.UpdateProfile),
		requireAuth,
	))
	mux.Handle("PUT /api/v1/me/plan", middleware.Chain(
		http.HandlerFunc(apiHandler.UpdatePlan),
		requireAuth,
	))
	mux.Handle("DELETE /api/v1/me", middleware.Chain(
		http.HandlerFunc(apiHandler.DeleteAccount),
		requireAuth,
	))
	mux.Handle("GET /api/v1/dashboard", middleware.Chain(
		http.HandlerFunc(apiHandler.Dashboard),
		requireAuth,
	))

	// API Key management
	mux.Handle("POST /api/v1/keys", middleware.Chain(
		http.HandlerFunc(apiHandler.CreateAPIKey),
		requireAuth,
	))
	mux.Handle("GET /api/v1/keys", middleware.Chain(
		http.HandlerFunc(apiHandler.ListAPIKeys),
		requireAuth,
	))
	mux.Handle("DELETE /api/v1/keys", middleware.Chain(
		http.HandlerFunc(apiHandler.DeleteAPIKey),
		requireAuth,
	))

	// Monitor management
	mux.Handle("POST /api/v1/monitors", middleware.Chain(
		http.HandlerFunc(apiHandler.CreateMonitor),
		requireAuth,
	))
	mux.Handle("GET /api/v1/monitors", middleware.Chain(
		http.HandlerFunc(apiHandler.ListMonitors),
		requireAuth,
	))
	mux.Handle("DELETE /api/v1/monitors", middleware.Chain(
		http.HandlerFunc(apiHandler.DeleteMonitor),
		requireAuth,
	))

	// Pro-only routes (example gated feature)
	mux.Handle("GET /api/v1/pro/analytics", middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"analytics": "premium data here", "revenue": 1000000}`))
		}),
		requireAuth,
		requirePro,
	))

	// Serve frontend static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("frontend"))))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend/index.html")
	})

	// Global middleware stack
	handler := middleware.Chain(mux,
		middleware.CORS(allowedOrigins),
		middleware.RateLimit(120),
		middleware.Logging,
	)

	// Start server
	fmt.Println(banner)
	log.Printf("Watchtower API running on http://localhost:%s", port)
	log.Printf("Endpoints:")
	log.Printf("   POST   /api/v1/auth/signup   - Create account")
	log.Printf("   POST   /api/v1/auth/login    - Login")
	log.Printf("   GET    /api/v1/me            - Get profile")
	log.Printf("   PUT    /api/v1/me            - Update profile")
	log.Printf("   PUT    /api/v1/me/plan       - Update plan")
	log.Printf("   DELETE /api/v1/me            - Delete account")
	log.Printf("   GET    /api/v1/dashboard     - Dashboard")
	log.Printf("   POST   /api/v1/keys          - Create API key")
	log.Printf("   GET    /api/v1/keys          - List API keys")
	log.Printf("   DELETE /api/v1/keys?id=N     - Delete API key")
	log.Printf("   POST   /api/v1/monitors      - Create monitor")
	log.Printf("   GET    /api/v1/monitors      - List monitors")
	log.Printf("   DELETE /api/v1/monitors?id=N - Delete monitor")
	log.Printf("   GET    /api/v1/pro/analytics - Pro analytics")
	log.Printf("   GET    /health               - Health check")

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

const banner = `
 __    __      _       _     _
/ / /\ \ \__ _| |_ ___| |__ | |_ _____      _____ _ __
\ \/  \/ / _' | __/ __| '_ \| __/ _ \ \ /\ / / _ \ '__|
 \  /\  / (_| | || (__| | | | || (_) \ V  V /  __/ |
  \/  \/ \__,_|\__\___|_| |_|\__\___/ \_/\_/ \___|_|  v1.0.0

Uptime Monitoring & Incident Intelligence
`

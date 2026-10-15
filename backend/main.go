package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"medplus-pos/db"
	"medplus-pos/handlers"
	"medplus-pos/middleware"

	_ "github.com/lib/pq"
)

func main() {
	loadEnvFileIfPresent()

	appEnv := getEnv("APP_ENV", "development")
	port := getEnv("PORT", "8080")

	if err := middleware.Configure(getEnv("JWT_SECRET", ""), getEnv("ALLOWED_ORIGINS", ""), appEnv); err != nil {
		log.Fatalf("Invalid middleware configuration: %v", err)
	}

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "medplus_pos")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")
	connStr := getEnv("DATABASE_URL", "")
	if connStr == "" {
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPass, dbHost, dbPort, dbName, dbSSLMode)
	}

	bootstrapDB := getBoolEnv("BOOTSTRAP_DB", strings.ToLower(appEnv) != "production")
	if bootstrapDB {
		err := ensureDatabaseExists(dbHost, dbPort, dbUser, dbPass, dbName, dbSSLMode)
		if err != nil {
			log.Printf("Warning: Database bootstrapping failed (will try direct connection): %v", err)
		}
	}

	log.Printf("Connecting to PostgreSQL for environment '%s'...", appEnv)
	database, err := db.InitDB(connStr)
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}
	defer database.Close()

	frontendDir, frontendErr := locateFrontendDir()
	if frontendErr != nil {
		log.Printf("Warning: frontend directory not found: %v", frontendErr)
	}

	mux := http.NewServeMux()
	if frontendErr == nil {
		assetsDir := filepath.Join(frontendDir, "assets")
		mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))
		
		docsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/docs" && r.URL.Path != "/docs/" && r.URL.Path != "/docs.html" {
				http.NotFound(w, r)
				return
			}
			http.ServeFile(w, r, filepath.Join(frontendDir, "docs.html"))
		})
		mux.Handle("/docs", docsHandler)
		mux.Handle("/docs/", docsHandler)
		mux.Handle("/docs.html", docsHandler)

		mux.Handle("/favicon.svg", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(frontendDir, "favicon.svg"))
		}))
	}
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if frontendErr != nil {
			http.Error(w, "frontend files not found", http.StatusInternalServerError)
			return
		}
		http.ServeFile(w, r, filepath.Join(frontendDir, "index.html"))
	}))

	mux.Handle("/api/health", middleware.CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respondSimpleJSON(w, http.StatusOK, `{"status":"ok","service":"medplus-pos"}`)
	})))
	mux.Handle("/api/ready", middleware.CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := db.DB.PingContext(ctx); err != nil {
			respondSimpleJSON(w, http.StatusServiceUnavailable, `{"status":"not_ready"}`)
			return
		}
		respondSimpleJSON(w, http.StatusOK, `{"status":"ready"}`)
	})))

	authRateLimit := middleware.RateLimit("auth", 10, time.Minute)
	mux.Handle("/api/auth/login", middleware.CORS(authRateLimit(http.HandlerFunc(handlers.LoginHandler))))
	mux.Handle("/api/auth/register", middleware.CORS(authRateLimit(http.HandlerFunc(handlers.RegisterHandler))))
	mux.Handle("/api/auth/me", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.MeHandler))))
	mux.Handle("/api/auth/logout", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.LogoutHandler))))
	mux.Handle("/api/profile", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.SaveProfileHandler))))

	mux.Handle("/api/products", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.GetProductsHandler))))
	mux.Handle("/api/products/", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.ProductItemHandler))))
	mux.Handle("/api/admin/products", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.ProductCollectionHandler))))

	mux.Handle("/api/inventory", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.GetInventoryHandler))))
	mux.Handle("/api/sales", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.ListSalesHandler(w, r)
		case http.MethodPost:
			handlers.ProcessSaleHandler(w, r)
		default:
			respondSimpleJSON(w, http.StatusMethodNotAllowed, `{"error":"Method not allowed"}`)
		}
	}))))
	mux.Handle("/api/receipts/", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.GetReceiptHandler))))

	mux.Handle("/api/users", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.UsersCollectionHandler))))
	mux.Handle("/api/users/", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.UserItemHandler))))

	mux.Handle("/api/suppliers", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.SuppliersCollectionHandler))))
	mux.Handle("/api/suppliers/", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.SupplierItemHandler))))
	mux.Handle("/api/purchase-orders", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.PurchaseOrdersCollectionHandler))))
	mux.Handle("/api/purchase-orders/", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.PurchaseOrderItemHandler))))

	mux.Handle("/api/customers", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.CustomersCollectionHandler))))
	mux.Handle("/api/customers/", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.CustomerItemHandler))))
	mux.Handle("/api/prescriptions", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.PrescriptionsCollectionHandler))))
	mux.Handle("/api/prescriptions/", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.PrescriptionItemHandler))))

	mux.Handle("/api/dashboard/stats", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.GetDashboardStatsHandler))))
	mux.Handle("/api/reports", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.ReportsHandler))))
	mux.Handle("/api/analytics", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.AnalyticsHandler))))
	mux.Handle("/api/database-overview", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.DatabaseOverviewHandler))))
	mux.Handle("/api/database-overview/access", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.VerifyDatabaseAccessHandler))))
	mux.Handle("/api/rbac/roles", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.RBACCollectionHandler))))
	mux.Handle("/api/rbac/roles/", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.RBACRoleHandler))))
	mux.Handle("/api/payments/config", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.PaymentsConfigHandler))))
	mux.Handle("/api/payments/mpesa/stk-push", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.MPesaSTKPushHandler))))
	mux.Handle("/api/payments/mpesa/status/", middleware.CORS(middleware.RequireAuth(http.HandlerFunc(handlers.MPesaPaymentStatusHandler))))
	mux.Handle("/api/payments/mpesa/callback", http.HandlerFunc(handlers.MPesaCallbackHandler))

	handler := middleware.Recoverer(middleware.RequestLogger(middleware.SecurityHeaders(mux)))
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("MedPlus POS Server is starting on port %s...", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	<-shutdownSignals
	log.Println("Shutdown signal received. Stopping server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}
}

func ensureDatabaseExists(host, port, user, password, dbName, sslMode string) error {
	adminConnStr := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=%s", user, password, host, port, sslMode)
	conn, err := sql.Open("postgres", adminConnStr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err = conn.Ping(); err != nil {
		return err
	}

	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	err = conn.QueryRow(query, dbName).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		log.Printf("Database '%s' does not exist. Creating it now...", dbName)
		_, err = conn.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		log.Printf("Database '%s' created successfully.", dbName)
	}

	return nil
}

func respondSimpleJSON(w http.ResponseWriter, code int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(body))
}

func locateFrontendDir() (string, error) {
	candidates := []string{"web", filepath.Join("..", "web")}
	for _, candidate := range candidates {
		indexPath := filepath.Join(candidate, "index.html")
		assetsPath := filepath.Join(candidate, "assets")
		if _, err := os.Stat(indexPath); err == nil {
			if info, assetsErr := os.Stat(assetsPath); assetsErr == nil && info.IsDir() {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("frontend directory not found")
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

func getBoolEnv(key string, defaultVal bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultVal
	}
}

func loadEnvFileIfPresent() {
	candidates := []string{".env", filepath.Join("backend", ".env")}
	for _, candidate := range candidates {
		if err := loadEnvFile(candidate); err == nil {
			log.Printf("Loaded environment from %s", candidate)
			return
		}
	}
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

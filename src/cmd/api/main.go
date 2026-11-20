package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
	_ "time/tzdata" // embed the timezone database so LoadLocation works in minimal containers

	"github.com/DanielDubon/Gimpy/internal/adapters/anthropic"
	"github.com/DanielDubon/Gimpy/internal/adapters/persistence"
	deliveryhttp "github.com/DanielDubon/Gimpy/internal/delivery/http"
	"github.com/DanielDubon/Gimpy/internal/usecases"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

// internalErrorMessage is the user-facing message returned when routine generation fails internally
const internalErrorMessage = "Lo siento, ocurrió un error técnico interno al procesar tu solicitud. Por favor, intenta de nuevo."

// defaultTimezone is used when APP_TIMEZONE is not set
const defaultTimezone = "America/Guatemala"

// Default per-user rate limit for POST /routine, used when RATE_LIMIT_RPS / RATE_LIMIT_BURST are not set
const (
	defaultRateLimitRPS   = 0.05 // 3 requests per minute
	defaultRateLimitBurst = 3
)

// envFloat reads a float64 from the environment, falling back to def when unset or invalid
func envFloat(key string, def float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 {
		slog.Warn("Invalid environment value, using default", "key", key, "value", raw, "default", def)
		return def
	}
	return value
}

// envInt reads an int from the environment, falling back to def when unset or invalid
func envInt(key string, def int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		slog.Warn("Invalid environment value, using default", "key", key, "value", raw, "default", def)
		return def
	}
	return value
}

type HealthResponse struct {
	Status string `json:"status"`
}

type RoutineRequest struct {
	Prompt string `json:"prompt"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := HealthResponse{
		Status: "ok",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Error encoding health response", "error", err)
	}
}

func routineHandler(generator *usecases.WorkoutGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		userID, ok := r.Context().Value(deliveryhttp.UserIDKey).(string)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "User ID not found in context",
			})
			return
		}

		var req RoutineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "Invalid request body: " + err.Error(),
			})
			return
		}

		// Validate prompt
		if req.Prompt == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "Prompt is required",
			})
			return
		}

		// SSE headers for streaming
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Get flusher for immediate streaming
		flusher, ok := w.(http.Flusher)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "Streaming not supported",
			})
			return
		}

		// Define callback for text chunks
		onChunk := func(text string) {
			chunkJSON, _ := json.Marshal(map[string]string{"type": "text", "content": text})
			fmt.Fprintf(w, "data: %s\n\n", chunkJSON)
			flusher.Flush()
		}

		// Generate workout with streaming
		workout, err := generator.Generate(r.Context(), userID, req.Prompt, onChunk)
		if err != nil {
			// Never expose raw internal errors (SQL, upstream APIs) to the client
			slog.Error("Database/Internal error", "operation", "GenerateRoutine", "user_id", userID, "error", err)
			errorJSON, _ := json.Marshal(map[string]string{"type": "error", "message": internalErrorMessage})
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", errorJSON)
			flusher.Flush()
			return
		}

		// Send final workout as complete event
		finalJSON, _ := json.Marshal(workout)
		fmt.Fprintf(w, "event: complete\ndata: %s\n\n", finalJSON)
		flusher.Flush()
	}
}

// openDatabase opens a connection pool compatible with Supabase's transaction pooler (PgBouncer/Supavisor).
// The pooler may route consecutive queries to different backend connections, so named prepared statements
// cached by pgx disappear between calls (SQLSTATE 26000). QueryExecModeExec uses a single-round-trip
// unnamed statement per query and never caches statements.
func openDatabase(dbURL string) (*sql.DB, error) {
	connConfig, err := pgx.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database url: %w", err)
	}

	connConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	connConfig.StatementCacheCapacity = 0
	connConfig.DescriptionCacheCapacity = 0

	return stdlib.OpenDB(*connConfig), nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// logs configg

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Info("Warning: unable to find .env using os variables")
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		slog.Error("API KEY environment variable is required")
		os.Exit(1)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Error("JWT_SECRET is not set")
		os.Exit(1)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DB url environment variable is required")
		os.Exit(1)
	}

	// db connection
	db, err := openDatabase(dbURL)
	if err != nil {
		slog.Error("Failed to open database connection", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("Database connection established successfully")

	userRepo := persistence.NewPostgresUserRepository(db)
	workoutRepo := persistence.NewPostgresWorkoutRepository(db)
	chatRepo := persistence.NewPostgresChatRepository(db)
	// Users' timezone decides what "today" means for the coach (defaults to Guatemala, UTC-6)
	timezone := os.Getenv("APP_TIMEZONE")
	if timezone == "" {
		timezone = defaultTimezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Error("Invalid APP_TIMEZONE", "timezone", timezone, "error", err)
		os.Exit(1)
	}
	slog.Info("Using application timezone", "timezone", timezone)

	llmService := anthropic.NewAnthropicClient(apiKey, workoutRepo, chatRepo, userRepo, location)

	authService := usecases.NewAuthService(userRepo, []byte(jwtSecret))
	workoutGenerator := usecases.NewWorkoutGenerator(llmService, userRepo, workoutRepo)

	authHandler := deliveryhttp.NewAuthHandler(authService)
	workoutHandler := deliveryhttp.NewWorkoutHandler(workoutRepo)
	authMiddleware := deliveryhttp.AuthMiddleware([]byte(jwtSecret))
	// Defaults: 3 requests per minute, burst of 3. Override with RATE_LIMIT_RPS / RATE_LIMIT_BURST (e.g. for k6 stress tests)
	rateLimitRPS := envFloat("RATE_LIMIT_RPS", defaultRateLimitRPS)
	rateLimitBurst := envInt("RATE_LIMIT_BURST", defaultRateLimitBurst)
	slog.Info("Using rate limit", "rps", rateLimitRPS, "burst", rateLimitBurst)
	rateLimitMiddleware := deliveryhttp.RateLimitMiddleware(rateLimitRPS, rateLimitBurst)

	mux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)

	// Protected endpoints
	mux.HandleFunc("POST /routine", authMiddleware(rateLimitMiddleware(routineHandler(workoutGenerator))))
	mux.HandleFunc("GET /routine/history", authMiddleware(workoutHandler.GetHistory))

	addr := ":8080"
	slog.Info("Starting Gimpy API server", "address", addr)
	slog.Info("Endpoints available:")
	slog.Info("  GET  /health              - Health check")
	slog.Info("  POST /auth/register       - User registration")
	slog.Info("  POST /auth/login          - User login")
	slog.Info("  POST /routine             - Generate workout routine (protected)")
	slog.Info("  GET  /routine/history     - Get workout history (protected)")

	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}

}

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync" // Added for local fallback cache
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/event"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/tracing"
)

var (
	jwtKey []byte
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type Handler struct {
	db         *sql.DB
	redis      *redis.Client
	producer   *event.Producer
	localCache sync.Map
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Missing required env var: %s", key)
	}
	return val
}

func main() {
	logger.Init("auth-service")

	tp, err := tracing.InitTracer("auth-service")
	if err != nil {
		logger.Log.Fatal("Failed to init tracer", zap.Error(err))
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	// JWT
	jwtKey = []byte(mustEnv("JWT_SECRET"))

	// PostgreSQL
	dsn := "postgres://" +
		mustEnv("DB_USER") + ":" +
		mustEnv("DB_PASSWORD") + "@" +
		mustEnv("DB_HOST") + ":" +
		mustEnv("DB_PORT") + "/" +
		mustEnv("DB_NAME") +
		"?sslmode=disable"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Log.Fatal("Failed to connect to Postgres", zap.Error(err))
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	// Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     mustEnv("REDIS_HOST") + ":" + mustEnv("REDIS_PORT"),
		Password: mustEnv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Kafka (OPTIONAL)
	var producer *event.Producer
	brokers := strings.Split(os.Getenv("KAFKA_BROKERS"), ",")
	if len(brokers) > 0 && brokers[0] != "" {
		p, err := event.NewProducer(brokers, "auth-service")
		if err != nil {
			logger.Log.Warn("Kafka unavailable, continuing without events", zap.Error(err))
		} else {
			producer = p
			defer producer.Close()
		}
	}

	h := &Handler{
		db:       db,
		redis:    redisClient,
		producer: producer,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", h.readyHandler)
	mux.HandleFunc("/signup", h.Signup)
	mux.HandleFunc("/login", h.Login)
	mux.HandleFunc("/validate", h.Validate)

	logger.Log.Info("Auth service running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *Handler) readyHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		http.Error(w, "DB not ready", http.StatusServiceUnavailable)
		return
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		http.Error(w, "Redis not ready", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(
		`INSERT INTO users (username, password) VALUES ($1, crypt($2, gen_salt('bf')))`,
		creds.Username,
		creds.Password,
	)
	if err != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	h.emitEvent(r.Context(), "user.signup", creds.Username)

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	var exists bool
	err := h.db.QueryRow(
		`SELECT true FROM users WHERE username=$1 AND password = crypt($2, password)`,
		creds.Username,
		creds.Password,
	).Scan(&exists)

	if err != nil || !exists {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Make token short-lived for high security (15 minutes instead of 1 hour)
	exp := time.Now().Add(15 * time.Minute)
	claims := &Claims{
		Username: creds.Username,
		Role:     "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(jwtKey)

	h.emitEvent(r.Context(), "user.login", creds.Username)

	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenStr,
	})
}

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !tkn.Valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Redis blacklist check with explicit timeout (Fail-Closed)
	redisCtx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()

	blacklisted, err := h.redis.Get(redisCtx, "bl:"+claims.ID).Result()
	if err == redis.Nil {
		// Token is valid
		h.localCache.Store(claims.ID, false)
	} else if err != nil {
		// Redis Failure! Attempt In-Memory Fallback
		logger.Log.Warn("Redis unreachable; checking local memory cache", zap.Error(err))
		if val, ok := h.localCache.Load(claims.ID); ok {
			if val.(bool) {
				http.Error(w, "Token revoked", http.StatusUnauthorized)
				return
			}
		} else {
			// Fail-Closed: If Redis fails and it's not in cache, reject traffic.
			http.Error(w, "Auth backend unavailable", http.StatusServiceUnavailable)
			return
		}
	} else if blacklisted == "1" {
		// Token is revoked
		h.localCache.Store(claims.ID, true)
		http.Error(w, "Token revoked", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) emitEvent(ctx context.Context, topic, key string) {
	if h.producer == nil {
		return
	}
	_ = h.producer.Emit(ctx, topic, key, map[string]interface{}{
		"id": uuid.New().String(),
		"ts": time.Now(),
	})
}

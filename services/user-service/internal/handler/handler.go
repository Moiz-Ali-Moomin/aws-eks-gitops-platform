package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib/pkg/logger"
	"go.uber.org/zap"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Handler struct{}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) HandleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var u User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Mock Save
	u.ID = "u_generated_123"
	u.CreatedAt = time.Now()

	logger.Log.Info("Created new user", zap.String("email", u.Email))

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

func (h *Handler) HandleUserByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/users/")

	// Mock Fetch
	u := User{
		ID:        id,
		Email:     "demo@example.com",
		Name:      "Demo User",
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}

	json.NewEncoder(w).Encode(u)
}

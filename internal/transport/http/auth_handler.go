package http

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/auth"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
)

type AuthHandler struct {
	auth *auth.Service
}

func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{auth: authService}
}

type registerRequest struct {
	Email    string    `json:"email"`
	Password string    `json:"password"`
	Role     user.Role `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authUserResponse struct {
	ID    string    `json:"id"`
	Email string    `json:"email"`
	Role  user.Role `json:"role"`
}

type loginResponse struct {
	Token string           `json:"token"`
	User  authUserResponse `json:"user"`
}

func (h *AuthHandler) Register(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req registerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		nethttp.Error(w, "invalid request body", nethttp.StatusBadRequest)
		return
	}

	createdUser, err := h.auth.Register(r.Context(), auth.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	})
	if err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(nethttp.StatusCreated)

	json.NewEncoder(w).Encode(authUserResponse{
		ID:    createdUser.ID,
		Email: createdUser.Email,
		Role:  createdUser.Role,
	})
}

func (h *AuthHandler) Login(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req loginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		nethttp.Error(w, "invalid request body", nethttp.StatusBadRequest)
		return
	}

	result, err := h.auth.Login(r.Context(), auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(loginResponse{
		Token: result.Token,
		User: authUserResponse{
			ID:    result.User.ID,
			Email: result.User.Email,
			Role:  result.User.Role,
		},
	})
}

func (h *AuthHandler) Me(w nethttp.ResponseWriter, r *nethttp.Request) {
	currentUser, ok := GetCurrentUser(r)
	if !ok {
		nethttp.Error(w, "not authenticated", nethttp.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(authUserResponse{
		ID:    currentUser.ID,
		Email: currentUser.Email,
		Role:  currentUser.Role,
	})
}

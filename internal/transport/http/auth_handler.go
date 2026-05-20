package http

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/auth"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/application/logactivity"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/activitylog"
	"github.com/EstateLinkAI/estatelink-lead-engine/internal/domain/user"
	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	auth            *auth.Service
	activityService *logactivity.Service
}

func NewAuthHandler(authService *auth.Service, activityService *logactivity.Service) *AuthHandler {
	return &AuthHandler{
		auth:            authService,
		activityService: activityService,
	}
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

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type updateUserRoleRequest struct {
	Role user.Role `json:"role"`
}

type authUserResponse struct {
	ID    string    `json:"id"`
	Email string    `json:"email"`
	Role  user.Role `json:"role"`
}

type loginResponse struct {
	AccessToken  string           `json:"accessToken"`
	RefreshToken string           `json:"refreshToken"`
	User         authUserResponse `json:"user"`
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

	_ = json.NewEncoder(w).Encode(authUserResponse{
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

	ipAddress, userAgent := requestMetadata(r)
	logActivityBestEffort(r.Context(), h.activityService, activitylog.ActivityLog{
		ActorUserID: result.User.ID,
		Action:      "user.logged_in",
		EntityType:  "user",
		Metadata: map[string]interface{}{
			"email": result.User.Email,
			"role":  result.User.Role,
		},
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})

	writeLoginResponse(w, result)
}

func (h *AuthHandler) Refresh(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req refreshRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		nethttp.Error(w, "invalid request body", nethttp.StatusBadRequest)
		return
	}

	result, err := h.auth.Refresh(r.Context(), auth.RefreshInput{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusUnauthorized)
		return
	}

	writeLoginResponse(w, result)
}

func (h *AuthHandler) Me(w nethttp.ResponseWriter, r *nethttp.Request) {
	currentUser, ok := GetCurrentUser(r)
	if !ok {
		nethttp.Error(w, "not authenticated", nethttp.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(authUserResponse{
		ID:    currentUser.ID,
		Email: currentUser.Email,
		Role:  currentUser.Role,
	})
}

func (h *AuthHandler) UpdateUserRole(w nethttp.ResponseWriter, r *nethttp.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeJSON(w, nethttp.StatusBadRequest, map[string]string{
			"error": "user id is required",
		})
		return
	}

	var req updateUserRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, nethttp.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if err := h.auth.UpdateUserRole(r.Context(), auth.UpdateUserRoleInput{
		UserID: userID,
		Role:   req.Role,
	}); err != nil {
		writeJSON(w, nethttp.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, nethttp.StatusOK, map[string]string{
		"message": "user role updated",
	})
}

func writeLoginResponse(w nethttp.ResponseWriter, result *auth.LoginOutput) {
	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(loginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User: authUserResponse{
			ID:    result.User.ID,
			Email: result.User.Email,
			Role:  result.User.Role,
		},
	})
}
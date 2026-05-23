package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/ebenamoafo2/workout-tracking/internal/store"
	"github.com/ebenamoafo2/workout-tracking/internal/tokens"
	"github.com/ebenamoafo2/workout-tracking/internal/utils"
)

type TokenHandler struct {
	tokenStore store.TokenStore
	userStore  store.UserStore
	logger *log.Logger
}

type createTokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewTokenHandler(tokenStore store.TokenStore, userStore store.UserStore, logger *log.Logger) *TokenHandler {
	return &TokenHandler{
		tokenStore: tokenStore,
		userStore:  userStore,
		logger: logger,
	}
}

func (h *TokenHandler) HandleCreateToken(w http.ResponseWriter, r *http.Request) {
	var req createTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Printf("ERROR: createTokenRequest: %v", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"error" : "invalid request payload"})
		return
	}

	// Validate user credentials
	user, err := h.userStore.GetUserByUsername(req.Username)
	if err != nil ||user == nil {
		h.logger.Printf("ERROR: GetUserByUsername: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error" : "internal server error"})
		return
	}

	// Check if the provided password matches the stored password hash
	passwordsDoMactch, err := user.PasswordHash.Matches(req.Password)
	if err != nil {
		h.logger.Printf("ERROR: PasswordMatches: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error" : "internal server error"})
		return
	}

	// If the passwords do not match, return an unauthorized error
	if !passwordsDoMactch {
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error" : "invalid credentials"})
		return
	}

	// Generate a new token for the authenticated user
	token, err := h.tokenStore.CreateNewToken(user.ID, 24 *time.Hour, tokens.ScopeAuth )
	if err != nil {
		h.logger.Printf("ERROR: CreateNewToken: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"error" : "internal server error"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"auth_token" : token})
}


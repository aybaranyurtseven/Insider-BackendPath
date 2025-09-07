package handler

import (
	"encoding/json"
	"insider-backend/internal/middleware"
	"insider-backend/internal/service"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type BalanceHandler struct {
	balanceService *service.BalanceService
}

func NewBalanceHandler(balanceService *service.BalanceService) *BalanceHandler {
	return &BalanceHandler{
		balanceService: balanceService,
	}
}

// GetCurrentBalance handles getting current balance for authenticated user
func (h *BalanceHandler) GetCurrentBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	balance, err := h.balanceService.GetBalance(r.Context(), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to get current balance")
		http.Error(w, "Failed to get balance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

// GetUserBalance handles getting balance for a specific user (admin only)
func (h *BalanceHandler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["user_id"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check permissions - only admin or the user themselves can view balance
	currentUserID, _ := middleware.GetUserIDFromContext(r.Context())
	currentUserRole, _ := middleware.GetUserRoleFromContext(r.Context())

	if currentUserRole != "admin" && currentUserID != userID {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	balance, err := h.balanceService.GetBalance(r.Context(), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("Failed to get user balance")
		http.Error(w, "Failed to get balance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

// GetBalanceHistory handles getting balance history
func (h *BalanceHandler) GetBalanceHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Check if requesting history for a different user
	if userIDParam := r.URL.Query().Get("user_id"); userIDParam != "" {
		if requestedUserID, err := uuid.Parse(userIDParam); err == nil {
			currentUserRole, _ := middleware.GetUserRoleFromContext(r.Context())
			if currentUserRole != "admin" && requestedUserID != userID {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}
			userID = requestedUserID
		}
	}

	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // default
	offset := 0 // default

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	history, err := h.balanceService.GetBalanceHistory(r.Context(), userID, limit, offset)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to get balance history")
		http.Error(w, "Failed to get balance history", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"history": history,
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
		"count":   len(history),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetBalanceAtTime handles getting balance at a specific time
func (h *BalanceHandler) GetBalanceAtTime(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	timestamp := r.URL.Query().Get("timestamp")
	if timestamp == "" {
		http.Error(w, "timestamp parameter is required", http.StatusBadRequest)
		return
	}

	// Check if requesting balance for a different user
	if userIDParam := r.URL.Query().Get("user_id"); userIDParam != "" {
		if requestedUserID, err := uuid.Parse(userIDParam); err == nil {
			currentUserRole, _ := middleware.GetUserRoleFromContext(r.Context())
			if currentUserRole != "admin" && requestedUserID != userID {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}
			userID = requestedUserID
		}
	}

	balance, err := h.balanceService.GetBalanceAtTime(r.Context(), userID, timestamp)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Str("timestamp", timestamp).Msg("Failed to get balance at time")
		http.Error(w, "Failed to get balance at time", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"user_id":   userID,
		"timestamp": timestamp,
		"balance":   balance,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetBalanceSnapshot handles getting a balance snapshot
func (h *BalanceHandler) GetBalanceSnapshot(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Check if requesting snapshot for a different user
	if userIDParam := r.URL.Query().Get("user_id"); userIDParam != "" {
		if requestedUserID, err := uuid.Parse(userIDParam); err == nil {
			currentUserRole, _ := middleware.GetUserRoleFromContext(r.Context())
			if currentUserRole != "admin" && requestedUserID != userID {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}
			userID = requestedUserID
		}
	}

	snapshot, err := h.balanceService.GetBalanceSnapshot(r.Context(), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to get balance snapshot")
		http.Error(w, "Failed to get balance snapshot", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

// RefreshBalance handles refreshing balance from database
func (h *BalanceHandler) RefreshBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// Check if refreshing balance for a different user
	if userIDParam := r.URL.Query().Get("user_id"); userIDParam != "" {
		if requestedUserID, err := uuid.Parse(userIDParam); err == nil {
			currentUserRole, _ := middleware.GetUserRoleFromContext(r.Context())
			if currentUserRole != "admin" && requestedUserID != userID {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}
			userID = requestedUserID
		}
	}

	balance, err := h.balanceService.RefreshBalance(r.Context(), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to refresh balance")
		http.Error(w, "Failed to refresh balance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

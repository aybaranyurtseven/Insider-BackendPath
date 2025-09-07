package handler

import (
	"encoding/json"
	"insider-backend/internal/domain"
	"insider-backend/internal/middleware"
	"insider-backend/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type TransactionHandler struct {
	transactionService *service.TransactionService
}

func NewTransactionHandler(transactionService *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
	}
}

// CreateCredit handles credit transaction creation
func (h *TransactionHandler) CreateCredit(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Type = string(domain.TransactionTypeCredit)

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()

	transaction, err := h.transactionService.CreateCredit(r.Context(), req, &userID, ipAddress, userAgent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create credit transaction")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}

// CreateDebit handles debit transaction creation
func (h *TransactionHandler) CreateDebit(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Type = string(domain.TransactionTypeDebit)

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()

	transaction, err := h.transactionService.CreateDebit(r.Context(), req, &userID, ipAddress, userAgent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create debit transaction")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}

// CreateTransfer handles transfer transaction creation
func (h *TransactionHandler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Type = string(domain.TransactionTypeTransfer)

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()

	transaction, err := h.transactionService.CreateTransfer(r.Context(), req, &userID, ipAddress, userAgent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create transfer transaction")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}

// GetTransaction handles getting transaction by ID
func (h *TransactionHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transactionIDStr := vars["id"]

	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	transaction, err := h.transactionService.GetTransaction(r.Context(), transactionID)
	if err != nil {
		log.Error().Err(err).Str("transaction_id", transactionIDStr).Msg("Failed to get transaction")
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	// Check if user has permission to view this transaction
	userID, _ := middleware.GetUserIDFromContext(r.Context())
	userRole, _ := middleware.GetUserRoleFromContext(r.Context())

	if userRole != "admin" {
		// Check if user is involved in the transaction
		isInvolved := false
		if transaction.FromUserID != nil && *transaction.FromUserID == userID {
			isInvolved = true
		}
		if transaction.ToUserID != nil && *transaction.ToUserID == userID {
			isInvolved = true
		}

		if !isInvolved {
			http.Error(w, "Insufficient permissions", http.StatusForbidden)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transaction)
}

// GetTransactionHistory handles getting transaction history with filters
func (h *TransactionHandler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	filter := domain.TransactionFilter{}

	// Parse query parameters
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}

	if txType := r.URL.Query().Get("type"); txType != "" {
		if domain.IsValidTransactionType(txType) {
			t := domain.TransactionType(txType)
			filter.Type = &t
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		if domain.IsValidTransactionStatus(status) {
			s := domain.TransactionStatus(status)
			filter.Status = &s
		}
	}

	if fromDateStr := r.URL.Query().Get("from_date"); fromDateStr != "" {
		if fromDate, err := time.Parse(time.RFC3339, fromDateStr); err == nil {
			filter.FromDate = &fromDate
		}
	}

	if toDateStr := r.URL.Query().Get("to_date"); toDateStr != "" {
		if toDate, err := time.Parse(time.RFC3339, toDateStr); err == nil {
			filter.ToDate = &toDate
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	} else {
		filter.Limit = 20 // default
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Check permissions
	currentUserID, _ := middleware.GetUserIDFromContext(r.Context())
	currentUserRole, _ := middleware.GetUserRoleFromContext(r.Context())

	// If not admin and no user_id specified, show current user's transactions
	if currentUserRole != "admin" && filter.UserID == nil {
		filter.UserID = &currentUserID
	}

	// If not admin and user_id specified, check if it's the current user
	if currentUserRole != "admin" && filter.UserID != nil && *filter.UserID != currentUserID {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	transactions, err := h.transactionService.GetTransactionHistory(r.Context(), filter)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get transaction history")
		http.Error(w, "Failed to get transaction history", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"transactions": transactions,
		"filter":       filter,
		"count":        len(transactions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserTransactions handles getting transactions for a specific user
func (h *TransactionHandler) GetUserTransactions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["user_id"]

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check permissions
	currentUserID, _ := middleware.GetUserIDFromContext(r.Context())
	currentUserRole, _ := middleware.GetUserRoleFromContext(r.Context())

	if currentUserRole != "admin" && currentUserID != userID {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
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

	transactions, err := h.transactionService.GetUserTransactions(r.Context(), userID, limit, offset)
	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("Failed to get user transactions")
		http.Error(w, "Failed to get user transactions", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"transactions": transactions,
		"user_id":      userID,
		"limit":        limit,
		"offset":       offset,
		"count":        len(transactions),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CancelTransaction handles transaction cancellation
func (h *TransactionHandler) CancelTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	transactionIDStr := vars["id"]

	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()

	if err := h.transactionService.CancelTransaction(r.Context(), transactionID, &userID, ipAddress, userAgent); err != nil {
		log.Error().Err(err).Str("transaction_id", transactionIDStr).Msg("Failed to cancel transaction")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"checkout-api/models"
)

// ItemStore defines the data operations the handler needs.
type ItemStore interface {
	GetItems(ctx context.Context) ([]*models.Item, error)
	GetItem(ctx context.Context, id int) (*models.Item, error)

	CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error)
	GetOrderById(ctx context.Context, orderID int) (*models.Order, error)

	CreateUserCart(ctx context.Context, cart *models.Cart) error
	GetUserCart(ctx context.Context, userID int) (*models.Cart, error)
	DeleteUserCart(ctx context.Context, userID int) error

	UpdateCartItem(ctx context.Context, userID int, itemID int, quantity int) (bool, error)
	RemoveCartItem(ctx context.Context, userID int, itemID int) (bool, error)

	SignUpUser(ctx context.Context, user *models.User) (*models.User, error)

	SaveIdempotencyKey(ctx context.Context, key string, orderID int) error
	GetOrderIDByIdempotencyKey(ctx context.Context, key string) (int, error)

	// TODO: refactor all methods to receive an extra context.Context as the first argument and return a second value of error type
}

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	store            ItemStore
	idempotencyCache map[string]*IdempotencyRecord
}

// NewHandler creates a Handler with the given store.
func NewHandler(s ItemStore) *Handler {
	return &Handler{
		store:            s,
		idempotencyCache: make(map[string]*IdempotencyRecord),
	}
}

// CreateOrderRequest is the payload for POST /orders.
type CreateOrderRequest struct {
	UserID int `json:"user_id"`
	Items  []struct {
		ItemID   int `json:"item_id"`
		Quantity int `json:"quantity"`
	} `json:"items"`
}

type CreateUserCartRequest struct {
	UserID int `json:"user_id"`
	Items  []struct {
		ItemID   int `json:"item_id"`
		Quantity int `json:"quantity"`
	} `json:"items"`
}

type AddItemToCartRequest struct {
	Items []struct {
		ItemID   int `json:"item_id"`
		Quantity int `json:"quantity"`
	} `json:"items"`
}

type UpdateCartItemRequest struct {
	UserID   int `json:"user_id"`
	Quantity int `json:"quantity"`
}

type RemoveCartItemRequest struct {
	UserID int `json:"user_id"`
}

type GetUserCartRequest struct {
	UserID int `json:"user_id"`
}

type CreateOrderFromCartRequest struct {
	UserID int `json:"user_id"`
}

type IdempotencyRecord struct {
	Response   []byte
	StatusCode int
	Expiry     time.Time
}

// PaymentResult represents a response from the payment provider.
type PaymentResult struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transaction_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

// mockProcessPayment simulates a payment provider call.
func mockProcessPayment(amount int) PaymentResult {
	if amount > 0 && amount < 1000000 {
		return PaymentResult{
			Success:       true,
			TransactionID: fmt.Sprintf("txn_%d", time.Now().UnixNano()),
		}
	}
	return PaymentResult{
		Success: false,
		Error:   "Payment declined",
	}
}

func (h *Handler) CreateUserCartAndAddItems(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateUserCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	cart, _ := h.store.GetUserCart(r.Context(), req.UserID)
	if cart != nil {
		http.Error(w, "cart already exists", http.StatusConflict)
		return
	}

	if len(req.Items) == 0 {
		http.Error(w, "can't create empty cart", http.StatusBadRequest)
		return
	}

	orderItems := make([]models.LineItem, 0, len(req.Items))
	for _, item := range req.Items {
		storeItem, err := h.store.GetItem(r.Context(), item.ItemID)
		if storeItem == nil || err != nil {
			http.Error(w, fmt.Sprintf("Item %d not found", item.ItemID), http.StatusBadRequest)
			return
		}
		orderItems = append(orderItems, models.LineItem{
			ItemID:   item.ItemID,
			Quantity: item.Quantity,
			Price:    storeItem.Price,
		})
	}

	userCart := &models.Cart{
		ID:     fmt.Sprintf("cart_%d", time.Now().UnixNano()),
		UserID: req.UserID,
		Items:  orderItems,
	}

	h.store.CreateUserCart(r.Context(), userCart)
	writeJSON(w, http.StatusCreated, userCart)

}

func (h *Handler) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.Atoi(pathParts[4])
	if err != nil {
		http.Error(w, "invalid item ID", http.StatusBadRequest)
		return
	}

	var req UpdateCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		http.Error(w, "quantity must be greater than 0", http.StatusBadRequest)
		return
	}

	isDone, err := h.store.UpdateCartItem(r.Context(), req.UserID, itemID, req.Quantity)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err.Error()), http.StatusBadRequest)
		return
	}
	if !isDone {
		writeJSON(w, http.StatusOK, map[string]string{"message": "item not found in cart or cart does not exist for this user"})
		return
	}

	cart, err := h.store.GetUserCart(r.Context(), req.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err.Error()), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, cart)
}

func (h *Handler) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	itemID, err := strconv.Atoi(pathParts[4])
	if err != nil {
		http.Error(w, "invalid item ID", http.StatusBadRequest)
		return
	}

	var req RemoveCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	isDone, err := h.store.RemoveCartItem(r.Context(), req.UserID, itemID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err.Error()), http.StatusBadRequest)
		return
	}
	if !isDone {
		writeJSON(w, http.StatusOK, map[string]string{"message": "item not found in cart or cart does not exist for this user"})
		return
	}

	cart, err := h.store.GetUserCart(r.Context(), req.UserID)
	if cart == nil {
		writeJSON(w, http.StatusOK, map[string]string{"message": "cart is now empty"})
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err.Error()), http.StatusBadRequest)
		return
	}
	if cart != nil && len(cart.Items) == 0 {
		err = h.store.DeleteUserCart(r.Context(), req.UserID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error: %v", err.Error()), http.StatusBadRequest)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetUserCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		http.Error(w, "missing X-User-ID header", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "invalid X-User-ID header", http.StatusBadRequest)
		return
	}

	cart, _ := h.store.GetUserCart(r.Context(), userID)
	//if err != nil { // catch errors for debug
	//	http.Error(w, err.Error(), http.StatusBadRequest)
	//	return
	//}
	if cart == nil || len(cart.Items) == 0 {
		emptyCart := &models.Cart{
			ID:     "",
			UserID: userID,
			Items:  []models.LineItem{},
		}
		writeJSON(w, http.StatusOK, emptyCart)
		return
	}

	writeJSON(w, http.StatusOK, cart)
}

func (h *Handler) CreateOrderFromCart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
		return
	}

	if record, exists := h.idempotencyCache[idempotencyKey]; exists {
		if time.Now().Before(record.Expiry) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(record.StatusCode)
			w.Write(record.Response)
			return
		}
		delete(h.idempotencyCache, idempotencyKey)
	}

	var req CreateOrderFromCartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	cart, err := h.store.GetUserCart(r.Context(), req.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: $1", err.Error()), http.StatusBadRequest)
		return
	}
	if cart == nil {
		writeJSON(w, http.StatusOK, map[string]string{"message": "no cart exists for this user"})
		return
	}

	if len(cart.Items) == 0 {
		http.Error(w, "cart is empty", http.StatusBadRequest)
		return
	}

	total := 0
	for _, item := range cart.Items {
		total += item.Price * item.Quantity
	}

	paymentResult := mockProcessPayment(total)

	status := "paid"
	if !paymentResult.Success {
		status = "failed"
	}

	order, err := h.store.CreateOrder(r.Context(), req.UserID, cart.Items, total, status)
	if err != nil {
		http.Error(w, "Error occured while CreateOrder", http.StatusBadRequest)
		return
	}

	if paymentResult.Success {
		err = h.store.DeleteUserCart(r.Context(), req.UserID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error: ", err.Error()), http.StatusBadRequest)
		}
	}

	responseData := map[string]any{
		"order":   order,
		"payment": paymentResult,
	}

	responseBody, _ := json.Marshal(responseData)

	statusCode := http.StatusCreated
	if !paymentResult.Success {
		statusCode = http.StatusPaymentRequired
	}

	h.idempotencyCache[idempotencyKey] = &IdempotencyRecord{
		Response:   responseBody,
		StatusCode: statusCode,
		Expiry:     time.Now().Add(24 * time.Hour),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(responseBody)
}

// GetItems handles GET /items — returns all available items.
func (h *Handler) GetItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	items, err := h.store.GetItems(r.Context())
	if err != nil {
		// return 500 to the client
		writeJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	// return  200 to client
	writeJSON(w, http.StatusOK, items)
}

// GetItemByID handles GET /items/{id} — returns a single item.
func (h *Handler) GetItemByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/items/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	item, err := h.store.GetItem(r.Context(), id)
	if item == nil || err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// CreateOrder handles POST /orders — creates an order with mock payment.
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
		return
	}

	orderID, err := h.store.GetOrderIDByIdempotencyKey(r.Context(), key)
	if err == nil {
		order, err := h.store.GetOrderById(r.Context(), orderID)
		if err != nil {
			http.Error(w, "failed to get cached order", http.StatusInternalServerError)
			return
		}

		fmt.Println("Cached response")
		writeJSON(w, http.StatusCreated, order)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate and calculate total
	total := 0
	orderItems := make([]models.LineItem, 0, len(req.Items))
	for _, item := range req.Items {
		storeItem, err := h.store.GetItem(r.Context(), item.ItemID)
		if storeItem == nil || err != nil {
			http.Error(w, fmt.Sprintf("Item %d not found", item.ItemID), http.StatusBadRequest)
			return
		}
		itemTotal := storeItem.Price * item.Quantity
		total += itemTotal
		orderItems = append(orderItems, models.LineItem{
			ItemID:   item.ItemID,
			Quantity: item.Quantity,
			Price:    storeItem.Price,
		})
	}

	// Process payment (mock)
	paymentResult := mockProcessPayment(total)

	status := "paid"
	if !paymentResult.Success {
		status = "failed"
	}

	order, err := h.store.CreateOrder(r.Context(), req.UserID, orderItems, total, status)

	if err != nil {
		http.Error(w, "Error occured while CreateOrder", http.StatusBadRequest)
		return
	}

	if paymentResult.Success {
		err = h.store.SaveIdempotencyKey(r.Context(), key, order.ID)
		if err != nil {
			http.Error(w, "failed to save idempotency key", http.StatusInternalServerError)
			return
		}
		fmt.Println("Non-Cached response")
		writeJSON(w, http.StatusCreated, map[string]any{
			"order":   order,
			"payment": paymentResult,
		})
	} else {
		fmt.Println("Non-Cached response")
		writeJSON(w, http.StatusPaymentRequired, map[string]any{
			"order":   order,
			"payment": paymentResult,
		})
	}
}

func (h *Handler) GetOrderById(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/orders/")
	orderID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := h.store.GetOrderById(r.Context(), orderID)
	if order == nil || err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, order)
}

type SignUpUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) SignUpUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SignUpUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "username, email, and password are required", http.StatusBadRequest)
		return
	}

	user := &models.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		CreatedAt: time.Now(),
	}

	createdUser, err := h.store.SignUpUser(r.Context(), user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, createdUser)
}

// writeJSON encodes v as JSON and writes it to the response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

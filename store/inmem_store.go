package store

import (
	"checkout-api/models"
	"context"
)

// InMemStore is an in-memory store for items and orders.
type InMemStore struct {
	items       map[int]*models.Item
	orders      map[int]*models.Order
	carts       map[int]*models.Cart
	nextOrderID int
}

// NewInMemStore creates a Store pre-loaded with seed data.
func NewInMemStore() *InMemStore {
	s := &InMemStore{
		items:       make(map[int]*models.Item),
		orders:      make(map[int]*models.Order),
		carts:       make(map[int]*models.Cart),
		nextOrderID: 1,
	}

	s.items[1] = &models.Item{ID: 1, Name: "Laptop", Description: "A fast laptop", Price: 120000, Stock: 10}
	s.items[2] = &models.Item{ID: 2, Name: "Mouse", Description: "Wireless mouse", Price: 2500, Stock: 50}
	s.items[3] = &models.Item{ID: 3, Name: "Keyboard", Description: "Mechanical keyboard", Price: 8000, Stock: 30}

	return s
}

// GetItems returns all available items.
func (s *InMemStore) GetItems(_ context.Context) ([]*models.Item, error) {
	items := make([]*models.Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

// GetItem returns a single item by ID, or nil if not found.
func (s *InMemStore) GetItem(id int) *models.Item {
	return s.items[id]
}

// CreateOrder creates a new order and returns it.
func (s *InMemStore) CreateOrder(userID int, items []models.LineItem, total int, status string) *models.Order {
	order := &models.Order{
		ID:     s.nextOrderID,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}
	s.orders[order.ID] = order
	s.nextOrderID++
	return order
}

func (s *InMemStore) CreateUserCart(cart *models.Cart) {
	s.carts[cart.UserID] = cart
}

func (s *InMemStore) GetUserCart(userID int) *models.Cart {
	var cart *models.Cart
	if c, ok := s.carts[userID]; ok {
		cart = c
		return cart
	}
	return cart
}

func (s *InMemStore) DeleteUserCart(userID int) {
	delete(s.carts, userID)
}

func (s *InMemStore) UpdateCartItem(userID int, itemID int, quantity int) bool {
	cart := s.carts[userID]
	if cart == nil {
		return false
	}

	for i := range cart.Items {
		if cart.Items[i].ItemID == itemID {
			cart.Items[i].Quantity = quantity
			return true
		}
	}
	return false
}

func (s *InMemStore) RemoveCartItem(userID int, itemID int) bool {
	cart := s.carts[userID]
	if cart == nil {
		return false
	}

	for i, item := range cart.Items {
		if item.ItemID == itemID {
			cart.Items = append(cart.Items[:i], cart.Items[i+1:]...)
			return true
		}
	}
	return false
}

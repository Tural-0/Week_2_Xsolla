package store

import (
	"checkout-api/models"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PostgresStore is an in-memory store for items and orders.
type PostgresStore struct {
	conn *pgx.Conn
}

// NewPostgresStore creates a Store pre-loaded with seed data.
func NewPostgresStore(conn *pgx.Conn) *PostgresStore {
	s := &PostgresStore{
		conn: conn,
	}
	return s
}

// GetItems returns all available items.
func (s *PostgresStore) GetItems(ctx context.Context) ([]*models.Item, error) {
	rows, err := s.conn.Query(ctx, "select * from items")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to run query on GetItems", err)
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		var item models.Item
		err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
		if err != nil {
			// Handle the scan error, potentially breaking the loop or logging and continuing
			fmt.Printf("unable to scan row: %w", err)
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}
		items = append(items, &item)
	}
	if rows.Err() != nil {
		return nil, err
	}

	return items, nil
}

// GetItem returns a single item by ID, or nil if not found.
func (s *PostgresStore) GetItem(id int) *models.Item {
	// TODO: query a single item with conn.QueryRow()
	return nil
}

func (s *PostgresStore) CreateOrder(userID int, items []models.LineItem, total int, status string) *models.Order {
	// TODO: create an order in a transaction
	// Use a context.Context passed as the first argument from your method
	// Use transaction with conn.Begin(), conn.Exec()
	return nil
}

func (s *PostgresStore) CreateUserCart(cart *models.Cart) {
	// TODO: implement
}

func (s *PostgresStore) GetUserCart(userID int) *models.Cart {
	// TODO: implement
	return nil
}

func (s *PostgresStore) DeleteUserCart(userID int) {
	// TODO: implement
}

func (s *PostgresStore) UpdateCartItem(userID int, itemID int, quantity int) bool {
	// TODO: implement
	return false
}

func (s *PostgresStore) RemoveCartItem(userID int, itemID int) bool {
	// TODO: implement
	return false
}

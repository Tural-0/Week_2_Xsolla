package store

import (
	"checkout-api/models"
	"context"
	"fmt"
	"time"

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
			fmt.Printf("unable to scan row: ")
			fmt.Print(err)
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
func (s *PostgresStore) GetItem(ctx context.Context, id int) (*models.Item, error) {
	// TODO: query a single item with conn.QueryRow()
	row := s.conn.QueryRow(ctx,
		"SELECT * FROM items WHERE id=$1", id)

	var item models.Item
	err := row.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("%w: No rows were returned from query on GetItemById", err)
		}
		fmt.Printf("unable to scan row: ")
		fmt.Print(err)
		return nil, fmt.Errorf("%w: failed to run query on GetItemById", err)
	}

	return &item, nil
}

func (s *PostgresStore) CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error) {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	createdAt := time.Now()

	var id int
	err = tx.QueryRow(
		ctx,
		`INSERT INTO orders (user_id, total, status, created_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		userID,
		total,
		status,
		createdAt,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	for _, item := range items {
		_, err = tx.Exec(
			ctx,
			`INSERT INTO order_items (order_id, item_id, price, quantity)
		 VALUES ($1, $2, $3, $4)`,
			id,
			item.ItemID,
			item.Price,
			item.Quantity,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to add items to order: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	order := &models.Order{
		ID:     id,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}

	return order, nil
}

func (s *PostgresStore) GetOrderById(ctx context.Context, orderID int) (*models.Order, error) {
	row := s.conn.QueryRow(ctx,
		"SELECT id, user_id, total, status FROM orders WHERE id=$1", orderID)

	var order models.Order
	err := row.Scan(&order.ID, &order.UserID, &order.Total, &order.Status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("%w: No rows were returned from query on GetOrderById", err)
		}
		fmt.Printf("unable to scan row: ")
		fmt.Print(err)
		return nil, fmt.Errorf("%w: failed to run query on GetOrderById", err)
	}

	// getting items
	rows, err := s.conn.Query(ctx, "SELECT item_id, price, quantity FROM order_items WHERE order_id = $1", order.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to run query on GetOrderById while getting order_items", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lItem models.LineItem
		err := rows.Scan(&lItem.ItemID, &lItem.Price, &lItem.Quantity)
		if err != nil {
			// Handle the scan error, potentially breaking the loop or logging and continuing
			fmt.Printf("unable to scan row: ")
			fmt.Print(err)
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}
		order.Items = append(order.Items, lItem)
	}
	if rows.Err() != nil {
		return nil, err
	}

	return &order, nil
}

func (s *PostgresStore) CreateUserCart(ctx context.Context, cart *models.Cart) error {
	// TODO: implement
	_, err := s.conn.Exec(ctx,
		"INSERT INTO carts (id, user_id) VALUES ($1, $2)", cart.ID, cart.UserID)
	if err != nil {
		return fmt.Errorf("%w: failed to run query on CreateUserCart while INSERT", err)
	}

	for _, item := range cart.Items {
		_, err := s.conn.Exec(ctx,
			"INSERT INTO cart_items (cart_id,item_id,price,quantity) VALUES ($1, $2, $3, $4)", cart.ID, item.ItemID, item.Price, item.Quantity)
		if err != nil {
			return fmt.Errorf("%w: failed to run query on CreateUserCart while INSERT of item", err)
		}
	}

	return nil
}

func (s *PostgresStore) GetUserCart(ctx context.Context, userID int) (*models.Cart, error) {
	// TODO: implement
	row := s.conn.QueryRow(ctx,
		"SELECT * FROM carts WHERE user_id=$1", userID)

	var cart models.Cart
	err := row.Scan(&cart.ID, &cart.UserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("%w: No rows were returned from query on GetUserCart", err)
		}
		fmt.Printf("unable to scan row: ")
		fmt.Print(err)
		return nil, fmt.Errorf("%w: failed to run query on GetUserCart", err)
	}

	// getting items
	rows, err := s.conn.Query(ctx, "SELECT item_id, price, quantity FROM cart_items WHERE cart_id = $1", cart.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to run query on GetUserCart while getting cart_items", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lItem models.LineItem
		err := rows.Scan(&lItem.ItemID, &lItem.Price, &lItem.Quantity)
		if err != nil {
			// Handle the scan error, potentially breaking the loop or logging and continuing
			fmt.Printf("unable to scan row: ")
			fmt.Print(err)
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}
		cart.Items = append(cart.Items, lItem)
	}
	if rows.Err() != nil {
		return nil, err
	}

	return &cart, nil
}

func (s *PostgresStore) DeleteUserCart(ctx context.Context, userID int) error {
	// TODO: implement
	var cartID string

	err := s.conn.QueryRow(
		ctx,
		"SELECT id FROM carts WHERE user_id=$1", userID,
	).Scan(&cartID)
	if err != nil {
		return fmt.Errorf("failed to find cart: %w", err)
	}

	_, err = s.conn.Exec(
		ctx,
		"DELETE FROM cart_items WHERE cart_id=$1",
		cartID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete cart items: %w", err)
	}

	_, err = s.conn.Exec(
		ctx,
		"DELETE FROM carts WHERE id=$1",
		cartID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}

	return nil

}

func (s *PostgresStore) UpdateCartItem(ctx context.Context, userID int, itemID int, quantity int) (bool, error) {
	// TODO: implement

	_, err := s.conn.Exec(ctx,
		"UPDATE cart_items SET quantity=$1 WHERE cart_id=(SELECT id FROM carts WHERE user_id=$2 LIMIT 1) AND item_id=$3", quantity, userID, itemID)

	if err != nil {
		return false, fmt.Errorf("%w: failed to run query on UpdateCartItem", err)
	}

	return true, nil
}

func (s *PostgresStore) RemoveCartItem(ctx context.Context, userID int, itemID int) (bool, error) {
	// TODO: implement
	_, err := s.conn.Exec(ctx,
		"DELETE FROM cart_items WHERE cart_id=(SELECT id FROM carts WHERE user_id=$1 LIMIT 1) AND item_id=$2", userID, itemID)
	if err != nil {
		return false, fmt.Errorf("%w: failed to run query on RemoveCartItem", err)
	}
	return true, nil
}

func (s *PostgresStore) SignUpUser(ctx context.Context, user *models.User) (*models.User, error) {
	_, err := s.conn.Exec(ctx,
		"INSERT INTO users (name, email, password, created_at) VALUES ($1, $2, $3, $4)", user.Username, user.Email, user.Password, time.Now())
	if err != nil {
		return nil, fmt.Errorf("%w: failed to run query on SignUpUser", err)
	}

	return user, nil
}

func (s *PostgresStore) SaveIdempotencyKey(ctx context.Context, key string, orderID int) error {
	_, err := s.conn.Exec(
		ctx,
		`INSERT INTO idempotency_keys (idempotency_key, order_id)
		 VALUES ($1, $2)`,
		key,
		orderID,
	)
	if err != nil {
		return fmt.Errorf("failed to save idempotency key: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetOrderIDByIdempotencyKey(ctx context.Context, key string) (int, error) {
	var orderID int

	err := s.conn.QueryRow(
		ctx,
		`SELECT order_id
		 FROM idempotency_keys
		 WHERE idempotency_key = $1`,
		key,
	).Scan(&orderID)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, pgx.ErrNoRows
		}
		return 0, fmt.Errorf("failed to get idempotency key: %w", err)
	}

	return orderID, nil
}

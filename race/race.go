package race

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

const connectionString = "postgresql://postgres:tural1234@localhost:5432/week2_xsolla"

func SimulateRaceCondition() {
	ctx := context.Background()

	// Two independent database connections.
	conn1, err := pgx.Connect(ctx, connectionString)
	if err != nil {
		panic(err)
	}
	defer conn1.Close(ctx)

	conn2, err := pgx.Connect(ctx, connectionString)
	if err != nil {
		panic(err)
	}
	defer conn2.Close(ctx)

	const mockID = 999999

	// Prepare a clean test item.
	_, _ = conn1.Exec(ctx, "DELETE FROM items WHERE id=$1", mockID)

	_, err = conn1.Exec(ctx,
		`INSERT INTO items
		(id, name, description, price, stock, created_at)
		VALUES ($1,$2,$3,$4,$5,NOW())`,
		mockID,
		"Race Condition Item",
		"Used only for SELECT FOR UPDATE demo",
		100,
		1,
	)

	if err != nil {
		panic(err)
	}

	fmt.Println("========================================")
	fmt.Println("Starting race condition simulation")
	fmt.Println("Initial stock = 1")
	fmt.Println("========================================")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		err := BuyItem(ctx, conn1, mockID, 1)

		if err != nil {
			fmt.Println("Goroutine 1:", err)
		} else {
			fmt.Println("Goroutine 1: Purchase successful")
		}
	}()

	go func() {
		defer wg.Done()

		err := BuyItem(ctx, conn2, mockID, 1)

		if err != nil {
			fmt.Println("Goroutine 2:", err)
		} else {
			fmt.Println("Goroutine 2: Purchase successful")
		}
	}()

	wg.Wait()

	var stock int

	err = conn1.QueryRow(ctx,
		"SELECT stock FROM items WHERE id=$1",
		mockID,
	).Scan(&stock)

	if err == nil {
		fmt.Println("----------------------------------------")
		fmt.Println("Final stock:", stock)
		fmt.Println("----------------------------------------")
	}
}

func BuyItem(ctx context.Context, conn *pgx.Conn, itemID int, quantity int) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var stock int

	///////////////////////////////////////////////////////////
	err = tx.QueryRow(
		ctx,
		`SELECT stock
		 FROM items
		 WHERE id=$1`,
		//FOR UPDATE`,
		itemID,
	).Scan(&stock)

	if err != nil {
		return fmt.Errorf("failed to get item stock: %w", err)
	}

	fmt.Printf("[%p] Read stock = %d\n", conn, stock)

	if stock < quantity {
		return fmt.Errorf("not enough stock")
	}

	// Make the race easier to observe.
	time.Sleep(2 * time.Second)

	_, err = tx.Exec(
		ctx,
		`UPDATE items
		 SET stock = stock - $1
		 WHERE id=$2`,
		quantity,
		itemID,
	)

	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

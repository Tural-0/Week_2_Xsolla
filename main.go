package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"checkout-api/handlers"
	"checkout-api/store"

	"github.com/jackc/pgx/v5"
)

func main() {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgresql://postgres:postgres@localhost:5432/postgres")
	if err != nil {
		panic(err)
	}

	err = conn.Ping(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("successfully connected to db")

	// s := store.NewInMemStore()
	postgresStore := store.NewPostgresStore(conn)
	h := handlers.NewHandler(postgresStore)

	http.HandleFunc("/user/cart/items/",
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPatch:
				h.UpdateCartItem(w, r)
			case http.MethodDelete:
				h.RemoveCartItem(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	http.HandleFunc("/user/cart",
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				h.GetUserCart(w, r)
			case http.MethodPost:
				h.CreateUserCartAndAddItems(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		})
	http.HandleFunc("/user/orders", h.CreateOrderFromCart)

	http.HandleFunc("/items", h.GetItems)
	http.HandleFunc("/items/", h.GetItemByID)

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

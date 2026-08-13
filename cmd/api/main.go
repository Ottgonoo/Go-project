package main

import (
	"context"
	"fmt"
	"log"

	"ledger-wallet/internal/database"
)

func main() {
	conn, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(context.Background())

	fmt.Println("Database connected successfully!")
}git 
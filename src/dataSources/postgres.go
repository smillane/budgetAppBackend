package dataSources

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateDatabasePool(ctx context.Context) {
	dbpool, err := pgxpool.New(ctx, "postgres://postgres:password@localhost:5433/budgetApp?sslmode=disable")
	if err != nil {
		fmt.Println("Unable to connect to db", err)
		os.Exit(1)
	}
	fmt.Println("Connection open")
	defer dbpool.Close()
}

package main

import (
	"context"
	"os"
	"os/signal"

	"goBackend/src/api"
	"goBackend/src/dataSources"
)

func init() {
	dataSources.CreatePlaidClient()
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	dataSources.CreateDatabasePool(ctx)
	api.Routes()
	os.Exit(0)
}

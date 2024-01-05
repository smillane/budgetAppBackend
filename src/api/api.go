package api

import (
	"flag"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"

	"goBackend/src/core/interactors"
	"goBackend/src/dataSources"
)

var routes = flag.Bool("routes", false, "Generate router documentation")

func Routes() {
	flag.Parse()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc: func(r *http.Request, origin string) bool { return true },
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	r.Route("/users", func(r chi.Router) {
		r.Get("/{userID}", interactors.GetUser)
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/set_access_token", dataSources.GetAccessToken)
		r.Get("/auth", dataSources.Auth)
		r.Get("/accounts", dataSources.Accounts)
		r.Get("/balance", dataSources.Balance)
		r.Get("/transactions", dataSources.Transactions)
		r.Post("/transactions", dataSources.Transactions)
		r.Get("/create_public_token", dataSources.CreatePublicToken)
		r.Post("/create_link_token", dataSources.CreateLinkToken)
		r.Get("/investments_transactions", dataSources.InvestmentTransactions)
		r.Get("/holdings", dataSources.Holdings)
		r.Get("/assets", dataSources.Assets)
	})

	http.ListenAndServe(":3000", r)
}

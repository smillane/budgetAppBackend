package interactors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"goBackend/src/core/entities"
)

func GetUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	fmt.Printf("ctx: %v\n", ctx)
	defer cancel()

	userID := chi.URLParam(r, "userID")
	fmt.Println(userID)
	user, err := entities.GetUser(userID)

	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(user)
}

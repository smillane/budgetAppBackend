package entities

import (
	"goBackend/src/core/repositories"
)

func GetUser(id string) (string, error) {
	return repositories.GetByUserID(id)
}

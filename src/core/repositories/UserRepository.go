package repositories

type UserRepository interface {
}

func GetByUserID(id string) (string, error) {
	// query := `SELECT id FROM accounts`
	return id, nil
}

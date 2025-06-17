package db

type User struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	Username     *string `json:"username"`
	FirstName    *string `json:"firstName"`
	LastName     *string `json:"lastName"`
	Avatar       *string `json:"avatar"`
	PasswordHash *[]byte `json:"-"`
	IsVerified   bool `json:"isVerified"`
}

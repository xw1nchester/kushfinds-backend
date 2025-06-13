package user

type User struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	Username     *string `json:"username"`
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	Avatar       *string `json:"avatar"`
	PasswordHash *[]byte `json:"-"`
	IsVerified   bool `json:"is_verified"`
}

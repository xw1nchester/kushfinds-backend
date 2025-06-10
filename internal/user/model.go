package user

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	PasswordHash []byte `json:"-"`
	Role     string `json:"role"`
}

package user

type User struct {
	ID            int     `json:"id"`
	Email         string  `json:"email"`
	Username      *string `json:"username"`
	FirstName     *string `json:"firstName"`
	LastName      *string `json:"lastName"`
	Avatar        *string `json:"avatar"`
	IsVerified    bool    `json:"isVerified"`
	PasswordHash  *[]byte `json:"-"`
	IsPasswordSet bool    `json:"isPasswordSet"`
}

type UserResponse struct {
	User User `json:"user"`
}

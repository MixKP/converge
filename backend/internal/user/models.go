package user

type User struct {
	ID   int    `json:"id"   example:"1"`
	Name string `json:"name" example:"alice"`
	Role string `json:"role" example:"teacher"`
}

type RegisterRequest struct {
	Name     string `json:"name"     binding:"required" example:"alice"`
	Password string `json:"password" binding:"required" example:"s3cret"`
	Role     string `json:"role"     binding:"required,oneof=admin teacher student parent" example:"teacher"`
}

type LoginRequest struct {
	Name     string `json:"name"     binding:"required" example:"alice"`
	Password string `json:"password" binding:"required" example:"s3cret"`
}

type AuthResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	User  User   `json:"user"`
}

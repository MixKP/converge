package user

type User struct {
	ID   int    `json:"id"   example:"1"`
	Name string `json:"name" example:"alice"`
}

type RegisterRequest struct {
	Name     string `json:"name"     binding:"required" example:"alice"`
	Password string `json:"password" binding:"required" example:"s3cret"`
}

type LoginRequest struct {
	Name     string `json:"name"     binding:"required" example:"alice"`
	Password string `json:"password" binding:"required" example:"s3cret"`
}

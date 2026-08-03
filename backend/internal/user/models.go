package user

type User struct {
	ID   int    `json:"id"   example:"1"`
	Name string `json:"name" example:"alice"`
	Role string `json:"role" example:"student"`
}

type RegisterRequest struct {
	Name     string `json:"name"     binding:"required" example:"alice"`
	Password string `json:"password" binding:"required" example:"s3cret"`
	Role     string `json:"role"     binding:"required,oneof=admin student parent" example:"student"`
	// StudentIDs links a parent to the students they guard. Required when role is parent, ignored otherwise.
	StudentIDs []int `json:"student_ids,omitempty" example:"3,4"`
}

type LoginRequest struct {
	Name     string `json:"name"     binding:"required" example:"alice"`
	Password string `json:"password" binding:"required" example:"s3cret"`
}

type AuthResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	User  User   `json:"user"`
}

type LinkStudentRequest struct {
	StudentID int `json:"student_id" binding:"required" example:"3"`
}

// ParentWithStudents is a parent user together with the students they guard.
type ParentWithStudents struct {
	User
	Students []User `json:"students"`
}

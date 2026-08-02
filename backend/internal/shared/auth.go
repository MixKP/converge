package shared

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
	RoleParent  Role = "parent"
	RoleGuest   Role = "guest"
)

func ParseRole(raw string) (Role, error) {
	switch Role(raw) {
	case RoleAdmin, RoleTeacher, RoleStudent, RoleParent:
		return Role(raw), nil
	default:
		return "", &ValidationError{Msg: "role must be one of admin, teacher, student, parent"}
	}
}

type Principal struct {
	UserID int
	Name   string
	Role   Role
}

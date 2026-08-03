package shared

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleStudent Role = "student"
	RoleParent  Role = "parent"
	RoleGuest   Role = "guest"
)

func ParseRole(raw string) (Role, error) {
	switch Role(raw) {
	case RoleAdmin, RoleStudent, RoleParent:
		return Role(raw), nil
	default:
		return "", &ValidationError{Msg: "role must be one of admin, student, parent"}
	}
}

type Principal struct {
	UserID int
	Name   string
	Role   Role
}

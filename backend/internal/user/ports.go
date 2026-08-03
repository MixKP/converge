package user

import (
	"context"

	"github.com/RinZ5/converge/backend/internal/shared"
)

type UserStore interface {
	CreateUser(ctx context.Context, name, passwordHash, role string) (*User, error)
	CreateParent(ctx context.Context, name, passwordHash string, studentIDs []int) (*User, error)
	GetCredentialByName(ctx context.Context, name string) (id int, passwordHash, role string, err error)
	StudentIDsForParent(ctx context.Context, parentID int) ([]int, error)
	ListStudents(ctx context.Context) ([]User, error)
	ListParentsWithStudents(ctx context.Context) ([]ParentWithStudents, error)
	StudentsForParent(ctx context.Context, parentID int) ([]User, error)
	LinkParentStudent(ctx context.Context, parentID, studentID int) error
	UnlinkParentStudent(ctx context.Context, parentID, studentID int) error
}

type TokenIssuer interface {
	Issue(p shared.Principal) (string, error)
}

package storage

const UserFieldUName = "UserName"
const UserFieldId = "Id"

type UserStorage interface {
	CreateUser(user *User) error
	QueryUser(field, val string) (*User, error)
	ListUser() ([]User, error)
}

type User struct {
	Id           string
	UserName     string
	PasswordHash string
	Email        string
}

func (p *psql) CreateUser(user *User) error {
	user.Id = "123"
	return nil
}

func (p *psql) QueryUser(field, val string) (*User, error) {
	return &User{
		Id:       "123",
		UserName: "VA",
	}, nil
}

func (p *psql) ListUser() ([]User, error) {
	return nil, nil
}

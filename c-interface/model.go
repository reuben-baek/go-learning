package c_interface

type User interface {
	GetID() int
	GetName() string
}

type UserEntity struct {
	ID   int
	Name string
}

func UserInstance(id int, name string) *UserEntity {
	return &UserEntity{ID: id, Name: name}
}

func (u *UserEntity) GetID() int {
	return u.ID
}

func (u *UserEntity) GetName() string {
	return u.Name
}

package repository

type User struct {
	ID   int
	Name string
}

type UserRepository struct {
	data map[int]User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		data: map[int]User{
			1: {ID: 1, Name: "spine-user"},
		},
	}
}

func (r *UserRepository) FindByID(id int) (User, bool) {
	user, ok := r.data[id]
	return user, ok
}

func (r *UserRepository) Save(user User) User {
	r.data[user.ID] = user
	return user
}

func (r *UserRepository) Delete(id int) {
	delete(r.data, id)
}

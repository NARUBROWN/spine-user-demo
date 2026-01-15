package service

import "spine-user-demo/repository"

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Get(id int) (repository.User, bool) {
	return s.repo.FindByID(id)
}

func (s *UserService) Create(id int, name string) repository.User {
	return s.repo.Save(repository.User{ID: id, Name: name})
}

func (s *UserService) Update(id int, name string) (repository.User, bool) {
	if _, ok := s.repo.FindByID(id); !ok {
		return repository.User{}, false
	}
	return s.repo.Save(repository.User{ID: id, Name: name}), true
}

func (s *UserService) Delete(id int) {
	s.repo.Delete(id)
}

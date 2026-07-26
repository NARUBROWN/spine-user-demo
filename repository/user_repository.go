package repository

import (
	"context"
	"spine-user-demo/entity"

	"github.com/uptrace/bun"
)

type UserRepository struct {
	db bun.IDB
}

func NewUserRepository(db bun.IDB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) FindByID(ctx context.Context, id int) (*entity.User, error) {
	return r.FindByIDWith(ctx, r.db, id)
}

func (r *UserRepository) FindByIDWith(ctx context.Context, db bun.IDB, id int) (*entity.User, error) {
	user := new(entity.User)

	err := db.NewSelect().
		Model(user).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) Save(ctx context.Context, tx bun.IDB, user *entity.User) error {
	_, err := tx.NewInsert().
		Model(user).
		Exec(ctx)

	return err
}

func (r *UserRepository) Update(ctx context.Context, tx bun.IDB, user *entity.User) error {
	_, err := tx.NewUpdate().
		Model(user).
		WherePK().
		Exec(ctx)

	return err
}

func (r *UserRepository) Delete(ctx context.Context, tx bun.IDB, id int) error {
	_, err := tx.NewDelete().
		Model((*entity.User)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

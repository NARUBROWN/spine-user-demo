package entity

import "time"

type User struct {
	ID        int64     `bun:",pk,autoincrement"`
	Name      string    `bun:",notnull"`
	Email     string    `bun:",unique,notnull"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

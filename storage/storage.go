package storage

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Storage interface {
	UserStorage
}

type psql struct {
	db *gorm.DB
}

type Config struct {
	Dns string `yaml:"dns"`
}

func NewStorage(c Config) (Storage, error) {
	db, err := gorm.Open(postgres.Open(c.Dns), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = autoMigrate(db)
	if err != nil {
		return nil, err
	}
	return &psql{
		db: db,
	}, err
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{})
}

package database

import "gorm.io/gorm"

type Store struct {
	DB *gorm.DB
}

func NewStore(databaseConnection *gorm.DB) *Store {
	return &Store{DB: databaseConnection}
}

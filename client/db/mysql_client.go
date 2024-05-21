package db

import "gorm.io/gorm"

type MySqlClient struct {
	client *gorm.DB
}

package models

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type URL struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	OriginalURL string    `json:"original_url" gorm:"not null"`
	ShortCode   string    `json:"short_code" gorm:"unique;not null"`
	Clicks      int       `json:"clicks" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func InitDatabase() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("urls.db"), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to database!")
	}

	// Auto-migrate the schema
	db.AutoMigrate(&URL{})

	return db
}

package main

import (
	"log"

	"url-shortener/handlers"
	"url-shortener/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Initialize database
	db, err := gorm.Open(sqlite.Open("urls.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate
	db.AutoMigrate(&models.URL{})

	// Initialize handlers
	handler := handlers.NewHandler(db)

	// Setup Gin router
	r := gin.Default()

	// Load HTML templates
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	// Routes
	r.GET("/", handler.HomePage)
	r.GET("/dashboard", handler.Dashboard)
	r.POST("/api/shorten", handler.ShortenURL)
	r.GET("/api/dashboard", handler.GetDashboardData)
	r.GET("/api/stats/:code", handler.GetStats)
	r.DELETE("/api/delete/:code", handler.DeleteURL)
	r.GET("/:code", handler.RedirectURL)

	// Start server
	log.Println("Server starting on :8080...")
	r.Run(":8080")
}

package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"net/url"
	"strings"

	"url-shortener/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	DB *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{DB: db}
}

// Generate random short code
func generateShortCode() string {
	bytes := make([]byte, 6)
	rand.Read(bytes)
	return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "=")
}

// Validate URL
func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// Shorten URL endpoint
func (h *Handler) ShortenURL(c *gin.Context) {
	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	// Add http:// if missing
	if !strings.HasPrefix(request.URL, "http://") && !strings.HasPrefix(request.URL, "https://") {
		request.URL = "http://" + request.URL
	}

	// Validate URL
	if !isValidURL(request.URL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}

	// Check if URL already exists
	var existingURL models.URL
	if err := h.DB.Where("original_url = ?", request.URL).First(&existingURL).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{
			"short_url":  c.Request.Host + "/" + existingURL.ShortCode,
			"short_code": existingURL.ShortCode,
		})
		return
	}

	// Generate unique short code
	var shortCode string
	for {
		shortCode = generateShortCode()
		var existing models.URL
		if err := h.DB.Where("short_code = ?", shortCode).First(&existing).Error; err != nil {
			break // Code is unique
		}
	}

	// Create new URL record
	newURL := models.URL{
		OriginalURL: request.URL,
		ShortCode:   shortCode,
	}

	if err := h.DB.Create(&newURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create short URL"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"short_url":  c.Request.Host + "/" + shortCode,
		"short_code": shortCode,
	})
}

// Redirect endpoint
func (h *Handler) RedirectURL(c *gin.Context) {
	shortCode := c.Param("code")

	var urlRecord models.URL
	if err := h.DB.Where("short_code = ?", shortCode).First(&urlRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	log.Println(urlRecord)

	// Raw SQL - simple and always works
	h.DB.Exec("UPDATE urls SET clicks = clicks + 1 WHERE short_code = ?", shortCode)

	c.Redirect(http.StatusMovedPermanently, urlRecord.OriginalURL)
}

// Get URL stats
func (h *Handler) GetStats(c *gin.Context) {
	shortCode := c.Param("code")

	var urlRecord models.URL
	if err := h.DB.Where("short_code = ?", shortCode).First(&urlRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	c.JSON(http.StatusOK, urlRecord)
}

// Dashboard endpoint
func (h *Handler) Dashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", nil)
}

// API endpoint for dashboard data
func (h *Handler) GetDashboardData(c *gin.Context) {
	var urls []models.URL
	var totalClicks int64
	var totalURLs int64

	// Get all URLs ordered by clicks
	h.DB.Order("clicks desc").Find(&urls)

	// Get total statistics
	h.DB.Model(&models.URL{}).Count(&totalURLs)
	h.DB.Model(&models.URL{}).Select("COALESCE(SUM(clicks), 0)").Scan(&totalClicks)

	// Get top 10 URLs
	var topURLs []models.URL
	h.DB.Order("clicks desc").Limit(10).Find(&topURLs)

	// Get recent URLs
	var recentURLs []models.URL
	h.DB.Order("created_at desc").Limit(10).Find(&recentURLs)

	// Prepare data for charts
	type ChartData struct {
		Labels []string `json:"labels"`
		Data   []int    `json:"data"`
	}

	// Top URLs chart data
	topChart := ChartData{
		Labels: make([]string, len(topURLs)),
		Data:   make([]int, len(topURLs)),
	}

	for i, url := range topURLs {
		if len(url.OriginalURL) > 30 {
			topChart.Labels[i] = url.OriginalURL[:30] + "..."
		} else {
			topChart.Labels[i] = url.OriginalURL
		}
		topChart.Data[i] = url.Clicks
	}

	c.JSON(http.StatusOK, gin.H{
		"total_urls":   totalURLs,
		"total_clicks": totalClicks,
		"all_urls":     urls,
		"top_urls":     topURLs,
		"recent_urls":  recentURLs,
		"top_chart":    topChart,
	})
}

// Serve home page
func (h *Handler) HomePage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

// Delete URL endpoint
func (h *Handler) DeleteURL(c *gin.Context) {
	shortCode := c.Param("code")

	result := h.DB.Where("short_code = ?", shortCode).Delete(&models.URL{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete URL"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "URL deleted successfully"})
}

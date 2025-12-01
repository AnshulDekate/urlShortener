package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time" 
	"strconv"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/AnshulDekate/urlShortener/service" 
)

type GinHandler struct {
	Service *service.Service
	Domain  string 
}

func NewGinHandler(svc *service.Service, domain string) *GinHandler {
	return &GinHandler{
		Service: svc,
		Domain:  domain,
	}
}

func (h *GinHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second) 
	defer cancel()

	err := h.Service.HealthCheck(ctx) 
	
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "Down", 
			"db_status": "connection failed",
			"error": err.Error(),
		}) 
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "Up", 
		"db_status": "ok",
	})
}

func (h *GinHandler) Shorten(c *gin.Context) {
    
	var req struct {
		LongURL string `json:"long_url" binding:"required"`
	}
    
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload (Expected JSON: {\"long_url\": \"...\"})",
		})
		return
	}

	shortCode, err := h.Service.CreateShortURL(req.LongURL)
	if err != nil {
		if strings.Contains(err.Error(), "invalid URL format") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "service capacity exhausted") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Short code generation failed. Try again later."})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error: Failed to process URL creation."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"short_url": h.Domain + shortCode,
	})
}

func (h *GinHandler) Redirect(c *gin.Context) {
	shortCode := c.Param("code")
	if shortCode == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}

	longURL, err := h.Service.GetLongURL(shortCode)
	
	if err != nil {
		if strings.Contains(err.Error(), "short code not found") || errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Short code not found"})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error during lookup"})
		return
	}

	c.Redirect(http.StatusFound, longURL) // 302 Found
}

func (h *GinHandler) ListURLs(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
    
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second) 
	defer cancel()

	listResponse, err := h.Service.ListURLs(ctx, page, limit)
	if err != nil {
		log.Printf("Service error during URL listing: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve URL list."})
		return
	}
	
	for i:=0; i<len(listResponse.URLs); i++ {
		listResponse.URLs[i].ShortCode = h.Domain + listResponse.URLs[i].ShortCode 
	}
	c.JSON(http.StatusOK, listResponse)
}

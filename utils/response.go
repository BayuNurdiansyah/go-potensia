package utils

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// OK sends a 200 OK response with data directly (no wrapper).
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// PaginatedOK sends paginated response.
func PaginatedOK(c *gin.Context, data interface{}, total int64, page, limit int) {
	c.JSON(http.StatusOK, gin.H{
		"data":        data,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
	})
}

// Paginate extracts page & limit from query params (defaults: page=1, limit=20).
func Paginate(c *gin.Context) (page int, limit int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return
}

// Success sends a 200 OK response.
func Success(c *gin.Context, data gin.H) {
	c.JSON(http.StatusOK, data)
}

// Created sends a 201 Created response.
func Created(c *gin.Context, data gin.H) {
	c.JSON(http.StatusCreated, data)
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"message": message})
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, gin.H{"message": message})
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, gin.H{"message": message})
}

// NotFound sends a 404 Not Found response.
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, gin.H{"message": message})
}

// Conflict sends a 409 Conflict response.
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, gin.H{"message": message})
}

// TooManyRequests sends a 429 response.
func TooManyRequests(c *gin.Context, message string, retryAfter int64) {
	c.JSON(http.StatusTooManyRequests, gin.H{
		"message":     message,
		"retry_after": retryAfter,
	})
}

// InternalError sends a 500 response.
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, gin.H{"message": message})
}
package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

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
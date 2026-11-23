package response

import "github.com/gin-gonic/gin"

// Body adalah format JSON standar seluruh endpoint: {status, message, data?}.
type Body struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Success untuk respons berhasil (2xx).
func Success(c *gin.Context, code int, message string, data any) {
	c.JSON(code, Body{Status: "success", Message: message, Data: data})
}

// Fail untuk kesalahan dari sisi klien (4xx).
func Fail(c *gin.Context, code int, message string) {
	c.JSON(code, Body{Status: "fail", Message: message})
}

// Error untuk kesalahan dari sisi server (5xx).
func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Body{Status: "error", Message: message})
}

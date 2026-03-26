package utils

import (
	"bili-auth-backend/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func OK(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, model.APIResponse{Code: 0, Message: msg, Data: data})
}

func Fail(c *gin.Context, httpStatus int, code int, msg string, details any) {
	c.JSON(httpStatus, model.APIResponse{Code: code, Message: msg, Details: details})
}

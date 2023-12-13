package routers

import "github.com/gin-gonic/gin"

// GetCommonCallbackErr return error message
func GetCommonCallbackErr(err error) gin.H {
	return gin.H{
		"message":  err.Error(),
	}
}

// GetCommonCallbackSuccess return success message
func GetCommonCallbackSuccess() gin.H {
	return gin.H{
		"message": "SUCCESS",
	}
}

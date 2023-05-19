package routers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/callback"
	opscloudclient "gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/client"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(Cors())
	altershieldOpenapi := r.Group("/openapi/altershield")
	{
		altershieldOpenapi.POST("/callback", callback.CheckCallBackHandler)
		altershieldOpenapi.POST("/liveTest", callback.LiveTest)

		altershieldOpenapi.GET("/suspend/deployment", callback.GetSuspendDeployment)
		altershieldOpenapi.PUT("/deployment/rollback", callback.DeploymentRollback)
	}

	// TODO delete
	altershieldClientweb := r.Group("/altershield/client")
	{
		altershieldClientweb.POST("/submitChangeExecOrderWeb", opscloudclient.SubmitChangeExecOrderWeb)
		altershieldClientweb.POST("/submitChangeStartNotifyWeb", opscloudclient.SubmitChangeStartNotifyWeb)
		altershieldClientweb.POST("/submitChangeFinishNotifyWeb", opscloudclient.SubmitChangeFinishNotifyWeb)
	}
	return r
}

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin") //请求头部
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Origin", "*") // 设置允许访问所有域: *
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar")
			c.Header("Access-Control-Max-Age", "172800")
			c.Header("Access-Control-Allow-Credentials", "false")
			c.Set("content-type", "application/json") //// 设置返回格式是js
		}

		//允许类型校验
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, "ok!")
		}

		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic info is: %v", err)
			}
		}()

		c.Next()
	}
}

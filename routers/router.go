package routers

import (
	metricprovider "github.com/traas-stack/altershield-operator/pkg/metric/provider"
	"log"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gin-gonic/gin"
)

func SetupRouter(apiClient client.Client, p metricprovider.Interface) *gin.Engine {
	r := gin.Default()
	r.Use(Cors())
	altershieldOpenapi := r.Group("/openapi")

	callbackHandler := &CallbackHandler{
		Client: apiClient,
	}
	metricHandler := &MetricHandler{
		metricProvider: p,
	}
	altershieldOpenapi.POST("/callback", callbackHandler.CheckCallBackHandler)
	altershieldOpenapi.POST("/liveTest", LiveTest)

	altershieldOpenapi.POST("/metric/query", metricHandler.Query)

	return r
}

func LiveTest(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Hello world"})
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

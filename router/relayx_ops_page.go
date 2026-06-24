package router

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed relayx_ops_page.html
var relayXOpsPageHTML []byte

func SetRelayXOpsPageRouter(router *gin.Engine) {
	router.GET("/relayx/ops", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", relayXOpsPageHTML)
	})
}

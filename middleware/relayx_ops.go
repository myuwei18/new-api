package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
)

const relayXOpsReadKeyOption = "RelayXOpsReadKey"

func RelayXOpsReadAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		configuredKey := relayXOpsReadKey()
		if configuredKey == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "relayx ops read key is not configured",
			})
			c.Abort()
			return
		}
		providedKey := relayXOpsBearerToken(c.GetHeader("Authorization"))
		if providedKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "relayx ops read key is required",
			})
			c.Abort()
			return
		}
		if providedKey != configuredKey {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "invalid relayx ops read key",
			})
			c.Abort()
			return
		}
		c.Set("relayx_ops_readonly", true)
		c.Next()
	}
}

func relayXOpsReadKey() string {
	if key := strings.TrimSpace(os.Getenv("RELAYX_OPS_READ_KEY")); key != "" {
		return key
	}
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	if common.OptionMap == nil {
		return ""
	}
	return strings.TrimSpace(common.OptionMap[relayXOpsReadKeyOption])
}

func relayXOpsBearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

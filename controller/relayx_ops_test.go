package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRelayXOpsTest(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Channel{}, &model.Log{}))
	model.DB = db
	model.LOG_DB = db
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
	t.Setenv("RELAYX_OPS_READ_KEY", "test-ops-key")

	now := time.Now().Unix()
	users := []model.User{
		{Id: 1, Username: "alpha-user", Email: "alpha@example.com", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "aff-alpha", Quota: 1000, UsedQuota: 100, RequestCount: 2},
		{Id: 2, Username: "root-user", Email: "root@example.com", Role: common.RoleRootUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "aff-root", Quota: 5000, UsedQuota: 0, RequestCount: 0},
	}
	require.NoError(t, db.Create(&users).Error)
	channels := []model.Channel{
		{Id: 10, Name: "primary-channel", Type: 1, Status: 1, Group: "default", ResponseTime: 100, TestTime: now},
	}
	require.NoError(t, db.Create(&channels).Error)
	logs := []model.Log{
		{UserId: 1, CreatedAt: now - 60, Type: model.LogTypeConsume, Username: "alpha-user", TokenName: "secret-token-name", ModelName: "gpt-5.5", Quota: 120, ChannelId: 10, TokenId: 3, Group: "default", RequestId: "req_abcdefghijklmnopqrstuvwxyz"},
		{UserId: 1, CreatedAt: now - 30, Type: model.LogTypeError, Username: "alpha-user", TokenName: "secret-token-name", ModelName: "gpt-5.5", Quota: 0, ChannelId: 10, TokenId: 3, Group: "default", Content: "prompt: do not leak this"},
		{UserId: 2, CreatedAt: now - 20, Type: model.LogTypeManage, Username: "root-user", Content: "rotated sensitive key abc123"},
	}
	require.NoError(t, db.Create(&logs).Error)

	router := gin.New()
	ops := router.Group("/api/relayx/ops")
	ops.Use(middleware.RelayXOpsReadAuth())
	ops.GET("/summary", GetRelayXOpsSummary)
	ops.GET("/users", GetRelayXOpsUsers)
	ops.GET("/logs", GetRelayXOpsLogs)
	ops.GET("/models", GetRelayXOpsModels)
	ops.GET("/channels", GetRelayXOpsChannels)
	ops.GET("/billing", GetRelayXOpsBilling)
	ops.GET("/security-events", GetRelayXOpsSecurityEvents)
	return router
}

func TestRelayXOpsReadAuthRequiresKey(t *testing.T) {
	router := setupRelayXOpsTest(t)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/relayx/ops/summary", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestRelayXOpsSummaryReturnsMaskedAggregates(t *testing.T) {
	router := setupRelayXOpsTest(t)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/relayx/ops/summary?window=1d", nil)
	req.Header.Set("Authorization", "Bearer test-ops-key")
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	assert.Contains(t, body, `"success":true`)
	assert.Contains(t, body, `"requests":2`)
	assert.Contains(t, body, `"success":1`)
	assert.Contains(t, body, `"failures":1`)
	assert.Contains(t, body, `"quota":120`)
	assert.NotContains(t, body, "alpha@example.com")
	assert.NotContains(t, body, "alpha-user")
	assert.NotContains(t, body, "primary-channel")
	assert.NotContains(t, body, "secret-token-name")
}

func TestRelayXOpsAllEndpointsReturnJSON(t *testing.T) {
	router := setupRelayXOpsTest(t)
	paths := []string{
		"/api/relayx/ops/summary",
		"/api/relayx/ops/users",
		"/api/relayx/ops/logs",
		"/api/relayx/ops/models",
		"/api/relayx/ops/channels",
		"/api/relayx/ops/billing",
		"/api/relayx/ops/security-events",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path+"?window=1d", nil)
			req.Header.Set("Authorization", "Bearer test-ops-key")
			router.ServeHTTP(recorder, req)
			require.Equal(t, http.StatusOK, recorder.Code)
			body := recorder.Body.String()
			assert.Contains(t, body, `"success":true`)
			assert.Contains(t, body, `"generated_at"`)
			assert.NotContains(t, body, "alpha@example.com")
			assert.NotContains(t, body, "alpha-user")
			assert.NotContains(t, body, "primary-channel")
			assert.NotContains(t, body, "secret-token-name")
			assert.NotContains(t, body, "prompt: do not leak this")
		})
	}
}

func TestRelayXOpsLogsDoNotReturnRawContent(t *testing.T) {
	router := setupRelayXOpsTest(t)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/relayx/ops/logs?window=1d", nil)
	req.Header.Set("Authorization", "Bearer test-ops-key")
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	assert.NotContains(t, body, "prompt: do not leak this")
	assert.NotContains(t, body, "secret-token-name")
	assert.Contains(t, body, `"error_digest":"error"`)
}

func TestRelayXOpsSecurityEventsDoNotReturnRawContent(t *testing.T) {
	router := setupRelayXOpsTest(t)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/relayx/ops/security-events?window=1d", nil)
	req.Header.Set("Authorization", "Bearer test-ops-key")
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)

	body := recorder.Body.String()
	assert.NotContains(t, body, "rotated sensitive key")
	assert.Contains(t, body, `"content":"manage"`)
}

func TestRelayXOpsReadAuthCanUseOptionMap(t *testing.T) {
	t.Setenv("RELAYX_OPS_READ_KEY", "")
	gin.SetMode(gin.TestMode)
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{"RelayXOpsReadKey": "option-key"}
	common.OptionMapRWMutex.Unlock()

	router := gin.New()
	router.GET("/protected", middleware.RelayXOpsReadAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer option-key")
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

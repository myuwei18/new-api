package controller

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	relayXOpsDefaultWindow = "7d"
	relayXOpsTopLimit      = 10
)

type relayXOpsEnvelope struct {
	Success     bool        `json:"success"`
	Data        interface{} `json:"data"`
	GeneratedAt string      `json:"generated_at"`
}

type relayXOpsWindow struct {
	Name           string `json:"name"`
	StartTimestamp int64  `json:"start_timestamp"`
	EndTimestamp   int64  `json:"end_timestamp"`
}

type relayXOpsMetric struct {
	Requests int64 `json:"requests"`
	Success  int64 `json:"success"`
	Failures int64 `json:"failures"`
	Quota    int64 `json:"quota"`
}

type relayXOpsTopItem struct {
	ID          int     `json:"id,omitempty"`
	Name        string  `json:"name,omitempty"`
	Requests    int64   `json:"requests"`
	Success     int64   `json:"success"`
	Failures    int64   `json:"failures"`
	FailureRate float64 `json:"failure_rate"`
	Quota       int64   `json:"quota"`
}

type relayXOpsSummary struct {
	Window      relayXOpsWindow    `json:"window"`
	Users       relayXOpsUsers     `json:"users"`
	Requests    relayXOpsMetric    `json:"requests"`
	FailureRate float64            `json:"failure_rate"`
	TopUsers    []relayXOpsTopItem `json:"top_users"`
	TopModels   []relayXOpsTopItem `json:"top_models"`
	TopChannels []relayXOpsTopItem `json:"top_channels"`
	Notes       []string           `json:"notes"`
}

type relayXOpsUsers struct {
	Total       int64               `json:"total"`
	Active      int64               `json:"active"`
	Admins      int64               `json:"admins"`
	Common      int64               `json:"common"`
	Disabled    int64               `json:"disabled"`
	TopByUsage  []relayXOpsUserItem `json:"top_by_usage,omitempty"`
	RecentUsers []relayXOpsUserItem `json:"recent_users,omitempty"`
}

type relayXOpsUserItem struct {
	UserID         int    `json:"user_id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	Role           int    `json:"role"`
	Status         int    `json:"status"`
	Group          string `json:"group"`
	Quota          int    `json:"quota"`
	UsedQuota      int    `json:"used_quota"`
	RequestCount   int    `json:"request_count"`
	WindowRequests int64  `json:"window_requests,omitempty"`
	WindowFailures int64  `json:"window_failures,omitempty"`
	WindowQuota    int64  `json:"window_quota,omitempty"`
}

type relayXOpsLogs struct {
	Window      relayXOpsWindow    `json:"window"`
	Requests    relayXOpsMetric    `json:"requests"`
	FailureRate float64            `json:"failure_rate"`
	Recent      []relayXOpsLogItem `json:"recent"`
}

type relayXOpsLogItem struct {
	ID          int    `json:"id"`
	CreatedAt   int64  `json:"created_at"`
	Type        int    `json:"type"`
	UserID      int    `json:"user_id"`
	ModelName   string `json:"model_name"`
	ChannelID   int    `json:"channel_id"`
	Quota       int    `json:"quota"`
	UseTime     int    `json:"use_time"`
	IsStream    bool   `json:"is_stream"`
	Group       string `json:"group"`
	RequestID   string `json:"request_id,omitempty"`
	ErrorDigest string `json:"error_digest,omitempty"`
}

type relayXOpsModels struct {
	Window relayXOpsWindow    `json:"window"`
	Models []relayXOpsTopItem `json:"models"`
}

type relayXOpsChannels struct {
	Window   relayXOpsWindow        `json:"window"`
	Channels []relayXOpsChannelItem `json:"channels"`
}

type relayXOpsChannelItem struct {
	ChannelID    int     `json:"channel_id"`
	Name         string  `json:"name"`
	Type         int     `json:"type"`
	Status       int     `json:"status"`
	Group        string  `json:"group"`
	Tag          string  `json:"tag,omitempty"`
	Requests     int64   `json:"requests"`
	Success      int64   `json:"success"`
	Failures     int64   `json:"failures"`
	FailureRate  float64 `json:"failure_rate"`
	Quota        int64   `json:"quota"`
	ResponseTime int     `json:"response_time"`
	TestTime     int64   `json:"test_time"`
}

type relayXOpsBilling struct {
	Window       relayXOpsWindow         `json:"window"`
	Quota        int64                   `json:"quota"`
	TopupQuota   int64                   `json:"topup_quota"`
	RefundQuota  int64                   `json:"refund_quota"`
	UserBalances relayXOpsBalanceSummary `json:"user_balances"`
	Notes        []string                `json:"notes"`
}

type relayXOpsBalanceSummary struct {
	TotalQuota     int64 `json:"total_quota"`
	TotalUsedQuota int64 `json:"total_used_quota"`
	NegativeUsers  int64 `json:"negative_users"`
}

type relayXOpsSecurityEvents struct {
	Window relayXOpsWindow          `json:"window"`
	Events []relayXOpsSecurityEvent `json:"events"`
	Notes  []string                 `json:"notes"`
}

type relayXOpsSecurityEvent struct {
	ID        int    `json:"id"`
	CreatedAt int64  `json:"created_at"`
	Type      int    `json:"type"`
	UserID    int    `json:"user_id"`
	Action    string `json:"action"`
	Content   string `json:"content"`
	RequestID string `json:"request_id,omitempty"`
}

type relayXOpsAggregateRow struct {
	ID       int
	Name     string
	Requests int64
	Success  int64
	Failures int64
	Quota    int64
}

func GetRelayXOpsSummary(c *gin.Context) {
	window := relayXOpsParseWindow(c)
	metrics, err := relayXOpsRequestMetrics(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	users, err := relayXOpsUserSummary(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	topUsers, err := relayXOpsTopUsers(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	topModels, err := relayXOpsTopModels(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	topChannels, err := relayXOpsTopChannels(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	relayXOpsJSON(c, relayXOpsSummary{
		Window:      window,
		Users:       users,
		Requests:    metrics,
		FailureRate: relayXOpsFailureRate(metrics.Failures, metrics.Requests),
		TopUsers:    topUsers,
		TopModels:   topModels,
		TopChannels: topChannels,
		Notes: []string{
			"This endpoint is read-only and returns masked aggregate data for external operations agents.",
			"Failure metrics count consume logs as success and error logs as failures.",
		},
	})
}

func GetRelayXOpsUsers(c *gin.Context) {
	window := relayXOpsParseWindow(c)
	users, err := relayXOpsUserSummary(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	relayXOpsJSON(c, struct {
		Window relayXOpsWindow `json:"window"`
		Users  relayXOpsUsers  `json:"users"`
	}{Window: window, Users: users})
}

func GetRelayXOpsLogs(c *gin.Context) {
	window := relayXOpsParseWindow(c)
	metrics, err := relayXOpsRequestMetrics(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	recent, err := relayXOpsRecentLogs(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	relayXOpsJSON(c, relayXOpsLogs{
		Window:      window,
		Requests:    metrics,
		FailureRate: relayXOpsFailureRate(metrics.Failures, metrics.Requests),
		Recent:      recent,
	})
}

func GetRelayXOpsModels(c *gin.Context) {
	window := relayXOpsParseWindow(c)
	models, err := relayXOpsTopModels(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	relayXOpsJSON(c, relayXOpsModels{Window: window, Models: models})
}

func GetRelayXOpsChannels(c *gin.Context) {
	window := relayXOpsParseWindow(c)
	channels, err := relayXOpsChannelSummary(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	relayXOpsJSON(c, relayXOpsChannels{Window: window, Channels: channels})
}

func GetRelayXOpsBilling(c *gin.Context) {
	window := relayXOpsParseWindow(c)
	quota, err := relayXOpsQuotaByType(window, model.LogTypeConsume)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	topupQuota, err := relayXOpsQuotaByType(window, model.LogTypeTopup)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	refundQuota, err := relayXOpsQuotaByType(window, model.LogTypeRefund)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	balances, err := relayXOpsBalanceSummaryData()
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	relayXOpsJSON(c, relayXOpsBilling{
		Window:       window,
		Quota:        quota,
		TopupQuota:   topupQuota,
		RefundQuota:  refundQuota,
		UserBalances: balances,
		Notes: []string{
			"Billing data is aggregate-only and does not include payment secrets or raw provider billing records.",
		},
	})
}

func GetRelayXOpsSecurityEvents(c *gin.Context) {
	window := relayXOpsParseWindow(c)
	events, err := relayXOpsSecurityEventLogs(window)
	if err != nil {
		relayXOpsError(c, err)
		return
	}
	relayXOpsJSON(c, relayXOpsSecurityEvents{
		Window: window,
		Events: events,
		Notes: []string{
			"This first skeleton uses existing manage/system/error logs as security-event signals until the System Event table lands.",
		},
	})
}

func relayXOpsJSON(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, relayXOpsEnvelope{
		Success:     true,
		Data:        data,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func relayXOpsError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"message": "failed to build relayx operations snapshot: " + err.Error(),
	})
}

func relayXOpsParseWindow(c *gin.Context) relayXOpsWindow {
	name := strings.ToLower(strings.TrimSpace(c.Query("window")))
	if name == "" {
		name = relayXOpsDefaultWindow
	}
	now := time.Now().UTC()
	duration := 7 * 24 * time.Hour
	switch name {
	case "1d":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	default:
		name = relayXOpsDefaultWindow
	}
	return relayXOpsWindow{
		Name:           name,
		StartTimestamp: now.Add(-duration).Unix(),
		EndTimestamp:   now.Unix(),
	}
}

func relayXOpsLogWindowQuery(window relayXOpsWindow) *gorm.DB {
	return model.LOG_DB.Model(&model.Log{}).Where("created_at >= ? AND created_at <= ?", window.StartTimestamp, window.EndTimestamp)
}

func relayXOpsRequestMetrics(window relayXOpsWindow) (relayXOpsMetric, error) {
	var success int64
	if err := relayXOpsLogWindowQuery(window).Where("type = ?", model.LogTypeConsume).Count(&success).Error; err != nil {
		return relayXOpsMetric{}, err
	}
	var failures int64
	if err := relayXOpsLogWindowQuery(window).Where("type = ?", model.LogTypeError).Count(&failures).Error; err != nil {
		return relayXOpsMetric{}, err
	}
	quota, err := relayXOpsQuotaByType(window, model.LogTypeConsume)
	if err != nil {
		return relayXOpsMetric{}, err
	}
	return relayXOpsMetric{Requests: success + failures, Success: success, Failures: failures, Quota: quota}, nil
}

func relayXOpsQuotaByType(window relayXOpsWindow, logType int) (int64, error) {
	var quota int64
	err := relayXOpsLogWindowQuery(window).Where("type = ?", logType).Select("COALESCE(sum(quota), 0)").Scan(&quota).Error
	return quota, err
}

func relayXOpsUserSummary(window relayXOpsWindow) (relayXOpsUsers, error) {
	var summary relayXOpsUsers
	if err := model.DB.Model(&model.User{}).Count(&summary.Total).Error; err != nil {
		return summary, err
	}
	if err := model.DB.Model(&model.User{}).Where("role >= ?", common.RoleAdminUser).Count(&summary.Admins).Error; err != nil {
		return summary, err
	}
	if err := model.DB.Model(&model.User{}).Where("role = ?", common.RoleCommonUser).Count(&summary.Common).Error; err != nil {
		return summary, err
	}
	if err := model.DB.Model(&model.User{}).Where("status <> ?", common.UserStatusEnabled).Count(&summary.Disabled).Error; err != nil {
		return summary, err
	}
	if err := relayXOpsLogWindowQuery(window).Distinct("user_id").Where("user_id > 0").Count(&summary.Active).Error; err != nil {
		return summary, err
	}
	top, err := relayXOpsTopUserDetails(window)
	if err != nil {
		return summary, err
	}
	recent, err := relayXOpsRecentUsers()
	if err != nil {
		return summary, err
	}
	summary.TopByUsage = top
	summary.RecentUsers = recent
	return summary, nil
}

func relayXOpsTopUsers(window relayXOpsWindow) ([]relayXOpsTopItem, error) {
	rows, err := relayXOpsGroupedStats(window, "user_id", "user_id > 0", "quota")
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	users, err := relayXOpsUsersByIDs(ids)
	if err != nil {
		return nil, err
	}
	items := make([]relayXOpsTopItem, 0, len(rows))
	for _, row := range rows {
		name := fmt.Sprintf("user-%d", row.ID)
		if user, ok := users[row.ID]; ok {
			name = relayXOpsMaskText(user.Username)
			if name == "" {
				name = relayXOpsMaskEmail(user.Email)
			}
		}
		items = append(items, relayXOpsTopItem{
			ID:          row.ID,
			Name:        name,
			Requests:    row.Requests,
			Success:     row.Success,
			Failures:    row.Failures,
			FailureRate: relayXOpsFailureRate(row.Failures, row.Requests),
			Quota:       row.Quota,
		})
	}
	return items, nil
}

func relayXOpsTopModels(window relayXOpsWindow) ([]relayXOpsTopItem, error) {
	rows, err := relayXOpsGroupedStats(window, "model_name", "model_name <> ''", "requests")
	if err != nil {
		return nil, err
	}
	items := make([]relayXOpsTopItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, relayXOpsTopItem{
			Name:        row.Name,
			Requests:    row.Requests,
			Success:     row.Success,
			Failures:    row.Failures,
			FailureRate: relayXOpsFailureRate(row.Failures, row.Requests),
			Quota:       row.Quota,
		})
	}
	return items, nil
}

func relayXOpsTopChannels(window relayXOpsWindow) ([]relayXOpsTopItem, error) {
	rows, err := relayXOpsGroupedStats(window, "channel_id", "channel_id > 0", "requests")
	if err != nil {
		return nil, err
	}
	channels, err := relayXOpsChannelsByIDs(relayXOpsRowIDs(rows))
	if err != nil {
		return nil, err
	}
	items := make([]relayXOpsTopItem, 0, len(rows))
	for _, row := range rows {
		name := fmt.Sprintf("channel-%d", row.ID)
		if channel, ok := channels[row.ID]; ok {
			name = relayXOpsMaskText(channel.Name)
		}
		items = append(items, relayXOpsTopItem{
			ID:          row.ID,
			Name:        name,
			Requests:    row.Requests,
			Success:     row.Success,
			Failures:    row.Failures,
			FailureRate: relayXOpsFailureRate(row.Failures, row.Requests),
			Quota:       row.Quota,
		})
	}
	return items, nil
}

func relayXOpsGroupedStats(window relayXOpsWindow, groupColumn string, extraCondition string, orderBy string) ([]relayXOpsAggregateRow, error) {
	selectColumn := groupColumn
	if groupColumn == "user_id" || groupColumn == "channel_id" {
		selectColumn += " AS id"
	} else {
		selectColumn += " AS name"
	}
	query := relayXOpsLogWindowQuery(window).
		Select(selectColumn+", count(*) AS requests, COALESCE(sum(CASE WHEN type = ? THEN 1 ELSE 0 END), 0) AS success, COALESCE(sum(CASE WHEN type = ? THEN 1 ELSE 0 END), 0) AS failures, COALESCE(sum(CASE WHEN type = ? THEN quota ELSE 0 END), 0) AS quota", model.LogTypeConsume, model.LogTypeError, model.LogTypeConsume).
		Where("type IN ?", []int{model.LogTypeConsume, model.LogTypeError}).
		Group(groupColumn).
		Limit(relayXOpsTopLimit)
	if extraCondition != "" {
		query = query.Where(extraCondition)
	}
	if orderBy == "quota" {
		query = query.Order("quota DESC")
	} else {
		query = query.Order("requests DESC")
	}
	var rows []relayXOpsAggregateRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func relayXOpsTopUserDetails(window relayXOpsWindow) ([]relayXOpsUserItem, error) {
	rows, err := relayXOpsGroupedStats(window, "user_id", "user_id > 0", "quota")
	if err != nil {
		return nil, err
	}
	users, err := relayXOpsUsersByIDs(relayXOpsRowIDs(rows))
	if err != nil {
		return nil, err
	}
	items := make([]relayXOpsUserItem, 0, len(rows))
	for _, row := range rows {
		user, ok := users[row.ID]
		if !ok {
			items = append(items, relayXOpsUserItem{UserID: row.ID, Username: fmt.Sprintf("user-%d", row.ID), WindowRequests: row.Requests, WindowFailures: row.Failures, WindowQuota: row.Quota})
			continue
		}
		items = append(items, relayXOpsUserItem{
			UserID:         user.Id,
			Username:       relayXOpsMaskText(user.Username),
			Email:          relayXOpsMaskEmail(user.Email),
			Role:           user.Role,
			Status:         user.Status,
			Group:          user.Group,
			Quota:          user.Quota,
			UsedQuota:      user.UsedQuota,
			RequestCount:   user.RequestCount,
			WindowRequests: row.Requests,
			WindowFailures: row.Failures,
			WindowQuota:    row.Quota,
		})
	}
	return items, nil
}

func relayXOpsRecentUsers() ([]relayXOpsUserItem, error) {
	var users []model.User
	if err := model.DB.Order("id DESC").Limit(relayXOpsTopLimit).Find(&users).Error; err != nil {
		return nil, err
	}
	items := make([]relayXOpsUserItem, 0, len(users))
	for _, user := range users {
		items = append(items, relayXOpsUserItem{
			UserID:       user.Id,
			Username:     relayXOpsMaskText(user.Username),
			Email:        relayXOpsMaskEmail(user.Email),
			Role:         user.Role,
			Status:       user.Status,
			Group:        user.Group,
			Quota:        user.Quota,
			UsedQuota:    user.UsedQuota,
			RequestCount: user.RequestCount,
		})
	}
	return items, nil
}

func relayXOpsRecentLogs(window relayXOpsWindow) ([]relayXOpsLogItem, error) {
	var logs []model.Log
	if err := relayXOpsLogWindowQuery(window).
		Where("type IN ?", []int{model.LogTypeConsume, model.LogTypeError}).
		Order("id DESC").
		Limit(relayXOpsTopLimit).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	items := make([]relayXOpsLogItem, 0, len(logs))
	for _, log := range logs {
		items = append(items, relayXOpsLogItem{
			ID:          log.Id,
			CreatedAt:   log.CreatedAt,
			Type:        log.Type,
			UserID:      log.UserId,
			ModelName:   log.ModelName,
			ChannelID:   log.ChannelId,
			Quota:       log.Quota,
			UseTime:     log.UseTime,
			IsStream:    log.IsStream,
			Group:       log.Group,
			RequestID:   relayXOpsShortID(log.RequestId),
			ErrorDigest: relayXOpsLogTypeName(log.Type),
		})
	}
	return items, nil
}

func relayXOpsChannelSummary(window relayXOpsWindow) ([]relayXOpsChannelItem, error) {
	var channels []model.Channel
	if err := model.DB.Omit("key", "base_url", "other", "other_info", "setting", "param_override", "header_override", "channel_info", "settings").Order("id DESC").Find(&channels).Error; err != nil {
		return nil, err
	}
	rows, err := relayXOpsGroupedStats(window, "channel_id", "channel_id > 0", "requests")
	if err != nil {
		return nil, err
	}
	stats := make(map[int]relayXOpsAggregateRow, len(rows))
	for _, row := range rows {
		stats[row.ID] = row
	}
	items := make([]relayXOpsChannelItem, 0, len(channels))
	for _, channel := range channels {
		row := stats[channel.Id]
		tag := ""
		if channel.Tag != nil {
			tag = relayXOpsMaskText(*channel.Tag)
		}
		items = append(items, relayXOpsChannelItem{
			ChannelID:    channel.Id,
			Name:         relayXOpsMaskText(channel.Name),
			Type:         channel.Type,
			Status:       channel.Status,
			Group:        channel.Group,
			Tag:          tag,
			Requests:     row.Requests,
			Success:      row.Success,
			Failures:     row.Failures,
			FailureRate:  relayXOpsFailureRate(row.Failures, row.Requests),
			Quota:        row.Quota,
			ResponseTime: channel.ResponseTime,
			TestTime:     channel.TestTime,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Requests == items[j].Requests {
			return items[i].ChannelID > items[j].ChannelID
		}
		return items[i].Requests > items[j].Requests
	})
	if len(items) > relayXOpsTopLimit {
		items = items[:relayXOpsTopLimit]
	}
	return items, nil
}

func relayXOpsBalanceSummaryData() (relayXOpsBalanceSummary, error) {
	var summary relayXOpsBalanceSummary
	if err := model.DB.Model(&model.User{}).Select("COALESCE(sum(quota), 0)").Scan(&summary.TotalQuota).Error; err != nil {
		return summary, err
	}
	if err := model.DB.Model(&model.User{}).Select("COALESCE(sum(used_quota), 0)").Scan(&summary.TotalUsedQuota).Error; err != nil {
		return summary, err
	}
	if err := model.DB.Model(&model.User{}).Where("quota < 0").Count(&summary.NegativeUsers).Error; err != nil {
		return summary, err
	}
	return summary, nil
}

func relayXOpsSecurityEventLogs(window relayXOpsWindow) ([]relayXOpsSecurityEvent, error) {
	var logs []model.Log
	if err := relayXOpsLogWindowQuery(window).
		Where("type IN ?", []int{model.LogTypeManage, model.LogTypeSystem, model.LogTypeError, model.LogTypeLogin}).
		Order("id DESC").
		Limit(relayXOpsTopLimit).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	events := make([]relayXOpsSecurityEvent, 0, len(logs))
	for _, log := range logs {
		events = append(events, relayXOpsSecurityEvent{
			ID:        log.Id,
			CreatedAt: log.CreatedAt,
			Type:      log.Type,
			UserID:    log.UserId,
			Action:    relayXOpsLogTypeName(log.Type),
			Content:   relayXOpsLogTypeName(log.Type),
			RequestID: relayXOpsShortID(log.RequestId),
		})
	}
	return events, nil
}

func relayXOpsUsersByIDs(ids []int) (map[int]model.User, error) {
	users := map[int]model.User{}
	if len(ids) == 0 {
		return users, nil
	}
	var list []model.User
	if err := model.DB.Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for _, user := range list {
		users[user.Id] = user
	}
	return users, nil
}

func relayXOpsChannelsByIDs(ids []int) (map[int]model.Channel, error) {
	channels := map[int]model.Channel{}
	if len(ids) == 0 {
		return channels, nil
	}
	var list []model.Channel
	if err := model.DB.Select("id", "name", "type", "status").Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	for _, channel := range list {
		channels[channel.Id] = channel
	}
	return channels, nil
}

func relayXOpsRowIDs(rows []relayXOpsAggregateRow) []int {
	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func relayXOpsFailureRate(failures int64, requests int64) float64 {
	if requests == 0 {
		return 0
	}
	return float64(failures) / float64(requests)
}

func relayXOpsMaskText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 2 {
		return string(runes[:1]) + "*"
	}
	if len(runes) <= 6 {
		return string(runes[:1]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1:])
	}
	return string(runes[:2]) + strings.Repeat("*", len(runes)-4) + string(runes[len(runes)-2:])
}

func relayXOpsMaskEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return relayXOpsMaskText(email)
	}
	return relayXOpsMaskText(parts[0]) + "@" + relayXOpsMaskText(parts[1])
}

func relayXOpsShortID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= 12 {
		return value
	}
	return string(runes[:6]) + "..." + string(runes[len(runes)-4:])
}

func relayXOpsDigestContent(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\r", " ")
	runes := []rune(content)
	if len(runes) <= 120 {
		return content
	}
	return string(runes[:120]) + "..."
}

func relayXOpsLogTypeName(logType int) string {
	switch logType {
	case model.LogTypeManage:
		return "manage"
	case model.LogTypeSystem:
		return "system"
	case model.LogTypeError:
		return "error"
	case model.LogTypeLogin:
		return "login"
	default:
		return "unknown"
	}
}

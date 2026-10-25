package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// isZhipuCodingPlanBase reports whether a ZhipuV4 channel targets a GLM coding
// plan special base, whose subscription usage is queryable upstream.
func isZhipuCodingPlanBase(baseURL string) bool {
	return baseURL == "glm-coding-plan" || baseURL == "glm-coding-plan-international"
}

// isOpenCodeBase reports whether an OpenAI channel points at an OpenCode
// gateway (OpenCode Go / Zen).
func isOpenCodeBase(baseURL string) bool {
	return strings.Contains(strings.ToLower(baseURL), "opencode.ai")
}

// GetChannelCodingPlanUsage fetches the subscription usage windows of coding
// plan channels (GLM Coding Plan, OpenCode Go) straight from the provider.
func GetChannelCodingPlanUsage(c *gin.Context) {
	channelId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, fmt.Errorf("invalid channel id: %w", err))
		return
	}

	ch, err := model.GetChannelById(channelId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if ch == nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "channel not found"})
		return
	}
	if ch.ChannelInfo.IsMultiKey {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "multi-key channel is not supported"})
		return
	}

	client, err := service.GetHttpClientWithProxy(ch.GetSetting().Proxy)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	var statusCode int
	var body []byte
	baseURL := ch.GetBaseURL()
	switch {
	case ch.Type == constant.ChannelTypeZhipu_v4 && isZhipuCodingPlanBase(baseURL):
		statusCode, body, err = service.FetchZhipuCodingPlanUsage(ctx, client, baseURL, ch.Key)
	case ch.Type == constant.ChannelTypeOpenAI && isOpenCodeBase(baseURL):
		statusCode, body, err = service.FetchOpenCodeUsage(ctx, client, baseURL, ch.Key)
	default:
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该渠道不支持订阅额度查询"})
		return
	}
	if err != nil {
		common.SysError("failed to fetch coding plan usage: " + err.Error())
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "获取用量信息失败，请稍后重试"})
		return
	}

	var payload any
	if common.Unmarshal(body, &payload) != nil {
		payload = string(body)
	}

	ok := statusCode >= 200 && statusCode < 300
	resp := gin.H{
		"success":         ok,
		"message":         "",
		"upstream_status": statusCode,
		"data":            payload,
	}
	if !ok {
		resp["message"] = fmt.Sprintf("upstream status: %d", statusCode)
	}
	c.JSON(http.StatusOK, resp)
}

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Coding plan subscription usage windows are exposed on account-portal
// endpoints that differ per provider:
//   - Zhipu (GLM Coding Plan): open.bigmodel.cn / api.z.ai, and the quota
//     endpoint requires the raw API key in Authorization (no Bearer prefix).
//   - OpenCode Go: {base}/v1/usage with a Bearer key.
const (
	zhipuCodingPlanUsagePath = "/api/monitor/usage/quota/limit"
	zhipuCodingPlanHostCN    = "https://open.bigmodel.cn"
	zhipuCodingPlanHostIntl  = "https://api.z.ai"
)

// ZhipuCodingPlanUsageURL routes the quota host by coding plan region: the
// international plan key and z.ai hosts use the international portal,
// everything else falls back to the domestic bigmodel.cn portal.
func ZhipuCodingPlanUsageURL(baseURL string) string {
	u := strings.ToLower(baseURL)
	if u == "glm-coding-plan-international" || strings.Contains(u, "z.ai") {
		return zhipuCodingPlanHostIntl + zhipuCodingPlanUsagePath
	}
	return zhipuCodingPlanHostCN + zhipuCodingPlanUsagePath
}

// OpenCodeUsageURL accepts the channel base URL with or without a trailing
// /v1 and always yields {base}/v1/usage.
func OpenCodeUsageURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	base = strings.TrimSuffix(base, "/v1")
	return base + "/v1/usage"
}

func FetchZhipuCodingPlanUsage(ctx context.Context, client *http.Client, baseURL string, apiKey string) (statusCode int, body []byte, err error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return 0, nil, fmt.Errorf("empty apiKey")
	}
	return fetchCodingPlanUsage(ctx, client, ZhipuCodingPlanUsageURL(baseURL), func(req *http.Request) {
		req.Header.Set("Authorization", key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept-Language", "en-US,en")
	})
}

func FetchOpenCodeUsage(ctx context.Context, client *http.Client, baseURL string, apiKey string) (statusCode int, body []byte, err error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return 0, nil, fmt.Errorf("empty apiKey")
	}
	return fetchCodingPlanUsage(ctx, client, OpenCodeUsageURL(baseURL), func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+key)
	})
}

func fetchCodingPlanUsage(ctx context.Context, client *http.Client, targetURL string, authorize func(*http.Request)) (statusCode int, body []byte, err error) {
	if client == nil {
		return 0, nil, fmt.Errorf("nil http client")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	authorize(req)

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

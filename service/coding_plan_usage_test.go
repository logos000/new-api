package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZhipuCodingPlanUsageURL(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{"domestic plan key", "glm-coding-plan", "https://open.bigmodel.cn/api/monitor/usage/quota/limit"},
		{"international plan key", "glm-coding-plan-international", "https://api.z.ai/api/monitor/usage/quota/limit"},
		{"z.ai inference base", "https://api.z.ai/api/coding/paas/v4", "https://api.z.ai/api/monitor/usage/quota/limit"},
		{"bigmodel inference base", "https://open.bigmodel.cn/api/coding/paas/v4", "https://open.bigmodel.cn/api/monitor/usage/quota/limit"},
		{"unknown host falls back to domestic", "https://custom.example.com/api/paas/v4", "https://open.bigmodel.cn/api/monitor/usage/quota/limit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ZhipuCodingPlanUsageURL(tt.base))
		})
	}
}

func TestOpenCodeUsageURL(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{"base without v1", "https://opencode.ai/zen/go", "https://opencode.ai/zen/go/v1/usage"},
		{"base with v1", "https://opencode.ai/zen/go/v1", "https://opencode.ai/zen/go/v1/usage"},
		{"trailing slash", "https://opencode.ai/zen/go/", "https://opencode.ai/zen/go/v1/usage"},
		{"trailing v1 slash", "https://opencode.ai/zen/go/v1/", "https://opencode.ai/zen/go/v1/usage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, OpenCodeUsageURL(tt.base))
		})
	}
}

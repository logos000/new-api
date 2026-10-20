package oairesponses

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requestWithInput(t *testing.T, input string) *dto.OpenAIResponsesRequest {
	t.Helper()
	require.True(t, json.Valid([]byte(input)), "input must be valid JSON")
	return &dto.OpenAIResponsesRequest{
		Model: "gemini-test",
		Input: []byte(input),
	}
}

func contentRoles(got *dto.GeminiChatRequest) []string {
	roles := make([]string, 0, len(got.Contents))
	for _, content := range got.Contents {
		roles = append(roles, content.Role)
	}
	return roles
}

func TestResponsesRequestToGeminiChatMergesCodexHistoryTurns(t *testing.T) {
	// Codex replay histories contain consecutive assistant messages and an
	// assistant message between function_call and function_call_output. Gemini
	// rejects consecutive model contents, so they must merge into one turn.
	got, err := OpenAIResponsesRequestToGeminiChat(context.Background(), requestWithInput(t, `[
		{"type":"message","role":"developer","content":[{"type":"input_text","text":"system rules"}]},
		{"type":"message","role":"user","content":[{"type":"input_text","text":"run it"}]},
		{"type":"message","role":"assistant","content":[{"type":"output_text","text":"on it"}]},
		{"type":"message","role":"assistant","content":[{"type":"output_text","text":"checking"}]},
		{"type":"function_call","name":"exec_command","call_id":"call_1","arguments":"{\"command\":\"echo a\"}"},
		{"type":"function_call","name":"view_image","call_id":"call_2","arguments":"{\"path\":\"a.png\"}"},
		{"type":"message","role":"assistant","content":[{"type":"output_text","text":"waiting for output"}]},
		{"type":"function_call_output","call_id":"call_1","output":"a"},
		{"type":"function_call_output","call_id":"call_2","output":"ok"},
		{"type":"message","role":"user","content":[{"type":"input_text","text":"and now?"}]}
	]`), &convmeta.Values{})
	require.NoError(t, err)

	// Everything folds into three alternating turns: the assistant texts share
	// the model turn with the function calls (call parts first), and the final
	// user message merges into the function-response turn.
	roles := contentRoles(got)
	assert.Equal(t, []string{"user", "model", "user"}, roles)

	modelTurn := got.Contents[1]
	require.Len(t, modelTurn.Parts, 5)
	assert.Equal(t, "exec_command", modelTurn.Parts[0].FunctionCall.FunctionName)
	assert.Equal(t, "view_image", modelTurn.Parts[1].FunctionCall.FunctionName)
	assert.Equal(t, "on it", modelTurn.Parts[2].Text)
	assert.Equal(t, "checking", modelTurn.Parts[3].Text)
	assert.Equal(t, "waiting for output", modelTurn.Parts[4].Text)

	userTurn := got.Contents[2]
	require.Len(t, userTurn.Parts, 3)
	assert.NotNil(t, userTurn.Parts[0].FunctionResponse)
	assert.NotNil(t, userTurn.Parts[1].FunctionResponse)
	assert.Equal(t, "and now?", userTurn.Parts[2].Text)

	require.NotNil(t, got.SystemInstructions)
	require.Len(t, got.SystemInstructions.Parts, 1)
	assert.Equal(t, "system rules", got.SystemInstructions.Parts[0].Text)
}

func TestResponsesRequestToGeminiChatAppendsContinuationForTrailingAssistant(t *testing.T) {
	// Gemini rejects requests ending with a model turn; a history that ends
	// with an assistant message must gain a minimal continuation user turn.
	got, err := OpenAIResponsesRequestToGeminiChat(context.Background(), requestWithInput(t, `[
		{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]},
		{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}
	]`), &convmeta.Values{})
	require.NoError(t, err)

	roles := contentRoles(got)
	assert.Equal(t, []string{"user", "model", "user"}, roles)
	last := got.Contents[len(got.Contents)-1]
	require.Len(t, last.Parts, 1)
	assert.Equal(t, "(continue)", last.Parts[0].Text)
}

func TestResponsesRequestToGeminiChatSimpleTurnUnchanged(t *testing.T) {
	got, err := OpenAIResponsesRequestToGeminiChat(context.Background(), requestWithInput(t, `[
		{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}
	]`), &convmeta.Values{})
	require.NoError(t, err)

	assert.Equal(t, []string{"user"}, contentRoles(got))
	require.Len(t, got.Contents, 1)
	require.Len(t, got.Contents[0].Parts, 1)
	assert.Equal(t, "hello", got.Contents[0].Parts[0].Text)
}

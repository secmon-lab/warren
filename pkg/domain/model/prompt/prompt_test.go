package prompt_test

import (
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
)

func TestGenerateWithStruct_MapInput(t *testing.T) {
	ctx := t.Context()
	tmpl := "Hello {{ .name }}, you are {{ .age }}."
	result, err := prompt.GenerateWithStruct(ctx, tmpl, map[string]any{
		"name": "Alice",
		"age":  30,
	})
	gt.NoError(t, err)
	gt.V(t, result).Equal("Hello Alice, you are 30.")
}

type testData struct {
	Name string
	Age  int
}

func TestGenerateWithStruct_StructInput(t *testing.T) {
	ctx := t.Context()
	tmpl := "Hello {{ .Name }}, you are {{ .Age }}."
	result, err := prompt.GenerateWithStruct(ctx, tmpl, testData{
		Name: "Bob",
		Age:  25,
	})
	gt.NoError(t, err)
	gt.V(t, result).Equal("Hello Bob, you are 25.")
}

type nestedData struct {
	User    userData
	Message string
}

type userData struct {
	Name string
	Role string
}

func TestGenerateWithStruct_NestedStruct(t *testing.T) {
	ctx := t.Context()
	tmpl := "{{ .User.Name }} ({{ .User.Role }}): {{ .Message }}"
	result, err := prompt.GenerateWithStruct(ctx, tmpl, nestedData{
		User:    userData{Name: "Charlie", Role: "admin"},
		Message: "hello world",
	})
	gt.NoError(t, err)
	gt.V(t, result).Equal("Charlie (admin): hello world")
}

func TestGenerateWithStruct_ConditionalStruct(t *testing.T) {
	ctx := t.Context()
	tmpl := `{{ if .Name }}Name: {{ .Name }}{{ end }}{{ if .Role }} Role: {{ .Role }}{{ end }}`

	result, err := prompt.GenerateWithStruct(ctx, tmpl, userData{Name: "Dave", Role: ""})
	gt.NoError(t, err)
	gt.True(t, strings.Contains(result, "Name: Dave"))
	gt.True(t, !strings.Contains(result, "Role:"))
}

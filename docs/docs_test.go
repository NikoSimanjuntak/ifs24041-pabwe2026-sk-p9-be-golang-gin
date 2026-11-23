package docs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggo/swag"
)

func TestSpecValid(t *testing.T) {
	var spec map[string]any
	require.NoError(t, json.Unmarshal([]byte(APIJSON), &spec))
	paths := spec["paths"].(map[string]any)
	for _, p := range []string{"/api/public/posts", "/api/auth/login", "/api/posts/ai-generate", "/api/admin/users"} {
		assert.Contains(t, paths, p)
	}
}

func TestRegistered(t *testing.T) {
	doc, err := swag.ReadDoc("swagger")
	require.NoError(t, err)
	assert.Contains(t, doc, "Delcom Blog API")
}

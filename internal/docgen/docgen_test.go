package docgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtract_MathModule(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "docgen", "math.fn")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	doc, err := Extract(data, path)
	require.NoError(t, err)
	require.Len(t, doc.Symbols, 2)
	assert.Equal(t, "add", doc.Symbols[0].Name)
	assert.Equal(t, "Add two integers", doc.Symbols[0].Summary)
	assert.Equal(t, "first summand", doc.Symbols[0].Args["a"])
	assert.Equal(t, "sum of a and b", doc.Symbols[0].Returns)
	md := RenderMarkdown(doc)
	assert.Contains(t, md, "## Symbols")
	assert.Contains(t, md, "pub fn add")
}

func TestGenerateAll(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "docgen")
	docs, err := GenerateAll(dir, false)
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

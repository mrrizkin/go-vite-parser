package goviteparser

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkNewVite(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewVite()
	}
}

func BenchmarkUseCspNonce(b *testing.B) {
	vite := NewVite()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		vite.UseCspNonce("")
	}
}

func BenchmarkParseAttributes(b *testing.B) {
	vite := NewVite()
	attributes := map[string]any{
		"src":         "test.js",
		"type":        "module",
		"defer":       true,
		"disabled":    false,
		"hidden":      nil,
		"data-value":  123,
		"crossorigin": "anonymous",
		"integrity":   "sha384-abc123def456",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vite.parseAttributes(attributes)
	}
}

func BenchmarkIsCssPath(b *testing.B) {
	vite := NewVite()
	paths := []string{
		"style.css",
		"style.scss",
		"script.js",
		"image.png",
		"style.css?v=123",
		"assets/main-abc123.css",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			_ = vite.isCSSPath(path)
		}
	}
}

func BenchmarkAssetPath(b *testing.B) {
	vite := NewVite()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vite.assetPath("assets/main-abc123.js", false)
	}
}

func BenchmarkManifestParsing(b *testing.B) {
	tmpDir := b.TempDir()
	buildDir := filepath.Join(tmpDir, "build")
	os.MkdirAll(buildDir, 0755)

	manifestPath := filepath.Join(buildDir, "manifest.json")
	os.WriteFile(manifestPath, []byte(testManifest), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	vite := NewVite()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear cache to force re-parsing
		vite.manifests = make(map[string]map[string]any)
		_, _ = vite.getManifest("build")
	}
}

func BenchmarkInvokeProduction(b *testing.B) {
	tmpDir := b.TempDir()
	buildDir := filepath.Join(tmpDir, "build")
	os.MkdirAll(buildDir, 0755)

	manifestPath := filepath.Join(buildDir, "manifest.json")
	os.WriteFile(manifestPath, []byte(testManifest), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	vite := NewVite()
	vite.WithEntryPoints([]string{"main.js"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = vite.Invoke([]string{"main.js"}, "build")
	}
}

func BenchmarkInvokeHot(b *testing.B) {
	tmpDir := b.TempDir()

	hotPath := filepath.Join(tmpDir, "hot")
	os.WriteFile(hotPath, []byte(testHotContent), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	vite := NewVite()
	vite.WithEntryPoints([]string{"main.js"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = vite.Invoke([]string{"main.js"}, "build")
	}
}

func BenchmarkUniqueStrings(b *testing.B) {
	vite := NewVite()
	slice := []string{
		"main.js", "app.js", "vendor.js", "main.js", "utils.js",
		"app.js", "style.css", "vendor.js", "components.js", "main.js",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vite.uniqueStrings(slice)
	}
}

func BenchmarkMakeTagForChunk(b *testing.B) {
	vite := NewVite()
	vite.UseCspNonce("test-nonce")

	chunk := map[string]any{
		"file": "assets/main-abc123.js",
		"src":  "main.js",
	}

	manifest := map[string]any{
		"main.js": chunk,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vite.makeTagForChunk("main.js", "/build/assets/main-abc123.js", chunk, manifest)
	}
}

func BenchmarkMakePreloadTagForChunk(b *testing.B) {
	vite := NewVite()
	vite.UseCspNonce("test-nonce")

	chunk := map[string]any{
		"file": "assets/main-abc123.js",
		"src":  "main.js",
	}

	manifest := map[string]any{
		"main.js": chunk,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vite.makePreloadTagForChunk("main.js", "/build/assets/main-abc123.js", chunk, manifest)
	}
}

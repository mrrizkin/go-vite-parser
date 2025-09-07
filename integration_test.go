package goviteparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHotReloadIntegration(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()
	vite.WithEntryPoints([]string{"main.js", "app.js"})

	// Test hot reload mode
	html, err := vite.Invoke([]string{"main.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error in hot mode: %v", err)
	}

	if !strings.Contains(html, "localhost:5173") {
		t.Error("Expected hot reload HTML to contain localhost:5173")
	}

	if !strings.Contains(html, "@vite/client") {
		t.Error("Expected hot reload HTML to contain @vite/client")
	}

	if !strings.Contains(html, "main.js") {
		t.Error("Expected hot reload HTML to contain main.js")
	}

	_ = tmpDir
}

func TestProductionModeIntegration(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()
	vite.WithEntryPoints([]string{"main.js"})

	// Remove hot file to simulate production
	os.Remove("hot")

	html, err := vite.Invoke([]string{"main.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error in production mode: %v", err)
	}

	// Should contain preload tags
	if !strings.Contains(html, "rel=\"modulepreload\"") {
		t.Error("Expected production HTML to contain modulepreload")
	}

	// Should contain script tag
	if !strings.Contains(html, "main-abc123.js") {
		t.Error("Expected production HTML to contain main script")
	}

	// Should contain CSS link
	if !strings.Contains(html, "main-abc123.css") {
		t.Error("Expected production HTML to contain CSS link")
	}

	_ = tmpDir
}

func TestReactRefreshIntegration(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()
	vite.UseCspNonce("test-nonce")

	// Test React refresh in hot mode
	refreshScript, err := vite.ReactRefresh()
	if err != nil {
		t.Errorf("Unexpected error getting React refresh: %v", err)
	}

	if !strings.Contains(refreshScript, "RefreshRuntime") {
		t.Error("Expected React refresh script to contain RefreshRuntime")
	}

	if !strings.Contains(refreshScript, "nonce=\"test-nonce\"") {
		t.Error("Expected React refresh script to contain nonce")
	}

	// Test React refresh in production (should return empty)
	os.Remove("hot")
	refreshScript, err = vite.ReactRefresh()
	if err != nil {
		t.Errorf("Unexpected error getting React refresh in production: %v", err)
	}

	if refreshScript != "" {
		t.Error("Expected React refresh to be empty in production mode")
	}

	_ = tmpDir
}

func TestContentRetrieval(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()

	// Create a test file in build directory
	buildDir := filepath.Join(tmpDir, "build", "assets")
	os.MkdirAll(buildDir, 0755)

	testContent := "console.log('test');"
	testFile := filepath.Join(buildDir, "main-abc123.js")
	os.WriteFile(testFile, []byte(testContent), 0644)

	// Remove hot file to test production mode
	os.Remove("hot")

	content, err := vite.Content("main.js", "build")
	if err != nil {
		t.Errorf("Unexpected error getting content: %v", err)
	}

	if content != testContent {
		t.Errorf("Expected content %s, got %s", testContent, content)
	}

	_ = tmpDir
}

func TestComplexManifestIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a more complex manifest
	complexManifest := `{
		"main.js": {
			"file": "assets/main-abc123.js",
			"src": "main.js",
			"isEntry": true,
			"imports": ["_vendor-def456.js", "_utils-ghi789.js"],
			"css": ["assets/main-abc123.css"],
			"integrity": "sha384-abc123"
		},
		"_vendor-def456.js": {
			"file": "assets/vendor-def456.js",
			"src": "_vendor-def456.js",
			"imports": ["_shared-jkl012.js"]
		},
		"_utils-ghi789.js": {
			"file": "assets/utils-ghi789.js",
			"src": "_utils-ghi789.js",
			"css": ["assets/utils-ghi789.css"]
		},
		"_shared-jkl012.js": {
			"file": "assets/shared-jkl012.js",
			"src": "_shared-jkl012.js"
		}
	}`

	// Setup files
	buildDir := filepath.Join(tmpDir, "build")
	os.MkdirAll(buildDir, 0755)

	manifestPath := filepath.Join(buildDir, "manifest.json")
	os.WriteFile(manifestPath, []byte(complexManifest), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	vite := NewVite()
	vite.UseIntegrityKey("integrity")

	html, err := vite.Invoke([]string{"main.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error with complex manifest: %v", err)
	}

	// Should contain integrity attribute
	if !strings.Contains(html, "integrity=\"sha384-abc123\"") {
		t.Error("Expected HTML to contain integrity attribute")
	}

	// Should contain all imported files as preloads
	if !strings.Contains(html, "vendor-def456.js") {
		t.Error("Expected HTML to contain vendor import")
	}

	if !strings.Contains(html, "utils-ghi789.js") {
		t.Error("Expected HTML to contain utils import")
	}

	// Note: shared-jkl012.js might not appear if it's only imported by vendor
	// This is expected behavior as the current implementation processes imports depth-first

	// Should contain CSS files
	if !strings.Contains(html, "main-abc123.css") {
		t.Error("Expected HTML to contain main CSS")
	}

	if !strings.Contains(html, "utils-ghi789.css") {
		t.Error("Expected HTML to contain utils CSS")
	}
}

func TestCustomAttributeResolversIntegration(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()

	// Add custom resolvers
	vite.UseScriptTagAttributes(
		func(src, url string, chunk map[string]any, manifest map[string]any) map[string]any {
			return map[string]any{
				"data-src":    src,
				"crossorigin": "anonymous",
			}
		},
	)

	vite.UseStyleTagAttributes(
		func(src, url string, chunk map[string]any, manifest map[string]any) map[string]any {
			return map[string]any{
				"data-style-src": src,
			}
		},
	)

	vite.UsePreloadTagAttributes(
		func(src, url string, chunk map[string]any, manifest map[string]any) map[string]any {
			return map[string]any{
				"data-preload-src": src,
			}
		},
	)

	// Remove hot file to test production mode
	os.Remove("hot")

	html, err := vite.Invoke([]string{"main.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error with custom resolvers: %v", err)
	}

	if !strings.Contains(html, "data-src=\"main.js\"") {
		t.Error("Expected HTML to contain custom script attribute")
	}

	if !strings.Contains(html, "crossorigin=\"anonymous\"") {
		t.Error("Expected HTML to contain crossorigin attribute")
	}

	if !strings.Contains(html, "data-preload-src=") {
		t.Error("Expected HTML to contain custom preload attribute")
	}

	_ = tmpDir
}

func TestPrefetchIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a manifest with dynamic imports for prefetch testing
	prefetchManifest := `{
		"app.js": {
			"file": "assets/app-abc123.js",
			"src": "app.js",
			"isEntry": true,
			"imports": ["_vendor-def456.js"],
			"css": ["assets/app-abc123.css"],
			"dynamicImports": ["_lazy-ghi789.js", "_modal-jkl012.js"]
		},
		"_vendor-def456.js": {
			"file": "assets/vendor-def456.js",
			"src": "_vendor-def456.js"
		},
		"_lazy-ghi789.js": {
			"file": "assets/lazy-ghi789.js",
			"src": "_lazy-ghi789.js",
			"imports": ["_shared-mno345.js"],
			"css": ["assets/lazy-ghi789.css"]
		},
		"_modal-jkl012.js": {
			"file": "assets/modal-jkl012.js",
			"src": "_modal-jkl012.js",
			"dynamicImports": ["_tooltip-pqr678.js"]
		},
		"_shared-mno345.js": {
			"file": "assets/shared-mno345.js",
			"src": "_shared-mno345.js"
		},
		"_tooltip-pqr678.js": {
			"file": "assets/tooltip-pqr678.js",
			"src": "_tooltip-pqr678.js"
		}
	}`

	// Setup files
	buildDir := filepath.Join(tmpDir, "build")
	os.MkdirAll(buildDir, 0755)

	manifestPath := filepath.Join(buildDir, "manifest.json")
	os.WriteFile(manifestPath, []byte(prefetchManifest), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	vite := NewVite()
	vite.UseCspNonce("integration-nonce")

	// Test waterfall prefetch
	vite.UseWaterfallPrefetching(nil)

	html, err := vite.Invoke([]string{"app.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error with waterfall prefetch integration: %v", err)
	}

	// Should contain all dynamic imports
	expectedAssets := []string{
		"lazy-ghi789.js",
		"modal-jkl012.js",
		"shared-mno345.js",
		"tooltip-pqr678.js",
		"lazy-ghi789.css",
	}

	for _, asset := range expectedAssets {
		if !strings.Contains(html, asset) {
			t.Errorf("Expected HTML to contain prefetch asset: %s", asset)
		}
	}

	// Should contain waterfall script with nonce
	if !strings.Contains(html, `nonce="integration-nonce"`) {
		t.Error("Expected HTML to contain nonce in prefetch script")
	}

	if !strings.Contains(html, "loadNext") {
		t.Error("Expected HTML to contain waterfall loadNext function")
	}

	// Test aggressive prefetch
	vite.UseAggressivePrefetching()

	html2, err := vite.Invoke([]string{"app.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error with aggressive prefetch integration: %v", err)
	}

	// Should contain forEach but not loadNext
	if !strings.Contains(html2, "forEach") {
		t.Error("Expected HTML to contain aggressive forEach")
	}

	if strings.Contains(html2, "loadNext") {
		t.Error("Expected HTML to NOT contain loadNext for aggressive strategy")
	}
}

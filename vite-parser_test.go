package goviteparser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test fixtures
const testManifest = `{
	"main.js": {
		"file": "assets/main-abc123.js",
		"src": "main.js",
		"isEntry": true,
		"imports": ["_vendor-def456.js"],
		"css": ["assets/main-abc123.css"],
		"dynamicImports": ["_dynamic-xyz789.js"]
	},
	"_vendor-def456.js": {
		"file": "assets/vendor-def456.js",
		"src": "_vendor-def456.js"
	},
	"_dynamic-xyz789.js": {
		"file": "assets/dynamic-xyz789.js",
		"src": "_dynamic-xyz789.js",
		"imports": ["_shared-abc123.js"],
		"css": ["assets/dynamic-xyz789.css"]
	},
	"_shared-abc123.js": {
		"file": "assets/shared-abc123.js",
		"src": "_shared-abc123.js"
	},
	"style.css": {
		"file": "assets/style-ghi789.css",
		"src": "style.css"
	}
}`

const testHotContent = "http://localhost:5173"

func setupTestFiles(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()

	// Create build directory
	buildDir := filepath.Join(tmpDir, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create manifest file
	manifestPath := filepath.Join(buildDir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(testManifest), 0644); err != nil {
		t.Fatal(err)
	}

	// Create hot file
	hotPath := filepath.Join(tmpDir, "hot")
	if err := os.WriteFile(hotPath, []byte(testHotContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)

	cleanup := func() {
		os.Chdir(oldWd)
	}

	return tmpDir, cleanup
}

func TestNewVite(t *testing.T) {
	vite := NewVite()

	if vite.config.BuildDirectory != "build" {
		t.Errorf("Expected BuildDirectory to be 'build', got %s", vite.config.BuildDirectory)
	}

	if vite.config.ManifestFilename != "manifest.json" {
		t.Errorf(
			"Expected ManifestFilename to be 'manifest.json', got %s",
			vite.config.ManifestFilename,
		)
	}

	if vite.config.HotFile != "hot" {
		t.Errorf("Expected HotFile to be 'hot', got %s", vite.config.HotFile)
	}
}

func TestUseCspNonce(t *testing.T) {
	vite := NewVite()

	// Test with provided nonce
	nonce := vite.UseCspNonce("test-nonce")
	if nonce != "test-nonce" {
		t.Errorf("Expected nonce to be 'test-nonce', got %s", nonce)
	}

	if vite.CspNonce() != "test-nonce" {
		t.Errorf("Expected stored nonce to be 'test-nonce', got %s", vite.CspNonce())
	}

	// Test with empty nonce (should generate)
	generatedNonce := vite.UseCspNonce("")
	if len(generatedNonce) == 0 {
		t.Error("Expected generated nonce to be non-empty")
	}

	if len(generatedNonce) != 60 { // 30 bytes * 2 (hex encoding)
		t.Errorf("Expected generated nonce length to be 60, got %d", len(generatedNonce))
	}
}

func TestUseIntegrityKey(t *testing.T) {
	vite := NewVite()
	result := vite.UseIntegrityKey("custom-integrity")

	if vite.integrityKey != "custom-integrity" {
		t.Errorf("Expected integrityKey to be 'custom-integrity', got %s", vite.integrityKey)
	}

	if result != vite {
		t.Error("Expected UseIntegrityKey to return the same Vite instance")
	}
}

func TestWithEntryPoints(t *testing.T) {
	vite := NewVite()
	entryPoints := []string{"main.js", "app.js"}

	result := vite.WithEntryPoints(entryPoints)

	if len(vite.config.EntryPoints) != 2 {
		t.Errorf("Expected 2 entry points, got %d", len(vite.config.EntryPoints))
	}

	if vite.config.EntryPoints[0] != "main.js" || vite.config.EntryPoints[1] != "app.js" {
		t.Errorf("Entry points not set correctly: %v", vite.config.EntryPoints)
	}

	if result != vite {
		t.Error("Expected WithEntryPoints to return the same Vite instance")
	}
}

func TestMergeEntryPoints(t *testing.T) {
	vite := NewVite()
	vite.WithEntryPoints([]string{"main.js", "app.js"})

	vite.MergeEntryPoints([]string{"vendor.js", "main.js"}) // main.js should be deduplicated

	expected := []string{"main.js", "app.js", "vendor.js"}
	if len(vite.config.EntryPoints) != 3 {
		t.Errorf("Expected 3 unique entry points, got %d", len(vite.config.EntryPoints))
	}

	for i, ep := range expected {
		if vite.config.EntryPoints[i] != ep {
			t.Errorf("Expected entry point %d to be %s, got %s", i, ep, vite.config.EntryPoints[i])
		}
	}
}

func TestConfigurationMethods(t *testing.T) {
	vite := NewVite()

	// Test UseManifestFilename
	vite.UseManifestFilename("custom-manifest.json")
	if vite.config.ManifestFilename != "custom-manifest.json" {
		t.Errorf(
			"Expected ManifestFilename to be 'custom-manifest.json', got %s",
			vite.config.ManifestFilename,
		)
	}

	// Test UseHotFile
	vite.UseHotFile("custom-hot")
	if vite.config.HotFile != "custom-hot" {
		t.Errorf("Expected HotFile to be 'custom-hot', got %s", vite.config.HotFile)
	}

	// Test UseBuildDirectory
	vite.UseBuildDirectory("dist")
	if vite.config.BuildDirectory != "dist" {
		t.Errorf("Expected BuildDirectory to be 'dist', got %s", vite.config.BuildDirectory)
	}

	// Test HotFile method
	if vite.HotFile() != "custom-hot" {
		t.Errorf("Expected HotFile() to return 'custom-hot', got %s", vite.HotFile())
	}
}

func TestCreateAssetPathsUsing(t *testing.T) {
	vite := NewVite()

	customResolver := func(path string, secure bool) string {
		return "https://cdn.example.com/" + path
	}

	vite.CreateAssetPathsUsing(customResolver)

	result := vite.assetPath("test.js", false)
	expected := "https://cdn.example.com/test.js"

	if result != expected {
		t.Errorf("Expected custom asset path %s, got %s", expected, result)
	}
}

func TestAttributeResolvers(t *testing.T) {
	vite := NewVite()

	scriptResolver := func(src, url string, chunk map[string]any, manifest map[string]any) map[string]any {
		return map[string]any{"data-script": "true"}
	}

	styleResolver := func(src, url string, chunk map[string]any, manifest map[string]any) map[string]any {
		return map[string]any{"data-style": "true"}
	}

	preloadResolver := func(src, url string, chunk map[string]any, manifest map[string]any) map[string]any {
		return map[string]any{"data-preload": "true"}
	}

	vite.UseScriptTagAttributes(scriptResolver)
	vite.UseStyleTagAttributes(styleResolver)
	vite.UsePreloadTagAttributes(preloadResolver)

	if len(vite.scriptTagAttributeResolvers) != 1 {
		t.Error("Expected 1 script tag attribute resolver")
	}

	if len(vite.styleTagAttributeResolvers) != 1 {
		t.Error("Expected 1 style tag attribute resolver")
	}

	if len(vite.preloadTagAttributeResolvers) != 1 {
		t.Error("Expected 1 preload tag attribute resolver")
	}
}

func TestPrefetchStrategies(t *testing.T) {
	vite := NewVite()

	// Test UseWaterfallPrefetching
	concurrency := 5
	vite.UseWaterfallPrefetching(&concurrency)

	if vite.prefetchStrategy != PrefetchWaterfall {
		t.Errorf(
			"Expected prefetch strategy to be %s, got %s",
			PrefetchWaterfall,
			vite.prefetchStrategy,
		)
	}

	if vite.prefetchConcurrently != 5 {
		t.Errorf("Expected prefetch concurrency to be 5, got %d", vite.prefetchConcurrently)
	}

	// Test UseAggressivePrefetching
	vite.UseAggressivePrefetching()

	if vite.prefetchStrategy != PrefetchAggressive {
		t.Errorf(
			"Expected prefetch strategy to be %s, got %s",
			PrefetchAggressive,
			vite.prefetchStrategy,
		)
	}

	// Test Prefetch method
	vite.Prefetch(&concurrency, "domcontentloaded")

	if vite.prefetchEvent != "domcontentloaded" {
		t.Errorf("Expected prefetch event to be 'domcontentloaded', got %s", vite.prefetchEvent)
	}
}

func TestIsRunningHot(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()

	// Test when hot file exists
	if !vite.IsRunningHot() {
		t.Error("Expected IsRunningHot to return true when hot file exists")
	}

	// Test when hot file doesn't exist
	os.Remove("hot")
	if vite.IsRunningHot() {
		t.Error("Expected IsRunningHot to return false when hot file doesn't exist")
	}

	_ = tmpDir
}

func TestManifestHash(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()

	// Remove hot file to test production mode
	os.Remove("hot")

	// Test with manifest file
	hash, err := vite.ManifestHash("build")
	if err != nil {
		t.Errorf("Unexpected error getting manifest hash: %v", err)
	}

	if len(hash) != 32 { // MD5 hash length
		t.Errorf("Expected hash length to be 32, got %d", len(hash))
	}

	// Test when running hot (should return empty string)
	// Create hot file again
	os.WriteFile("hot", []byte(testHotContent), 0644)

	hash2, err := vite.ManifestHash("build")
	if err != nil {
		t.Errorf("Unexpected error getting manifest hash when hot: %v", err)
	}

	if hash2 != "" {
		t.Error("Expected empty hash when running hot")
	}

	_ = tmpDir
}

func TestAsset(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()

	// Remove hot file to test production mode
	os.Remove("hot")

	assetURL, err := vite.Asset("main.js", "build")
	if err != nil {
		t.Errorf("Unexpected error getting asset: %v", err)
	}

	expected := "/build/assets/main-abc123.js"
	if assetURL != expected {
		t.Errorf("Expected asset URL %s, got %s", expected, assetURL)
	}

	// Test with non-existent asset
	_, err = vite.Asset("nonexistent.js", "build")
	if err == nil {
		t.Error("Expected error for non-existent asset")
	}

	_ = tmpDir
}

func TestFlushAndPreloadedAssets(t *testing.T) {
	vite := NewVite()

	// Add some preloaded assets
	vite.preloadedAssets["test.js"] = map[string]any{"rel": "preload"}

	assets := vite.PreloadedAssets()
	if len(assets) != 1 {
		t.Errorf("Expected 1 preloaded asset, got %d", len(assets))
	}

	vite.Flush()

	assets = vite.PreloadedAssets()
	if len(assets) != 0 {
		t.Errorf("Expected 0 preloaded assets after flush, got %d", len(assets))
	}
}

func TestIsCssPath(t *testing.T) {
	vite := NewVite()

	testCases := []struct {
		path     string
		expected bool
	}{
		{"style.css", true},
		{"style.scss", true},
		{"style.less", true},
		{"style.sass", true},
		{"style.styl", true},
		{"style.stylus", true},
		{"style.pcss", true},
		{"style.postcss", true},
		{"style.css?v=123", true},
		{"script.js", false},
		{"image.png", false},
		{"", false},
	}

	for _, tc := range testCases {
		result := vite.isCSSPath(tc.path)
		if result != tc.expected {
			t.Errorf("isCssPath(%s) = %v, expected %v", tc.path, result, tc.expected)
		}
	}
}

func TestParseAttributes(t *testing.T) {
	vite := NewVite()

	attributes := map[string]any{
		"src":        "test.js",
		"type":       "module",
		"defer":      true,
		"disabled":   false,
		"hidden":     nil,
		"data-value": 123,
	}

	result := vite.parseAttributes(attributes)

	// Should contain src, type, defer, and data-value
	// Should not contain disabled or hidden
	expectedCount := 4
	if len(result) != expectedCount {
		t.Errorf("Expected %d attributes, got %d: %v", expectedCount, len(result), result)
	}

	// Check for specific attributes
	found := make(map[string]bool)
	for _, attr := range result {
		if strings.Contains(attr, "src=") {
			found["src"] = true
		}
		if strings.Contains(attr, "type=") {
			found["type"] = true
		}
		if attr == "defer" {
			found["defer"] = true
		}
		if strings.Contains(attr, "data-value=") {
			found["data-value"] = true
		}
	}

	if !found["src"] || !found["type"] || !found["defer"] || !found["data-value"] {
		t.Errorf("Missing expected attributes in result: %v", result)
	}
}

func TestToHTML(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()
	vite.WithEntryPoints([]string{"main.js"})

	// Remove hot file to test production mode
	os.Remove("hot")

	html, err := vite.ToHTML()
	if err != nil {
		t.Errorf("Unexpected error generating HTML: %v", err)
	}

	if !strings.Contains(html, "main-abc123.js") {
		t.Error("Expected HTML to contain main script reference")
	}

	if !strings.Contains(html, "main-abc123.css") {
		t.Error("Expected HTML to contain main CSS reference")
	}

	_ = tmpDir
}

func TestPrefetchWaterfallStrategy(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()
	vite.UseWaterfallPrefetching(nil) // Use default concurrency

	// Remove hot file to test production mode
	os.Remove("hot")

	html, err := vite.Invoke([]string{"main.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error with waterfall prefetch: %v", err)
	}

	// Should contain prefetch script
	if !strings.Contains(html, "window.addEventListener('load'") {
		t.Error("Expected HTML to contain prefetch event listener")
	}

	// Should contain waterfall-specific loadNext function
	if !strings.Contains(html, "loadNext") {
		t.Error("Expected HTML to contain loadNext function for waterfall strategy")
	}

	// Should contain dynamic imports
	if !strings.Contains(html, "dynamic-xyz789.js") {
		t.Error("Expected HTML to contain dynamic import for prefetching")
	}

	// Should contain concurrency setting
	if !strings.Contains(html, "loadNext(") {
		t.Error("Expected HTML to contain loadNext call with concurrency")
	}

	_ = tmpDir
}

func TestPrefetchAggressiveStrategy(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()
	vite.UseAggressivePrefetching()

	// Remove hot file to test production mode
	os.Remove("hot")

	html, err := vite.Invoke([]string{"main.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error with aggressive prefetch: %v", err)
	}

	// Should contain prefetch script
	if !strings.Contains(html, "window.addEventListener('load'") {
		t.Error("Expected HTML to contain prefetch event listener")
	}

	// Should contain aggressive-specific forEach
	if !strings.Contains(html, "forEach") {
		t.Error("Expected HTML to contain forEach for aggressive strategy")
	}

	// Should contain dynamic imports
	if !strings.Contains(html, "dynamic-xyz789.js") {
		t.Error("Expected HTML to contain dynamic import for prefetching")
	}

	// Should NOT contain loadNext function (specific to waterfall)
	if strings.Contains(html, "loadNext") {
		t.Error("Expected HTML to NOT contain loadNext function for aggressive strategy")
	}

	_ = tmpDir
}

func TestPrefetchCustomEvent(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()
	concurrency := 2
	vite.Prefetch(&concurrency, "domcontentloaded")

	// Remove hot file to test production mode
	os.Remove("hot")

	html, err := vite.Invoke([]string{"main.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error with custom prefetch event: %v", err)
	}

	// Should contain custom event
	if !strings.Contains(html, "window.addEventListener('domcontentloaded'") {
		t.Error("Expected HTML to contain custom prefetch event")
	}

	// Should contain custom concurrency
	if !strings.Contains(html, "loadNext(") {
		t.Error("Expected HTML to contain loadNext call")
	}

	_ = tmpDir
}

func TestPrefetchWithNonce(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()
	vite.UseCspNonce("test-nonce-123")
	vite.UseAggressivePrefetching()

	// Remove hot file to test production mode
	os.Remove("hot")

	html, err := vite.Invoke([]string{"main.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error with nonce prefetch: %v", err)
	}

	// Should contain nonce in prefetch script
	if !strings.Contains(html, `nonce="test-nonce-123"`) {
		t.Error("Expected HTML to contain nonce in prefetch script")
	}

	_ = tmpDir
}

func TestNoPrefetchWhenNoDynamicImports(t *testing.T) {
	tmpDir := t.TempDir()

	// Create manifest without dynamic imports
	simpleManifest := `{
		"main.js": {
			"file": "assets/main-abc123.js",
			"src": "main.js",
			"isEntry": true,
			"imports": ["_vendor-def456.js"],
			"css": ["assets/main-abc123.css"]
		},
		"_vendor-def456.js": {
			"file": "assets/vendor-def456.js",
			"src": "_vendor-def456.js"
		}
	}`

	// Setup files
	buildDir := filepath.Join(tmpDir, "build")
	os.MkdirAll(buildDir, 0755)

	manifestPath := filepath.Join(buildDir, "manifest.json")
	os.WriteFile(manifestPath, []byte(simpleManifest), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	vite := NewVite()
	vite.UseAggressivePrefetching()

	html, err := vite.Invoke([]string{"main.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error with no dynamic imports: %v", err)
	}

	// Should NOT contain prefetch script when no dynamic imports
	if strings.Contains(html, "window.addEventListener") {
		t.Error("Expected HTML to NOT contain prefetch script when no dynamic imports")
	}

	// Should still contain regular preload and script tags
	if !strings.Contains(html, "main-abc123.js") {
		t.Error("Expected HTML to contain main script")
	}
}

func TestPrefetchInHotMode(t *testing.T) {
	tmpDir, cleanup := setupTestFiles(t)
	defer cleanup()

	vite := NewVite()
	vite.UseAggressivePrefetching()

	// Keep hot file to test hot mode
	html, err := vite.Invoke([]string{"main.js"}, "build")
	if err != nil {
		t.Errorf("Unexpected error in hot mode with prefetch: %v", err)
	}

	// Should NOT contain prefetch script in hot mode
	if strings.Contains(html, "window.addEventListener") {
		t.Error("Expected HTML to NOT contain prefetch script in hot mode")
	}

	// Should contain hot reload tags
	if !strings.Contains(html, "localhost:5173") {
		t.Error("Expected HTML to contain hot reload URL")
	}

	_ = tmpDir
}

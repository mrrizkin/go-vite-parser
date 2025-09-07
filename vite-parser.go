// Package goviteparser provides a Go implementation of the Vite parser.
package goviteparser

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AttributeResolver is a function type for resolving tag attributes
type AttributeResolver func(src, url string, chunk map[string]any, manifest map[string]any) map[string]any

// AssetPathResolver is a function type for resolving asset paths
type AssetPathResolver func(path string, secure bool) string

// PrefetchStrategy defines the prefetching strategy
type PrefetchStrategy string

const (
	PrefetchWaterfall  PrefetchStrategy = "waterfall"
	PrefetchAggressive PrefetchStrategy = "aggressive"
)

// PrefetchAsset represents an asset to be prefetched
type PrefetchAsset struct {
	Rel           string `json:"rel"`
	FetchPriority string `json:"fetchpriority"`
	Href          string `json:"href"`
	As            string `json:"as,omitempty"`
	Nonce         string `json:"nonce,omitempty"`
	Crossorigin   string `json:"crossorigin,omitempty"`
	Integrity     string `json:"integrity,omitempty"`
}

// ViteConfig represents the configuration for the Vite instance
type ViteConfig struct {
	BuildDirectory   string
	ManifestFilename string
	HotFile          string
	EntryPoints      []string
}

// NewVite creates a new Vite instance with default configuration
func NewVite() *Vite {
	return &Vite{
		config: ViteConfig{
			BuildDirectory:   "build",
			ManifestFilename: "manifest.json",
			HotFile:          "hot",
		},
		integrityKey:                 "integrity",
		scriptTagAttributeResolvers:  []AttributeResolver{},
		styleTagAttributeResolvers:   []AttributeResolver{},
		preloadTagAttributeResolvers: []AttributeResolver{},
		preloadedAssets:              make(map[string]map[string]any),
		manifests:                    make(map[string]map[string]any),
		prefetchConcurrently:         3,
		prefetchEvent:                "load",
	}
}

// Vite represents the main Vite instance with Laravel-like functionality
type Vite struct {
	config                       ViteConfig
	nonce                        string
	integrityKey                 string
	assetPathResolver            AssetPathResolver
	scriptTagAttributeResolvers  []AttributeResolver
	styleTagAttributeResolvers   []AttributeResolver
	preloadTagAttributeResolvers []AttributeResolver
	preloadedAssets              map[string]map[string]any
	manifests                    map[string]map[string]any
	prefetchStrategy             PrefetchStrategy
	prefetchConcurrently         int
	prefetchEvent                string
}

// UseCspNonce generates or sets a Content Security Policy nonce
func (v *Vite) UseCspNonce(nonce string) string {
	if nonce == "" {
		nonce = v.generateNonce()
	}
	v.nonce = nonce
	return nonce
}

// CspNonce returns the current CSP nonce
func (v *Vite) CspNonce() string {
	return v.nonce
}

// UseIntegrityKey sets the key to detect integrity hashes in the manifest
func (v *Vite) UseIntegrityKey(key string) *Vite {
	v.integrityKey = key
	return v
}

// WithEntryPoints sets the Vite entry points
func (v *Vite) WithEntryPoints(entryPoints []string) *Vite {
	v.config.EntryPoints = entryPoints
	return v
}

// MergeEntryPoints merges additional entry points with the current set
func (v *Vite) MergeEntryPoints(entryPoints []string) *Vite {
	merged := append(v.config.EntryPoints, entryPoints...)
	v.config.EntryPoints = v.uniqueStrings(merged)
	return v
}

// UseManifestFilename sets the filename for the manifest file
func (v *Vite) UseManifestFilename(filename string) *Vite {
	v.config.ManifestFilename = filename
	return v
}

// CreateAssetPathsUsing sets a custom asset path resolver
func (v *Vite) CreateAssetPathsUsing(resolver AssetPathResolver) *Vite {
	v.assetPathResolver = resolver
	return v
}

// HotFile returns the path to the hot file
func (v *Vite) HotFile() string {
	if v.config.HotFile == "" {
		return "hot"
	}
	return v.config.HotFile
}

// UseHotFile sets the Vite "hot" file path
func (v *Vite) UseHotFile(path string) *Vite {
	v.config.HotFile = path
	return v
}

// UseBuildDirectory sets the Vite build directory
func (v *Vite) UseBuildDirectory(path string) *Vite {
	v.config.BuildDirectory = path
	return v
}

// UseScriptTagAttributes adds a script tag attribute resolver
func (v *Vite) UseScriptTagAttributes(resolver AttributeResolver) *Vite {
	v.scriptTagAttributeResolvers = append(v.scriptTagAttributeResolvers, resolver)
	return v
}

// UseStyleTagAttributes adds a style tag attribute resolver
func (v *Vite) UseStyleTagAttributes(resolver AttributeResolver) *Vite {
	v.styleTagAttributeResolvers = append(v.styleTagAttributeResolvers, resolver)
	return v
}

// UsePreloadTagAttributes adds a preload tag attribute resolver
func (v *Vite) UsePreloadTagAttributes(resolver AttributeResolver) *Vite {
	v.preloadTagAttributeResolvers = append(v.preloadTagAttributeResolvers, resolver)
	return v
}

// Prefetch sets up prefetching with optional concurrency
func (v *Vite) Prefetch(concurrency *int, event string) *Vite {
	if event == "" {
		event = "load"
	}
	v.prefetchEvent = event

	if concurrency == nil {
		return v.UsePrefetchStrategy(PrefetchAggressive, nil)
	}
	return v.UsePrefetchStrategy(PrefetchWaterfall, map[string]any{"concurrency": *concurrency})
}

// UseWaterfallPrefetching sets the waterfall prefetching strategy
func (v *Vite) UseWaterfallPrefetching(concurrency *int) *Vite {
	config := map[string]any{}
	if concurrency != nil {
		config["concurrency"] = *concurrency
	} else {
		config["concurrency"] = v.prefetchConcurrently
	}
	return v.UsePrefetchStrategy(PrefetchWaterfall, config)
}

// UseAggressivePrefetching sets the aggressive prefetching strategy
func (v *Vite) UseAggressivePrefetching() *Vite {
	return v.UsePrefetchStrategy(PrefetchAggressive, nil)
}

// UsePrefetchStrategy sets the prefetching strategy
func (v *Vite) UsePrefetchStrategy(strategy PrefetchStrategy, config map[string]any) *Vite {
	v.prefetchStrategy = strategy
	if strategy == PrefetchWaterfall && config != nil {
		if concurrency, ok := config["concurrency"].(int); ok {
			v.prefetchConcurrently = concurrency
		}
	}
	return v
}

// Invoke generates Vite tags for entrypoints (main method like Laravel's __invoke)
func (v *Vite) Invoke(entrypoints []string, buildDirectory string) (string, error) {
	if buildDirectory == "" {
		buildDirectory = v.config.BuildDirectory
	}

	if v.IsRunningHot() {
		return v.generateHotTags(entrypoints)
	}

	manifest, err := v.getManifest(buildDirectory)
	if err != nil {
		return "", err
	}

	return v.generateProductionTags(entrypoints, buildDirectory, manifest)
}

// IsRunningHot determines if the HMR server is running
func (v *Vite) IsRunningHot() bool {
	_, err := os.Stat(v.HotFile())
	return err == nil
}

// Asset gets the URL for an asset
func (v *Vite) Asset(asset, buildDirectory string) (string, error) {
	if buildDirectory == "" {
		buildDirectory = v.config.BuildDirectory
	}

	if v.IsRunningHot() {
		return v.hotAsset(asset)
	}

	manifest, err := v.getManifest(buildDirectory)
	if err != nil {
		return "", err
	}

	chunk, err := v.getChunk(manifest, asset)
	if err != nil {
		return "", err
	}

	file, ok := chunk["file"].(string)
	if !ok {
		return "", fmt.Errorf("invalid file in chunk for asset: %s", asset)
	}

	return v.assetPath(filepath.Join(buildDirectory, file), false), nil
}

// Content gets the content of a given asset
func (v *Vite) Content(asset, buildDirectory string) (string, error) {
	if buildDirectory == "" {
		buildDirectory = v.config.BuildDirectory
	}

	manifest, err := v.getManifest(buildDirectory)
	if err != nil {
		return "", err
	}

	chunk, err := v.getChunk(manifest, asset)
	if err != nil {
		return "", err
	}

	file, ok := chunk["file"].(string)
	if !ok {
		return "", fmt.Errorf("invalid file in chunk for asset: %s", asset)
	}

	path := filepath.Join(buildDirectory, file)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("unable to locate file from Vite manifest: %s", path)
	}

	return string(content), nil
}

// ManifestHash gets a unique hash representing the current manifest
func (v *Vite) ManifestHash(buildDirectory string) (string, error) {
	if buildDirectory == "" {
		buildDirectory = v.config.BuildDirectory
	}

	if v.IsRunningHot() {
		return "", nil
	}

	manifestPath := v.manifestPath(buildDirectory)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return "", nil
	}

	file, err := os.Open(manifestPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ReactRefresh generates React refresh runtime script
func (v *Vite) ReactRefresh() (string, error) {
	if !v.IsRunningHot() {
		return "", nil
	}

	refreshAsset, err := v.hotAsset("@react-refresh")
	if err != nil {
		return "", err
	}

	attributes := v.parseAttributes(map[string]any{
		"nonce": v.nonce,
	})

	return fmt.Sprintf(`<script type="module" %s>
    import RefreshRuntime from '%s'
    RefreshRuntime.injectIntoGlobalHook(window)
    window.$RefreshReg$ = () => {}
    window.$RefreshSig$ = () => (type) => type
    window.__vite_plugin_react_preamble_installed__ = true
</script>`, strings.Join(attributes, " "), refreshAsset), nil
}

// PreloadedAssets returns the preloaded assets
func (v *Vite) PreloadedAssets() map[string]map[string]any {
	return v.preloadedAssets
}

// Flush clears the state
func (v *Vite) Flush() {
	v.preloadedAssets = make(map[string]map[string]any)
}

// ToHTML gets the Vite tag content as a string of HTML using configured entry points
func (v *Vite) ToHTML() (string, error) {
	return v.Invoke(v.config.EntryPoints, "")
}

// Private methods

func (v *Vite) generateNonce() string {
	bytes := make([]byte, 30)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (v *Vite) uniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}

func (v *Vite) generateHotTags(entrypoints []string) (string, error) {
	origin, err := v.getHotOrigin()
	if err != nil {
		return "", err
	}

	var tags []string

	// Add @vite/client first
	clientURL, err := url.JoinPath(origin, "@vite/client")
	if err != nil {
		return "", err
	}
	tags = append(tags, v.makeTagForChunk("@vite/client", clientURL, nil, nil))

	// Add entrypoint tags
	for _, entrypoint := range entrypoints {
		entrypointURL, err := url.JoinPath(origin, entrypoint)
		if err != nil {
			continue
		}
		tags = append(tags, v.makeTagForChunk(entrypoint, entrypointURL, nil, nil))
	}

	return strings.Join(tags, ""), nil
}

func (v *Vite) generateProductionTags(
	entrypoints []string,
	buildDirectory string,
	manifest map[string]any,
) (string, error) {
	var tags []string
	var preloads []string
	processedPreloads := make(map[string]bool)

	for _, entrypoint := range entrypoints {
		chunk, err := v.getChunk(manifest, entrypoint)
		if err != nil {
			continue
		}

		file, ok := chunk["file"].(string)
		if !ok {
			continue
		}

		url := v.assetPath(filepath.Join(buildDirectory, file), false)

		// Add preload for main chunk
		if !processedPreloads[url] {
			preloadTag := v.makePreloadTagForChunk(entrypoint, url, chunk, manifest)
			if preloadTag != "" {
				preloads = append(preloads, preloadTag)
				processedPreloads[url] = true
			}
		}

		// Process imports
		if imports, ok := chunk["imports"].([]any); ok {
			for _, importInterface := range imports {
				if importStr, ok := importInterface.(string); ok {
					if importChunk, exists := manifest[importStr].(map[string]any); exists {
						if importFile, ok := importChunk["file"].(string); ok {
							importURL := v.assetPath(
								filepath.Join(buildDirectory, importFile),
								false,
							)
							if !processedPreloads[importURL] {
								preloadTag := v.makePreloadTagForChunk(
									importStr,
									importURL,
									importChunk,
									manifest,
								)
								if preloadTag != "" {
									preloads = append(preloads, preloadTag)
									processedPreloads[importURL] = true
								}
							}
						}

						// Process CSS from imports
						if css, ok := importChunk["css"].([]any); ok {
							for _, cssInterface := range css {
								if cssStr, ok := cssInterface.(string); ok {
									cssURL := v.assetPath(
										filepath.Join(buildDirectory, cssStr),
										false,
									)
									cssChunk := map[string]any{"file": cssStr}
									if !processedPreloads[cssURL] {
										preloadTag := v.makePreloadTagForChunk(
											cssStr,
											cssURL,
											cssChunk,
											manifest,
										)
										if preloadTag != "" {
											preloads = append(preloads, preloadTag)
											processedPreloads[cssURL] = true
										}
									}
									tags = append(
										tags,
										v.makeTagForChunk(cssStr, cssURL, cssChunk, manifest),
									)
								}
							}
						}
					}
				}
			}
		}

		// Add main chunk tag
		tags = append(tags, v.makeTagForChunk(entrypoint, url, chunk, manifest))

		// Process CSS from main chunk
		if css, ok := chunk["css"].([]any); ok {
			for _, cssInterface := range css {
				if cssStr, ok := cssInterface.(string); ok {
					cssURL := v.assetPath(filepath.Join(buildDirectory, cssStr), false)
					cssChunk := map[string]any{"file": cssStr}
					if !processedPreloads[cssURL] {
						preloadTag := v.makePreloadTagForChunk(cssStr, cssURL, cssChunk, manifest)
						if preloadTag != "" {
							preloads = append(preloads, preloadTag)
							processedPreloads[cssURL] = true
						}
					}
					tags = append(tags, v.makeTagForChunk(cssStr, cssURL, cssChunk, manifest))
				}
			}
		}
	}

	// Combine preloads and tags
	base := strings.Join(preloads, "") + strings.Join(tags, "")

	// Handle prefetch strategy
	if v.prefetchStrategy == "" {
		return base, nil
	}

	// Collect dynamic imports for prefetching
	prefetchAssets := v.collectDynamicImports(entrypoints, manifest)
	if len(prefetchAssets) == 0 {
		return base, nil
	}

	// Generate prefetch script
	prefetchScript := v.generatePrefetchScript(prefetchAssets, v.prefetchStrategy)

	return base + prefetchScript, nil
}

func (v *Vite) getHotOrigin() (string, error) {
	content, err := os.ReadFile(v.HotFile())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

func (v *Vite) hotAsset(asset string) (string, error) {
	origin, err := v.getHotOrigin()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(origin, "/") + "/" + asset, nil
}

func (v *Vite) getManifest(buildDirectory string) (map[string]any, error) {
	manifestPath := v.manifestPath(buildDirectory)

	if cached, exists := v.manifests[manifestPath]; exists {
		return cached, nil
	}

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("vite manifest not found at: %s", manifestPath)
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest map[string]any
	if err := json.Unmarshal(content, &manifest); err != nil {
		return nil, err
	}

	v.manifests[manifestPath] = manifest
	return manifest, nil
}

func (v *Vite) manifestPath(buildDirectory string) string {
	return filepath.Join(buildDirectory, v.config.ManifestFilename)
}

func (v *Vite) getChunk(manifest map[string]any, file string) (map[string]any, error) {
	chunk, exists := manifest[file]
	if !exists {
		return nil, fmt.Errorf("unable to locate file in Vite manifest: %s", file)
	}

	chunkMap, ok := chunk.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid chunk format for file: %s", file)
	}

	return chunkMap, nil
}

func (v *Vite) makeTagForChunk(
	src, url string,
	chunk map[string]any,
	manifest map[string]any,
) string {
	if v.isCSSPath(url) {
		return v.makeStylesheetTagWithAttributes(
			url,
			v.resolveStylesheetTagAttributes(src, url, chunk, manifest),
		)
	}
	return v.makeScriptTagWithAttributes(
		url,
		v.resolveScriptTagAttributes(src, url, chunk, manifest),
	)
}

func (v *Vite) makePreloadTagForChunk(
	src, url string,
	chunk map[string]any,
	manifest map[string]any,
) string {
	attributes := v.resolvePreloadTagAttributes(src, url, chunk, manifest)
	if attributes == nil {
		return ""
	}

	// Store preloaded asset
	if v.preloadedAssets == nil {
		v.preloadedAssets = make(map[string]map[string]any)
	}

	preloadAttrs := make(map[string]any)
	for k, v := range attributes {
		if k != "href" {
			preloadAttrs[k] = v
		}
	}
	v.preloadedAssets[url] = preloadAttrs

	return "<link " + strings.Join(v.parseAttributes(attributes), " ") + " />"
}

func (v *Vite) resolveScriptTagAttributes(
	src, url string,
	chunk map[string]any,
	manifest map[string]any,
) map[string]any {
	attributes := make(map[string]any)

	if v.integrityKey != "" && chunk != nil {
		if integrity, exists := chunk[v.integrityKey]; exists {
			attributes["integrity"] = integrity
		}
	}

	for _, resolver := range v.scriptTagAttributeResolvers {
		resolved := resolver(src, url, chunk, manifest)
		for k, v := range resolved {
			attributes[k] = v
		}
	}

	return attributes
}

func (v *Vite) resolveStylesheetTagAttributes(
	src, url string,
	chunk map[string]any,
	manifest map[string]any,
) map[string]any {
	attributes := make(map[string]any)

	if v.integrityKey != "" && chunk != nil {
		if integrity, exists := chunk[v.integrityKey]; exists {
			attributes["integrity"] = integrity
		}
	}

	for _, resolver := range v.styleTagAttributeResolvers {
		resolved := resolver(src, url, chunk, manifest)
		for k, v := range resolved {
			attributes[k] = v
		}
	}

	return attributes
}

func (v *Vite) resolvePreloadTagAttributes(
	src, url string,
	chunk map[string]any,
	manifest map[string]any,
) map[string]any {
	var attributes map[string]any

	if v.isCSSPath(url) {
		attributes = map[string]any{
			"rel":         "preload",
			"as":          "style",
			"href":        url,
			"nonce":       v.nonce,
			"crossorigin": v.resolveStylesheetTagAttributes(src, url, chunk, manifest)["crossorigin"],
		}
	} else {
		attributes = map[string]any{
			"rel":         "modulepreload",
			"as":          "script",
			"href":        url,
			"nonce":       v.nonce,
			"crossorigin": v.resolveScriptTagAttributes(src, url, chunk, manifest)["crossorigin"],
		}
	}

	if v.integrityKey != "" && chunk != nil {
		if integrity, exists := chunk[v.integrityKey]; exists {
			attributes["integrity"] = integrity
		}
	}

	for _, resolver := range v.preloadTagAttributeResolvers {
		resolved := resolver(src, url, chunk, manifest)
		if resolved == nil {
			return nil
		}
		for k, v := range resolved {
			attributes[k] = v
		}
	}

	return attributes
}

func (v *Vite) makeScriptTagWithAttributes(url string, attributes map[string]any) string {
	allAttributes := map[string]any{
		"type":  "module",
		"src":   url,
		"nonce": v.nonce,
	}

	for k, v := range attributes {
		allAttributes[k] = v
	}

	return "<script " + strings.Join(v.parseAttributes(allAttributes), " ") + "></script>"
}

func (v *Vite) makeStylesheetTagWithAttributes(url string, attributes map[string]any) string {
	allAttributes := map[string]any{
		"rel":   "stylesheet",
		"href":  url,
		"nonce": v.nonce,
	}

	for k, v := range attributes {
		allAttributes[k] = v
	}

	return "<link " + strings.Join(v.parseAttributes(allAttributes), " ") + " />"
}

func (v *Vite) isCSSPath(path string) bool {
	cssRegex := regexp.MustCompile(`\.(css|less|sass|scss|styl|stylus|pcss|postcss)(\?[^\.]*)?$`)
	return cssRegex.MatchString(path)
}

func (v *Vite) parseAttributes(attributes map[string]any) []string {
	var result []string
	for key, value := range attributes {
		if value == nil || value == false {
			continue
		}
		if value == true {
			result = append(result, key)
		} else {
			result = append(result, fmt.Sprintf(`%s="%v"`, key, value))
		}
	}
	return result
}

func (v *Vite) assetPath(path string, secure bool) string {
	if v.assetPathResolver != nil {
		return v.assetPathResolver(path, secure)
	}
	// Default implementation - just return the path as-is
	// In a real application, this would generate proper URLs
	return "/" + strings.TrimLeft(path, "/")
}

// collectDynamicImports recursively collects all dynamic imports from the manifest
func (v *Vite) collectDynamicImports(
	entrypoints []string,
	manifest map[string]any,
) []PrefetchAsset {
	var assets []PrefetchAsset
	discoveredImports := make(map[string]bool)

	for _, entrypoint := range entrypoints {
		chunk, exists := manifest[entrypoint].(map[string]any)
		if !exists {
			continue
		}

		dynamicImports, ok := chunk["dynamicImports"].([]any)
		if !ok {
			continue
		}

		for _, importInterface := range dynamicImports {
			importStr, ok := importInterface.(string)
			if !ok {
				continue
			}

			importChunk, exists := manifest[importStr].(map[string]any)
			if !exists {
				continue
			}

			file, ok := importChunk["file"].(string)
			if !ok {
				continue
			}

			// Only process JS and CSS files
			if !strings.HasSuffix(file, ".js") && !strings.HasSuffix(file, ".css") {
				continue
			}

			// Recursively collect imports
			collected := v.collectImportsRecursively(importChunk, manifest, &discoveredImports)
			assets = append(assets, collected...)
		}
	}

	return v.uniquePrefetchAssets(assets)
}

// collectImportsRecursively recursively collects imports and dynamic imports
func (v *Vite) collectImportsRecursively(
	chunk map[string]any,
	manifest map[string]any,
	discoveredImports *map[string]bool,
) []PrefetchAsset {
	var assets []PrefetchAsset

	// Add the current chunk
	if file, ok := chunk["file"].(string); ok {
		src := chunk["src"]
		if src == nil {
			// Find the src by looking for the chunk in manifest
			for key, value := range manifest {
				if chunkValue, ok := value.(map[string]any); ok {
					if chunkFile, ok := chunkValue["file"].(string); ok && chunkFile == file {
						src = key
						break
					}
				}
			}
		}

		asset := v.createPrefetchAsset(src, file, chunk, manifest)
		if asset != nil {
			assets = append(assets, *asset)
		}
	}

	// Process imports
	if imports, ok := chunk["imports"].([]any); ok {
		for _, importInterface := range imports {
			importStr, ok := importInterface.(string)
			if !ok {
				continue
			}

			if (*discoveredImports)[importStr] {
				continue
			}
			(*discoveredImports)[importStr] = true

			if importChunk, exists := manifest[importStr].(map[string]any); exists {
				collected := v.collectImportsRecursively(importChunk, manifest, discoveredImports)
				assets = append(assets, collected...)
			}
		}
	}

	// Process dynamic imports
	if dynamicImports, ok := chunk["dynamicImports"].([]any); ok {
		for _, importInterface := range dynamicImports {
			importStr, ok := importInterface.(string)
			if !ok {
				continue
			}

			if (*discoveredImports)[importStr] {
				continue
			}
			(*discoveredImports)[importStr] = true

			if importChunk, exists := manifest[importStr].(map[string]any); exists {
				collected := v.collectImportsRecursively(importChunk, manifest, discoveredImports)
				assets = append(assets, collected...)
			}
		}
	}

	// Process CSS files
	if css, ok := chunk["css"].([]any); ok {
		for _, cssInterface := range css {
			cssStr, ok := cssInterface.(string)
			if !ok {
				continue
			}

			// Find the CSS chunk in manifest or create a minimal one
			var cssChunk map[string]any
			for _, value := range manifest {
				if chunkValue, ok := value.(map[string]any); ok {
					if chunkFile, ok := chunkValue["file"].(string); ok && chunkFile == cssStr {
						cssChunk = chunkValue
						break
					}
				}
			}

			if cssChunk == nil {
				cssChunk = map[string]any{"file": cssStr}
			}

			asset := v.createPrefetchAsset(cssStr, cssStr, cssChunk, manifest)
			if asset != nil {
				assets = append(assets, *asset)
			}
		}
	}

	return assets
}

// createPrefetchAsset creates a prefetch asset from chunk information
func (v *Vite) createPrefetchAsset(
	src any,
	file string,
	chunk map[string]any,
	manifest map[string]any,
) *PrefetchAsset {
	buildDirectory := v.config.BuildDirectory
	url := v.assetPath(fmt.Sprintf("%s/%s", buildDirectory, file), false)

	// Check if already preloaded
	if _, exists := v.preloadedAssets[url]; exists {
		return nil
	}

	srcStr := ""
	if src != nil {
		srcStr = fmt.Sprintf("%v", src)
	}

	attributes := v.resolvePreloadTagAttributes(srcStr, url, chunk, manifest)
	if attributes == nil {
		return nil
	}

	asset := &PrefetchAsset{
		Rel:           "prefetch",
		FetchPriority: "low",
		Href:          url,
	}

	// Set additional attributes
	if as, ok := attributes["as"].(string); ok {
		asset.As = as
	}
	if v.nonce != "" {
		asset.Nonce = v.nonce
	}
	if crossorigin, ok := attributes["crossorigin"].(string); ok {
		asset.Crossorigin = crossorigin
	}
	if integrity, ok := attributes["integrity"].(string); ok {
		asset.Integrity = integrity
	}

	return asset
}

// uniquePrefetchAssets removes duplicate assets based on href
func (v *Vite) uniquePrefetchAssets(assets []PrefetchAsset) []PrefetchAsset {
	seen := make(map[string]bool)
	var unique []PrefetchAsset

	for _, asset := range assets {
		if !seen[asset.Href] {
			seen[asset.Href] = true
			unique = append(unique, asset)
		}
	}

	return unique
}

// generatePrefetchScript generates the JavaScript for prefetching assets
func (v *Vite) generatePrefetchScript(assets []PrefetchAsset, strategy PrefetchStrategy) string {
	if len(assets) == 0 {
		return ""
	}

	assetsJSON, err := json.Marshal(assets)
	if err != nil {
		return ""
	}

	nonceAttr := ""
	if v.nonce != "" {
		nonceAttr = fmt.Sprintf(` nonce="%s"`, v.nonce)
	}

	switch strategy {
	case PrefetchWaterfall:
		return fmt.Sprintf(`
<script%s>
     window.addEventListener('%s', () => window.setTimeout(() => {
        const makeLink = (asset) => {
            const link = document.createElement('link')

            Object.keys(asset).forEach((attribute) => {
                link.setAttribute(attribute, asset[attribute])
            })

            return link
        }

        const loadNext = (assets, count) => window.setTimeout(() => {
            if (count > assets.length) {
                count = assets.length

                if (count === 0) {
                    return
                }
            }

            const fragment = new DocumentFragment

            while (count > 0) {
                const link = makeLink(assets.shift())
                fragment.append(link)
                count--

                if (assets.length) {
                    link.onload = () => loadNext(assets, 1)
                    link.onerror = () => loadNext(assets, 1)
                }
            }

            document.head.append(fragment)
        })

        loadNext(%s, %d)
    }))
</script>`, nonceAttr, v.prefetchEvent, string(assetsJSON), v.prefetchConcurrently)

	case PrefetchAggressive:
		return fmt.Sprintf(`
<script%s>
     window.addEventListener('%s', () => window.setTimeout(() => {
        const makeLink = (asset) => {
            const link = document.createElement('link')

            Object.keys(asset).forEach((attribute) => {
                link.setAttribute(attribute, asset[attribute])
            })

            return link
        }

        const fragment = new DocumentFragment;
        %s.forEach((asset) => fragment.append(makeLink(asset)))
        document.head.append(fragment)
     }))
</script>`, nonceAttr, v.prefetchEvent, string(assetsJSON))

	default:
		return ""
	}
}

package goviteparser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
)

type (
	Config struct {
		OutDir       string
		ManifestPath string
		HotFilePath  string
	}

	EntryInfo struct {
		File    string   `json:"file"`
		CSS     []string `json:"css"`
		Imports []string `json:"imports"`
	}

	HTMLTags struct {
		Preload string
		CSS     string
		JS      string
	}

	Manifest     map[string]EntryInfo
	ManifestTags map[string]HTMLTags

	ViteManifestInfo struct {
		Origin       string
		Manifest     Manifest
		ManifestTags ManifestTags
		Client       string
		ClientTag    string
		ReactRefresh string
	}
)

var (
	scriptExtensions = []string{".js", ".ts", ".jsx", ".tsx", ".mjs", ".cjs", ".wasm", ".vue", ".svelte"}
	styleExtensions  = []string{".css", ".scss", ".sass", ".less", ".styl", ".stylus", ".pcss", ".postcss"}
)

func Parse(config Config) ViteManifestInfo {
	var err error

	origin := ""
	hotFilePath := path.Clean(config.HotFilePath)
	_, err = os.Stat(hotFilePath)
	if err == nil {
		content, err := os.ReadFile(hotFilePath)
		if err == nil {
			origin = string(content)
		}
	}

	manifest := make(Manifest)
	if origin == "" {
		manifestPath := path.Join(config.ManifestPath)
		content, err := os.ReadFile(manifestPath)
		if err == nil {
			_ = json.Unmarshal(content, &manifest)
		}
	}

	client := ""
	clientTag := ""
	if origin != "" {
		client, err = url.JoinPath(origin, "/@vite/client")
		if err == nil {
			clientTag = createScriptTag(client)
		}
	}

	manifestTags := make(ManifestTags)

	prefix := origin
	if prefix == "" {
		prefix = config.OutDir
	}

	for entry, entryInfo := range manifest {
		manifestTags[entry] = resolveTagEntry(manifest, entryInfo, prefix)
	}

	return ViteManifestInfo{
		Origin:       origin,
		Manifest:     manifest,
		Client:       client,
		ClientTag:    clientTag,
		ReactRefresh: createReactRefreshTag(origin),
	}
}

func (tags *HTMLTags) Render() string {
	return tags.Preload + tags.CSS + tags.JS
}

func resolveTagEntry(manifest Manifest, entryInfo EntryInfo, prefix string) HTMLTags {
	preload := ""
	style := ""
	script := ""

	preload += createPreloadTag(prefix + entryInfo.File)
	for _, cssPath := range entryInfo.CSS {
		style += createStyleTag(prefix + cssPath)
	}

	for _, importPath := range entryInfo.Imports {
		importEntryInfo, ok := manifest[importPath]
		if ok && importEntryInfo.File != "" {
			preload += createPreloadTag(prefix + importEntryInfo.File)
		}

		if ok && len(importEntryInfo.CSS) > 0 {
			for _, cssPath := range importEntryInfo.CSS {
				style += createStyleTag(prefix + cssPath)
			}
		}
	}

	file := entryInfo.File
	extension := path.Ext(file)
	if inArray(extension, scriptExtensions) {
		script += createScriptTag(prefix + file)
	} else if inArray(extension, styleExtensions) {
		style += createStyleTag(prefix + file)
	}

	return HTMLTags{
		Preload: preload,
		CSS:     style,
		JS:      script,
	}
}

func inArray(needle string, haystack []string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}

	return false
}

func createReactRefreshTag(origin string) string {
	return fmt.Sprintf(`<script type="module">
    import RefreshRuntime from '%s/@react-refresh';
    RefreshRuntime.injectIntoGlobalHook(window);
    window.$RefreshReg$ = () => {};
    window.$RefreshSig$ = () => (type) => type;
    window.__vite_plugin_react_preamble_installed__ = true;
	</script>`, origin)
}

func createPreloadTag(path string) string {
	return fmt.Sprintf(`<link rel="modulepreload" href="%s" />`, path)
}

func createStyleTag(path string) string {
	return fmt.Sprintf(`<link rel="stylesheet" href="%s" />`, path)
}

func createScriptTag(path string) string {
	return fmt.Sprintf(`<script type="module" src="%s"></script>`, path)
}
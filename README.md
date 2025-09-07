# Go Vite Parser

A Laravel-inspired Vite integration for Go applications. This package provides a fluent API for generating Vite asset tags, handling both development (hot reload) and production modes seamlessly.

## Features

- 🔥 **Hot Module Replacement (HMR)** support
- 📦 **Production asset bundling** with manifest parsing
- 🎨 **CSS preprocessing** support (Sass, Less, Stylus, etc.)
- ⚡ **Preload generation** for optimal performance
- 🔒 **CSP nonce** support for security
- 🎯 **Integrity hashes** for asset verification
- 🔧 **Customizable attribute resolvers**
- 📊 **React Refresh** integration
- 🚀 **Prefetching strategies** (waterfall/aggressive)

## Installation

```bash
go get github.com/mrrizkin/go-vite-parser
```

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/mrrizkin/go-vite-parser"
)

func main() {
    // Create a new Vite instance
    vite := goviteparser.NewVite()
    
    // Configure entry points
    vite.WithEntryPoints([]string{"main.js", "app.css"})
    
    // Optional: Configure build directory (default: "build")
    vite.UseBuildDirectory("dist")
    
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Generate Vite tags
        viteTags, err := vite.ToHTML()
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        
        html := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Go Vite App</title>
    %s
</head>
<body>
    <div id="app">
        <h1>Hello from Go + Vite!</h1>
    </div>
</body>
</html>`, viteTags)
        
        w.Header().Set("Content-Type", "text/html")
        fmt.Fprint(w, html)
    })
    
    fmt.Println("Server running at http://localhost:8080")
    http.ListenAndServe(":8080", nil)
}
```

### Advanced Configuration

```go
vite := goviteparser.NewVite()

// Security: Add CSP nonce
nonce := vite.UseCspNonce("") // Auto-generates secure nonce

// Asset integrity
vite.UseIntegrityKey("integrity")

// Custom asset path resolver
vite.CreateAssetPathsUsing(func(path string, secure bool) string {
    if secure {
        return "https://cdn.example.com/" + path
    }
    return "https://assets.example.com/" + path
})

// Custom script attributes
vite.UseScriptTagAttributes(func(src, url string, chunk, manifest map[string]interface{}) map[string]interface{} {
    return map[string]interface{}{
        "crossorigin": "anonymous",
        "data-turbo-track": "reload",
    }
})

// Prefetching strategy
vite.UseWaterfallPrefetching(&[]int{3}[0]) // Prefetch 3 assets concurrently
// OR
vite.UseAggressivePrefetching() // Prefetch all assets immediately
```

## Development vs Production

The package automatically detects the environment:

- **Development**: Looks for a `hot` file (created by Vite dev server)
- **Production**: Reads from `manifest.json` in the build directory

### Development Mode (Hot Reload)

When the `hot` file exists, the package:
- Injects `@vite/client` for HMR
- Serves assets directly from the Vite dev server
- Supports React Refresh automatically

### Production Mode

When no `hot` file is found, the package:
- Reads the Vite manifest
- Generates optimized preload tags
- Includes integrity hashes (if configured)
- Resolves all asset dependencies

## API Reference

### Core Methods

```go
// Create new instance
vite := goviteparser.NewVite()

// Configuration
vite.WithEntryPoints([]string{"main.js"})
vite.UseBuildDirectory("dist")
vite.UseManifestFilename("manifest.json")
vite.UseHotFile("hot")

// Security
vite.UseCspNonce("custom-nonce")
vite.UseIntegrityKey("integrity")

// Generate HTML
html, err := vite.ToHTML()
html, err := vite.Invoke([]string{"main.js"}, "")

// Asset utilities
assetUrl, err := vite.Asset("main.js", "")
content, err := vite.Content("main.js", "")
hash, err := vite.ManifestHash("")

// React support
refreshScript, err := vite.ReactRefresh()

// State management
vite.Flush() // Clear preloaded assets
assets := vite.PreloadedAssets()
```

### Attribute Resolvers

Customize HTML attributes for different tag types:

```go
// Script tags
vite.UseScriptTagAttributes(func(src, url string, chunk, manifest map[string]interface{}) map[string]interface{} {
    return map[string]interface{}{
        "crossorigin": "anonymous",
        "defer": true,
    }
})

// Style tags
vite.UseStyleTagAttributes(func(src, url string, chunk, manifest map[string]interface{}) map[string]interface{} {
    return map[string]interface{}{
        "media": "screen",
    }
})

// Preload tags
vite.UsePreloadTagAttributes(func(src, url string, chunk, manifest map[string]interface{}) map[string]interface{} {
    return map[string]interface{}{
        "crossorigin": "anonymous",
    }
})
```

## Vite Configuration

### vite.config.js

```javascript
import { defineConfig } from 'vite'

export default defineConfig({
  build: {
    manifest: true,
    outDir: 'public/build',
    rollupOptions: {
      input: {
        main: 'resources/js/main.js',
        app: 'resources/css/app.css'
      }
    }
  },
  server: {
    hmr: {
      host: 'localhost'
    }
  }
})
```

### Hot File Setup

For development, create a `hot` file containing your dev server URL:

```bash
echo "http://localhost:5173" > hot
```

Or use the `vite-plugin-backend` plugin to handle this automatically.

## Testing

The package includes comprehensive tests:

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run benchmarks
make bench

# Run full CI pipeline
make ci
```

## Performance

Benchmark results on modern hardware:
- NewVite(): ~28ns per operation
- Asset resolution: ~66ns per operation  
- Tag generation: ~15μs per operation
- Manifest parsing: ~28μs per operation

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests (`make test`)
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [Laravel Vite](https://laravel.com/docs/vite)
- Built for the Go ecosystem with performance in mind

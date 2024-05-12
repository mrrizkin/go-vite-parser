### Go Vite Parser

#### Overview

This package helps parse Vite manifests, generating HTML tags for web development projects. It offers functionality to extract data from Vite manifests and create HTML tags for scripts and styles.

#### Usage

```go
// main.go
package main

import (
    "fmt"
    "github.com/mrrizkin/go-vite-parser"
    "net/http"
)

func main() {
    // you can get the hot file path using the vite-plugin-backend
    viteManifestInfo := goviteparser.Parse(goviteparser.Config{
        OutDir:       "/public/build/",
        ManifestPath: "/public/build/manifest.json",
        HotFilePath:  "/public/hot",
    })

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // enter entrypoint name to get the tags
        mainTags := viteManifestInfo.ManifestTags["main.js"].Render()

        fmt.Fprintf(w, `
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <title>Example Page</title>
                %s
            </head>
            <body>
                <h1>Welcome to Example Page</h1>
                <p>This page is using Vite for frontend development.</p>
            </body>
            </html>
        `, mainTags)
    })

    http.ListenAndServe(":8080", nil)
}
```

```javascript
// main.js
document.addEventListener("DOMContentLoaded", () => {
  console.log("Hello, world!");
});
```

```javascript
// vite.config.js
import { defineConfig } from "vite";
import backendPlugin from "vite-plugin-backend";

export default defineConfig({
  plugins: [
    backendPlugin({
      input: ["main.js"],
    }),
  ],
});
```

#### Structs

- **Config**: Configuration options for parsing Vite manifest.
- **HTMLTags**: HTML tags for preload, CSS, and JavaScript.
- **Manifest**: Represents the Vite manifest.
- **ManifestTags**: HTML tags for entries in the manifest.
- **ViteManifestInfo**: Information parsed from the Vite manifest, including origin, manifest data, client URL, client tag, and React refresh tag.

#### Functions

- **Parse**: Parses the Vite manifest and generates ViteManifestInfo.
- **(HTMLTags) Render**: Renders HTML tags for preload, CSS, and JavaScript.

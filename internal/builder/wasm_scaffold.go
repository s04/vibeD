package builder

import "fmt"

// GenerateWasmScaffold returns additional files (wasmcloud.toml, WIT interfaces, etc.)
// needed to compile source code as a wasmCloud component for the given language.
// The returned map is filename → content. These files are written alongside the
// user's source before `wash build` runs.
func GenerateWasmScaffold(language, imageName string, files map[string]string) map[string]string {
	if language == "" || language == "auto" {
		language = DetectLanguage(files)
	}

	switch language {
	case "go":
		return scaffoldGo(imageName)
	case "rust":
		return scaffoldRust(imageName)
	default:
		return nil
	}
}

// scaffoldGo generates wasmcloud.toml, WIT world, and go:generate directive
// for a TinyGo-based wasmCloud HTTP component.
func scaffoldGo(imageName string) map[string]string {
	toml := fmt.Sprintf(`name = "component"
language = "tinygo"
type = "component"
version = "0.1.0"

[component]
wasm_target = "wasm32-wasi-preview2"
wit_world = "hello"

[tinygo]
scheduler = "none"
garbage_collector = "conservative"

[registry]
push_to = "%s"
`, imageName)

	wit := `package vibed:component;

world hello {
  include wasmcloud:component-go/imports@0.1.0;

  export wasi:http/incoming-handler@0.2.0;
}
`

	generateGo := `package main

//go:generate go run go.bytecodealliance.org/cmd/wit-bindgen-go generate --world hello --out gen ./wit
`

	toolsGo := `//go:build tools

package main

import (
	_ "go.bytecodealliance.org/cmd/wit-bindgen-go"
)
`

	return map[string]string{
		"wasmcloud.toml": toml,
		"wit/world.wit":  wit,
		"generate.go":    generateGo,
		"tools.go":       toolsGo,
	}
}

// scaffoldRust generates wasmcloud.toml and WIT world for a Rust wasmCloud HTTP component.
func scaffoldRust(imageName string) map[string]string {
	toml := fmt.Sprintf(`name = "component"
language = "rust"
type = "component"
version = "0.1.0"

[component]
wasm_target = "wasm32-wasip2"
wit_world = "hello"

[registry]
push_to = "%s"
`, imageName)

	wit := `package vibed:component;

world hello {
  export wasi:http/incoming-handler@0.2.0;
}
`

	return map[string]string{
		"wasmcloud.toml": toml,
		"wit/world.wit":  wit,
	}
}

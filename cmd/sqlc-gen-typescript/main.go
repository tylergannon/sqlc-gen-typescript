package main

import (
	"github.com/sqlc-dev/plugin-sdk-go/codegen"

	"github.com/tylergannon/sqlc-gen-typescript/internal/generator"
)

func main() {
	codegen.Run(generator.Generate)
}

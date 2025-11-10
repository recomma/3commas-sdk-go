//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi.yaml 3commas-openapi/3commas/openapi.yaml
//go:generate go run error_helpers_gen.go -input ./threecommas/openapi.gen.go -output ./threecommas/apierr_gen.go -package threecommas
//go:generate go run options_generator.go -input ./threecommas/openapi.gen.go -output ./threecommas/options_gen.go -package threecommas
//go:generate go tool golang.org/x/tools/cmd/goimports -w ./threecommas/options_gen.go
package main

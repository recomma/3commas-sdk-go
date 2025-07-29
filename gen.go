//go:generate oapi-codegen -config oapi.yaml ../3commas/openapi.yaml
//go:generate go run error_helpers_gen.go -input ./threecommas/openapi.gen.go -output ./threecommas/apierr_gen.go -package threecommas
//go:generate go run options_generator.go -input ./threecommas/openapi.gen.go -output ./threecommas/options_gen.go -package threecommas
package main

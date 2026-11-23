// Package docs menyediakan spesifikasi OpenAPI/Swagger (dibaca oleh gin-swagger).
package docs

import (
	_ "embed"

	"github.com/swaggo/swag"
)

// APIJSON adalah spesifikasi OpenAPI dalam format JSON (disajikan di GET /api.json).
//
//go:embed swagger.json
var APIJSON string

// SwaggerInfo mendaftarkan dokumen ke registri swag dengan nama instance "swagger".
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8081",
	BasePath:         "/",
	Schemes:          []string{"http"},
	Title:            "Delcom Blog API",
	Description:      "REST API Delcom Blog",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  APIJSON,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}

Transform your JSON Schema draft-07 to OpenAPI 3.0

[![Build Status](https://travis-ci.org/bunyk/jsonschema2openapi.svg?branch=master)](https://travis-ci.org/bunyk/jsonschema2openapi)
[![GoDoc](https://godoc.org/github.com/bunyk/jsonschema2openapi?status.svg)](https://godoc.org/github.com/bunyk/jsonschema2openapi)

PutSchemaIntoOpenAPI will put `definitions` from provided JSON Schema into your OpenAPI 3.0 specification `component.schemas`. Also it will
* Cleanup any `$schema` or other keys from JSON Reference objects
* Any reference to `#/definitions` will lead to `#/component/schemas`
* `"oneOf": [{"type": X}, {"type": "null"}]` will be replaced with `"type": X, "nullable": true`
* `oneOf` with multiple `if`s inside around one property with different values, will be transformed to oneOf with discriminate, see [here](https://github.com/bunyk/jsonschema2openapi/blob/master/translator.go#L81)

## Installation

```
go get github.com/bunyk/jsonschema2openapi
```

## Usage example

```go
import (
    "fmt"
    "github.com/bunyk/jsonschema2openapi"
)

..

APISpec, err := PutSchemaIntoOpenAPI(schema, openapitemplate)
if err == nil {
    fmt.Println(APISpec)
} else {
    fmt.Println(err.Error())
}
```
See more complete example in docs: https://godoc.org/github.com/bunyk/jsonschema2openapi#PutSchemaIntoOpenAPI

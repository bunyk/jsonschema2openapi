package jsonschema2openapi

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bunyk/jsonschema2openapi/fixtures"

	"github.com/jmoiron/jsonq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var minOpenAPI string = `{
	"components": {
		"schemas": {
			"something": "here"
		}
	}
}`

func Jq(data string) *jsonq.JsonQuery {
	var j map[string]interface{}
	err := json.Unmarshal([]byte(data), &j)
	if err != nil {
		panic(err)
	}
	return jsonq.NewQuery(j)
}

var _ = Describe("PutSchemaIntoOpenAPI", func() {
	It("should return put schema into OpenAPI successfully", func() {
		api, err := PutSchemaIntoOpenAPI(`{
			"definitions": {
				"someData": {
					"type": "string"
				}
			}
		}`, minOpenAPI)
		Expect(err).To(BeNil())

		jq := Jq(api)
		Expect(jq.String("components", "schemas", "someData", "type")).To(Equal("string"))
	})

	It("should add discriminator to oneOf generated from case", func() {
		api, err := PutSchemaIntoOpenAPI(fixtures.DiscriminatorJSON, minOpenAPI)
		Expect(err).To(BeNil())

		jq := Jq(api)
		oneOf, err := jq.Array("components", "schemas", "events.Event", "oneOf")
		Expect(err).To(BeNil())

		Expect(oneOf).To(ConsistOf(
			map[string]interface{}{
				"$ref": "#/components/schemas/v1events.Event",
			},
			map[string]interface{}{
				"$ref": "#/components/schemas/v2events.Event",
			},
		))

		discriminator, err := jq.Object("components", "schemas", "events.Event", "discriminator")
		Expect(err).To(BeNil())

		discriminatorMarshalled, err := json.Marshal(discriminator)
		Expect(err).To(BeNil())

		Expect(discriminatorMarshalled).To(MatchJSON(`{
			"propertyName": "version",
			"mapping": {
				"v1": "#/components/schemas/v1events.Event",
				"v2": "#/components/schemas/v2events.Event"
			}
		}`))
	})
})

var _ = Describe("getConstant", func() {
	It("Should return name and value of constant successfully", func() {
		var jsonData map[string]interface{}
		_ = json.Unmarshal([]byte(`{
			"properties": { "PROPERTY": { "enum": [ "CASE1" ] } }
		}`), &jsonData)
		ok, name, value := getConstant(jsonData)
		Expect(ok).To(BeTrue())
		Expect(name).To(Equal("PROPERTY"))
		Expect(value).To(Equal("CASE1"))
	})
	It("Should fail when there are more than one value", func() {
		var jsonData map[string]interface{}
		_ = json.Unmarshal([]byte(`{
			"properties": { "PROPERTY": { "enum": [ "CASE1", "CASE2" ] } }
		}`), &jsonData)
		ok, _, _ := getConstant(jsonData)
		Expect(ok).To(BeFalse())
	})
	It("Should fail for json without properties", func() {
		var jsonData map[string]interface{}
		_ = json.Unmarshal([]byte(`{
			"properties": "schmoperties"
		}`), &jsonData)
		ok, _, _ := getConstant(jsonData)
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("getRef", func() {
	It("Should return reference from object successfully", func() {
		var jsonData map[string]interface{}
		_ = json.Unmarshal([]byte(`{ "$ref": "somewhere in time"}`), &jsonData)
		ok, ref := getRef(jsonData)
		Expect(ok).To(BeTrue())
		Expect(ref).To(Equal("somewhere in time"))
	})
	It("Should return false where there are no $ref field", func() {
		var jsonData map[string]interface{}
		_ = json.Unmarshal([]byte(`{ "ref": "lost somewhere in time"}`), &jsonData)
		ok, _ := getRef(jsonData)
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("getCases", func() {
	It("Should return cases from object successfully", func() {
		var jsonData map[string]interface{}
		err := json.Unmarshal([]byte(`{"oneOf": [
			{
				"if": { "properties": { "PROPERTY": { "enum": [ "CASE1" ] } } },
				"then": { "$ref": "REF1" },
				"else": { "properties": { "PROPERTY": { "enum": [ "CASE1" ] } } }
			},
			{
				"if": { "properties": { "PROPERTY": { "enum": [ "CASE2" ] } } },
				"then": { "$ref": "REF2" },
				"else": { "properties": { "PROPERTY": { "enum": [ "CASE2" ] } } }
			}
		]}`), &jsonData)
		Expect(err).To(BeNil())
		ok, cases := getCases(jsonData)
		Expect(ok).To(BeTrue())
		Expect(cases).To(Equal(casesResult{
			Property: "PROPERTY",
			Cases:    []string{"CASE1", "CASE2"},
			Refs:     []string{"REF1", "REF2"},
		}))
	})
	It("Should return false when that is not cases", func() {
		var jsonData map[string]interface{}
		_ = json.Unmarshal([]byte(`{ "ref": "lost somewhere in time"}`), &jsonData)
		ok, _ := getCases(jsonData)
		Expect(ok).To(BeFalse())
	})
})

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "All Tests Suite")
}

func ExamplePutSchemaIntoOpenAPI() {
	fullAPI, _ := PutSchemaIntoOpenAPI(
		`{
		"definitions": {
			"Data": {
				"oneOf": [
					{
						"if": { "properties": { "type": { "enum": [ "stringornull" ] } } },
						"then": { "properties": {"payload": {"$ref": "#/definitions/Payload1" } } },
						"else": { "properties": { "type": { "enum": [ "stringornull" ] } } }
					},
					{
						"if": { "properties": { "type": { "enum": [ "int" ] } } },
						"then": { "properties": {"payload": {"$ref": "#/definitions/Payload2" } } },
						"else": { "properties": { "type": { "enum": [ "int" ] } } }
					}
				]
			},
			"Payload1": {
				"oneOf": [
					{ "type": "string" },
					{ "type": "null" }
				]
			}, 
			"Payload2": {
				"type": "int"
			}
		}
	}`, `{
  "openapi": "3.0.0",
  "info": {
    "version": "1.0.0",
    "title": "My API"
  },
  "servers": [],
  "paths": {
    "/data": {
      "get": {
        "summary": "Get data",
        "responses": {
          "200": {
            "description": "Data response",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/Data"
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
    }
  }
}`)
	fmt.Println(fullAPI)
	// Output:
	// {
	//  "components": {
	//   "schemas": {
	//    "Data": {
	//     "oneOf": [
	//      {
	//       "anyOf": [
	//        {
	//         "allOf": [
	//          {
	//           "properties": {
	//            "type": {
	//             "enum": [
	//              "stringornull"
	//             ]
	//            }
	//           }
	//          },
	//          {
	//           "properties": {
	//            "payload": {
	//             "$ref": "#/components/schemas/Payload1"
	//            }
	//           }
	//          }
	//         ]
	//        },
	//        {
	//         "allOf": [
	//          {
	//           "not": {
	//            "properties": {
	//             "type": {
	//              "enum": [
	//               "stringornull"
	//              ]
	//             }
	//            }
	//           }
	//          },
	//          {
	//           "properties": {
	//            "type": {
	//             "enum": [
	//              "stringornull"
	//             ]
	//            }
	//           }
	//          }
	//         ]
	//        }
	//       ]
	//      },
	//      {
	//       "anyOf": [
	//        {
	//         "allOf": [
	//          {
	//           "properties": {
	//            "type": {
	//             "enum": [
	//              "int"
	//             ]
	//            }
	//           }
	//          },
	//          {
	//           "properties": {
	//            "payload": {
	//             "$ref": "#/components/schemas/Payload2"
	//            }
	//           }
	//          }
	//         ]
	//        },
	//        {
	//         "allOf": [
	//          {
	//           "not": {
	//            "properties": {
	//             "type": {
	//              "enum": [
	//               "int"
	//              ]
	//             }
	//            }
	//           }
	//          },
	//          {
	//           "properties": {
	//            "type": {
	//             "enum": [
	//              "int"
	//             ]
	//            }
	//           }
	//          }
	//         ]
	//        }
	//       ]
	//      }
	//     ]
	//    },
	//    "Payload1": {
	//     "nullable": true,
	//     "type": "string"
	//    },
	//    "Payload2": {
	//     "type": "int"
	//    }
	//   }
	//  },
	//  "info": {
	//   "title": "My API",
	//   "version": "1.0.0"
	//  },
	//  "openapi": "3.0.0",
	//  "paths": {
	//   "/data": {
	//    "get": {
	//     "responses": {
	//      "200": {
	//       "content": {
	//        "application/json": {
	//         "schema": {
	//          "$ref": "#/components/schemas/Data"
	//         }
	//        }
	//       },
	//       "description": "Data response"
	//      }
	//     },
	//     "summary": "Get data"
	//    }
	//   }
	//  },
	//  "servers": []
	// }
}

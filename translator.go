package jsonschema2openapi

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/jmoiron/jsonq"
)

// PutSchemaIntoOpenAPI returns OpenAPI spec based on template and JSONSchema which is added to its components/schemas
func PutSchemaIntoOpenAPI(schemaJSON, openAPITemplate string) (string, error) {
	var schema map[string]interface{}
	err := json.Unmarshal([]byte(schemaJSON), &schema)
	if err != nil {
		return "", fmt.Errorf("Error %s. Was not able to parse JSON schema", err.Error())
	}

	// Load OpenAPI spec from string constant
	var tmpl map[string]interface{}
	err = json.Unmarshal([]byte(openAPITemplate), &tmpl)
	if err != nil {
		return "", fmt.Errorf("Error %s. Not able to parse OpenAPI template", err.Error())
	}

	// Get componets.schemas object to fill up
	jq := jsonq.NewQuery(tmpl)
	schemas, err := jq.Object("components", "schemas")
	if err != nil {
		return "", fmt.Errorf("Error %s. Bad JSON template, no component.schemas object", err.Error())
	}

	// Now add definitions from our schema to that OpenAPI
	schema4OpenAPI := schema["definitions"].(map[string]interface{})
	for k, v := range TranslateDefinitions(schema4OpenAPI) {
		schemas[k] = v
	}

	// And output what we got
	res, _ := json.MarshalIndent(tmpl, "", " ")
	return string(res), nil
}

// TranslateDefinitions translates JSON Schema definitons object to components/schemas of OpenAPI
func TranslateDefinitions(definitions map[string]interface{}) map[string]interface{} {
	schema4OpenAPI := replaceRefs(
		definitions,
		"#/definitions/", "#/components/schemas/",
	).(map[string]interface{})
	schema4OpenAPI = replaceNullable(schema4OpenAPI).(map[string]interface{})
	schema4OpenAPI = discriminate(schema4OpenAPI).(map[string]interface{})
	return materialImplication(schema4OpenAPI).(map[string]interface{})
}

// Recursively replace old substring to new in any value of $ref key in json
func replaceRefs(jsonData interface{}, old, new string) interface{} {
	switch jsonData.(type) {
	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, v := range jsonData.(map[string]interface{}) {
			if k == "$ref" { // if schema has $ref
				return map[string]interface{}{ // we do not need any other fields there
					"$ref": strings.Replace(v.(string), old, new, 1),
				}
			}
			res[k] = replaceRefs(v, old, new)
		}
		return res
	case []interface{}:
		res := make([]interface{}, 0)
		for _, v := range jsonData.([]interface{}) {
			res = append(res, replaceRefs(v, old, new))
		}
		return res
	default:
	}
	return jsonData
}

// discriminate replaces any occurences of
//
//	"oneOf": [
//		{
//			"if": { "properties": { "PROPERTY": { "enum": [ "CASE1" ] } } },
//			"then": { "$ref": "REF1" }
//			"else": { "properties": { "PROPERTY": { "enum": [ "CASE1" ] } } }
//		},
//		{
//			"if": { "properties": { "PROPERTY": { "enum": [ "CASE2" ] } } },
//			"then": { "$ref": "REF2" }
//			"else": { "properties": { "PROPERTY": { "enum": [ "CASE2" ] } } }
//		}
//	]
//
// with
//
// "oneOf": [
//		{ "$ref": "REF1" },
//		{ "$ref": "REF2" },
//	],
//	"discriminator": {
//		"propertyName": "PROPERTY",
//		"mapping": {
//			"CASE1": "REF1",
//			"CASE2": "REF2"
//		}
//	}
//
func discriminate(jsonData interface{}) interface{} {
	switch v := jsonData.(type) {
	case map[string]interface{}:
		ok, cases := getCases(v)
		if ok {
			v["oneOf"] = reflist(cases.Refs)
			discriminator := make(map[string]interface{})
			discriminator["propertyName"] = cases.Property
			discriminator["mapping"] = cases2refmapping(cases.Cases, cases.Refs)
			v["discriminator"] = discriminator
		} else { // Go deeper
			for k, subschema := range v {
				v[k] = discriminate(subschema)
			}
		}
		return v
	case []interface{}:
		res := make([]interface{}, 0)
		for _, elem := range v {
			res = append(res, discriminate(elem))
		}
		return res
	default:
		return v
	}
}

// https://en.wikipedia.org/wiki/Material_implication_(rule_of_inference)
// Turn
// {
// 	"if": CONDITION
// 	"then": SCHEMA1
// 	"else": SCHEMA2
// }
//
// To
//
// {
//   "anyOf": [
//     { "allOf": [ CONDITION, SCHEMA1 ] },
//     { "allOf": [ {"not": CONDITION }, SCHEMA2 ] }
//   ]
// }
func materialImplication(jsonData interface{}) interface{} {
	switch v := jsonData.(type) {
	case map[string]interface{}:
		ok, ifschema, thenschema, elseschema := getCondition(v)
		if ok {
			v["anyOf"] = []interface{}{
				map[string][]interface{}{
					"allOf": []interface{}{
						ifschema, thenschema,
					},
				},
				map[string][]interface{}{
					"allOf": []interface{}{
						map[string]interface{}{
							"not": ifschema,
						},
						elseschema,
					},
				},
			}
			delete(v, "if")
			delete(v, "then")
			delete(v, "else")
		} else { // Go deeper
			for k, subschema := range v {
				v[k] = materialImplication(subschema)
			}
		}
		return v
	case []interface{}:
		res := make([]interface{}, 0)
		for _, elem := range v {
			res = append(res, materialImplication(elem))
		}
		return res
	default:
		return v
	}
}

func getCondition(jsonData interface{}) (ok bool, ifschema, thenschema, elseschema interface{}) {
	obj, ok := jsonData.(map[string]interface{})
	if !ok {
		return
	}
	ok = false
	jq := jsonq.NewQuery(obj)
	ifschema, err := jq.Object("if")
	if err != nil {
		return
	}
	thenschema, err = jq.Object("then")
	if err != nil {
		return
	}
	elseschema, err = jq.Object("else")
	if err != nil {
		return
	}
	ok = true
	return
}

// ["a", "b"], ["A", "B"] => {"a": "A", "b":"B"}
func cases2refmapping(cases, refs []string) map[string]string {
	res := make(map[string]string)
	for i, ref := range refs {
		res[cases[i]] = ref
	}
	return res
}

// ["a", "b"] => [{"$ref": "a"}, {"$ref": "b"}]
func reflist(refs []string) []map[string]string {
	res := make([]map[string]string, len(refs))
	for i, r := range refs {
		res[i] = map[string]string{
			"$ref": r,
		}
	}
	return res
}

// casesResult is struct to hold data from cases pattern in JSON
type casesResult struct {
	Property string
	Cases    []string
	Refs     []string
}

// getCases checks if JSON matches oneOf pattern described in comment for discriminate
// Returns ok true in case of match, name of property to discriminate by, and list of cases and respective references
func getCases(jsonData interface{}) (ok bool, res casesResult) {
	obj, ok := jsonData.(map[string]interface{})
	if !ok {
		return
	}
	jq := jsonq.NewQuery(obj)
	oneOf, err := jq.Array("oneOf")
	if err != nil {
		return false, res
	}
	if len(oneOf) < 1 {
		return false, res
	}
	for _, caseIface := range oneOf {
		ok, name, value, ref := getCase(caseIface)
		if !ok {
			return false, res
		}
		if res.Property != "" && res.Property != name {
			// Properties are different in different oneOf cases
			return false, res
		}
		res.Property = name
		res.Cases = append(res.Cases, value)
		res.Refs = append(res.Refs, ref)
	}
	if res.Property == "" { // No cycle iterations happened above
		return false, res
	}
	return true, res
}

func getCase(jsonData interface{}) (ok bool, name, value, ref string) {
	ok, ifschema, thenschema, elseschema := getCondition(jsonData)
	if !ok {
		return
	}
	ok, name, value = getConstant(ifschema)
	if !ok {
		return
	}
	ok = false
	if !reflect.DeepEqual(ifschema, elseschema) {
		return // else schema should equal condition schema, to always fail in else
	}
	ok, ref = getRef(thenschema)
	return
}

// getRef checks if JSON matches { "$ref": "REF"}
// and returns match result, and REF value
func getRef(jsonData interface{}) (ok bool, ref string) {
	obj, ok := jsonData.(map[string]interface{})
	if !ok {
		return false, ""
	}
	jq := jsonq.NewQuery(obj)
	ref, err := jq.String("$ref")
	if err != nil {
		return false, ""
	}
	return true, ref
}

// getConstant checks if JSON matches pattern { "properties": { "PROPERTY": { "enum": [ "CASE1" ] } } }
// and returns match result, constant name and value
func getConstant(jsonData interface{}) (ok bool, name string, value string) {
	obj, ok := jsonData.(map[string]interface{})
	if !ok {
		return false, "", ""
	}
	jq := jsonq.NewQuery(obj)
	properties, err := jq.Object("properties")
	if err != nil {
		return false, "", ""
	}
	for k, v := range properties {
		if name != "" {
			return false, "", "" // properties have more than one property
		}
		name = k
		enum, ok := v.(map[string]interface{})
		if !ok {
			return false, "", ""
		}
		casesIface, ok := enum["enum"]
		if !ok {
			return false, "", ""
		}
		cases, ok := casesIface.([]interface{})
		if !ok || (len(cases) != 1) {
			return false, "", ""
		}
		value, ok = cases[0].(string)
		if !ok {
			return false, "", ""
		}
	}
	return true, name, value
}

// Recursively replace "oneOf": [{"type": X}, {"type": "null"}]
// with "type": X, "nullable": true
func replaceNullable(jsonData interface{}) interface{} {
	switch jsonData.(type) {
	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, v := range jsonData.(map[string]interface{}) {
			nullableType := isNullable(v)
			if k == "oneOf" && nullableType != "" {
				res["type"] = nullableType
				res["nullable"] = true
			} else {
				res[k] = replaceNullable(v)
			}
		}
		return res
	case []interface{}:
		res := make([]interface{}, 0)
		for _, v := range jsonData.([]interface{}) {
			res = append(res, replaceNullable(v))
		}
		return res
	default:
	}
	return jsonData
}

// Check if json is of the form [{"type": X}, {"type": "null"}]
// and return type that is nullable
// Otherwise return empty string
func isNullable(jsonData interface{}) string {
	js, ok := jsonData.([]interface{})
	if !ok { // not an array
		return ""
	}
	if len(js) != 2 {
		return ""
	}
	res := ""
	nulled := false
	for _, v := range js {
		typeDeclaration, ok := v.(map[string]interface{})
		if !ok {
			return ""
		}
		tIf, ok := typeDeclaration["type"]
		if !ok {
			return ""
		}
		t, ok := tIf.(string)
		if !ok {
			return ""
		}
		if t == "null" {
			nulled = true
		} else {
			res = t
		}
	}
	if !nulled {
		return ""
	}
	return res
}

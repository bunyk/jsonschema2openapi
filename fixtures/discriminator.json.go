package fixtures

var DiscriminatorJSON = `
{
    "definitions": {
        "Events": {
            "items": {
                "$ref": "#/components/schemas/events.Event"
            },
            "type": "array"
        },
        "events.Event": {
            "additionalProperties": true,
            "oneOf": [
                {
                    "else": {
                        "properties": {
                            "version": {
                                "enum": [
                                    "v1"
                                ]
                            }
                        }
                    },
                    "if": {
                        "properties": {
                            "version": {
                                "enum": [
                                    "v1"
                                ]
                            }
                        }
                    },
                    "then": {
                        "$ref": "#/components/schemas/v1events.Event"
                    }
                },
                {
                    "else": {
                        "properties": {
                            "version": {
                                "enum": [
                                    "v2"
                                ]
                            }
                        }
                    },
                    "if": {
                        "properties": {
                            "version": {
                                "enum": [
                                    "v2"
                                ]
                            }
                        }
                    },
                    "then": {
                        "$ref": "#/components/schemas/v2events.Event"
                    }
                }
            ],
            "properties": {
                "timestamp": {
                    "format": "date-time",
                    "type": "string"
                },
                "type": {
                    "pattern": "^\\S",
                    "type": "string"
                },
                "uuid": {
                    "type": "string"
                },
                "version": {
                    "pattern": "^v\\d+$",
                    "type": "string"
                }
            },
            "required": [
                "type",
                "version",
                "timestamp"
            ],
            "type": "object"
        }
    }
}
`

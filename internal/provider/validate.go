package provider

import "encoding/json"

func ValidateJSON(content string) error {
	var v interface{}
	return json.Unmarshal([]byte(content), &v)
}

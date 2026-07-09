package provider

import (
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type diagList = diag.Diagnostics

// jsonSemanticallyEqual reports whether two JSON documents are equal after
// decoding, ignoring whitespace and key order. Invalid JSON is never equal.
func jsonSemanticallyEqual(a, b string) bool {
	var av, bv any
	if err := json.Unmarshal([]byte(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

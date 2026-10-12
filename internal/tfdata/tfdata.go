// Package tfdata converts Terraform schema values without unchecked type assertions.
// Each function returns the zero value if v holds another type.
package tfdata

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

// String returns v as a string.
func String(v any) string {
	s, _ := v.(string)

	return s
}

// Int returns v as an int.
func Int(v any) int {
	i, _ := v.(int)

	return i
}

// Bool returns v as a bool.
func Bool(v any) bool {
	b, _ := v.(bool)

	return b
}

// List returns v as a []any, the SDK's TypeList value.
func List(v any) []any {
	l, _ := v.([]any)

	return l
}

// Map returns v as a map[string]any, the SDK's nested block value.
func Map(v any) map[string]any {
	m, _ := v.(map[string]any)

	return m
}

// Set returns v as a *schema.Set, the SDK's TypeSet value.
func Set(v any) *schema.Set {
	s, _ := v.(*schema.Set)

	return s
}

// Strings returns a TypeList of strings as a []string.
func Strings(v any) []string {
	raw := List(v)
	out := make([]string, 0, len(raw))

	for _, item := range raw {
		out = append(out, String(item))
	}

	return out
}

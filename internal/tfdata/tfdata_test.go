package tfdata_test

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/yellowhat/terraform-provider-hetznerrobot/internal/tfdata"
)

func TestConvert(t *testing.T) {
	t.Parallel()

	set := schema.NewSet(schema.HashInt, []any{1})

	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "String", got: tfdata.String("a"), want: "a"},
		{name: "String wrong type", got: tfdata.String(1), want: ""},
		{name: "Int", got: tfdata.Int(1), want: 1},
		{name: "Int nil", got: tfdata.Int(nil), want: 0},
		{name: "Bool", got: tfdata.Bool(true), want: true},
		{name: "List", got: tfdata.List([]any{"a"}), want: []any{"a"}},
		{name: "Map", got: tfdata.Map(map[string]any{"k": "v"}), want: map[string]any{"k": "v"}},
		{name: "Set", got: tfdata.Set(set), want: set},
		{name: "Strings", got: tfdata.Strings([]any{"1", 2, "3"}), want: []string{"1", "", "3"}},
		{name: "Strings nil", got: tfdata.Strings(nil), want: []string{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if !reflect.DeepEqual(test.want, test.got) {
				t.Errorf("want %#v, got %#v", test.want, test.got)
			}
		})
	}
}

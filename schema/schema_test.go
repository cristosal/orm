package schema_test

import (
	"testing"

	"github.com/cristosal/orm/schema"
)

func TestInferStructBase(t *testing.T) {
	type tc struct {
		name               string
		value              any
		expectedTypeName   string
		expectedFieldValue string
	}

	type foo struct {
		Name string
	}

	tt := []tc{
		{
			name:               "single struct",
			value:              foo{Name: "bar"},
			expectedTypeName:   "foo",
			expectedFieldValue: "bar",
		},
		{
			name:               "ptr to struct",
			value:              &foo{Name: "bar"},
			expectedTypeName:   "foo",
			expectedFieldValue: "bar",
		},
		{
			name:               "struct slice",
			value:              []foo{},
			expectedTypeName:   "foo",
			expectedFieldValue: "",
		},
		{
			name:               "ptr to struct slice",
			value:              &[]foo{},
			expectedTypeName:   "foo",
			expectedFieldValue: "",
		},
	}

	for _, tc := range tt {
		typ, val, err := schema.Reflect(tc.value)
		if err != nil {
			t.Fatal(err)
		}

		if typ.Name() != tc.expectedTypeName {
			t.Fatalf("%s: expected name to be foo", tc.name)
		}

		if val.FieldByName("Name").String() != tc.expectedFieldValue {
			t.Fatalf("%s: expected name to be bar", tc.name)
		}
	}
}

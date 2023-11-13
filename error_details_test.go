package errkit_test

import (
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/kastenhq/errkit"
)

type testStruct struct {
	Foo string
	Bar int
	Baz *testStruct
}

func TestToErrorDetails(t *testing.T) {
	cases := []struct {
		testName string
		args     []any
		expected errkit.ErrorDetails
	}{
		{
			testName: "ErrorDetails as an argument",
			args:     []any{errkit.ErrorDetails{"key": "value"}},
			expected: errkit.ErrorDetails{"key": "value"},
		},
		{
			testName: "Sequence of keys and values of any type",
			args:     []any{"string_key", "string value", "int key", 123, "struct key", testStruct{Foo: "aaa", Bar: 123, Baz: &testStruct{Foo: "bbb", Bar: 234}}},
			expected: errkit.ErrorDetails{"string_key": "string value", "int key": 123, "struct key": testStruct{Foo: "aaa", Bar: 123, Baz: &testStruct{Foo: "bbb", Bar: 234}}},
		},
		{
			testName: "Odd number of arguments",
			args:     []any{"key_1", 1, "key_2"},
			expected: errkit.ErrorDetails{"key_1": 1, "key_2": "NOVAL"},
		},
		{
			testName: "Argument which is supposed to be a key is not a string",
			args:     []any{123, 456},
			expected: errkit.ErrorDetails{"BADKEY:(123)": 456},
		},
	}

	for _, tc := range cases {
		t.Run(tc.testName, func(t *testing.T) {
			c := qt.New(t)
			result := errkit.ToErrorDetails(tc.args)
			c.Assert(result, qt.DeepEquals, tc.expected)
		})
	}
}

package b_types

import (
	"fmt"
	"testing"
)

func TestStructValueParam(t *testing.T) {

	type T struct {
		Buffer string
	}

	valueFunc := func(v T) T {
		fmt.Printf("valueFunc-input: %p\n", &v)
		fmt.Printf("valueFunc-input.value: %+v\n", v)
		o := v
		fmt.Printf("valueFunc-output: %p\n", &o)
		return o
	}

	v := T{Buffer: "hello"}
	fmt.Printf("input: %p\n", &v)

	o := valueFunc(v)
	fmt.Printf("output: %p\n", &o)
}

type TT struct {
	Body string
}

func (t TT) Len() int {
	fmt.Printf("TT.Len: %p\n", &t)
	return len(t.Body)
}
func (t TT) Clear() {
	fmt.Printf("TT.Clear: %p\n", &t)
	t.Body = ""
}

func TestStructMethod(t *testing.T) {
	tt := TT{
		Body: "hello",
	}
	fmt.Printf("input: %p\n", &tt)
	tt.Len()
	tt.Clear()
}

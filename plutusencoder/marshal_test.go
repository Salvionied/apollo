package plutusencoder

import "testing"

func TestMarshalNilPointer(t *testing.T) {
	_, err := MarshalPlutus((*SimpleDatum)(nil))
	if err == nil {
		t.Error("expected error for nil pointer")
	}
}

func TestMarshalNonStruct(t *testing.T) {
	x := 42
	_, err := MarshalPlutus(&x)
	if err == nil {
		t.Error("expected error for non-struct")
	}
}

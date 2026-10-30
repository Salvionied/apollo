package backendutil

import (
	"strings"
	"testing"
)

func TestDecodeLimitedArray(t *testing.T) {
	rows, err := DecodeLimitedArray[int]([]byte("[1,2]"), 2)
	if err != nil || len(rows) != 2 || rows[0] != 1 || rows[1] != 2 {
		t.Fatalf("decoded rows = %v, %v", rows, err)
	}
	for _, input := range []string{"[1,2,3]", "null", "[1] true"} {
		if _, err := DecodeLimitedArray[int]([]byte(input), 2); err == nil {
			t.Fatalf("expected %q to fail", input)
		}
	}
	if _, err := DecodeLimitedArray[int]([]byte("["+strings.Repeat("1,", 1000)+"1]"), 1000); err == nil {
		t.Fatal("expected row 1001 to fail before decoding")
	}
}

package blueprint

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/blinklabs-io/plutigo/data"
)

func TestSharedRefChainParsesQuickly(t *testing.T) {
	const levels = 22
	var b strings.Builder
	b.WriteString(`{"preamble":{"title":"chain","version":"1.0.0","plutusVersion":"v3"},"validators":[{"title":"v","redeemer":{"schema":{"$ref":"#/definitions/D0"}}}],"definitions":{`)
	for i := 0; i < levels; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"D%d":{"dataType":"constructor","index":0,"fields":[{"$ref":"#/definitions/D%d"},{"$ref":"#/definitions/D%d"}]}`, i, i+1, i+1)
	}
	fmt.Fprintf(&b, `,"D%d":{"dataType":"integer"}}}`, levels)

	start := time.Now()
	if _, err := Parse([]byte(b.String())); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("shared $ref chain took %s", elapsed)
	}
}

func TestTypedNilPlutusDataIsRejected(t *testing.T) {
	const raw = `{
		"preamble": {"title": "nil", "version": "1.0.0", "plutusVersion": "v3"},
		"validators": [{"title": "v", "redeemer": {"schema": {"dataType": "integer"}}}]
	}`
	bp, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	val, err := bp.Validator("v")
	if err != nil {
		t.Fatal(err)
	}
	var integer *data.Integer
	if err := bp.ValidateData(&val.Redeemer.Single.Schema, integer); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("typed nil integer: %v", err)
	}
	list := data.NewList((*data.Integer)(nil))
	const listRaw = `{
		"preamble": {"title": "nil-list", "version": "1.0.0", "plutusVersion": "v3"},
		"validators": [{"title": "v", "redeemer": {"schema": {"dataType": "list", "items": {"dataType": "integer"}}}}]
	}`
	bp, err = Parse([]byte(listRaw))
	if err != nil {
		t.Fatal(err)
	}
	val, err = bp.Validator("v")
	if err != nil {
		t.Fatal(err)
	}
	if err := bp.ValidateData(&val.Redeemer.Single.Schema, list); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("nil list item: %v", err)
	}
}

package backendutil

import "testing"

func TestUtxoQueryBudgetBoundsTotals(t *testing.T) {
	var items UtxoQueryBudget
	if err := items.AddPage(MaxUtxosPerQuery, 0); err != nil {
		t.Fatal(err)
	}
	if err := items.AddPage(1, 0); err == nil {
		t.Fatal("expected the aggregate item limit to reject another UTxO")
	}

	var bytes UtxoQueryBudget
	if err := bytes.AddPage(0, MaxUtxoBytesPerQuery-1); err != nil {
		t.Fatal(err)
	}
	if err := bytes.AddBytes(1); err != nil {
		t.Fatal(err)
	}
	if err := bytes.AddPage(0, 1); err == nil {
		t.Fatal("expected the aggregate byte limit to reject another byte")
	}
}

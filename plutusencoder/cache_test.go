package plutusencoder

import (
	"fmt"
	"sync"
	"testing"

	"github.com/blinklabs-io/plutigo/data"
)

type cacheList struct {
	_ struct{} `plutusType:"DefList" plutusConstr:"1"`
	N int64    `plutusType:"Int"`
}

type cacheMap struct {
	_ struct{} `plutusType:"Map"`
	N int64    `plutusType:"Int" plutusKey:"n"`
}

type firstBlankWins struct {
	_ struct{} `plutusType:"DefList" plutusConstr:"1"`
	_ struct{} `plutusType:"Map"`
	N int64    `plutusType:"Int"`
}

type badContainer struct {
	_ struct{} `plutusType:"NotAContainer"`
	N int64    `plutusType:"Int"`
}

type wideList struct {
	_     struct{} `plutusType:"DefList" plutusConstr:"0"`
	Items []int64  `plutusType:"DefList"`
}

type wideMap struct {
	_ struct{}         `plutusType:"DefList" plutusConstr:"0"`
	M map[string]int64 `plutusType:"Map"`
}

type twelveFields struct {
	_                struct{} `plutusType:"DefList" plutusConstr:"0"`
	A, B, C, D, E, F int64    `plutusType:"Int"`
	G, H, I, J, K, L int64    `plutusType:"Int"`
}

func TestStructCacheKeepsEachType(t *testing.T) {
	listPD, err := MarshalPlutus(&cacheList{N: 5})
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%T", listPD); got != "*data.Constr" {
		t.Fatalf("list type encoded as %s", got)
	}
	mapPD, err := MarshalPlutus(&cacheMap{N: 5})
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%T", mapPD); got != "*data.Map" {
		t.Fatalf("map type encoded as %s", got)
	}
	var decoded cacheList
	if err := UnmarshalPlutus(listPD, &decoded); err != nil || decoded.N != 5 {
		t.Fatalf("list decode: %v %+v", err, decoded)
	}
}

func TestFirstBlankFieldWins(t *testing.T) {
	pd, err := MarshalPlutus(&firstBlankWins{N: 5})
	if err != nil {
		t.Fatal(err)
	}
	constr, ok := pd.(*data.Constr)
	if !ok || !constrTagEqual(constr.Tag, 1) || len(constr.Fields) != 1 {
		t.Fatalf("expected constructor 1 with one field, got %#v", pd)
	}
	integer, ok := constr.Fields[0].(*data.Integer)
	if !ok || integer.Inner == nil || integer.Inner.Int64() != 5 {
		t.Fatalf("expected integer 5, got %#v", constr.Fields[0])
	}
}

func TestCachedTagErrorIsStable(t *testing.T) {
	first, err := MarshalPlutus(&badContainer{N: 1})
	if err == nil {
		t.Fatal("expected a bad container tag to fail")
	}
	second, err2 := MarshalPlutus(&badContainer{N: 1})
	if err2 == nil || err2.Error() != err.Error() || first != nil || second != nil {
		t.Fatalf("cached error changed: %v then %v", err, err2)
	}
}

func TestConcurrentFirstUse(t *testing.T) {
	var wg sync.WaitGroup
	errCh := make(chan error, 32)
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pd, err := MarshalPlutus(&cacheList{N: 7})
			if err != nil {
				errCh <- err
				return
			}
			var decoded cacheList
			if err := UnmarshalPlutus(pd, &decoded); err != nil || decoded.N != 7 {
				errCh <- fmt.Errorf("decode %v %+v", err, decoded)
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}

func TestPreallocCapStillDecodes(t *testing.T) {
	items := make([]int64, 1025)
	for i := range items {
		items[i] = int64(i)
	}
	pd, err := MarshalPlutus(&wideList{Items: items})
	if err != nil {
		t.Fatal(err)
	}
	var decoded wideList
	if err := UnmarshalPlutus(pd, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Items) != 1025 || decoded.Items[1024] != 1024 {
		t.Fatalf("list length %d", len(decoded.Items))
	}

	values := make(map[string]int64, 1025)
	for i := 0; i < 1025; i++ {
		values[fmt.Sprintf("k%04d", i)] = int64(i)
	}
	pd, err = MarshalPlutus(&wideMap{M: values})
	if err != nil {
		t.Fatal(err)
	}
	var decodedMap wideMap
	if err := UnmarshalPlutus(pd, &decodedMap); err != nil {
		t.Fatal(err)
	}
	if len(decodedMap.M) != 1025 || decodedMap.M["k1024"] != 1024 {
		t.Fatalf("map length %d", len(decodedMap.M))
	}
}

func BenchmarkTwelveFieldStruct(b *testing.B) {
	original := twelveFields{A: 1, B: 2, C: 3, D: 4, E: 5, F: 6, G: 7, H: 8, I: 9, J: 10, K: 11, L: 12}
	pd, err := MarshalPlutus(&original)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		encoded, err := MarshalPlutus(&original)
		if err != nil {
			b.Fatal(err)
		}
		var decoded twelveFields
		if err := UnmarshalPlutus(encoded, &decoded); err != nil {
			b.Fatal(err)
		}
	}
	_ = pd
}

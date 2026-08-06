package apollo

import (
	"math/rand"
	"testing"

	"github.com/blinklabs-io/gouroboros/ledger/common"
)

// The ledger cannot hold a change output below the minimum UTxO value, so a
// selection whose leftover lands between zero and that minimum leaves the
// builder no choice but to fold the remainder into the fee, where the user
// loses it outright. MACS scores candidates by
//
//	P(u,c) = v / (|v - avg| + 1)
//
// which peaks near the pool mean, so a payment near the mean is exactly the
// case that strands its change — and preferring the candidate that covers the
// deficit alone makes it worse, because the tightest cover leaves the smallest
// change. These tests pin the floor that steers around the dead band.

// realisticPool is a mixed wallet of ADA-only UTxOs from 1 to 21 ADA, the shape
// where the mean sits in the middle of the payment range.
func realisticPool(t *testing.T, n int, seed int64) []common.Utxo {
	t.Helper()
	addr := testAddress(t)
	rng := rand.New(rand.NewSource(seed))
	pool := make([]common.Utxo, 0, n)
	for i := range n {
		//nolint:gosec // test sequence
		pool = append(pool, benchUtxo(
			addr, uint64(i), 1_000_000+rng.Uint64()%20_000_000, nil,
		))
	}
	return pool
}

// buildWith builds a real transfer of pay lovelace out of pool using sel.
func buildWith(
	t *testing.T,
	sel CoinSelector,
	pool []common.Utxo,
	pay int64,
) (*Apollo, error) {
	t.Helper()
	addr := testAddress(t)
	cc := setupFixedContext()
	for _, u := range pool {
		cc.AddUtxo(addr, u)
	}
	p, err := NewPayment(validTestAddrBech32, pay, nil)
	if err != nil {
		t.Fatal(err)
	}
	return New(cc).
		SetCoinSelector(sel).
		SetWallet(NewExternalWallet(addr)).
		AddPayment(p).
		SetTtl(50_000_000).
		Complete()
}

// TestMACSDoesNotStrandChangeAsFee is the regression guard. Without the
// MinChange floor the configured selector burned its change on a majority of
// payment amounts against this pool, paying up to 1,011,827 lovelace where the
// size-based fee is 168,537 — six times the fee, with the difference simply
// lost.
func TestMACSDoesNotStrandChangeAsFee(t *testing.T) {
	pool := realisticPool(t, 400, 7)

	var stranded, n int
	var worst uint64
	for pay := int64(2_000_000); pay <= 18_000_000; pay += 250_000 {
		a, err := buildWith(t, NewMACSSelector(), pool, pay)
		if err != nil {
			t.Fatalf("pay=%d: %v", pay, err)
		}
		n++
		body := a.GetTx().Body
		if body.TxFee > worst {
			worst = body.TxFee
		}
		// A single output means the change output was dropped, so its value
		// went into the fee.
		if len(body.Outputs()) == 1 {
			stranded++
			t.Errorf(
				"pay=%d stranded its change: fee=%d with one output",
				pay, body.TxFee,
			)
		}
	}
	if n == 0 {
		t.Fatal("no transactions were built")
	}
	// Every transaction here is one input and two outputs, so the fee should
	// not exceed the size-based fee for that shape by any margin.
	const plainTransferFee = 168_537
	if worst > plainTransferFee {
		t.Errorf(
			"worst fee %d exceeds the plain 1-in 2-out fee %d, so change is "+
				"still being absorbed",
			worst, plainTransferFee,
		)
	}
	t.Logf("%d payment amounts, %d stranded, worst fee %d", n, stranded, worst)
}

// TestMACSSelectsWhenEveryCloserStrandsChange covers the cost of the floor: it
// must steer selection, never block it. When no candidate can leave an
// emittable change amount the selector still has to return a covering set.
func TestMACSSelectsWhenEveryCloserStrandsChange(t *testing.T) {
	addr := testAddress(t)
	// One UTxO, and it leaves change far inside the dead band.
	pool := []common.Utxo{benchUtxo(addr, 1, 10_400_000, nil)}

	a, err := buildWith(t, NewMACSSelector(), pool, 10_000_000)
	if err != nil {
		t.Fatalf("selection blocked by the MinChange floor: %v", err)
	}
	body := a.GetTx().Body
	if len(body.Inputs()) != 1 {
		t.Errorf("expected the single UTxO to be spent, got %d inputs",
			len(body.Inputs()))
	}
	// The leftover is genuinely unemittable, so it is expected to be absorbed;
	// what matters is that the transaction was built at all.
	t.Logf("fee=%d outputs=%d", body.TxFee, len(body.Outputs()))
}

// TestMACSPrefersLargerCloserOverStrandedChange pins the choice directly: given
// a tight cover that strands its change and a larger one that does not, the
// larger is taken even though the tight one scores higher on priority.
func TestMACSPrefersLargerCloserOverStrandedChange(t *testing.T) {
	addr := testAddress(t)
	// Paying 10 ADA: the 10.4 ADA UTxO leaves 0.4 ADA (unemittable), the
	// 14 ADA UTxO leaves 4 ADA (fine). Filler pulls the mean toward 10.4 so
	// priority alone would take the tight one.
	pool := make([]common.Utxo, 0, 22)
	pool = append(pool,
		benchUtxo(addr, 1, 10_400_000, nil),
		benchUtxo(addr, 2, 14_000_000, nil),
	)
	for i := range 20 {
		//nolint:gosec // test sequence
		pool = append(pool, benchUtxo(addr, uint64(100+i), 10_400_000, nil))
	}

	selected, err := NewMACSSelector().
		Select(t.Context(), pool, NewSimpleValue(10_000_000))
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) == 0 {
		t.Fatal("nothing selected")
	}
	// The first pick covers the target; dust sweeping may append more.
	got := selected[0].Output.Amount().Uint64()
	if got != 14_000_000 {
		t.Errorf(
			"selected a %d lovelace UTxO, want 14000000: a tight cover that "+
				"strands its change was preferred over one that does not",
			got,
		)
	}
}

// TestMACSZeroValueKeepsPaperBehavior confirms the floor is opt-in through the
// constructor, so the zero value still runs the published algorithm.
func TestMACSZeroValueKeepsPaperBehavior(t *testing.T) {
	if got := (&MACSSelector{}).MinChange; got != 0 {
		t.Errorf("zero value MinChange = %d, want 0", got)
	}
	if got := NewMACSSelector().MinChange; got != macsSelectorMinChange {
		t.Errorf(
			"NewMACSSelector MinChange = %d, want %d",
			got, macsSelectorMinChange,
		)
	}
}

// TestMACSMinChangeIsDeterministic guards the fallback comparator: equal-value
// closers must break ties by ref, or selection stops being reproducible.
func TestMACSMinChangeIsDeterministic(t *testing.T) {
	pool := realisticPool(t, 300, 19)
	target := NewSimpleValue(9_500_000)

	first, err := NewMACSSelector().Select(t.Context(), pool, target)
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		again, err := NewMACSSelector().Select(t.Context(), pool, target)
		if err != nil {
			t.Fatal(err)
		}
		if len(again) != len(first) {
			t.Fatalf("selection size changed: %d then %d", len(first), len(again))
		}
		for i := range first {
			if utxoRef(first[i]) != utxoRef(again[i]) {
				t.Fatalf("selection differs at %d: %s vs %s",
					i, utxoRef(first[i]), utxoRef(again[i]))
			}
		}
	}
}

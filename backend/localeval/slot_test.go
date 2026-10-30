package localeval

import (
	"math"
	"testing"
	"time"
)

func TestSlotToTimePublicNetworks(t *testing.T) {
	tests := []struct {
		name   string
		config SlotConfig
		slot   uint64
		want   string
	}{
		// The first Shelley slot on each network.
		{"mainnet Shelley start", MainnetSlotConfig, 4492800, "2020-07-29T21:44:51Z"},
		{"preprod Shelley start", PreprodSlotConfig, 86400, "2022-06-21T00:00:00Z"},
		{"preview start", PreviewSlotConfig, 0, "2022-10-25T00:00:00Z"},
		// cardano-cli, given mainnet's era history, reports this slot's start
		// as POSIX time 1741566291000.
		{"mainnet slot 150000000", MainnetSlotConfig, 150000000, "2025-03-10T00:24:51Z"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.config.SlotToTime(test.slot)
			if err != nil {
				t.Fatal(err)
			}
			if got.UTC().Format(time.RFC3339) != test.want {
				t.Errorf("SlotToTime(%d) = %s, want %s", test.slot, got.UTC().Format(time.RFC3339), test.want)
			}
			slot, err := test.config.TimeToSlot(got)
			if err != nil {
				t.Fatal(err)
			}
			if slot != test.slot {
				t.Errorf("TimeToSlot(SlotToTime(%d)) = %d", test.slot, slot)
			}
		})
	}
}

func TestTimeToSlotWithinSlot(t *testing.T) {
	config := SlotConfig{ZeroTime: time.Unix(1000, 0), ZeroSlot: 10, SlotLength: 2 * time.Second}
	slot, err := config.TimeToSlot(time.Unix(1003, 999))
	if err != nil {
		t.Fatal(err)
	}
	if slot != 11 {
		t.Errorf("TimeToSlot = %d, want 11", slot)
	}
}

func TestSlotConfigRejects(t *testing.T) {
	valid := SlotConfig{ZeroTime: time.Unix(1000, 0), ZeroSlot: 10, SlotLength: time.Second}
	if _, err := valid.SlotToTime(9); err == nil {
		t.Error("SlotToTime before the zero slot succeeded")
	}
	if _, err := valid.TimeToSlot(time.Unix(999, 0)); err == nil {
		t.Error("TimeToSlot before the zero time succeeded")
	}
	if _, err := valid.SlotToTime(math.MaxUint64); err == nil {
		t.Error("SlotToTime of an overflowing slot succeeded")
	}
	for name, config := range map[string]SlotConfig{
		"zero time":       {SlotLength: time.Second},
		"zero length":     {ZeroTime: time.Unix(1000, 0)},
		"negative length": {ZeroTime: time.Unix(1000, 0), SlotLength: -time.Second},
	} {
		if _, err := config.SlotToTime(0); err == nil {
			t.Errorf("%s: SlotToTime succeeded", name)
		}
		if _, err := config.TimeToSlot(time.Unix(2000, 0)); err == nil {
			t.Errorf("%s: TimeToSlot succeeded", name)
		}
	}
}

func TestSlotConfigForNetworkMagic(t *testing.T) {
	for magic, want := range map[uint32]SlotConfig{
		MainnetNetworkMagic: MainnetSlotConfig,
		PreprodNetworkMagic: PreprodSlotConfig,
		PreviewNetworkMagic: PreviewSlotConfig,
	} {
		got, ok := SlotConfigForNetworkMagic(magic)
		if !ok || got != want {
			t.Errorf("SlotConfigForNetworkMagic(%d) = %+v, %v", magic, got, ok)
		}
	}
	if _, ok := SlotConfigForNetworkMagic(42); ok {
		t.Error("SlotConfigForNetworkMagic(42) reported a known network")
	}
}

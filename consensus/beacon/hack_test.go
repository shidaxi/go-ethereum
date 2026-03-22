package beacon

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	contractA = common.HexToAddress("0x851356ae760d987E095750cCeb3bC6014560891C")
	contractB = common.HexToAddress("0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B")
	slotA     = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000033")
	slotB     = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000058")
	oldValA   = common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	oldValB   = common.HexToHash("0x000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	newValA   = common.HexToHash("0x000000000000000000000000deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	newValB   = common.HexToHash("0x000000000000000000000000cafebabecafebabecafebabecafebabecafebabe")
)

func makeTestDB(t *testing.T) *state.StateDB {
	t.Helper()
	db, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	db.SetState(contractA, slotA, oldValA)
	db.SetState(contractB, slotB, oldValB)
	return db
}

// TestHackStateOverrideLegacy verifies the original single-override behaviour.
func TestHackStateOverrideLegacy(t *testing.T) {
	t.Setenv("HACK_STATE_OVERRIDE_BLOCK", "100")
	t.Setenv("HACK_STATE_OVERRIDE_ADDRESS", contractA.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_SLOT", slotA.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_VALUE", newValA.Hex())

	t.Run("storage overridden at target block", func(t *testing.T) {
		db := makeTestDB(t)
		hackStateOverride(db, 100)
		if got := db.GetState(contractA, slotA); got != newValA {
			t.Errorf("slot not overridden\n got:  %v\n want: %v", got, newValA)
		}
	})

	t.Run("storage unchanged before target block", func(t *testing.T) {
		db := makeTestDB(t)
		hackStateOverride(db, 99)
		if got := db.GetState(contractA, slotA); got != oldValA {
			t.Errorf("slot must not change at block 99\n got:  %v\n want: %v", got, oldValA)
		}
	})

	t.Run("storage unchanged after target block", func(t *testing.T) {
		db := makeTestDB(t)
		hackStateOverride(db, 101)
		if got := db.GetState(contractA, slotA); got != oldValA {
			t.Errorf("slot must not change at block 101\n got:  %v\n want: %v", got, oldValA)
		}
	})

	t.Run("no trigger when env var unset", func(t *testing.T) {
		t.Setenv("HACK_STATE_OVERRIDE_BLOCK", "")
		db := makeTestDB(t)
		hackStateOverride(db, 0)
		if got := db.GetState(contractA, slotA); got != oldValA {
			t.Errorf("slot must not change when block var is unset\n got:  %v\n want: %v", got, oldValA)
		}
	})
}

// TestHackStateOverrideIndexed verifies the multi-override indexed style.
func TestHackStateOverrideIndexed(t *testing.T) {
	// Two overrides at the same block.
	t.Setenv("HACK_STATE_OVERRIDE_0_BLOCK", "200")
	t.Setenv("HACK_STATE_OVERRIDE_0_ADDRESS", contractA.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_0_SLOT", slotA.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_0_VALUE", newValA.Hex())

	t.Setenv("HACK_STATE_OVERRIDE_1_BLOCK", "200")
	t.Setenv("HACK_STATE_OVERRIDE_1_ADDRESS", contractB.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_1_SLOT", slotB.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_1_VALUE", newValB.Hex())

	t.Run("both slots overridden at target block", func(t *testing.T) {
		db := makeTestDB(t)
		hackStateOverride(db, 200)
		if got := db.GetState(contractA, slotA); got != newValA {
			t.Errorf("slotA not overridden\n got:  %v\n want: %v", got, newValA)
		}
		if got := db.GetState(contractB, slotB); got != newValB {
			t.Errorf("slotB not overridden\n got:  %v\n want: %v", got, newValB)
		}
	})

	t.Run("no change before target block", func(t *testing.T) {
		db := makeTestDB(t)
		hackStateOverride(db, 199)
		if got := db.GetState(contractA, slotA); got != oldValA {
			t.Errorf("slotA must not change at block 199")
		}
		if got := db.GetState(contractB, slotB); got != oldValB {
			t.Errorf("slotB must not change at block 199")
		}
	})

	t.Run("scanning stops at first gap", func(t *testing.T) {
		// index 2 is unset; index 3 should be ignored even if set
		t.Setenv("HACK_STATE_OVERRIDE_3_BLOCK", "200")
		t.Setenv("HACK_STATE_OVERRIDE_3_ADDRESS", contractB.Hex())
		t.Setenv("HACK_STATE_OVERRIDE_3_SLOT", slotB.Hex())
		t.Setenv("HACK_STATE_OVERRIDE_3_VALUE", oldValB.Hex()) // would revert slotB if applied

		db := makeTestDB(t)
		hackStateOverride(db, 200)
		// slotB should have newValB (from index 1), not oldValB (index 3 skipped)
		if got := db.GetState(contractB, slotB); got != newValB {
			t.Errorf("index 3 must be skipped due to gap at 2\n got:  %v\n want: %v", got, newValB)
		}
	})
}

// TestHackStateOverrideDifferentBlocks verifies overrides can fire at different block numbers.
func TestHackStateOverrideDifferentBlocks(t *testing.T) {
	t.Setenv("HACK_STATE_OVERRIDE_0_BLOCK", "300")
	t.Setenv("HACK_STATE_OVERRIDE_0_ADDRESS", contractA.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_0_SLOT", slotA.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_0_VALUE", newValA.Hex())

	t.Setenv("HACK_STATE_OVERRIDE_1_BLOCK", "400")
	t.Setenv("HACK_STATE_OVERRIDE_1_ADDRESS", contractB.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_1_SLOT", slotB.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_1_VALUE", newValB.Hex())

	t.Run("only override 0 fires at block 300", func(t *testing.T) {
		db := makeTestDB(t)
		hackStateOverride(db, 300)
		if got := db.GetState(contractA, slotA); got != newValA {
			t.Errorf("slotA not overridden at 300")
		}
		if got := db.GetState(contractB, slotB); got != oldValB {
			t.Errorf("slotB must not change at 300")
		}
	})

	t.Run("only override 1 fires at block 400", func(t *testing.T) {
		db := makeTestDB(t)
		hackStateOverride(db, 400)
		if got := db.GetState(contractA, slotA); got != oldValA {
			t.Errorf("slotA must not change at 400")
		}
		if got := db.GetState(contractB, slotB); got != newValB {
			t.Errorf("slotB not overridden at 400")
		}
	})
}

package beacon

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestHackStateOverride verifies that hackStateOverride writes the storage slot
// exactly at the configured block number and is a no-op at any other number,
// including when no transactions are present (empty blocks).
func TestHackStateOverride(t *testing.T) {
	var (
		contractAddr = common.HexToAddress("0x851356ae760d987E095750cCeb3bC6014560891C")
		slot         = common.HexToHash("0x33")
		oldOwner     = common.HexToHash("0x000000000000000000000000aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		newOwner     = common.HexToHash("0x000000000000000000000000deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	)

	t.Setenv("HACK_STATE_OVERRIDE_BLOCK", "100")
	t.Setenv("HACK_STATE_OVERRIDE_ADDRESS", contractAddr.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_SLOT", slot.Hex())
	t.Setenv("HACK_STATE_OVERRIDE_VALUE", newOwner.Hex())

	makeDB := func() *state.StateDB {
		db, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		db.SetState(contractAddr, slot, oldOwner)
		return db
	}

	t.Run("storage overridden at target block", func(t *testing.T) {
		db := makeDB()
		hackStateOverride(db, 100)
		if got := db.GetState(contractAddr, slot); got != newOwner {
			t.Errorf("slot not overridden\n got:  %v\n want: %v", got, newOwner)
		}
	})

	t.Run("storage unchanged before target block", func(t *testing.T) {
		db := makeDB()
		hackStateOverride(db, 99)
		if got := db.GetState(contractAddr, slot); got != oldOwner {
			t.Errorf("slot must not change at block 99\n got:  %v\n want: %v", got, oldOwner)
		}
	})

	t.Run("storage unchanged after target block", func(t *testing.T) {
		db := makeDB()
		hackStateOverride(db, 101)
		if got := db.GetState(contractAddr, slot); got != oldOwner {
			t.Errorf("slot must not change at block 101\n got:  %v\n want: %v", got, oldOwner)
		}
	})

	t.Run("no trigger when env var unset (block 0 safe)", func(t *testing.T) {
		t.Setenv("HACK_STATE_OVERRIDE_BLOCK", "")
		db := makeDB()
		hackStateOverride(db, 0)
		if got := db.GetState(contractAddr, slot); got != oldOwner {
			t.Errorf("slot must not change when block var is unset\n got:  %v\n want: %v", got, oldOwner)
		}
	})
}

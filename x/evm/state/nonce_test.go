package state_test

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/cosmos/cosmos-sdk/testutil/keeper"
	"github.com/cosmos/cosmos-sdk/x/evm/state"
)

func TestNonce(t *testing.T) {
	k := &testkeeper.EVMTestApp.EvmKeeper
	ctx := testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{}).WithBlockTime(time.Now())
	stateDB := state.NewDBImpl(ctx, k, false)
	_, addr := testkeeper.MockAddressPair()
	stateDB.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
	nonce := stateDB.GetNonce(addr)
	require.Equal(t, nonce, uint64(1))
	stateDB.SetNonce(addr, 2, tracing.NonceChangeUnspecified)
	nonce = stateDB.GetNonce(addr)
	require.Equal(t, nonce, uint64(2))
}

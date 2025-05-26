package keeper_test

import (
	"bytes"
	"testing"

	"github.com/cosmos/cosmos-sdk/x/evm/keeper"
	"github.com/stretchr/testify/require"
)

func TestInitGenesis(t *testing.T) {
	ctx, k := createTestContext(t)
	// coinbase address must be associated
	coinbaseEniAddr, associated := k.GetEniAddress(ctx, keeper.GetCoinbaseAddress())
	require.True(t, associated)
	require.True(t, bytes.Equal(coinbaseEniAddr, k.AccountKeeper().GetModuleAddress("fee_collector")))
}

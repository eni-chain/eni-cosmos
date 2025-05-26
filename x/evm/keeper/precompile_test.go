package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/testutil/keeper"
	evmkeeper "github.com/cosmos/cosmos-sdk/x/evm/keeper"
)

func TestIsPayablePrecompile(t *testing.T) {
	_, evmAddr := keeper.MockAddressPair()
	require.False(t, evmkeeper.IsPayablePrecompile(&evmAddr))
	require.False(t, evmkeeper.IsPayablePrecompile(nil))
}

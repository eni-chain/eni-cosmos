package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/cosmos/cosmos-sdk/testutil/keeper"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.EvmKeeper(t)
	params := types.DefaultParams()

	require.EqualValues(t, params, k.GetParams(ctx))
}

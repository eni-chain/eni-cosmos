package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	ctx, k := createTestContext(t)
	params := types.DefaultParams()

	require.EqualValues(t, params, k.GetParams(ctx))
}

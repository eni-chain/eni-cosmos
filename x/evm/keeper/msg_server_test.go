package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/x/evm/keeper"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
)

func setupMsgServer(t *testing.T) (*keeper.Keeper, types.MsgServer, context.Context) {
	ctx, k := createTestContext(t)
	return k, keeper.NewMsgServerImpl(k), ctx
}

func TestMsgServer(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

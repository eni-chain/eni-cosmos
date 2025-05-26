package evm_test

import (
	"testing"

	testkeeper "github.com/cosmos/cosmos-sdk/testutil/keeper"
	"github.com/cosmos/cosmos-sdk/testutil/nullify"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/keeper"
	evm "github.com/cosmos/cosmos-sdk/x/evm/module"
	evmtypes "github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/stretchr/testify/require"
)

func createTestContext(t *testing.T) (types.Context, *keeper.Keeper) {
	app, ctx := testkeeper.NewMockApp(t, false)
	return ctx, app.GetEVMKeeper()
}

func TestGenesis(t *testing.T) {
	genesisState := evmtypes.GenesisState{
		Params: evmtypes.DefaultParams(),
	}
	ctx, k := createTestContext(t)
	evm.InitGenesis(ctx, k, genesisState)
	got := evm.ExportGenesis(ctx, k)
	require.NotNil(t, got)
	nullify.Fill(&genesisState)
	nullify.Fill(got)
}

package state_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	testkeeper "github.com/cosmos/cosmos-sdk/testutil/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/state"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestAddBalance(t *testing.T) {
	ctx, k := createTestContext(t)
	ctx = ctx.WithBlockTime(time.Now())
	db := state.NewDBImpl(ctx, k, false)
	eniAddr, evmAddr := testkeeper.MockAddressPair()
	require.Equal(t, uint256.NewInt(0), db.GetBalance(evmAddr))
	db.AddBalance(evmAddr, uint256.NewInt(0), tracing.BalanceChangeUnspecified)

	// set association
	k.SetAddressMapping(db.Ctx(), eniAddr, evmAddr)
	require.Equal(t, uint256.NewInt(0), db.GetBalance(evmAddr))
	db.AddBalance(evmAddr, uint256.NewInt(10000000000000), tracing.BalanceChangeUnspecified)
	require.Nil(t, db.Err())
	require.Equal(t, db.GetBalance(evmAddr), uint256.NewInt(10000000000000))

	_, evmAddr2 := testkeeper.MockAddressPair()
	_, evmAddr3 := testkeeper.MockAddressPair()
	db.SelfDestruct(evmAddr3)
	db.AddBalance(evmAddr2, uint256.NewInt(5000000000000), tracing.BalanceChangeUnspecified)
	require.Nil(t, db.Err())
	require.Equal(t, db.GetBalance(evmAddr3), uint256.NewInt(0))
}

func TestSubBalance(t *testing.T) {
	ctx, k := createTestContext(t)
	ctx = ctx.WithBlockTime(time.Now())
	db := state.NewDBImpl(ctx, k, false)
	eniAddr, evmAddr := testkeeper.MockAddressPair()
	require.Equal(t, uint256.NewInt(0), db.GetBalance(evmAddr))
	db.SubBalance(evmAddr, uint256.NewInt(0), tracing.BalanceChangeUnspecified)

	// set association
	k.SetAddressMapping(db.Ctx(), eniAddr, evmAddr)
	require.Equal(t, uint256.NewInt(0), db.GetBalance(evmAddr))
	amt := sdk.NewCoins(sdk.NewCoin(k.GetBaseDenom(ctx), math.NewInt(20000000000000)))
	k.BankKeeper().MintCoins(db.Ctx(), types.ModuleName, amt)
	k.BankKeeper().SendCoinsFromModuleToAccount(db.Ctx(), types.ModuleName, eniAddr, amt)
	db.SubBalance(evmAddr, uint256.NewInt(10000000000000), tracing.BalanceChangeUnspecified)
	require.Nil(t, db.Err())
	require.Equal(t, db.GetBalance(evmAddr), uint256.NewInt(10000000000000))

	_, evmAddr2 := testkeeper.MockAddressPair()
	amt = sdk.NewCoins(sdk.NewCoin(k.GetBaseDenom(ctx), math.NewInt(10000000000000)))
	k.BankKeeper().MintCoins(db.Ctx(), types.ModuleName, amt)
	k.BankKeeper().SendCoinsFromModuleToAccount(db.Ctx(), types.ModuleName, sdk.AccAddress(evmAddr2[:]), amt)
	db.SubBalance(evmAddr2, uint256.NewInt(5000000000000), tracing.BalanceChangeUnspecified) // should redirect to SubBalance
	require.Nil(t, db.Err())
	require.Equal(t, db.GetBalance(evmAddr2), uint256.NewInt(5000000000000))

	// insufficient balance
	db.SubBalance(evmAddr2, uint256.NewInt(10000000000000), tracing.BalanceChangeUnspecified)
	require.NotNil(t, db.Err())

	db.WithErr(nil)
	_, evmAddr3 := testkeeper.MockAddressPair()
	db.SelfDestruct(evmAddr3)
	db.SubBalance(evmAddr2, uint256.NewInt(5000000000000), tracing.BalanceChangeUnspecified)
	require.Nil(t, db.Err())
	require.Equal(t, db.GetBalance(evmAddr3), uint256.NewInt(0))
}

func TestSetBalance(t *testing.T) {
	ctx, k := createTestContext(t)
	ctx = ctx.WithBlockTime(time.Now())
	db := state.NewDBImpl(ctx, k, true)
	_, evmAddr := testkeeper.MockAddressPair()
	db.SetBalance(evmAddr, uint256.NewInt(10000000000000), tracing.BalanceChangeUnspecified)
	require.Equal(t, uint256.NewInt(10000000000000), db.GetBalance(evmAddr))

	eniAddr2, evmAddr2 := testkeeper.MockAddressPair()
	k.SetAddressMapping(db.Ctx(), eniAddr2, evmAddr2)
	db.SetBalance(evmAddr2, uint256.NewInt(10000000000000), tracing.BalanceChangeUnspecified)
	require.Equal(t, uint256.NewInt(10000000000000), db.GetBalance(evmAddr2))
}

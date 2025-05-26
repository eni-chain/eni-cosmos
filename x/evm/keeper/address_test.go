package keeper_test

import (
	"testing"

	testkeeper "github.com/cosmos/cosmos-sdk/testutil/keeper"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func createTestContext(t *testing.T) (types.Context, *keeper.Keeper) {
	app, ctx := testkeeper.NewMockApp(t, false)
	return ctx, app.GetEVMKeeper()
}

func padTo20Bytes(input []byte) []byte {
	if len(input) >= 20 {
		return input[:20]
	}
	padded := make([]byte, 20)
	copy(padded, input)
	return padded
}

func TestAddressMapping(t *testing.T) {
	ctx, k := createTestContext(t)

	eniAddr := types.AccAddress(padTo20Bytes([]byte("eni-address-addr1")))
	evmAddr := common.BytesToAddress(padTo20Bytes([]byte("evm-address-addr2")))

	k.SetAddressMapping(ctx, eniAddr, evmAddr)

	gotEvmAddr, found := k.GetEVMAddress(ctx, eniAddr)
	require.True(t, found)
	require.Equal(t, evmAddr, gotEvmAddr)

	gotEvmAddr2 := k.GetEVMAddressOrDefault(ctx, eniAddr)
	require.Equal(t, evmAddr, gotEvmAddr2)

	gotEniAddr, found := k.GetEniAddress(ctx, evmAddr)
	require.True(t, found)
	require.Equal(t, eniAddr, gotEniAddr)

	// our feature
	gotEniAddr2 := k.GetEniAddressOrDefault(ctx, evmAddr)
	require.Equal(t, evmAddr.Bytes(), gotEniAddr2.Bytes())

	collected := false
	k.IterateEniAddressMapping(ctx, func(e common.Address, a types.AccAddress) bool {
		if e == evmAddr && a.Equals(eniAddr) {
			collected = true
			return true
		}
		return false
	})
	require.True(t, collected)

	require.True(t, k.CanAddressReceive(ctx, eniAddr))

	k.DeleteAddressMapping(ctx, eniAddr, evmAddr)
	_, found = k.GetEVMAddress(ctx, eniAddr)
	require.False(t, found)
}

func TestGetEniAddressFromString(t *testing.T) {
	ctx, k := createTestContext(t)
	h := keeper.NewEvmAddressHandler(k)

	// Hex string test
	evmAddr := common.BytesToAddress(padTo20Bytes([]byte("evm-hex-addr")))
	k.SetAddressMapping(ctx, padTo20Bytes([]byte("evm-hex-addr")), evmAddr)
	res, err := h.GetEniAddressFromString(ctx, evmAddr.Hex())
	require.NoError(t, err)
	require.Equal(t, types.AccAddress(evmAddr[:]), res)

	// Bech32 test
	acc := types.AccAddress(padTo20Bytes([]byte("addr-b32-format")))
	res2, err := h.GetEniAddressFromString(ctx, acc.String())
	require.NoError(t, err)
	require.Equal(t, acc, res2)
}

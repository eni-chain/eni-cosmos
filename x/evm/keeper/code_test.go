package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/testutil/keeper"
	testkeeper "github.com/cosmos/cosmos-sdk/testutil/keeper"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestCode(t *testing.T) {
	ctx, k := createTestContext(t)

	_, addr := testkeeper.MockAddressPair()

	require.Equal(t, common.Hash{}, k.GetCodeHash(ctx, addr))

	code := []byte{1, 2, 3, 4, 5}
	k.SetCode(ctx, addr, code)
	require.Equal(t, crypto.Keccak256Hash(code), k.GetCodeHash(ctx, addr))
	require.Equal(t, code, k.GetCode(ctx, addr))
	require.Equal(t, 5, k.GetCodeSize(ctx, addr))
}

func TestNilCode(t *testing.T) {
	ctx, k := createTestContext(t)

	_, addr := keeper.MockAddressPair()

	k.SetCode(ctx, addr, nil)
	require.Nil(t, k.GetCode(ctx, addr))
	require.Equal(t, 0, k.GetCodeSize(ctx, addr))
	require.Equal(t, ethtypes.EmptyCodeHash, k.GetCodeHash(ctx, addr))
}

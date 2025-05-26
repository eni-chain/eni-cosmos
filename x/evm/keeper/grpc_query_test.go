package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/evm/keeper"
	evmtypes "github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestEniAddressByEVMAddress(t *testing.T) {
	ctx, k := createTestContext(t)
	querier := keeper.NewQuerier(k)

	evmbz := common.HexToAddress("0x1234567890123456789012345678901234567890")
	eni := types.AccAddress([]byte("eni-address-00000000000000")[:20])

	k.SetAddressMapping(ctx, eni, evmbz)

	t.Run("valid associated evm address", func(t *testing.T) {
		resp, err := querier.EniAddressByEVMAddress(ctx, &evmtypes.QueryEniAddressByEVMAddressRequest{
			EvmAddress: evmbz.Hex(),
		})
		require.NoError(t, err)
		require.True(t, resp.Associated)
		require.Equal(t, eni.String(), resp.EniAddress)
	})

	t.Run("empty address", func(t *testing.T) {
		_, err := querier.EniAddressByEVMAddress(ctx, &evmtypes.QueryEniAddressByEVMAddressRequest{})
		require.ErrorIs(t, err, sdkerrors.ErrInvalidRequest)
	})

	t.Run("unassociated address", func(t *testing.T) {
		resp, err := querier.EniAddressByEVMAddress(ctx, &evmtypes.QueryEniAddressByEVMAddressRequest{
			EvmAddress: common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa").Hex(),
		})
		require.NoError(t, err)
		require.False(t, resp.Associated)
	})
}

func TestEVMAddressByEniAddress(t *testing.T) {
	ctx, k := createTestContext(t)
	querier := keeper.NewQuerier(k)

	evmbz := common.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd")
	eni := types.AccAddress([]byte("eni-address-00000000000000")[:20])
	k.SetAddressMapping(ctx, eni, evmbz)

	t.Run("valid associated eni address", func(t *testing.T) {
		resp, err := querier.EVMAddressByEniAddress(ctx, &evmtypes.QueryEVMAddressByEniAddressRequest{
			EniAddress: eni.String(),
		})
		require.NoError(t, err)
		require.True(t, resp.Associated)
		require.Equal(t, evmbz.Hex(), resp.EvmAddress)
	})

	t.Run("empty address", func(t *testing.T) {
		_, err := querier.EVMAddressByEniAddress(ctx, &evmtypes.QueryEVMAddressByEniAddressRequest{})
		require.ErrorIs(t, err, sdkerrors.ErrInvalidRequest)
	})

	t.Run("unassociated eni address", func(t *testing.T) {
		unassociated := types.AccAddress([]byte("eni-other-00000000000000")[:20])
		resp, err := querier.EVMAddressByEniAddress(ctx, &evmtypes.QueryEVMAddressByEniAddressRequest{
			EniAddress: unassociated.String(),
		})
		require.NoError(t, err)
		require.False(t, resp.Associated)
	})
}

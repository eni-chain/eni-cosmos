package types_test

import (
	"testing"

	cosmossdk_io_math "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/stretchr/testify/require"
)

func TestDefaultParams(t *testing.T) {
	require.Equal(t, types.Params{
		PriorityNormalizer:                  types.DefaultPriorityNormalizer,
		BaseFeePerGas:                       types.DefaultBaseFeePerGas,
		MinimumFeePerGas:                    types.DefaultMinFeePerGas,
		MaximumFeePerGas:                    types.DefaultMaxFeePerGas,
		MaxDynamicBaseFeeUpwardAdjustment:   types.DefaultMaxDynamicBaseFeeUpwardAdjustment,
		MaxDynamicBaseFeeDownwardAdjustment: types.DefaultMaxDynamicBaseFeeDownwardAdjustment,
		TargetGasUsedPerBlock:               types.DefaultTargetGasUsedPerBlock,
		InitEniAddress:                      "",
		InitEniAmount:                       "",
	}, types.DefaultParams())
	require.Nil(t, types.DefaultParams().Validate())
}

func TestValidateParamsInvalidPriorityNormalizer(t *testing.T) {
	params := types.DefaultParams()
	params.PriorityNormalizer = cosmossdk_io_math.LegacyNewDec(-1) // Set to invalid negative value

	err := params.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonpositive priority normalizer")
}

func TestValidateParamsNegativeBaseFeePerGas(t *testing.T) {
	params := types.DefaultParams()
	params.BaseFeePerGas = cosmossdk_io_math.LegacyNewDec(-1) // Set to invalid negative value

	err := params.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "negative base fee per gas")
}

func TestBaseFeeMinimumFee(t *testing.T) {
	params := types.DefaultParams()
	params.MinimumFeePerGas = cosmossdk_io_math.LegacyNewDec(1)
	params.BaseFeePerGas = params.MinimumFeePerGas.Add(cosmossdk_io_math.LegacyNewDec(1))
	err := params.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "minimum fee cannot be lower than base fee")
}

func TestValidateParamsInvalidMaxDynamicBaseFeeUpwardAdjustment(t *testing.T) {
	params := types.DefaultParams()
	params.MaxDynamicBaseFeeUpwardAdjustment = cosmossdk_io_math.LegacyNewDec(-1) // Set to invalid negative value

	err := params.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "negative base fee adjustment")

	params.MaxDynamicBaseFeeUpwardAdjustment = cosmossdk_io_math.LegacyNewDec(2)
	err = params.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "base fee adjustment must be less than or equal to 1")
}

func TestValidateParamsInvalidMaxDynamicBaseFeeDownwardAdjustment(t *testing.T) {
	params := types.DefaultParams()
	params.MaxDynamicBaseFeeDownwardAdjustment = cosmossdk_io_math.LegacyNewDec(-1) // Set to invalid negative value

	err := params.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "negative base fee adjustment")

	params.MaxDynamicBaseFeeDownwardAdjustment = cosmossdk_io_math.LegacyNewDec(2)
	err = params.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "base fee adjustment must be less than or equal to 1")
}

func TestValidateParamsInvalidMaxFeePerGas(t *testing.T) {
	params := types.DefaultParams()
	params.MaximumFeePerGas = cosmossdk_io_math.LegacyNewDec(-1) // Set to invalid negative value

	err := params.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "negative max fee per gas")
}

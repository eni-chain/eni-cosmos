package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/stretchr/testify/require"
)

func TestBaseFeePerGas(t *testing.T) {
	ctx, k := createTestContext(t)
	require.Equal(t, k.GetMinimumFeePerGas(ctx), k.GetCurrBaseFeePerGas(ctx))
	fmt.Println("gas fee", k.GetCurrBaseFeePerGas(ctx), k.GetMaximumFeePerGas(ctx))
	require.True(t, k.GetCurrBaseFeePerGas(ctx).GTE(k.GetMaximumFeePerGas(ctx))) // GetCurrBaseFeePerGas default is 1000000000
	originalbf := k.GetCurrBaseFeePerGas(ctx)
	k.SetCurrBaseFeePerGas(ctx, math.LegacyOneDec())
	require.Equal(t, math.LegacyNewDecFromInt(math.NewInt(1000000000)), k.GetCurrBaseFeePerGas(ctx))
	k.SetCurrBaseFeePerGas(ctx, originalbf)
}

func TestAdjustBaseFeePerGas(t *testing.T) {
	ctx, k := createTestContext(t)
	testCases := []struct {
		name            string
		currentBaseFee  float64
		minimumFee      float64
		maximumFee      float64
		blockGasUsed    uint64
		blockGasLimit   uint64
		upwardAdj       math.LegacyDec
		downwardAdj     math.LegacyDec
		targetGasUsed   uint64
		expectedBaseFee uint64
	}{
		{
			name:            "Block gas usage exactly half of limit, 0% up, 0% down, no fee change",
			currentBaseFee:  100,
			minimumFee:      10,
			maximumFee:      1000,
			blockGasUsed:    500000,
			blockGasLimit:   1000000,
			upwardAdj:       math.LegacyZeroDec(),
			downwardAdj:     math.LegacyZeroDec(),
			targetGasUsed:   500000,
			expectedBaseFee: 100,
		},
		{
			name:            "Block gas usage 50%, 50% up, 50% down, no fee change",
			currentBaseFee:  100,
			minimumFee:      10,
			maximumFee:      1000,
			blockGasUsed:    500000,
			blockGasLimit:   1000000,
			upwardAdj:       math.LegacyNewDecWithPrec(5, 1),
			downwardAdj:     math.LegacyNewDecWithPrec(5, 1),
			targetGasUsed:   500000,
			expectedBaseFee: 100,
		},
		{
			name:            "Block gas usage 75%, 0% up, 0% down, base fee stays the same",
			currentBaseFee:  10000,
			minimumFee:      10,
			maximumFee:      100000,
			blockGasUsed:    750000,
			blockGasLimit:   1000000,
			upwardAdj:       math.LegacyZeroDec(),
			downwardAdj:     math.LegacyZeroDec(),
			targetGasUsed:   500000,
			expectedBaseFee: 10000,
		},
		{
			name:            "Block gas usage 25%, 0% up, 0% down, base fee stays the same",
			currentBaseFee:  10000,
			minimumFee:      10,
			maximumFee:      100000,
			blockGasUsed:    250000,
			blockGasLimit:   1000000,
			upwardAdj:       math.LegacyZeroDec(),
			downwardAdj:     math.LegacyZeroDec(),
			targetGasUsed:   500000,
			expectedBaseFee: 10000,
		},
		{
			name:            "Block gas usage 75%, 50% up, 0% down, base fee increases by 25%",
			currentBaseFee:  10000,
			minimumFee:      10,
			maximumFee:      100000,
			blockGasUsed:    750000,
			blockGasLimit:   1000000,
			upwardAdj:       math.LegacyNewDecWithPrec(5, 1),
			downwardAdj:     math.LegacyZeroDec(),
			targetGasUsed:   500000,
			expectedBaseFee: 12500,
		},
		{
			name:            "Block gas usage 25%, 0% up, 50% down, base fee decreases by 25%",
			currentBaseFee:  10000,
			minimumFee:      10,
			maximumFee:      100000,
			blockGasUsed:    250000,
			blockGasLimit:   1000000,
			upwardAdj:       math.LegacyZeroDec(),
			downwardAdj:     math.LegacyNewDecWithPrec(5, 1),
			targetGasUsed:   500000,
			expectedBaseFee: 7500,
		},
		{
			name:            "Block gas usage low, new base fee below minimum, set to minimum",
			currentBaseFee:  100,
			minimumFee:      99,
			maximumFee:      1000,
			blockGasUsed:    0,
			blockGasLimit:   1000000,
			upwardAdj:       math.LegacyNewDecWithPrec(5, 2),
			downwardAdj:     math.LegacyNewDecWithPrec(5, 2),
			targetGasUsed:   500000,
			expectedBaseFee: 99, // Should not go below the minimum fee
		},
		{
			name:            "Block gas usage high, new base fee above maximum, set to maximum",
			currentBaseFee:  999,
			minimumFee:      10,
			maximumFee:      1000,
			blockGasUsed:    1000000, // completely full block
			blockGasLimit:   1000000,
			upwardAdj:       math.LegacyNewDecWithPrec(5, 1),
			downwardAdj:     math.LegacyNewDecWithPrec(5, 1),
			targetGasUsed:   500000,
			expectedBaseFee: 1000, // Should not go above the maximum fee
		},
		{
			name:            "target gas used is 0",
			currentBaseFee:  10000,
			minimumFee:      10,
			maximumFee:      1000,
			blockGasUsed:    0,
			blockGasLimit:   1000000,
			upwardAdj:       math.LegacyNewDecWithPrec(5, 1),
			downwardAdj:     math.LegacyNewDecWithPrec(5, 1),
			targetGasUsed:   0,
			expectedBaseFee: 10000,
		},
		{
			name: "cap block gas used to block gas limit",
			// block gas used is 1.5x block gas limit
			currentBaseFee:  10000,
			minimumFee:      10,
			maximumFee:      100000,
			blockGasUsed:    1500000,
			blockGasLimit:   1000000,
			upwardAdj:       math.LegacyNewDecWithPrec(5, 1),
			downwardAdj:     math.LegacyNewDecWithPrec(5, 1),
			targetGasUsed:   500000,
			expectedBaseFee: 15000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx = ctx.WithConsensusParams(cmtproto.ConsensusParams{
				Block: &cmtproto.BlockParams{MaxGas: int64(tc.blockGasLimit)},
			})
			k.SetNextBaseFeePerGas(ctx, math.LegacyNewDecFromInt(math.NewInt(int64(tc.currentBaseFee))))
			p := k.GetParams(ctx)
			p.MinimumFeePerGas = math.LegacyNewDecFromInt(math.NewInt(int64(tc.minimumFee)))
			p.MaximumFeePerGas = math.LegacyNewDecFromInt(math.NewInt(int64(tc.maximumFee)))
			p.MaxDynamicBaseFeeUpwardAdjustment = tc.upwardAdj
			p.MaxDynamicBaseFeeDownwardAdjustment = tc.downwardAdj
			p.TargetGasUsedPerBlock = tc.targetGasUsed
			k.SetParams(ctx, p)
			adjustGasFee := k.AdjustDynamicBaseFeePerGas(ctx, tc.blockGasUsed)
			expected := math.LegacyNewDecFromInt(math.NewInt(int64(tc.expectedBaseFee)))
			require.Equal(t, expected.MustFloat64(), adjustGasFee.MustFloat64(), 0.001, "prev block base fee did not match expected value")
		})
	}
}

func TestGetDynamicBaseFeePerGasWithNilMinFee(t *testing.T) {
	ctx, k := createTestContext(t)

	// Test case 1: When dynamic base fee doesn't exist and minimum fee is nil
	store := ctx.KVStore(k.GetStoreKey())
	store.Delete(types.BaseFeePerGasPrefix)

	// Clear the dynamic base fee from store
	fee := k.GetCurrBaseFeePerGas(ctx)
	require.Equal(t, types.DefaultParams().MinimumFeePerGas, fee)
	require.False(t, fee.IsNil())

	// Test case 2: When dynamic base fee exists
	expectedFee := math.LegacyNewDec(1000000000)
	k.SetCurrBaseFeePerGas(ctx, expectedFee)

	fee = k.GetCurrBaseFeePerGas(ctx)
	require.Equal(t, expectedFee, fee)
	require.False(t, fee.IsNil())
}

func TestGetPrevBlockBaseFeePerGasWithNilMinFee(t *testing.T) {
	ctx, k := createTestContext(t)

	// Test case 1: When dynamic base fee doesn't exist and minimum fee is nil
	store := ctx.KVStore(k.GetStoreKey())
	store.Delete(types.BaseFeePerGasPrefix)

	// Clear the dynamic base fee from store
	fee := k.GetCurrBaseFeePerGas(ctx)
	require.Equal(t, types.DefaultParams().MinimumFeePerGas, fee)
	require.False(t, fee.IsNil())
}

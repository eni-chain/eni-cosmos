package keeper

import (
	cosmossdk_io_math "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"math/big"

	"github.com/cosmos/cosmos-sdk/utils"
	"github.com/cosmos/cosmos-sdk/x/evm/config"

	"github.com/cosmos/cosmos-sdk/x/evm/types"
)

const BaseDenom = "ueni"

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.Paramstore.SetParamSet(ctx, &params)
}

func (k *Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	return k.GetParamsIfExists(ctx)
}

func (k *Keeper) GetParamsIfExists(ctx sdk.Context) types.Params {
	params := types.Params{}
	params = types.DefaultParams()
	return params
	k.Paramstore.GetParamSetIfExists(ctx, &params)
	if params.BaseFeePerGas.IsNil() {
		params = types.DefaultParams()
	}
	return params
}

func (k *Keeper) GetBaseDenom(ctx sdk.Context) string {
	return BaseDenom
}

func (k *Keeper) GetPriorityNormalizer(ctx sdk.Context) cosmossdk_io_math.LegacyDec {
	return k.GetParams(ctx).PriorityNormalizer
}

func (k *Keeper) GetBaseFeePerGas(ctx sdk.Context) cosmossdk_io_math.LegacyDec {
	return k.GetParams(ctx).BaseFeePerGas
}

func (k *Keeper) GetMaxDynamicBaseFeeUpwardAdjustment(ctx sdk.Context) cosmossdk_io_math.LegacyDec {
	return k.GetParams(ctx).MaxDynamicBaseFeeUpwardAdjustment
}

func (k *Keeper) GetMaxDynamicBaseFeeDownwardAdjustment(ctx sdk.Context) cosmossdk_io_math.LegacyDec {
	return k.GetParams(ctx).MaxDynamicBaseFeeDownwardAdjustment
}

func (k *Keeper) GetMinimumFeePerGas(ctx sdk.Context) cosmossdk_io_math.LegacyDec {
	return k.GetParams(ctx).MinimumFeePerGas
}

func (k *Keeper) GetMaximumFeePerGas(ctx sdk.Context) cosmossdk_io_math.LegacyDec {
	return k.GetParams(ctx).MaximumFeePerGas
}

func (k *Keeper) GetTargetGasUsedPerBlock(ctx sdk.Context) uint64 {
	return k.GetParams(ctx).TargetGasUsedPerBlock
}

func (k *Keeper) ChainID(ctx sdk.Context) *big.Int {
	if k.EthBlockTestConfig.Enabled {
		// replay is for eth mainnet so always return 1
		return utils.Big1
	}
	// return mapped chain ID
	return config.GetEVMChainID(ctx.ChainID())

}

/*
*
eni gas = evm gas * multiplier
eni gas price = fee / eni gas = fee / (evm gas * multiplier) = evm gas / multiplier
*/
func (k *Keeper) GetEVMGasLimitFromCtx(ctx sdk.Context) uint64 {
	return k.getEvmGasLimitFromCtx(ctx)
}

func (k *Keeper) GetCosmosGasLimitFromEVMGas(ctx sdk.Context, evmGas uint64) uint64 {
	gasMultipler := k.GetPriorityNormalizer(ctx)
	gasLimitBigInt := cosmossdk_io_math.LegacyNewDecFromInt(cosmossdk_io_math.NewIntFromUint64(evmGas)).Mul(gasMultipler).TruncateInt().BigInt()
	if gasLimitBigInt.Cmp(utils.BigMaxU64) > 0 {
		gasLimitBigInt = utils.BigMaxU64
	}
	return gasLimitBigInt.Uint64()
}

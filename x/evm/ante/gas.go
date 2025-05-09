package ante

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmkeeper "github.com/cosmos/cosmos-sdk/x/evm/keeper"
	evmtypes "github.com/cosmos/cosmos-sdk/x/evm/types"
)

type GasLimitDecorator struct {
	evmKeeper *evmkeeper.Keeper
}

func NewGasLimitDecorator(evmKeeper *evmkeeper.Keeper) *GasLimitDecorator {
	return &GasLimitDecorator{evmKeeper: evmKeeper}
}

// Called at the end of the ante chain to set gas limit properly
func (gl GasLimitDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	return ctx, nil
	msg := evmtypes.MustGetEVMTransactionMessage(tx)
	txData, err := evmtypes.UnpackTxData(msg.Data)
	if err != nil {
		return ctx, err
	}

	adjustedGasLimit := gl.evmKeeper.GetPriorityNormalizer(ctx).MulInt64(int64(txData.GetGas()))
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(adjustedGasLimit.TruncateInt().Uint64()))
	return next(ctx, tx, simulate)
}

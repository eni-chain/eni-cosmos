package keeper

import (
	"context"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/utils/config"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"time"
)

func (msg msgServer) AddBlackLists(goCtx context.Context, req *types.MsgAddBlackLists) (*types.MsgAddBlackListsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	gasParamsManager := config.DefaultUpdateConfig.GasParamsManager

	if gasParamsManager != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidGasManager, "invalid gas manager; expected %s, got %s", gasParamsManager, req.Authority)
	}

	if len(req.Addresses) == 0 || len(req.Addresses) > 10 {
		return nil, errorsmod.Wrapf(types.ErrBlackListsUpdate, "addresses length must less than 10,got %d", len(req.Addresses))
	}

	kv := msg.Keeper.PrefixStore(ctx, types.BlackListsPrefix)
	data := time.Now().Format(time.DateTime)
	for _, addr := range req.Addresses {
		if common.IsHexAddress(addr) {
			evmAddr := common.HexToAddress(addr)
			oldData := kv.Get(evmAddr[:])
			if oldData != nil {
				return nil, errorsmod.Wrapf(types.ErrBlackListsUpdate, "address %s already in the blacklists ", evmAddr)
			}

			kv.Set(evmAddr[:], []byte(data))
		} else {
			return nil, errorsmod.Wrapf(types.ErrBlackListsUpdate, "req Addresses not evmAddress %s", addr)
		}

	}

	return &types.MsgAddBlackListsResponse{}, nil
}

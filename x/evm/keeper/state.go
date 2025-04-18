package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
)

func (k *Keeper) GetState(ctx sdk.Context, addr common.Address, hash common.Hash) common.Hash {
	val := k.PrefixStore(ctx, types.StateKey(addr)).Get(hash[:])
	if val == nil {
		k.logger.Debug("GetState", "address", addr.Hex(), "key", hash.Hex(), "value", "nil")
		return common.Hash{}
	}
	return common.BytesToHash(val)
}

func (k *Keeper) SetState(ctx sdk.Context, addr common.Address, key common.Hash, val common.Hash) {
	k.PrefixStore(ctx, types.StateKey(addr)).Set(key[:], val[:])
	k.logger.Debug("SetState", "address", addr.Hex(), "key", key.Hex(), "value", val.Hex())
}

func (k *Keeper) IterateState(ctx sdk.Context, cb func(addr common.Address, key common.Hash, val common.Hash) bool) {
	iter := prefix.NewStore(ctx.KVStore(k.storeKey), types.StateKeyPrefix).Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		k := iter.Key()
		evmAddr := common.BytesToAddress(k[:common.AddressLength])
		if cb(evmAddr, common.BytesToHash(k[common.AddressLength:]), common.BytesToHash(iter.Value())) {
			break
		}
	}
}

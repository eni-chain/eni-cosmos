package keeper

import (
	"encoding/binary"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
)

func (k *Keeper) GetNonce(ctx sdk.Context, addr common.Address) uint64 {
	bz := k.PrefixStore(ctx, types.NonceKeyPrefix).Get(addr[:])
	if bz == nil {
		return 0
	}
	nonce := binary.BigEndian.Uint64(bz)
	k.logger.Debug("GetNonce", "address", addr.Hex(), "nonce", nonce)
	return nonce
}

func (k *Keeper) SetNonce(ctx sdk.Context, addr common.Address, nonce uint64) {
	length := make([]byte, 8)
	binary.BigEndian.PutUint64(length, nonce)
	k.PrefixStore(ctx, types.NonceKeyPrefix).Set(addr[:], length)
	k.logger.Debug("SetNonce", "address", addr.Hex(), "nonce", nonce)
}

func (k *Keeper) IterateAllNonces(ctx sdk.Context, cb func(addr common.Address, nonce uint64) bool) {
	iter := prefix.NewStore(ctx.KVStore(k.storeKey), types.NonceKeyPrefix).Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		evmAddr := common.BytesToAddress(iter.Key())
		if cb(evmAddr, binary.BigEndian.Uint64(iter.Value())) {
			break
		}
	}
}

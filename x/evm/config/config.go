package config

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const DefaultChainID = int64(0x697873)

// ChainIDMapping is a mapping of cosmos chain IDs to their respective chain IDs.
var ChainIDMapping = map[string]int64{
	"eni-mainnet": int64(173),      //ENI
	"eni-testnet": int64(174),      // ENI Testnet
	"eni-devnet":  int64(0x697873), // ENI Devnet
}

func GetEVMChainID(cosmosChainID string) *big.Int {
	if evmChainID, ok := ChainIDMapping[cosmosChainID]; ok {
		return big.NewInt(evmChainID)
	}
	return big.NewInt(DefaultChainID)
}

func GetVersionWthDefault(ctx sdk.Context, override uint16, defaultVersion uint16) uint16 {
	// overrides are only available on non-live chain IDs
	if override > 0 && !IsLiveChainID(ctx) {
		return override
	}
	return defaultVersion
}

// IsLiveChainID return true if one of the live chainIDs
func IsLiveChainID(ctx sdk.Context) bool {
	_, ok := ChainIDMapping[ctx.ChainID()]
	return ok
}

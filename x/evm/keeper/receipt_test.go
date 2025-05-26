package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReceipt(t *testing.T) {
	ctx, k := createTestContext(t)
	txHash := common.HexToHash("0x0750333eac0be1203864220893d8080dd8a8fd7a2ed098dfd92a718c99d437f2")
	_, err := k.GetReceipt(ctx, txHash)
	require.NotNil(t, err)
	err = k.MockReceipt(ctx, txHash, &types.Receipt{TxHashHex: txHash.Hex()})
	assert.Nil(t, err)
	_, err = k.GetTransientReceipt(ctx, txHash)
	require.Nil(t, err)
	k.AppendToEvmTxDeferredInfo(ctx, ethtypes.Bloom{}, common.Hash{1}, math.NewInt(1)) // make sure this isn't flushed into receipt store
	_, err = k.GetReceipt(ctx, common.Hash{1})
	require.Equal(t, "not found", err.Error())
}

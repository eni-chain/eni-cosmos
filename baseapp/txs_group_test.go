package baseapp

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestTxGroup_GroupByTxs(t *testing.T) {
	// Create addresses
	fromAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	toAddr := common.HexToAddress("0x0987654321098765432109876543210987654321")

	// Create a TxGroup with mock decoder
	mockDecoder := func(tx []byte) (sdk.Tx, error) {
		return nil, nil
	}
	group := NewTxGroup(mockDecoder, 2)

	// Add transactions to the group
	group.evmTxMetas = append(group.evmTxMetas, &TxMeta{
		RawTx: []byte("tx1"),
		From:  fromAddr,
		To:    &toAddr,
		Nonce: 1,
		Data:  []byte("data1"),
		Index: 0,
	})
	group.evmTxMetas = append(group.evmTxMetas, &TxMeta{
		RawTx: []byte("tx2"),
		From:  fromAddr,
		To:    &toAddr,
		Nonce: 2,
		Data:  []byte("data2"),
		Index: 1,
	})

	// Test grouping
	err := group.groupByAddress()
	require.NoError(t, err)

	// Verify the transactions are in order
	require.Equal(t, 1, len(group.groups))
	groupTxs := group.groups[fromAddr.Hex()]
	require.Equal(t, 2, len(groupTxs))
	require.Equal(t, uint64(1), groupTxs[0].Nonce)
	require.Equal(t, uint64(2), groupTxs[1].Nonce)
}

func TestTxGroup_GroupByTxs_Empty(t *testing.T) {
	mockDecoder := func(tx []byte) (sdk.Tx, error) {
		return nil, nil
	}
	group := NewTxGroup(mockDecoder, 0)
	err := group.groupByAddress()
	require.NoError(t, err)
	require.Equal(t, 0, len(group.groups))
}

func TestTxGroup_GroupByTxs_DifferentSenders(t *testing.T) {
	// Create different addresses
	fromAddr1 := common.HexToAddress("0x1234567890123456789012345678901234567890")
	fromAddr2 := common.HexToAddress("0x0987654321098765432109876543210987654321")
	toAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")

	// Create a TxGroup with mock decoder
	mockDecoder := func(tx []byte) (sdk.Tx, error) {
		return nil, nil
	}
	group := NewTxGroup(mockDecoder, 2)

	// Add transactions from different senders
	group.evmTxMetas = append(group.evmTxMetas, &TxMeta{
		RawTx: []byte("tx1"),
		From:  fromAddr1,
		To:    &toAddr,
		Nonce: 1,
		Data:  []byte("data1"),
		Index: 0,
	})
	group.evmTxMetas = append(group.evmTxMetas, &TxMeta{
		RawTx: []byte("tx2"),
		From:  fromAddr2,
		To:    &toAddr,
		Nonce: 1,
		Data:  []byte("data2"),
		Index: 1,
	})

	// Test grouping
	err := group.groupByAddress()
	require.NoError(t, err)

	// Verify the transactions are in separate groups
	require.Equal(t, 2, len(group.groups))
	require.Equal(t, 1, len(group.groups[fromAddr1.Hex()]))
	require.Equal(t, 1, len(group.groups[fromAddr2.Hex()]))
}

func TestTxGroup_GroupByTxs_AssociateTxs(t *testing.T) {
	// Create addresses
	fromAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	toAddr := common.HexToAddress("0x0987654321098765432109876543210987654321")

	// Create a TxGroup with mock decoder
	mockDecoder := func(tx []byte) (sdk.Tx, error) {
		return nil, nil
	}
	group := NewTxGroup(mockDecoder, 3)

	// Add regular and associate transactions
	group.evmTxMetas = append(group.evmTxMetas, &TxMeta{
		RawTx:       []byte("tx1"),
		From:        fromAddr,
		To:          &toAddr,
		Nonce:       1,
		Data:        []byte("data1"),
		IsAssociate: false,
		Index:       0,
	})
	group.evmTxMetas = append(group.evmTxMetas, &TxMeta{
		RawTx:       []byte("tx2"),
		From:        fromAddr,
		To:          nil,
		Nonce:       0,
		Data:        nil,
		IsAssociate: true,
		Index:       1,
	})
	group.evmTxMetas = append(group.evmTxMetas, &TxMeta{
		RawTx:       []byte("tx3"),
		From:        fromAddr,
		To:          &toAddr,
		Nonce:       2,
		Data:        []byte("data3"),
		IsAssociate: false,
		Index:       2,
	})

	// Test grouping
	err := group.groupByAddress()
	require.NoError(t, err)
	err = group.getAssociateTxs()
	require.NoError(t, err)

	// Verify associate transactions are separated
	require.Equal(t, 1, len(group.associateTxs))
	require.Equal(t, []byte("tx2"), group.associateTxs[0])
	require.Equal(t, 2, len(group.groups[fromAddr.Hex()]))
}

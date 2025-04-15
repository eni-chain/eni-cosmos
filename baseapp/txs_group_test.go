package baseapp

import (
	"github.com/cosmos/cosmos-sdk/baseapp/mocks"
	"github.com/golang/mock/gomock"
	"testing"

	"errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types/ethtx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestNewTxGroup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDecoder := mocks.NewMockTxDecoder(ctrl)
	capacity := 100

	txGroup := NewTxGroup(mockDecoder, capacity)

	assert.NotNil(t, txGroup)
	assert.Equal(t, mockDecoder, txGroup.txDecoder)
	assert.Equal(t, 0, len(txGroup.evmTxMetas))
	assert.Equal(t, capacity, cap(txGroup.evmTxMetas))
	assert.NotNil(t, txGroup.groups)
	assert.Equal(t, 0, len(txGroup.associateTxs))
	assert.Equal(t, 0, len(txGroup.otherTxs))
	assert.Equal(t, 0, len(txGroup.sequentialTxs))
	assert.NotNil(t, txGroup.txGroupDAG)
}

func TestGroupByTxs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	to := &common.Address{1}
	t.Run("Successful EVM Transaction", func(t *testing.T) {
		mockDecoder := mocks.NewMockTxDecoder(ctrl)
		mockTx := mocks.NewMockTx(ctrl)
		mockMsg := mocks.NewMockMsgEVMTransaction(ctrl)
		dynamicFeeTx := &ethtx.DynamicFeeTx{
			To:    to.String(),
			Nonce: 1,
			Data:  []byte("test"),
		}

		mockDecoder.EXPECT().Decode([]byte("tx1")).Return(mockTx, nil)
		mockTx.EXPECT().GetMsgs().Return([]sdk.Msg{mockMsg})
		mockMsg.EXPECT().GetEvmSender().Return(common.Address{2}, nil)
		mockMsg.EXPECT().GetCachedValue().Return(dynamicFeeTx)

		app := &BaseApp{txDecoder: mockDecoder.Decode}
		txs := [][]byte{[]byte("tx1")}

		txGroup, err := app.GroupByTxs(txs)

		assert.NoError(t, err)
		assert.NotNil(t, txGroup)
		assert.Equal(t, 1, len(txGroup.evmTxMetas))
		assert.Equal(t, common.Address{2}, txGroup.evmTxMetas[0].From)
		assert.Equal(t, &common.Address{1}, txGroup.evmTxMetas[0].To)
		assert.Equal(t, uint64(1), txGroup.evmTxMetas[0].Nonce)
	})

	t.Run("Decode Error", func(t *testing.T) {
		mockDecoder := mocks.NewMockTxDecoder(ctrl)
		mockDecoder.EXPECT().Decode([]byte("tx1")).Return(nil, errors.New("decode error"))

		app := &BaseApp{txDecoder: mockDecoder.Decode}
		txs := [][]byte{[]byte("tx1")}

		txGroup, err := app.GroupByTxs(txs)

		assert.Error(t, err)
		assert.Nil(t, txGroup)
		assert.Contains(t, err.Error(), "failed to process 1 transactions")
	})

	t.Run("Non-EVM Transaction", func(t *testing.T) {
		mockDecoder := mocks.NewMockTxDecoder(ctrl)
		mockTx := mocks.NewMockTx(ctrl)
		mockMsg := mocks.NewMockMsgEVMTransaction(ctrl)

		mockDecoder.EXPECT().Decode([]byte("tx1")).Return(mockTx, nil)
		mockTx.EXPECT().GetMsgs().Return([]sdk.Msg{mockMsg})

		app := &BaseApp{txDecoder: mockDecoder.Decode}
		txs := [][]byte{[]byte("tx1")}

		txGroup, err := app.GroupByTxs(txs)

		assert.NoError(t, err)
		assert.NotNil(t, txGroup)
		assert.Equal(t, 1, len(txGroup.otherTxs))
		assert.Equal(t, []byte("tx1"), txGroup.otherTxs[0])
	})
}

func TestProcessTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	txGroup := NewTxGroup(mocks.NewMockTxDecoder(ctrl), 10)

	t.Run("Decode Error", func(t *testing.T) {
		mockDecoder := mocks.NewMockTxDecoder(ctrl)
		txGroup.txDecoder = mockDecoder
		mockDecoder.EXPECT().Decode([]byte("tx1")).Return(nil, errors.New("decode error"))

		result := processTransaction(txGroup, 0, []byte("tx1"))

		assert.Error(t, result.err)
		assert.Contains(t, result.err.Error(), "decode error")
		assert.Nil(t, result.meta)
		assert.Nil(t, result.entry)
	})

	t.Run("Non-EVM Transaction", func(t *testing.T) {
		mockDecoder := mocks.NewMockTxDecoder(ctrl)
		mockTx := mocks.NewMockTx(ctrl)
		mockMsg := mocks.NewMockMsgEVMTransaction(ctrl)

		txGroup.txDecoder = mockDecoder
		mockDecoder.EXPECT().Decode([]byte("tx1")).Return(mockTx, nil)
		mockTx.EXPECT().GetMsgs().Return([]sdk.Msg{mockMsg})

		result := processTransaction(txGroup, 0, []byte("tx1"))

		assert.NoError(t, result.err)
		assert.NotNil(t, result.entry)
		assert.Nil(t, result.meta)
		assert.Equal(t, []byte("tx1"), result.entry.Tx)
	})

	t.Run("EVM Transaction", func(t *testing.T) {
		to := &common.Address{1}
		mockDecoder := mocks.NewMockTxDecoder(ctrl)
		mockTx := mocks.NewMockTx(ctrl)
		mockMsg := mocks.NewMockMsgEVMTransaction(ctrl)
		dynamicFeeTx := &ethtx.DynamicFeeTx{
			To:    to.String(),
			Nonce: 1,
			Data:  []byte("test"),
		}

		txGroup.txDecoder = mockDecoder
		mockDecoder.EXPECT().Decode([]byte("tx1")).Return(mockTx, nil)
		mockTx.EXPECT().GetMsgs().Return([]sdk.Msg{mockMsg})
		mockMsg.EXPECT().GetEvmSender().Return(common.Address{2}, nil)
		mockMsg.EXPECT().GetCachedValue().Return(dynamicFeeTx)

		result := processTransaction(txGroup, 0, []byte("tx1"))

		assert.NoError(t, result.err)
		assert.Nil(t, result.entry)
		assert.NotNil(t, result.meta)
		assert.Equal(t, common.Address{2}, result.meta.From)
		assert.Equal(t, &common.Address{1}, result.meta.To)
		assert.Equal(t, uint64(1), result.meta.Nonce)
	})
}

func TestFilterEvmTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	txGroup := NewTxGroup(mocks.NewMockTxDecoder(ctrl), 10)

	t.Run("Successful Case", func(t *testing.T) {
		mockTx := mocks.NewMockTx(ctrl)
		mockMsg := mocks.NewMockMsgEVMTransaction(ctrl)
		dynamicFeeTx := &ethtx.DynamicFeeTx{}

		mockTx.EXPECT().GetMsgs().Return([]sdk.Msg{mockMsg})
		mockMsg.EXPECT().GetEvmSender().Return(common.Address{1}, nil)
		mockMsg.EXPECT().GetCachedValue().Return(dynamicFeeTx)

		data, sender, isAssociate, err := txGroup.filterEvmTx(mockTx)

		assert.NoError(t, err)
		assert.Equal(t, common.Address{1}, sender)
		assert.False(t, isAssociate)
		assert.Equal(t, dynamicFeeTx, data)
	})

	t.Run("Nil Cached Value", func(t *testing.T) {
		mockTx := mocks.NewMockTx(ctrl)
		mockMsg := mocks.NewMockMsgEVMTransaction(ctrl)

		mockTx.EXPECT().GetMsgs().Return([]sdk.Msg{mockMsg})
		mockMsg.EXPECT().GetEvmSender().Return(common.Address{1}, nil)
		mockMsg.EXPECT().GetCachedValue().Return(nil)

		_, _, _, err := txGroup.filterEvmTx(mockTx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil cached value")
	})
}

func TestDecodeEvmTx(t *testing.T) {
	txGroup := NewTxGroup(mocks.NewMockTxDecoder(gomock.NewController(t)), 10)
	to := &common.Address{1}
	t.Run("DynamicFeeTx", func(t *testing.T) {
		data := &ethtx.DynamicFeeTx{
			To:    to.String(),
			Nonce: 1,
			Data:  []byte("test"),
		}

		meta, err := txGroup.decodeEvmTx(data, common.Address{2}, 0, []byte("tx1"), false)

		assert.NoError(t, err)
		assert.Equal(t, common.Address{2}, meta.From)
		assert.Equal(t, &common.Address{1}, meta.To)
		assert.Equal(t, uint64(1), meta.Nonce)
		assert.Equal(t, []byte("test"), meta.Data)
		assert.False(t, meta.IsAssociate)
	})

	t.Run("AssociateTx", func(t *testing.T) {
		data := &ethtx.AssociateTx{}

		meta, err := txGroup.decodeEvmTx(data, common.Address{2}, 0, []byte("tx1"), true)

		assert.NoError(t, err)
		assert.Equal(t, common.Address{2}, meta.From)
		assert.Nil(t, meta.To)
		assert.Equal(t, uint64(0), meta.Nonce)
		assert.Nil(t, meta.Data)
		assert.True(t, meta.IsAssociate)
	})

	t.Run("Unsupported Tx Type", func(t *testing.T) {
		data := struct{}{}

		_, err := txGroup.decodeEvmTx(data, common.Address{2}, 0, []byte("tx1"), false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported transaction type")
	})
}

func TestGroupByAddress(t *testing.T) {
	txGroup := NewTxGroup(mocks.NewMockTxDecoder(gomock.NewController(t)), 10)
	txGroup.evmTxMetas = []*TxMeta{
		{From: common.Address{1}, Nonce: 2, RawTx: []byte("tx2")},
		{From: common.Address{1}, Nonce: 1, RawTx: []byte("tx1")},
		{From: common.Address{2}, Nonce: 1, RawTx: []byte("tx3")},
	}

	err := txGroup.groupByAddress()

	assert.NoError(t, err)
	assert.Equal(t, 2, len(txGroup.groups))
	assert.Equal(t, 2, len(txGroup.groups[common.Address{1}.Hex()]))
	assert.Equal(t, 1, len(txGroup.groups[common.Address{2}.Hex()]))
	assert.Equal(t, uint64(1), txGroup.groups[common.Address{1}.Hex()][0].Nonce)
	assert.Equal(t, uint64(2), txGroup.groups[common.Address{1}.Hex()][1].Nonce)
}

func TestGetAssociateTxs(t *testing.T) {
	txGroup := NewTxGroup(mocks.NewMockTxDecoder(gomock.NewController(t)), 10)
	txGroup.groups = map[string][]*TxMeta{
		"addr1": {
			{RawTx: []byte("tx1"), IsAssociate: true},
			{RawTx: []byte("tx2"), IsAssociate: false},
		},
		"addr2": {
			{RawTx: []byte("tx3"), IsAssociate: false},
		},
	}

	err := txGroup.getAssociateTxs()

	assert.NoError(t, err)
	assert.Equal(t, 1, len(txGroup.associateTxs))
	assert.Equal(t, []byte("tx1"), txGroup.associateTxs[0])
	assert.Equal(t, 1, len(txGroup.groups["addr1"]))
	assert.Equal(t, []byte("tx2"), txGroup.groups["addr1"][0].RawTx)
	assert.Equal(t, 1, len(txGroup.groups["addr2"]))
}

func TestBuildSampDag(t *testing.T) {
	txGroup := NewTxGroup(mocks.NewMockTxDecoder(gomock.NewController(t)), 10)
	txGroup.associateTxs = [][]byte{[]byte("assoc1"), []byte("assoc2")}
	txGroup.otherTxs = [][]byte{[]byte("other1")}
	txGroup.groups = map[string][]*TxMeta{
		"addr1": {
			{RawTx: []byte("tx1")},
			{RawTx: []byte("tx2")},
		},
		"addr2": {
			{RawTx: []byte("tx3")},
		},
	}

	txGroup.buildSampDag()

	assert.Equal(t, 6, len(txGroup.txGroupDAG.txs))
	assert.Equal(t, 3, len(txGroup.txGroupDAG.dag))
	assert.Equal(t, []byte("assoc1"), txGroup.txGroupDAG.txs[0])
	assert.Equal(t, []byte("assoc2"), txGroup.txGroupDAG.txs[1])
	assert.Equal(t, []byte("other1"), txGroup.txGroupDAG.txs[2])
	assert.Equal(t, int64(2), txGroup.txGroupDAG.dag[0]) // associate txs
	assert.Equal(t, int64(1), txGroup.txGroupDAG.dag[1]) // other txs
	assert.Equal(t, int64(3), txGroup.txGroupDAG.dag[2]) // account group
}

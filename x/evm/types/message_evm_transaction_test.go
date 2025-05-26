package types_test

import (
	"encoding/hex"
	"math/big"
	"testing"

	testkeeper "github.com/cosmos/cosmos-sdk/testutil/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	protov2 "google.golang.org/protobuf/proto"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/derived"
	"github.com/cosmos/cosmos-sdk/x/evm/keeper"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types/ethtx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func createTestContext(t *testing.T) (sdktypes.Context, *keeper.Keeper) {
	app, ctx := testkeeper.NewMockApp(t, false)
	return ctx, app.GetEVMKeeper()
}

func TestIsAssociate(t *testing.T) {
	tx, err := types.NewMsgEVMTransaction(&ethtx.AssociateTx{})
	require.Nil(t, err)
	require.True(t, tx.IsAssociateTx())
}

func TestIsNotAssociate(t *testing.T) {
	tx, err := types.NewMsgEVMTransaction(nil)
	require.Error(t, err)

	tx, err = types.NewMsgEVMTransaction(&ethtx.AccessTuple{})
	require.Nil(t, err)
	require.False(t, tx.IsAssociateTx())
}

func TestAsTransaction(t *testing.T) {
	ctx, k := createTestContext(t)
	chainID := k.ChainID(ctx)
	chainCfg := types.DefaultChainConfig()
	ethCfg := chainCfg.EthereumConfig(chainID)
	blockNum := big.NewInt(ctx.BlockHeight())
	privKey := testkeeper.MockPrivateKey()
	testPrivHex := hex.EncodeToString(privKey.Bytes())
	key, _ := crypto.HexToECDSA(testPrivHex)
	to := new(common.Address)
	txData := ethtypes.DynamicFeeTx{
		Nonce:     1,
		GasFeeCap: big.NewInt(10000000000000),
		Gas:       1000,
		To:        to,
		Value:     big.NewInt(1000000000000000),
		Data:      []byte("abc"),
		ChainID:   chainID,
	}

	signer := ethtypes.MakeSigner(ethCfg, blockNum, uint64(ctx.BlockTime().Unix()))
	tx, err := ethtypes.SignTx(ethtypes.NewTx(&txData), signer, key)
	typedTx, err := ethtx.NewDynamicFeeTx(tx)
	msg, err := types.NewMsgEVMTransaction(typedTx)
	require.Nil(t, err)
	ethTx, ethTxData := msg.AsTransaction()
	require.Equal(t, chainID, ethTx.ChainId())
	require.Equal(t, uint64(1), ethTx.Nonce())
	require.Equal(t, []byte("abc"), ethTx.Data())
	require.Nil(t, ethTxData.Validate())

}

func TestMustGetEVMTransactionMessage(t *testing.T) {
	testMsg := types.MsgEVMTransaction{
		Data:    nil,
		Derived: nil,
	}
	testTx := NewTestTx([]sdk.Msg{&testMsg})

	types.MustGetEVMTransactionMessage(testTx)
}

func TestMustGetEVMTransactionMessageMultipleMsgs(t *testing.T) {
	testMsg := types.MsgEVMTransaction{
		Data:    nil,
		Derived: nil,
	}
	testTx := NewTestTx([]sdk.Msg{&testMsg, &testMsg})

	defer func() { recover() }()
	types.MustGetEVMTransactionMessage(testTx)
	t.Errorf("Should not be able to convert a non evm emssage")
}

func TestAttackerUnableToSetDerived(t *testing.T) {
	msg := types.MsgEVMTransaction{Derived: &derived.Derived{SenderEVMAddr: common.BytesToAddress([]byte("abc"))}}
	bz, err := msg.Marshal()
	require.Nil(t, err)
	decoded := types.MsgEVMTransaction{}
	err = decoded.Unmarshal(bz)
	require.Nil(t, err)
	require.Equal(t, common.Address{}, decoded.Derived.SenderEVMAddr)
}

type TestTx struct {
	msgs []sdk.Msg
}

func (t TestTx) GetMsgsV2() ([]protov2.Message, error) {
	//TODO implement me
	panic("implement me")
}

func NewTestTx(msgs []sdk.Msg) TestTx {
	return TestTx{msgs: msgs}
}

func (t TestTx) GetMsgs() []sdk.Msg {
	return t.msgs
}

func (t TestTx) ValidateBasic() error {
	return nil
}

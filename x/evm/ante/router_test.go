package ante_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/evm/ante"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types/ethtx"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type mockAnteState struct {
	call string
}

func (m *mockAnteState) regularAnteHandler(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
	m.call = "regular"
	return ctx, nil
}

func (m *mockAnteState) evmAnteHandler(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
	m.call = "evm"
	return ctx, nil
}

func TestRouter(t *testing.T) {
	bankMsg := &banktypes.MsgSend{}
	ethTx := mockLegacyTransaction(big.NewInt(20))
	tx, err := ethtx.NewLegacyTx(ethTx)
	evmMsg, _ := types.NewMsgEVMTransaction(tx)
	mockAnte := mockAnteState{}
	router := ante.NewEVMRouterDecorator(mockAnte.regularAnteHandler, mockAnte.evmAnteHandler)
	_, err = router.AnteHandle(sdk.Context{}, mockTx{msgs: []sdk.Msg{bankMsg}}, false)
	require.Nil(t, err)
	require.Equal(t, "regular", mockAnte.call)
	_, err = router.AnteHandle(sdk.Context{}, mockTx{msgs: []sdk.Msg{evmMsg}}, false)
	require.Nil(t, err)
	require.Equal(t, "evm", mockAnte.call)
	_, err = router.AnteHandle(sdk.Context{}, mockTx{msgs: []sdk.Msg{evmMsg, bankMsg}}, false)
	require.NotNil(t, err)
}

func mockLegacyTransaction(value *big.Int) *ethtypes.Transaction {
	inner := &ethtypes.LegacyTx{
		GasPrice: big.NewInt(100),
		Gas:      1000,
		To:       &common.Address{'a'},
		Value:    value,
		Data:     []byte{'b'},
		V:        big.NewInt(3),
		R:        big.NewInt(5),
		S:        big.NewInt(7),
	}
	return ethtypes.NewTx(inner)
}

package keeper

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/evm/keeper"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/go-bip39"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

type MockApp struct {
	App       *baseapp.BaseApp
	EVMKeeper *keeper.Keeper
}

func (app *MockApp) SetApp(a *baseapp.BaseApp) {
	app.App = a
}

func (app *MockApp) SetEVMKeeper(evm *keeper.Keeper) {
	app.EVMKeeper = evm
}

func (app *MockApp) GetEVMKeeper() *keeper.Keeper {
	return app.EVMKeeper
}

func (app *MockApp) NewContext(isCheckTx bool) sdk.Context {
	return app.App.NewContext(isCheckTx)
}

func NewMockApp(t testing.TB, isCheckTx bool) (*MockApp, sdk.Context) {
	app := &MockApp{}

	k, ctx := NewEvmKeeper(t, isCheckTx)

	ba := setup()

	app.SetApp(ba)
	app.SetEVMKeeper(k)

	return app, ctx
}

func setup() (res *baseapp.BaseApp) {
	db := dbm.NewMemDB()
	res = baseapp.NewBaseApp("goeni", log.NewNopLogger(), db, nil, baseapp.SetChainID("goeni"))
	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)
	res.SetTxEncoder(txConfig.TxEncoder())
	res.SetTxDecoder(txConfig.TxDecoder())

	return res
}

func MockPrivateKey() cryptotypes.PrivKey {
	// Generate a new Sei private key
	entropySeed, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropySeed)
	algo := hd.Secp256k1
	derivedPriv, _ := algo.Derive()(mnemonic, "", "")
	return algo.Generate()(derivedPriv)
}

func PrivateKeyToAddresses(privKey cryptotypes.PrivKey) (sdk.AccAddress, common.Address) {
	// Encode the private key to hex (i.e. what wallets do behind the scene when users reveal private keys)
	testPrivHex := hex.EncodeToString(privKey.Bytes())

	// Sign an Ethereum transaction with the hex private key
	key, _ := crypto.HexToECDSA(testPrivHex)
	msg := crypto.Keccak256([]byte("foo"))
	sig, _ := crypto.Sign(msg, key)

	// Recover the public keys from the Ethereum signature
	recoveredPub, _ := crypto.Ecrecover(msg, sig)
	pubKey, _ := crypto.UnmarshalPubkey(recoveredPub)

	evmAddr := crypto.PubkeyToAddress(*pubKey)
	eniAddr := sdk.AccAddress(evmAddr[:])

	return eniAddr, evmAddr
}

func NewEvmKeeper(t testing.TB, isCheckTx bool) (*keeper.Keeper, sdk.Context) {
	evmStoreKey := storetypes.NewKVStoreKey(types.StoreKey)
	transientStoreKey := storetypes.NewTransientStoreKey(types.TransientStoreKey)
	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	bankStoreKey := storetypes.NewKVStoreKey(banktypes.StoreKey)
	paramstore := NewMockParamStore()

	registry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)

	maccPerms := map[string][]string{
		"fee_collector":          {"fee_collector"},
		"mint":                   {"minter"},
		"bonded_tokens_pool":     {"burner", "staking"},
		"not_bonded_tokens_pool": {"burner", "staking"},
		"multiPerm":              {"burner", "minter", "staking"},
		"random":                 {"random"},
		"evm":                    {"minter", "evm", "staking", "random"},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		cdc, runtime.NewKVStoreService(authStoreKey), authtypes.ProtoBaseAccount, maccPerms, authcodec.NewBech32Codec("eni"),
		sdk.Bech32MainPrefix, authtypes.NewModuleAddress("gov").String(),
	)

	bankKeeper := bankkeeper.NewBaseKeeper(
		cdc,
		runtime.NewKVStoreService(bankStoreKey),
		accountKeeper,
		nil,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)

	evmKeeper := keeper.NewKeeper(evmStoreKey, transientStoreKey, paramstore, bankKeeper, &accountKeeper, nil, cdc, log.NewNopLogger())

	cms := store.NewCommitMultiStore(paramstore.db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(evmStoreKey, storetypes.StoreTypeIAVL, paramstore.db)
	cms.MountStoreWithDB(authStoreKey, storetypes.StoreTypeIAVL, paramstore.db)
	cms.MountStoreWithDB(bankStoreKey, storetypes.StoreTypeIAVL, paramstore.db)
	cms.MountStoreWithDB(transientStoreKey, storetypes.StoreTypeTransient, paramstore.db)
	err := cms.LoadLatestVersion()
	assert.NoError(t, err)

	ctx := sdk.NewContext(cms, cmtproto.Header{Time: time.Now()}, isCheckTx, log.NewNopLogger())

	evmKeeper.SetParams(ctx, types.DefaultParams())

	feeEniAddr := accountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	feeEvmAddr := common.BytesToAddress([]byte("fee_collector"))
	coinbaseAddr := keeper.GetCoinbaseAddress()
	evmKeeper.SetAddressMapping(ctx, feeEniAddr, feeEvmAddr)
	evmKeeper.SetAddressMapping(ctx, feeEniAddr, coinbaseAddr)

	return evmKeeper, ctx
}

type mockParamStore struct {
	db *dbm.MemDB
}

func NewMockParamStore() *mockParamStore {
	return &mockParamStore{
		db: dbm.NewMemDB(),
	}
}

func (m *mockParamStore) GetParamSetIfExists(ctx sdk.Context, ps paramtypes.ParamSet) {
	for _, pair := range ps.ParamSetPairs() {
		m.GetIfExists(pair.Key, pair.Value)
	}
}

func (m *mockParamStore) SetParamSet(ctx sdk.Context, ps paramtypes.ParamSet) {
	for _, pair := range ps.ParamSetPairs() {
		v := reflect.Indirect(reflect.ValueOf(pair.Value)).Interface()

		if err := pair.ValidatorFn(v); err != nil {
			panic(fmt.Sprintf("value from ParamSetPair is invalid: %s", err))
		}

		m.Set(pair.Key, v)
	}
}

func (m *mockParamStore) HasKeyTable() bool {
	return true
}

func (m *mockParamStore) WithKeyTable(table paramtypes.KeyTable) paramtypes.Subspace {
	return paramtypes.Subspace{}
}

func (m *mockParamStore) Set(key []byte, value interface{}) {
	bz, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("marshal param store error, %s", err))
	}
	err = m.db.Set(key, bz)
	if err != nil {
		panic(fmt.Sprintf("set param store error, %s", err))
	}
}

func (m *mockParamStore) GetIfExists(key []byte, ptr interface{}) {
	bz, err := m.db.Get(key)
	if err != nil {
		panic(fmt.Sprintf("get param store error, %s", err))
	}
	if bz == nil {
		return
	}
	err = json.Unmarshal(bz, ptr)
	if err != nil {
		panic(fmt.Sprintf("unmarshal param store error, %s", err))
	}
}

func MockAddressPair() (sdk.AccAddress, common.Address) {
	return PrivateKeyToAddresses(MockPrivateKey())
}

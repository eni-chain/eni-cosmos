package types

import (
	"context"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/gogoproto/proto"

	"cosmossdk.io/core/comet"
	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	"cosmossdk.io/store/gaskv"
	storetypes "cosmossdk.io/store/types"
)

// ExecMode defines the execution mode which can be set on a Context.
type ExecMode uint8

// All possible execution modes.
const (
	ExecModeCheck ExecMode = iota
	ExecModeReCheck
	ExecModeSimulate
	ExecModePrepareProposal
	ExecModeProcessProposal
	ExecModeVoteExtension
	ExecModeVerifyVoteExtension
	ExecModeFinalize
)

/*
Context is an immutable object contains all information needed to
process a request.

It contains a context.Context object inside if you want to use that,
but please do not over-use it. We try to keep all data structured
and standard additions here would be better just to add to the Context struct
*/
type Context struct {
	baseCtx context.Context
	ms      storetypes.MultiStore
	// Deprecated: Use HeaderService for height, time, and chainID and CometService for the rest
	header cmtproto.Header
	// Deprecated: Use HeaderService for hash
	headerHash []byte
	// Deprecated: Use HeaderService for chainID and CometService for the rest
	chainID              string
	txBytes              []byte
	logger               log.Logger
	voteInfo             []abci.VoteInfo
	gasMeter             storetypes.GasMeter
	blockGasMeter        storetypes.GasMeter
	checkTx              bool
	recheckTx            bool // if recheckTx == true, then checkTx must also be true
	sigverifyTx          bool // when run simulation, because the private key corresponding to the account in the genesis.json randomly generated, we must skip the sigverify.
	execMode             ExecMode
	minGasPrice          DecCoins
	consParams           cmtproto.ConsensusParams
	eventManager         EventManagerI
	priority             int64 // The tx priority, only relevant in CheckTx
	kvGasConfig          storetypes.GasConfig
	transientKVGasConfig storetypes.GasConfig
	streamingManager     storetypes.StreamingManager
	cometInfo            comet.BlockInfo
	headerInfo           header.Info
	isParallelExec       bool
	isSimpleDag          bool

	txIndex           int
	txSum             [32]byte
	pendingTxChecker  abci.PendingTxChecker // Checker for pending transaction, only relevant in CheckTx
	checkTxCallback   func(Context, error)  // callback to make at the end of CheckTx. Input param is the error (nil-able) of `runMsgs`
	deliverTxCallback func(Context)         // callback to make at the end of DeliverTx.
	expireTxHandler   func()                // callback that the mempool invokes when a tx is expired

	// EVM properties
	isEvm                               bool   // EVM transaction flag
	evmNonce                            uint64 // EVM Transaction nonce
	evmSenderAddress                    string // EVM Sender address
	evmTxHash                           string // EVM TX hash
	evmVmError                          string // EVM VM error during execution
	evmEntryViaWasmdPrecompile          bool   // EVM is entered via wasmd precompile directly
	evmPrecompileCalledFromDelegateCall bool   // EVM precompile is called from a delegate call
	isPendingTransaction                bool

	isFastMempool    bool
	traceSpanContext context.Context
}

// Proposed rename, not done to avoid API breakage
type Request = Context

// Read-only accessors
func (c Context) Context() context.Context                      { return c.baseCtx }
func (c Context) MultiStore() storetypes.MultiStore             { return c.ms }
func (c Context) BlockHeight() int64                            { return c.header.Height }
func (c Context) BlockTime() time.Time                          { return c.header.Time }
func (c Context) ChainID() string                               { return c.chainID }
func (c Context) TxBytes() []byte                               { return c.txBytes }
func (c Context) Logger() log.Logger                            { return c.logger }
func (c Context) VoteInfos() []abci.VoteInfo                    { return c.voteInfo }
func (c Context) GasMeter() storetypes.GasMeter                 { return c.gasMeter }
func (c Context) BlockGasMeter() storetypes.GasMeter            { return c.blockGasMeter }
func (c Context) IsCheckTx() bool                               { return c.checkTx }
func (c Context) IsReCheckTx() bool                             { return c.recheckTx }
func (c Context) IsPendingTransaction() bool                    { return c.isPendingTransaction }
func (c Context) IsSigverifyTx() bool                           { return c.sigverifyTx }
func (c Context) ExecMode() ExecMode                            { return c.execMode }
func (c Context) MinGasPrices() DecCoins                        { return c.minGasPrice }
func (c Context) EventManager() EventManagerI                   { return c.eventManager }
func (c Context) Priority() int64                               { return c.priority }
func (c Context) KVGasConfig() storetypes.GasConfig             { return c.kvGasConfig }
func (c Context) TransientKVGasConfig() storetypes.GasConfig    { return c.transientKVGasConfig }
func (c Context) StreamingManager() storetypes.StreamingManager { return c.streamingManager }
func (c Context) CometInfo() comet.BlockInfo                    { return c.cometInfo }
func (c Context) HeaderInfo() header.Info                       { return c.headerInfo }
func (c Context) TxIndex() int                                  { return c.txIndex }
func (c Context) TxSum() [32]byte                               { return c.txSum }
func (c Context) DeliverTxCallback() func(Context)              { return c.deliverTxCallback }
func (c Context) PendingTxChecker() abci.PendingTxChecker       { return c.pendingTxChecker }
func (c Context) CheckTxCallback() func(Context, error)         { return c.checkTxCallback }
func (c Context) ExpireTxHandler() func()                       { return c.expireTxHandler }
func (c Context) IsParallelTx() bool                            { return c.isParallelExec }
func (c Context) IsSimpleDag() bool                             { return c.isSimpleDag }
func (c Context) IsFastMempool() bool                           { return c.isFastMempool }

// Getters
func (c Context) IsEVM() bool {
	return c.isEvm
}

func (c Context) EVMNonce() uint64 {
	return c.evmNonce
}

func (c Context) EVMSenderAddress() string {
	return c.evmSenderAddress
}

func (c Context) EVMTxHash() string {
	return c.evmTxHash
}

func (c Context) EVMVmError() string {
	return c.evmVmError
}

func (c Context) EVMEntryViaWasmdPrecompile() bool {
	return c.evmEntryViaWasmdPrecompile
}

func (c Context) EVMPrecompileCalledFromDelegateCall() bool {
	return c.evmPrecompileCalledFromDelegateCall
}

// Setters
func (c Context) WithIsEVM(evm bool) Context {
	c.isEvm = evm
	return c
}

func (c Context) WithIsPendingTx(is bool) Context {
	c.isPendingTransaction = is
	return c
}

func (c Context) WithIsFastMempool(is bool) Context {
	c.isFastMempool = is
	return c
}

func (c Context) WithEVMNonce(evmNonce uint64) Context {
	c.evmNonce = evmNonce
	return c
}

func (c Context) WithEVMSenderAddress(evmSenderAddress string) Context {
	c.evmSenderAddress = evmSenderAddress
	return c
}

func (c Context) WithEVMTxHash(evmTxHash string) Context {
	c.evmTxHash = evmTxHash
	return c
}

func (c Context) WithEVMVmError(evmVmError string) Context {
	c.evmVmError = evmVmError
	return c
}

func (c Context) WithEVMEntryViaWasmdPrecompile(evmEntryViaWasmdPrecompile bool) Context {
	c.evmEntryViaWasmdPrecompile = evmEntryViaWasmdPrecompile
	return c
}

func (c Context) WithEVMPrecompileCalledFromDelegateCall(evmPrecompileCalledFromDelegateCall bool) Context {
	c.evmPrecompileCalledFromDelegateCall = evmPrecompileCalledFromDelegateCall
	return c
}

func (c Context) WithParallelExec(isParallelExec bool) Context {
	c.isParallelExec = isParallelExec
	return c
}

func (c Context) WithSimpleDag(isSimpleDag bool) Context {
	c.isSimpleDag = isSimpleDag
	return c
}

// clone the header before returning
func (c Context) BlockHeader() cmtproto.Header {
	t := c.header.Time
	h := c.header
	h.Time = time.Time{} //set time to zero, so proto.Clone can work
	msg := proto.Clone(&h).(*cmtproto.Header)
	msg.Time = t //set time back
	return *msg
}

func (c Context) TraceSpanContext() context.Context {
	return c.traceSpanContext
}

// HeaderHash returns a copy of the header hash obtained during abci.RequestBeginBlock
func (c Context) HeaderHash() []byte {
	hash := make([]byte, len(c.headerHash))
	copy(hash, c.headerHash)
	return hash
}

func (c Context) ConsensusParams() cmtproto.ConsensusParams {
	return c.consParams
}

func (c Context) Deadline() (deadline time.Time, ok bool) {
	return c.baseCtx.Deadline()
}

func (c Context) Done() <-chan struct{} {
	return c.baseCtx.Done()
}

func (c Context) Err() error {
	return c.baseCtx.Err()
}

// create a new context
func NewContext(ms storetypes.MultiStore, header cmtproto.Header, isCheckTx bool, logger log.Logger) Context {
	// https://github.com/gogo/protobuf/issues/519
	header.Time = header.Time.UTC()
	return Context{
		baseCtx:              context.Background(),
		ms:                   ms,
		header:               header,
		chainID:              header.ChainID,
		checkTx:              isCheckTx,
		sigverifyTx:          true,
		logger:               logger,
		gasMeter:             storetypes.NewInfiniteGasMeter(),
		minGasPrice:          DecCoins{},
		eventManager:         NewEventManager(),
		kvGasConfig:          storetypes.KVGasConfig(),
		transientKVGasConfig: storetypes.TransientGasConfig(),
	}
}
func (c Context) WithTxSum(txSum [32]byte) Context {
	c.txSum = txSum
	return c
}

// WithTxIndex returns a Context with the current transaction index that's being processed
func (c Context) WithTxIndex(txIndex int) Context {
	c.txIndex = txIndex
	return c
}

// WithContext returns a Context with an updated context.Context.
func (c Context) WithContext(ctx context.Context) Context {
	c.baseCtx = ctx
	return c
}

// WithMultiStore returns a Context with an updated MultiStore.
func (c Context) WithMultiStore(ms storetypes.MultiStore) Context {
	c.ms = ms
	return c
}

// WithBlockHeader returns a Context with an updated CometBFT block header in UTC time.
func (c Context) WithBlockHeader(header cmtproto.Header) Context {
	// https://github.com/gogo/protobuf/issues/519
	header.Time = header.Time.UTC()
	c.header = header
	return c
}

// WithHeaderHash returns a Context with an updated CometBFT block header hash.
func (c Context) WithHeaderHash(hash []byte) Context {
	temp := make([]byte, len(hash))
	copy(temp, hash)

	c.headerHash = temp
	return c
}

// WithBlockTime returns a Context with an updated CometBFT block header time in UTC with no monotonic component.
// Stripping the monotonic component is for time equality.
func (c Context) WithBlockTime(newTime time.Time) Context {
	newHeader := c.BlockHeader()
	// https://github.com/gogo/protobuf/issues/519
	newHeader.Time = newTime.Round(0).UTC()
	return c.WithBlockHeader(newHeader)
}

// WithProposer returns a Context with an updated proposer consensus address.
func (c Context) WithProposer(addr ConsAddress) Context {
	newHeader := c.BlockHeader()
	newHeader.ProposerAddress = addr.Bytes()
	return c.WithBlockHeader(newHeader)
}

// WithBlockHeight returns a Context with an updated block height.
func (c Context) WithBlockHeight(height int64) Context {
	newHeader := c.BlockHeader()
	newHeader.Height = height
	return c.WithBlockHeader(newHeader)
}

// WithChainID returns a Context with an updated chain identifier.
func (c Context) WithChainID(chainID string) Context {
	c.chainID = chainID
	return c
}

// WithTxBytes returns a Context with an updated txBytes.
func (c Context) WithTxBytes(txBytes []byte) Context {
	c.txBytes = txBytes
	return c
}

// WithLogger returns a Context with an updated logger.
func (c Context) WithLogger(logger log.Logger) Context {
	c.logger = logger
	return c
}

// WithVoteInfos returns a Context with an updated consensus VoteInfo.
func (c Context) WithVoteInfos(voteInfo []abci.VoteInfo) Context {
	c.voteInfo = voteInfo
	return c
}

// WithGasMeter returns a Context with an updated transaction GasMeter.
func (c Context) WithGasMeter(meter storetypes.GasMeter) Context {
	c.gasMeter = meter
	return c
}

// WithBlockGasMeter returns a Context with an updated block GasMeter
func (c Context) WithBlockGasMeter(meter storetypes.GasMeter) Context {
	c.blockGasMeter = meter
	return c
}

// WithKVGasConfig returns a Context with an updated gas configuration for
// the KVStore
func (c Context) WithKVGasConfig(gasConfig storetypes.GasConfig) Context {
	c.kvGasConfig = gasConfig
	return c
}

// WithTransientKVGasConfig returns a Context with an updated gas configuration for
// the transient KVStore
func (c Context) WithTransientKVGasConfig(gasConfig storetypes.GasConfig) Context {
	c.transientKVGasConfig = gasConfig
	return c
}

// WithIsCheckTx enables or disables CheckTx value for verifying transactions and returns an updated Context
func (c Context) WithIsCheckTx(isCheckTx bool) Context {
	c.checkTx = isCheckTx
	c.execMode = ExecModeCheck
	return c
}

// WithIsRecheckTx called with true will also set true on checkTx in order to
// enforce the invariant that if recheckTx = true then checkTx = true as well.
func (c Context) WithIsReCheckTx(isRecheckTx bool) Context {
	if isRecheckTx {
		c.checkTx = true
	}
	c.recheckTx = isRecheckTx
	c.execMode = ExecModeReCheck
	return c
}

// WithIsSigverifyTx called with true will sigverify in auth module
func (c Context) WithIsSigverifyTx(isSigverifyTx bool) Context {
	c.sigverifyTx = isSigverifyTx
	return c
}

// WithExecMode returns a Context with an updated ExecMode.
func (c Context) WithExecMode(m ExecMode) Context {
	c.execMode = m
	return c
}

// WithMinGasPrices returns a Context with an updated minimum gas price value
func (c Context) WithMinGasPrices(gasPrices DecCoins) Context {
	c.minGasPrice = gasPrices
	return c
}

// WithConsensusParams returns a Context with an updated consensus params
func (c Context) WithConsensusParams(params cmtproto.ConsensusParams) Context {
	c.consParams = params
	return c
}

// WithEventManager returns a Context with an updated event manager
func (c Context) WithEventManager(em EventManagerI) Context {
	c.eventManager = em
	return c
}

// WithPriority returns a Context with an updated tx priority
func (c Context) WithPriority(p int64) Context {
	c.priority = p
	return c
}

// WithStreamingManager returns a Context with an updated streaming manager
func (c Context) WithStreamingManager(sm storetypes.StreamingManager) Context {
	c.streamingManager = sm
	return c
}

// WithCometInfo returns a Context with an updated comet info
func (c Context) WithCometInfo(cometInfo comet.BlockInfo) Context {
	c.cometInfo = cometInfo
	return c
}

// WithHeaderInfo returns a Context with an updated header info
func (c Context) WithHeaderInfo(headerInfo header.Info) Context {
	// Settime to UTC
	headerInfo.Time = headerInfo.Time.UTC()
	c.headerInfo = headerInfo
	return c
}
func (c Context) WithDeliverTxCallback(deliverTxCallback func(Context)) Context {
	c.deliverTxCallback = deliverTxCallback
	return c
}
func (c Context) WithPendingTxChecker(pendingTxChecker abci.PendingTxChecker) Context {
	c.pendingTxChecker = pendingTxChecker
	return c
}
func (c Context) WithCheckTxCallback(checkTxCallback func(Context, error)) Context {
	c.checkTxCallback = checkTxCallback
	return c
}
func (c Context) WithExpireTxHandler(expireTxHandler func()) Context {
	c.expireTxHandler = expireTxHandler
	return c
}

func (c Context) WithTraceSpanContext(ctx context.Context) Context {
	c.traceSpanContext = ctx
	return c
}

// TODO: remove???
func (c Context) IsZero() bool {
	return c.ms == nil
}

func (c Context) WithValue(key, value interface{}) Context {
	c.baseCtx = context.WithValue(c.baseCtx, key, value)
	return c
}

func (c Context) Value(key interface{}) interface{} {
	if key == SdkContextKey {
		return c
	}

	return c.baseCtx.Value(key)
}

// ----------------------------------------------------------------------------
// Store / Caching
// ----------------------------------------------------------------------------

// KVStore fetches a KVStore from the MultiStore.
func (c Context) KVStore(key storetypes.StoreKey) storetypes.KVStore {
	return gaskv.NewStore(c.ms.GetKVStore(key), c.gasMeter, c.kvGasConfig)
}

// TransientStore fetches a TransientStore from the MultiStore.
func (c Context) TransientStore(key storetypes.StoreKey) storetypes.KVStore {
	return gaskv.NewStore(c.ms.GetKVStore(key), c.gasMeter, c.transientKVGasConfig)
}

// CacheContext returns a new Context with the multi-store cached and a new
// EventManager. The cached context is written to the context when writeCache
// is called. Note, events are automatically emitted on the parent context's
// EventManager when the caller executes the write.
func (c Context) CacheContext() (cc Context, writeCache func()) {
	cms := c.ms.CacheMultiStore()
	cc = c.WithMultiStore(cms).WithEventManager(NewEventManager())

	writeCache = func() {
		c.EventManager().EmitEvents(cc.EventManager().Events())
		cms.Write()
	}

	return cc, writeCache
}

var (
	_ context.Context    = Context{}
	_ storetypes.Context = Context{}
)

// ContextKey defines a type alias for a stdlib Context key.
type ContextKey string

// SdkContextKey is the key in the context.Context which holds the sdk.Context.
const SdkContextKey ContextKey = "sdk-context"

// WrapSDKContext returns a stdlib context.Context with the provided sdk.Context's internal
// context as a value. It is useful for passing an sdk.Context  through methods that take a
// stdlib context.Context parameter such as generated gRPC methods. To get the original
// sdk.Context back, call UnwrapSDKContext.
//
// Deprecated: there is no need to wrap anymore as the Cosmos SDK context implements context.Context.
func WrapSDKContext(ctx Context) context.Context {
	return ctx
}

// UnwrapSDKContext retrieves a Context from a context.Context instance
// attached with WrapSDKContext. It panics if a Context was not properly
// attached
func UnwrapSDKContext(ctx context.Context) Context {
	if sdkCtx, ok := ctx.(Context); ok {
		return sdkCtx
	}
	return ctx.Value(SdkContextKey).(Context)
}

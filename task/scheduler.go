package tasks

import (
	"context"
	"cosmossdk.io/log"
	"cosmossdk.io/store/multiversion"
	"cosmossdk.io/store/multiversion/occ"
	store "cosmossdk.io/store/types"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type status int32

func (s status) String() string {
	switch s {
	case statusPendingInt:
		return "pending"
	case statusExecutedInt:
		return "executed"
	case statusAbortedInt:
		return "aborted"
	case statusValidatedInt:
		return "validated"
	case statusWaitingInt:
		return "waiting"
	default:
		return fmt.Sprintf("unknown status (%d)", s) //
	}
}

const (
	maximumIterations = 10
)
const (
	statusPendingInt   = 0
	statusExecutedInt  = 1
	statusAbortedInt   = 2
	statusValidatedInt = 3
	statusWaitingInt   = 4
)

type deliverTxTask struct {
	Ctx     sdk.Context
	AbortCh chan occ.Abort

	mx            sync.RWMutex
	Status        status
	Dependencies  map[int]struct{}
	Abort         *occ.Abort
	Incarnation   int
	SdkTx         sdk.Tx
	Checksum      [32]byte
	Tx            []byte
	AbsoluteIndex int
	Response      *abci.ExecTxResult
	VersionStores map[sdk.StoreKey]*multiversion.VersionIndexedStore
	TxTracer      sdk.TxTracer
}

// AppendDependencies appends the given indexes to the task's dependencies
func (dt *deliverTxTask) AppendDependencies(deps []int) {
	dt.mx.Lock()
	defer dt.mx.Unlock()
	for _, taskIdx := range deps {
		dt.Dependencies[taskIdx] = struct{}{}
	}
}

func (dt *deliverTxTask) IsStatus(s status) bool {
	return atomic.LoadInt32((*int32)(&dt.Status)) == int32(s)
}

func (dt *deliverTxTask) SetStatus(s status) {
	atomic.StoreInt32((*int32)(&dt.Status), int32(s))
}

func (dt *deliverTxTask) Reset() {
	dt.SetStatus(statusPendingInt)
	dt.Response = nil
	dt.Abort = nil
	dt.AbortCh = nil
	dt.VersionStores = nil

	if dt.TxTracer != nil {
		dt.TxTracer.Reset()
	}
}

func (dt *deliverTxTask) Increment() {
	dt.Incarnation++
}

// Scheduler processes tasks concurrently
type Scheduler interface {
	ProcessAll(ctx sdk.Context, reqs *sdk.DeliverTxBatchRequest) ([]*abci.ExecTxResult, error)
}

type scheduler struct {
	deliverTx          func(ctx sdk.Context, tx []byte) *abci.ExecTxResult
	workers            int
	multiVersionStores map[sdk.StoreKey]multiversion.MultiVersionStore
	allTasksMap        map[int]*deliverTxTask
	allTasks           []*deliverTxTask
	executeCh          chan func()
	validateCh         chan func()
	metrics            *schedulerMetrics
	synchronous        bool // true if maxIncarnation exceeds threshold
	maxIncarnation     int  // current highest incarnation
	loger              log.Logger
	conflictCache      map[int]struct {
		valid     bool
		conflicts []int
	}
	cacheMu sync.RWMutex
}

// NewScheduler creates a new scheduler
func NewScheduler(workers int, deliverTxFunc func(ctx sdk.Context, tx []byte) *abci.ExecTxResult, loger log.Logger) Scheduler {
	return &scheduler{
		workers:   workers,
		deliverTx: deliverTxFunc,
		metrics:   &schedulerMetrics{},
		loger:     loger,
		conflictCache: make(map[int]struct {
			valid     bool
			conflicts []int
		}),
	}
}

func (s *scheduler) invalidateTask(task *deliverTxTask) {
	for _, mv := range s.multiVersionStores {
		mv.InvalidateWriteset(task.AbsoluteIndex, task.Incarnation)
		mv.ClearReadset(task.AbsoluteIndex)
		mv.ClearIterateset(task.AbsoluteIndex)
	}
	s.cacheMu.Lock()
	delete(s.conflictCache, task.AbsoluteIndex)
	s.cacheMu.Unlock()
}

func start(ctx context.Context, ch chan func(), workers int) {
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case work := <-ch:
					work()
				}
			}
		}()
	}
}

func (s *scheduler) DoValidate(work func()) {
	if s.synchronous {
		work()
		return
	}
	s.validateCh <- work
}

func (s *scheduler) DoExecute(work func()) {
	if s.synchronous {
		work()
		return
	}
	s.executeCh <- work
}

func (s *scheduler) computeConflicts(task *deliverTxTask) (bool, []int) {
	var conflicts []int
	uniq := make(map[int]struct{})
	valid := true
	for _, mv := range s.multiVersionStores {
		ok, mvConflicts := mv.ValidateTransactionState(task.AbsoluteIndex)
		for _, c := range mvConflicts {
			if _, ok := uniq[c]; !ok {
				conflicts = append(conflicts, c)
				uniq[c] = struct{}{}
			}
		}
		valid = valid && ok
	}
	sort.Ints(conflicts)
	return valid, conflicts
}

func (s *scheduler) findConflicts(task *deliverTxTask) (bool, []int) {
	s.cacheMu.RLock()
	if cached, ok := s.conflictCache[task.AbsoluteIndex]; ok {
		s.cacheMu.RUnlock()
		return cached.valid, cached.conflicts
	}
	s.cacheMu.RUnlock()

	valid, conflicts := s.computeConflicts(task)
	s.cacheMu.Lock()
	if s.conflictCache == nil {
		s.conflictCache = make(map[int]struct {
			valid     bool
			conflicts []int
		})
	}

	s.conflictCache[task.AbsoluteIndex] = struct {
		valid     bool
		conflicts []int
	}{valid, conflicts}

	s.cacheMu.Unlock()
	return valid, conflicts
}

func toTasks(reqs []*sdk.DeliverTxEntry) ([]*deliverTxTask, map[int]*deliverTxTask) {
	tasksMap := make(map[int]*deliverTxTask)
	allTasks := make([]*deliverTxTask, 0, len(reqs))
	for _, r := range reqs {
		task := &deliverTxTask{
			Tx:            r.Tx,
			SdkTx:         r.SdkTx,
			Checksum:      r.Checksum,
			AbsoluteIndex: r.AbsoluteIndex,
			Status:        statusPendingInt,
			Dependencies:  map[int]struct{}{},
			TxTracer:      r.TxTracer,
		}

		tasksMap[r.AbsoluteIndex] = task
		allTasks = append(allTasks, task)
	}
	return allTasks, tasksMap
}

func (s *scheduler) collectResponses(tasks []*deliverTxTask) []*abci.ExecTxResult {
	res := make([]*abci.ExecTxResult, 0, len(tasks))
	for _, t := range tasks {
		res = append(res, t.Response)

		if t.TxTracer != nil {
			t.TxTracer.Commit()
		}
	}
	return res
}

func (s *scheduler) tryInitMultiVersionStore(ctx sdk.Context, txNum int) {
	if s.multiVersionStores != nil {
		return
	}
	mvs := make(map[sdk.StoreKey]multiversion.MultiVersionStore)
	keys := ctx.MultiStore().StoreKeys()
	for _, sk := range keys {
		mvs[sk] = multiversion.NewMultiXSyncVersionStore(ctx.MultiStore().GetKVStore(sk), txNum)
	}
	s.multiVersionStores = mvs
}

func dependenciesValidated(tasksMap map[int]*deliverTxTask, deps map[int]struct{}) bool {
	for i := range deps {
		// because idx contains absoluteIndices, we need to fetch from map
		task := tasksMap[i]
		if !task.IsStatus(statusValidatedInt) {
			return false
		}
	}
	return true
}

func filterTasks(tasks []*deliverTxTask, filter func(*deliverTxTask) bool) []*deliverTxTask {
	var res []*deliverTxTask
	for _, t := range tasks {
		if filter(t) {
			res = append(res, t)
		}
	}
	return res
}

func allValidated(tasks []*deliverTxTask) bool {
	for _, t := range tasks {
		if !t.IsStatus(statusValidatedInt) {
			return false
		}
	}
	return true
}

func (s *scheduler) PrefillEstimates(reqs []*sdk.DeliverTxEntry) {
	// iterate over TXs, update estimated writesets where applicable
	for _, req := range reqs {
		mappedWritesets := req.EstimatedWritesets
		// order shouldnt matter for storeKeys because each storeKey partitioned MVS is independent
		for storeKey, writeset := range mappedWritesets {
			// we use `-1` to indicate a prefill incarnation
			s.multiVersionStores[storeKey].SetEstimatedWriteset(req.AbsoluteIndex, -1, writeset)
		}
	}
}

// schedulerMetrics contains metrics for the scheduler
type schedulerMetrics struct {
	// maxIncarnation is the highest incarnation seen in this set
	maxIncarnation int
	// retries is the number of tx attempts beyond the first attempt
	retries int
}

func (s *scheduler) emitMetrics() {
	telemetry.IncrCounter(float32(s.metrics.retries), "scheduler", "retries")
	telemetry.IncrCounter(float32(s.metrics.maxIncarnation), "scheduler", "incarnations")
}

func (s *scheduler) ProcessAll(ctx sdk.Context, reqs *sdk.DeliverTxBatchRequest) ([]*abci.ExecTxResult, error) {
	startTime := time.Now()
	var iterations int
	// initialize mutli-version stores if they haven't been initialized yet
	s.tryInitMultiVersionStore(ctx, len(reqs.TxEntries))
	tasks, tasksMap := toTasks(reqs.TxEntries)
	s.allTasks = tasks
	s.allTasksMap = tasksMap
	s.executeCh = make(chan func(), len(tasks))
	s.validateCh = make(chan func(), len(tasks))
	defer s.emitMetrics()

	// default to number of tasks if workers is negative or 0 by this point
	workers := s.workers
	if s.workers < 1 || len(tasks) < s.workers {
		workers = len(tasks)
	}

	workerCtx, cancel := context.WithCancel(ctx.Context())
	defer cancel()

	// execution tasks are limited by workers
	start(workerCtx, s.executeCh, workers)

	// validation tasks uses length of tasks to avoid blocking on validation
	start(workerCtx, s.validateCh, len(tasks))

	toExecute := tasks
	execStart := time.Now()
	for !allValidated(tasks) {
		// if the max incarnation >= x, we should revert to synchronous
		if iterations >= maximumIterations {
			// process synchronously
			s.synchronous = true
			startIdx, anyLeft := s.findFirstNonValidated()
			if !anyLeft {
				break
			}
			toExecute = tasks[startIdx:]
		}

		execStart := time.Now()
		// execute sets statuses of tasks to either executed or aborted
		if ctx.IsSimpleDag() && iterations == 0 && len(reqs.SimpleDag) > 0 && len(reqs.TxEntries) != len(reqs.SimpleDag) {
			if err := s.executeAllWithDag(ctx, toExecute, reqs.SimpleDag); err != nil {
				return nil, err
			}
		} else {
			if err := s.executeAll(ctx, toExecute); err != nil {
				return nil, err
			}
		}

		s.loger.Info("execute all", "spend time ", time.Since(execStart).Milliseconds(), "exec txs len", len(toExecute), "iterations count", iterations)

		validateStart := time.Now()
		// validate returns any that should be re-executed
		// note this processes ALL tasks, not just those recently executed
		var err error
		toExecute, err = s.validateAll(ctx, tasks)
		if err != nil {
			return nil, err
		}
		s.loger.Info("validate all", "spend time ", time.Since(validateStart).Milliseconds(), "validate txs len", len(toExecute), "iterations count", iterations)
		// these are retries which apply to metrics
		s.metrics.retries += len(toExecute)
		iterations++
	}
	s.loger.Info("execute all and validate all", "spend time ", time.Since(execStart).Milliseconds(), "iterations count", iterations)

	writeStoreStart := time.Now()
	for _, mv := range s.multiVersionStores {
		mv.WriteLatestToStore()
	}
	s.metrics.maxIncarnation = s.maxIncarnation
	s.loger.Info("write latest to store", "spend time ", time.Since(writeStoreStart).Milliseconds(), "write len", len(s.multiVersionStores))

	ctx.Logger().Info("occ scheduler", "height", ctx.BlockHeight(), "txs", len(tasks), "latency_ms", time.Since(startTime).Milliseconds(), "retries", s.metrics.retries, "maxIncarnation", s.maxIncarnation, "iterations", iterations, "sync", s.synchronous, "workers", s.workers)

	return s.collectResponses(tasks), nil
}

func (s *scheduler) shouldRerun(task *deliverTxTask) bool {
	switch task.Status {

	case statusAbortedInt, statusPendingInt:
		return true

	// validated tasks can become unvalidated if an earlier re-run task now conflicts
	case statusExecutedInt, statusValidatedInt:
		// With the current scheduler, we won't actually get to this step if a previous task has already been determined to be invalid,
		// since we choose to fail fast and mark the subsequent tasks as invalid as well.
		// TODO: in a future async scheduler that no longer exhaustively validates in order, we may need to carefully handle the `valid=true` with conflicts case
		if valid, conflicts := s.computeConflicts(task); !valid {
			s.invalidateTask(task)
			task.AppendDependencies(conflicts)

			// if the conflicts are now validated, then rerun this task
			if dependenciesValidated(s.allTasksMap, task.Dependencies) {
				return true
			} else {
				// otherwise, wait for completion
				task.SetStatus(statusWaitingInt)
				return false
			}
		} else if len(conflicts) == 0 {
			// mark as validated, which will avoid re-validating unless a lower-index re-validates
			task.SetStatus(statusValidatedInt)
			return false
		}
		// conflicts and valid, so it'll validate next time
		return false

	case statusWaitingInt:
		// if conflicts are done, then this task is ready to run again
		return dependenciesValidated(s.allTasksMap, task.Dependencies)
	}
	panic("unexpected status: " + task.Status.String())
}

func (s *scheduler) validateTask(ctx sdk.Context, task *deliverTxTask) bool {
	if s.shouldRerun(task) {
		return false
	}
	return true
}

func (s *scheduler) findFirstNonValidated() (int, bool) {
	for i, t := range s.allTasks {
		if t.Status != statusValidatedInt {
			return i, true
		}
	}
	return 0, false
}

func (s *scheduler) validateAll(ctx sdk.Context, tasks []*deliverTxTask) ([]*deliverTxTask, error) {
	var mx sync.Mutex
	var res []*deliverTxTask

	startIdx, anyLeft := s.findFirstNonValidated()

	if !anyLeft {
		return nil, nil
	}

	wg := &sync.WaitGroup{}
	for i := startIdx; i < len(tasks); i++ {
		wg.Add(1)
		t := tasks[i]
		s.DoValidate(func() {
			defer wg.Done()
			if !s.validateTask(ctx, t) {
				mx.Lock()
				defer mx.Unlock()
				t.Reset()
				t.Increment()
				// update max incarnation for scheduler
				if t.Incarnation > s.maxIncarnation {
					s.maxIncarnation = t.Incarnation
				}
				res = append(res, t)
			}
		})
	}
	wg.Wait()

	return res, nil
}

// ExecuteAllWithDag executes all tasks concurrently
func (s *scheduler) executeAllWithDag(ctx sdk.Context, tasks []*deliverTxTask, simpleDag []int64) error {
	var iterations int
	if len(tasks) == 0 {
		return nil
	}
	wg := &sync.WaitGroup{}

	for i := 0; i < len(tasks); i += int(simpleDag[iterations]) {
		batch := int(simpleDag[iterations])
		end := i + batch
		if end > len(tasks) {
			end = len(tasks)
		}

		wg.Add(batch)
		batchStart := time.Now()
		for j := i; j < end; j++ {
			t := tasks[j]
			s.DoExecute(func() {
				s.prepareAndRunTaskWithDag(wg, ctx, t) // no need to wait for previous tasks
			})
		}
		wg.Wait()
		s.loger.Info("execute batch with dag", "index", iterations, "spend time ", time.Since(batchStart).Milliseconds(), "batch txs len", end-i)
		iterations++
		if iterations >= len(simpleDag) {
			break
		}
	}

	return nil
}

// ExecuteAll executes all tasks concurrently
func (s *scheduler) executeAll(ctx sdk.Context, tasks []*deliverTxTask) error {
	if len(tasks) == 0 {
		return nil
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(tasks))

	for _, task := range tasks {
		t := task
		s.DoExecute(func() {
			s.prepareAndRunTask(wg, ctx, t)
		})
	}

	wg.Wait()
	return nil
}

func (s *scheduler) prepareAndRunTaskWithDag(wg *sync.WaitGroup, ctx sdk.Context, task *deliverTxTask) {
	s.executeTask(task, ctx)
	wg.Done()
}

func (s *scheduler) prepareAndRunTask(wg *sync.WaitGroup, ctx sdk.Context, task *deliverTxTask) {
	s.executeTask(task, ctx)
	wg.Done()
}

// prepareTask initializes the context and version stores for a task
func (s *scheduler) prepareTask(task *deliverTxTask) {
	ctx := task.Ctx.WithTxIndex(task.AbsoluteIndex)

	// initialize the context
	abortCh := make(chan occ.Abort, len(s.multiVersionStores))

	// if there are no stores, don't try to wrap, because there's nothing to wrap
	if len(s.multiVersionStores) > 0 {
		// non-blocking
		cms := ctx.MultiStore().CacheMultiStore()

		// init version stores by store key
		vs := make(map[store.StoreKey]*multiversion.VersionIndexedStore)
		for storeKey, mvs := range s.multiVersionStores {
			vs[storeKey] = mvs.VersionedIndexedStore(task.AbsoluteIndex, task.Incarnation, abortCh)
		}

		// save off version store so we can ask it things later
		task.VersionStores = vs
		ms := cms.SetKVStores(func(k store.StoreKey, kvs sdk.KVStore) store.CacheWrap {
			return vs[k]
		})

		ctx = ctx.WithMultiStore(ms)
	}

	if task.TxTracer != nil {
		ctx = task.TxTracer.InjectInContext(ctx)
	}

	task.AbortCh = abortCh
	task.Ctx = ctx
}

func (s *scheduler) executeTask(task *deliverTxTask, ctx sdk.Context) {
	task.Ctx = ctx
	//startTime := time.Now()
	// in the synchronous case, we only want to re-execute tasks that need re-executing
	if s.synchronous {
		// even if already validated, it could become invalid again due to preceeding
		// reruns. Make sure previous writes are invalidated before rerunning.
		if task.IsStatus(statusValidatedInt) {
			s.invalidateTask(task)
		}

		// waiting transactions may not yet have been reset
		// this ensures a task has been reset and incremented
		if !task.IsStatus(statusPendingInt) {
			task.Reset()
			task.Increment()
		}
	}

	s.prepareTask(task)
	//deliverTime := time.Now()
	resp := s.deliverTx(task.Ctx, task.Tx)
	// close the abort channel
	close(task.AbortCh)
	abort, ok := <-task.AbortCh
	if ok {
		s.loger.Info("executeTask abort ", "abort contains", fmt.Sprintf("%+v", abort))
		// if there is an abort item that means we need to wait on the dependent tx
		task.SetStatus(statusAbortedInt)
		task.Abort = &abort
		task.AppendDependencies([]int{abort.DependentTxIdx})
		// write from version store to multiversion stores
		for _, v := range task.VersionStores {
			v.WriteEstimatesToMultiVersionStore()
		}
		return
	}

	task.SetStatus(statusExecutedInt)
	task.Response = resp

	//writeTime := time.Now()
	// write from version store to multiversion stores
	for _, v := range task.VersionStores {
		v.WriteToMultiVersionStore()
	}
	//allTime := time.Since(startTime).Microseconds()
	//deliveTime := time.Since(deliverTime).Microseconds()
	//writTime := time.Since(writeTime).Microseconds()
	//s.loger.Info("executeTask RWList ", "txIndex", task.AbsoluteIndex, "prepareTask", allTime-deliveTime, "elapsed time delive", deliveTime-writTime, "elapsed time", allTime, "elapsed time of write", writTime)
}

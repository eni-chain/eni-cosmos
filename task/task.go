package tasks

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/store/rwset"
	store "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type status string

const (
	// statusPending tasks are ready for execution
	// all executing tasks are in pending state
	statusPending status = "pending"
	// statusExecuted tasks are ready for validation
	// these tasks did not abort during execution
	statusExecuted status = "executed"
	// statusValidated means the task has been validated
	// tasks in this status can be reset if an earlier task fails validation
	statusValidated status = "validated"
)

type deliverTxTask struct {
	Ctx sdk.Context

	mx               sync.RWMutex
	Status           status
	Result           *abci.ExecTxResult
	TxBytes          []byte
	TxIndex          int
	TxExecStores     map[store.StoreKey]*rwset.TxExecutionStore
	ValidationResult bool  // cache validate result
	Conflicts        []int // cache conflicts list
}

type sortTxTasks []*deliverTxTask

func (s sortTxTasks) Len() int           { return len(s) }
func (s sortTxTasks) Swap(i, j int)      { (s)[i], (s)[j] = (s)[j], (s)[i] }
func (s sortTxTasks) Less(i, j int) bool { return (s)[i].TxIndex < (s)[j].TxIndex }

func (dt *deliverTxTask) IsStatus(s status) bool {
	dt.mx.RLock()
	defer dt.mx.RUnlock()
	return dt.Status == s
}

func (dt *deliverTxTask) SetStatus(s status) {
	dt.mx.Lock()
	defer dt.mx.Unlock()
	dt.Status = s
}

func (dt *deliverTxTask) Reset() {
	dt.SetStatus(statusPending)
	dt.TxExecStores = nil
	dt.ValidationResult = false // reset validation result
	dt.Conflicts = nil          // reset conflicts list
}

// Scheduler processes tasks concurrently
type Scheduler interface {
	ProcessAll(ctx sdk.Context, reqs sdk.DeliverTxBatchRequest) ([]*abci.ExecTxResult, error)
}

type scheduler struct {
	deliverTx   func(ctx sdk.Context, tx []byte) *abci.ExecTxResult
	rwSetStores map[store.StoreKey]rwset.RwSetStore
	synchronous bool // true if maxIncarnation exceeds threshold
}

// NewScheduler creates a new scheduler
func NewScheduler(deliverTxFunc func(ctx sdk.Context, tx []byte) *abci.ExecTxResult) Scheduler {
	return &scheduler{
		deliverTx: deliverTxFunc,
	}
}

func (s *scheduler) invalidateTask(task *deliverTxTask) {
	for _, mv := range s.rwSetStores {
		mv.InvalidateWriteSet(task.TxIndex)
		mv.ClearReadSet(task.TxIndex)
	}
}

func (s *scheduler) findConflicts(task *deliverTxTask) (bool, []int) {
	var conflicts []int
	uniq := make(map[int]struct{})
	valid := true
	for _, mv := range s.rwSetStores {
		ok, mvConflicts := mv.ValidateTransactionState(task.TxIndex, s.synchronous)
		for _, c := range mvConflicts {
			if _, ok := uniq[c]; !ok {
				conflicts = append(conflicts, c)
				uniq[c] = struct{}{}
			}
		}
		// any non-ok value makes valid false
		valid = valid && ok
	}
	// remove sort operation to improve performance
	return valid, conflicts
}

func toTasks(ctx sdk.Context, entries []*sdk.DeliverTxEntry) []*deliverTxTask {
	if len(entries) == 0 {
		return []*deliverTxTask{}
	}
	allTasks := make([]*deliverTxTask, 0, len(entries))
	for _, r := range entries {
		task := &deliverTxTask{
			TxIndex: r.TxIndex,
			Status:  statusPending,
			TxBytes: r.Tx,
			Ctx:     ctx,
		}
		allTasks = append(allTasks, task)
	}
	return allTasks
}

func (s *scheduler) tryInitRwSetStore(ctx sdk.Context) {
	if s.rwSetStores != nil {
		return
	}
	rws := make(map[store.StoreKey]rwset.RwSetStore)
	keys := ctx.MultiStore().StoreKeys()
	for _, sk := range keys {
		rws[sk] = rwset.NewRwSetStore(ctx.MultiStore().GetKVStore(sk))
	}
	s.rwSetStores = rws
}

func (s *scheduler) ProcessAll(ctx sdk.Context, req sdk.DeliverTxBatchRequest) ([]*abci.ExecTxResult, error) {
	if len(req.SeqEntries)+len(req.OtherEntries) == 0 {
		return []*abci.ExecTxResult{}, nil
	}
	startTime := time.Now()
	s.tryInitRwSetStore(ctx)
	otherTasks := toTasks(ctx, req.OtherEntries)
	allSeqTasks := toTasks(ctx, req.SeqEntries)
	allTasks := append(otherTasks, allSeqTasks...)

	pTasks, sTasks, err := s.preprocessTask(otherTasks)
	if err != nil {
		return nil, err
	}

	allSeqTasks = append(allSeqTasks, sTasks...)
	sort.Sort(sortTxTasks(allSeqTasks))

	if err = s.finishParaTasks(pTasks); err != nil {
		return nil, fmt.Errorf("parallel otherTasks failed: %w", err)
	}

	if err = s.finishSerialTasks(allSeqTasks); err != nil {
		return nil, fmt.Errorf("serial otherTasks failed: %w", err)
	}

	for _, mv := range s.rwSetStores {
		mv.WriteLatestToStore()
	}

	ctx.Logger().Info("occ scheduler", "height", ctx.BlockHeight(), "txs", len(otherTasks), "latency_ms", time.Since(startTime).Milliseconds(), "sync", s.synchronous)

	return s.collectResponses(allTasks), nil
}

func (s *scheduler) collectResponses(tasks []*deliverTxTask) []*abci.ExecTxResult {
	res := make([]*abci.ExecTxResult, 0, len(tasks))
	for _, t := range tasks {
		res = append(res, t.Result)
	}
	return res
}

func (s *scheduler) shouldValid(task *deliverTxTask) error {
	if task.Status != statusValidated && task.Status != statusExecuted {
		return fmt.Errorf("expected task status is statusValidated or statusExecuted, but actual is %v", task.Status)
	}

	// if validate result of cache not available,it is recalculated
	if !task.ValidationResult && len(task.Conflicts) == 0 {
		valid, conflicts := s.findConflicts(task)
		task.ValidationResult = valid
		task.Conflicts = conflicts
	}

	if !task.ValidationResult {
		ctx := task.Ctx
		ctx.Logger().Error("Validation failed for task",
			"task_index", task.TxIndex,
			"status", task.Status,
			"conflicts", task.Conflicts)
		for storeKey, mv := range s.rwSetStores {
			ok, mvConflicts := mv.ValidateTransactionState(task.TxIndex, s.synchronous)
			if !ok {
				ctx.Logger().Error("Validation failed for store",
					"store_key", storeKey.String(),
					"conflicts", mvConflicts)
			}
		}
		return fmt.Errorf("task %v verify result false, conflicts found: %v", task.TxIndex, task.Conflicts)
	}

	task.SetStatus(statusValidated)
	return nil
}

func (s *scheduler) validateTask(task *deliverTxTask) error {
	return s.shouldValid(task)
}

// use channel to optimize task classification
func (s *scheduler) defineTasksType(tasks []*deliverTxTask) ([]*deliverTxTask, []*deliverTxTask, error) {
	if len(tasks) == 0 {
		return nil, nil, nil
	}

	pChan := make(chan *deliverTxTask)
	sChan := make(chan *deliverTxTask)
	var wg sync.WaitGroup
	wg.Add(len(tasks))

	for _, t := range tasks {
		go func(task *deliverTxTask) {
			defer wg.Done()
			valid, conflicts := s.findConflicts(task)
			task.ValidationResult = valid
			task.Conflicts = conflicts
			if valid {
				pChan <- task
			} else {
				sChan <- task
			}
		}(t)
	}
	wg.Wait()
	close(pChan)
	close(sChan)

	pTasks := make([]*deliverTxTask, 0, len(tasks))
	sTasks := make([]*deliverTxTask, 0, len(tasks))
	for task := range pChan {
		pTasks = append(pTasks, task)
	}
	for task := range sChan {
		sTasks = append(sTasks, task)
	}

	return pTasks, sTasks, nil
}

func (s *scheduler) validateAll(tasks []*deliverTxTask) error {
	if len(tasks) == 0 {
		return nil
	}

	errChan := make(chan error, len(tasks))
	wg := &sync.WaitGroup{}
	wg.Add(len(tasks))

	for _, t := range tasks {
		go func(task *deliverTxTask) {
			defer wg.Done()
			if err := s.validateTask(task); err != nil {
				errChan <- err
			}
		}(t)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *scheduler) finishParaTasks(pTasks []*deliverTxTask) error {
	if len(pTasks) == 0 {
		return nil
	}
	err := s.validateAll(pTasks)
	if err != nil {
		return err
	}
	return nil
}

func (s *scheduler) finishSerialTasks(sTasks []*deliverTxTask) error {
	if len(sTasks) == 0 {
		return nil
	}

	s.synchronous = true

	for _, task := range sTasks {
		// rest task status and clear rwset to ensure clean execution
		task.Reset()
		s.invalidateTask(task)
		s.prepareAndRunTask(nil, task)
	}

	err := s.validateAll(sTasks)
	if err != nil {
		return fmt.Errorf("serial otherTasks failed: %w", err)
	}
	return nil
}

func (s *scheduler) preprocessTask(tasks []*deliverTxTask) ([]*deliverTxTask, []*deliverTxTask, error) {
	s.parallelExec(tasks)

	pTasks, sTasks, err := s.defineTasksType(tasks)
	if err != nil {
		return nil, nil, err
	}

	return pTasks, sTasks, nil
}

func (s *scheduler) parallelExec(tasks []*deliverTxTask) {
	if len(tasks) == 0 {
		return
	}

	workerCount := min(len(tasks), runtime.NumCPU())
	taskChan := make(chan *deliverTxTask, len(tasks))
	wg := &sync.WaitGroup{}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				s.executeTask(task)
			}
		}()
	}

	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)
	wg.Wait()
}

func (s *scheduler) prepareAndRunTask(wg *sync.WaitGroup, task *deliverTxTask) {
	s.executeTask(task)
	if wg != nil {
		wg.Done()
	}
}

func (s *scheduler) prepareTask(task *deliverTxTask) {
	ctx := task.Ctx.WithTxIndex(task.TxIndex)

	if len(s.rwSetStores) > 0 {
		cms := ctx.MultiStore().CacheMultiStore()

		vs := make(map[store.StoreKey]*rwset.TxExecutionStore)
		for storeKey, mvs := range s.rwSetStores {
			vs[storeKey] = mvs.TxExecutionStore(task.TxIndex)
		}

		task.TxExecStores = vs
		ms := cms.SetKVStores(func(k store.StoreKey, kvs store.KVStore) store.CacheWrap {
			return vs[k]
		})

		ctx = ctx.WithMultiStore(ms)
	}

	task.Ctx = ctx
}

func (s *scheduler) executeTask(task *deliverTxTask) {
	if s.synchronous {
		if task.IsStatus(statusValidated) {
			s.invalidateTask(task)
		}
		if !task.IsStatus(statusPending) {
			task.Reset()
		}
	}

	s.prepareTask(task)

	resp := s.deliverTx(task.Ctx, task.TxBytes)

	task.SetStatus(statusExecuted)
	task.Result = resp

	for _, v := range task.TxExecStores {
		v.WriteToRwSetStore()
	}
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

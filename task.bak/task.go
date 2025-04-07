package tasks

//
//import (
//	"fmt"
//	abci "github.com/cometbft/cometbft/abci/types"
//	"runtime"
//	"sort"
//	"sync"
//	"time"
//
//	"cosmossdk.io/store/rwset"
//	store "cosmossdk.io/store/types"
//	sdk "cosmossdk.io/types"
//)
//
//type status string
//
//const (
//	// statusPending tasks are ready for execution
//	// all executing tasks are in pending state
//	statusPending status = "pending"
//	// statusExecuted tasks are ready for validation
//	// these tasks did not abort during execution
//	statusExecuted status = "executed"
//	// statusValidated means the task has been validated
//	// tasks in this status can be reset if an earlier task fails validation
//	statusValidated status = "validated"
//)
//
//type deliverTxTask struct {
//	Ctx sdk.Context
//
//	mx           sync.RWMutex
//	Status       status
//	Result       *abci.ExecTxResult
//	TxBytes      []byte
//	TxIndex      int
//	TxExecStores map[store.StoreKey]*rwset.TxExecutionStore
//}
//
//type sortTxTasks []*deliverTxTask
//
//func (s sortTxTasks) Len() int           { return len(s) }
//func (s sortTxTasks) Swap(i, j int)      { (s)[i], (s)[j] = (s)[j], (s)[i] }
//func (s sortTxTasks) Less(i, j int) bool { return (s)[i].TxIndex < (s)[j].TxIndex }
//
//func (dt *deliverTxTask) IsStatus(s status) bool {
//	dt.mx.RLock()
//	defer dt.mx.RUnlock()
//	return dt.Status == s
//}
//
//func (dt *deliverTxTask) SetStatus(s status) {
//	dt.mx.Lock()
//	defer dt.mx.Unlock()
//	dt.Status = s
//}
//
//func (dt *deliverTxTask) Reset() {
//	dt.SetStatus(statusPending)
//	dt.TxExecStores = nil
//
//}
//
//// Scheduler processes tasks concurrently
//type Scheduler interface {
//	ProcessAll(ctx sdk.Context, reqs sdk.DeliverTxBatchRequest, txs [][]byte, SimpleDag []int64) ([]*abci.ExecTxResult, error)
//}
//
//type scheduler struct {
//	deliverTx   func(ctx sdk.Context, tx []byte) *abci.ExecTxResult
//	rwSetStores map[store.StoreKey]rwset.RwSetStore
//	synchronous bool // true if maxIncarnation exceeds threshold
//}
//
//// NewScheduler creates a new scheduler
//func NewScheduler(deliverTxFunc func(ctx sdk.Context, tx []byte) *abci.ExecTxResult) Scheduler {
//	return &scheduler{
//		deliverTx: deliverTxFunc,
//	}
//}
//
//func (s *scheduler) invalidateTask(task *deliverTxTask) {
//	for _, mv := range s.rwSetStores {
//		mv.InvalidateWriteSet(task.TxIndex)
//		mv.ClearReadSet(task.TxIndex)
//	}
//}
//
//func (s *scheduler) findConflicts(task *deliverTxTask) (bool, []int) {
//	var conflicts []int
//	uniq := make(map[int]struct{})
//	valid := true
//	for _, mv := range s.rwSetStores {
//		ok, mvConflicts := mv.ValidateTransactionState(task.TxIndex, s.synchronous)
//		for _, c := range mvConflicts {
//			if _, ok := uniq[c]; !ok {
//				conflicts = append(conflicts, c)
//				uniq[c] = struct{}{}
//			}
//		}
//		// any non-ok value makes valid false
//		valid = valid && ok
//	}
//	sort.Ints(conflicts)
//
//	return valid, conflicts
//}
//
//func toTasks(ctx sdk.Context, entries []*sdk.DeliverTxEntry) []*deliverTxTask {
//	if len(entries) == 0 {
//		return []*deliverTxTask{}
//	}
//	allTasks := make([]*deliverTxTask, 0, len(entries))
//	for _, r := range entries {
//		task := &deliverTxTask{
//			TxIndex: r.TxIndex,
//			Status:  statusPending,
//			TxBytes: r.Tx,
//			Ctx:     ctx,
//		}
//
//		allTasks = append(allTasks, task)
//	}
//	return allTasks
//}
//
//func (s *scheduler) tryInitRwSetStore(ctx sdk.Context) {
//	if s.rwSetStores != nil {
//		return
//	}
//	rws := make(map[store.StoreKey]rwset.RwSetStore)
//	keys := ctx.MultiStore().StoreKeys()
//	for _, sk := range keys {
//		rws[sk] = rwset.NewRwSetStore(ctx.MultiStore().GetKVStore(sk))
//	}
//	s.rwSetStores = rws
//}
//
//func (s *scheduler) ProcessAll(ctx sdk.Context, req sdk.DeliverTxBatchRequest, txs [][]byte, SimpleDag []int64) ([]*abci.ExecTxResult, error) {
//	if len(txs) == 0 {
//		return []*abci.ExecTxResult{}, nil
//	}
//
//	// get batch txs by simple dag
//	batchTxs := s.getBatchTxs(txs, SimpleDag)
//	startTime := time.Now()
//	s.tryInitRwSetStore(ctx)
//
//	var allResults []*abci.ExecTxResult
//	// group out serial: process each batch sequentially
//	// group in parallel: preprocess tasks, then execute parallel tasks
//	for batchIdx, batch := range batchTxs {
//		if len(batch) == 0 {
//			continue
//		}
//
//		txTasks := toTasks(ctx, batch)
//		if err := s.finishParaTasks(txTasks); err != nil {
//			return nil, fmt.Errorf("parallel tasks in batch %d failed: %w", batchIdx, err)
//		}
//
//		batchResults := s.collectResponses(txTasks)
//		allResults = append(allResults, batchResults...)
//	}
//
//	for _, mv := range s.rwSetStores {
//		mv.WriteLatestToStore()
//	}
//
//	ctx.Logger().Info("occ scheduler", "height", ctx.BlockHeight(), "txs", len(txs), "latency_ms",
//		time.Since(startTime).Milliseconds(), "sync", s.synchronous)
//
//	return allResults, nil
//}
//
//func (s *scheduler) getBatchTxs(txs [][]byte, SimpleDag []int64) [][]*sdk.DeliverTxEntry {
//	batchTxs := make([][]*sdk.DeliverTxEntry, 0, len(SimpleDag))
//	currentPos := 0
//	for _, batchSize := range SimpleDag {
//		if batchSize <= 0 {
//			continue
//		}
//
//		endPos := currentPos + int(batchSize)
//		if currentPos >= len(txs) {
//			break
//		}
//		if endPos > len(txs) {
//			endPos = len(txs)
//		}
//
//		batch := make([]*sdk.DeliverTxEntry, 0, endPos-currentPos)
//		for i := currentPos; i < endPos; i++ {
//			entry := &sdk.DeliverTxEntry{
//				Tx:      txs[i],
//				TxIndex: i,
//			}
//			batch = append(batch, entry)
//		}
//
//		batchTxs = append(batchTxs, batch)
//		currentPos = endPos
//	}
//	return batchTxs
//}
//func (s *scheduler) collectResponses(tasks []*deliverTxTask) []*abci.ExecTxResult {
//	res := make([]*abci.ExecTxResult, 0, len(tasks))
//	for _, t := range tasks {
//		res = append(res, t.Result)
//	}
//	return res
//}
//
//func (s *scheduler) shouldValid(task *deliverTxTask) error {
//	if task.Status != statusValidated && task.Status != statusExecuted {
//		return fmt.Errorf("expected task status is statusValidated or statusExecuted, but actual is %v", task.Status)
//	}
//
//	if valid, conflicts := s.findConflicts(task); !valid || len(conflicts) != 0 {
//		s.invalidateTask(task)
//		return fmt.Errorf("task %v verify result %v,conflicts found: %v, ", task.TxIndex, valid, conflicts)
//	}
//
//	task.SetStatus(statusValidated)
//	return nil
//}
//
//func (s *scheduler) validateTask(task *deliverTxTask) error {
//	return s.shouldValid(task)
//}
//
//// By verifying tasks, differentiate between tasks that can be executed in parallel and tasks that can be executed sequentially.
//func (s *scheduler) defineTasksType(tasks []*deliverTxTask) ([]*deliverTxTask, []*deliverTxTask, error) {
//
//	var (
//		mx     sync.Mutex
//		pTasks []*deliverTxTask
//		sTasks []*deliverTxTask
//	)
//
//	wg := &sync.WaitGroup{}
//	for i := 0; i < len(tasks); i++ {
//		wg.Add(1)
//		t := tasks[i]
//		go func() {
//			mx.Lock()
//			defer mx.Unlock()
//			if valid, conflicts := s.findConflicts(t); !valid || len(conflicts) != 0 {
//				sTasks = append(sTasks, t)
//			} else {
//				pTasks = append(pTasks, t)
//			}
//			wg.Done()
//		}()
//	}
//	wg.Wait()
//
//	return pTasks, sTasks, nil
//}
//
//func (s *scheduler) validateAll(tasks []*deliverTxTask) error {
//	wg := &sync.WaitGroup{}
//	errChan := make(chan error, len(tasks))
//
//	for i := 0; i < len(tasks); i++ {
//		wg.Add(1)
//		t := tasks[i]
//		go func(task *deliverTxTask) {
//			defer wg.Done()
//			if err := s.validateTask(task); err != nil {
//				errChan <- err
//			}
//		}(t)
//	}
//
//	go func() {
//		wg.Wait()
//		close(errChan)
//	}()
//
//	for err := range errChan {
//		if err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
//func (s *scheduler) finishParaTasks(pTasks []*deliverTxTask) error {
//	if len(pTasks) == 0 {
//		return nil
//	}
//	s.parallelExec(pTasks)
//
//	err := s.validateAll(pTasks)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (s *scheduler) finishSerialTasks(sTasks []*deliverTxTask) error {
//	if len(sTasks) == 0 {
//		return nil
//	}
//
//	s.synchronous = true
//
//	for _, task := range sTasks {
//		s.prepareAndRunTask(nil, task)
//	}
//
//	err := s.validateAll(sTasks)
//	if err != nil {
//		return err
//	}
//	return nil
//}
//
//func (s *scheduler) preprocessTask(tasks []*deliverTxTask) ([]*deliverTxTask, []*deliverTxTask, error) {
//	s.parallelExec(tasks)
//
//	pTasks, sTasks, err := s.defineTasksType(tasks)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	return pTasks, sTasks, nil
//}
//
//func (s *scheduler) parallelExec(tasks []*deliverTxTask) {
//	if len(tasks) == 0 {
//		return
//	}
//	// set GOMAXPROCS to the number of CPUs to use all available cores
//	runtime.GOMAXPROCS(runtime.NumCPU())
//
//	// use worker pooled by runtime to execute tasks in parallel
//	workerCount := runtime.NumCPU()
//	taskChan := make(chan *deliverTxTask, len(tasks))
//	wg := &sync.WaitGroup{}
//
//	// start worker
//	for i := 0; i < workerCount; i++ {
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			for task := range taskChan {
//				s.executeTask(task)
//			}
//		}()
//	}
//
//	// dispatch tasks to workers
//	for _, task := range tasks {
//		taskChan <- task
//	}
//	close(taskChan)
//	wg.Wait()
//}
//
//func (s *scheduler) prepareAndRunTask(wg *sync.WaitGroup, task *deliverTxTask) {
//	s.executeTask(task)
//	if wg != nil {
//		wg.Done()
//	}
//}
//
//// prepareTask initializes the context and version stores for a task
//func (s *scheduler) prepareTask(task *deliverTxTask) {
//	ctx := task.Ctx.WithTxIndex(task.TxIndex)
//
//	// if there are no stores, don't try to wrap, because there's nothing to wrap
//	if len(s.rwSetStores) > 0 {
//		// non-blocking
//		cms := ctx.MultiStore().CacheMultiStore()
//
//		// init version stores by store key
//		vs := make(map[store.StoreKey]*rwset.TxExecutionStore)
//		for storeKey, mvs := range s.rwSetStores {
//			vs[storeKey] = mvs.TxExecutionStore(task.TxIndex)
//		}
//
//		// save off version store so we can ask it things later
//		task.TxExecStores = vs
//		ms := cms.SetKVStores(func(k store.StoreKey, kvs store.KVStore) store.CacheWrap {
//			return vs[k]
//		})
//
//		ctx = ctx.WithMultiStore(ms)
//	}
//
//	task.Ctx = ctx
//}
//
//func (s *scheduler) executeTask(task *deliverTxTask) {
//	if s.synchronous {
//		if task.IsStatus(statusValidated) {
//			s.invalidateTask(task)
//		}
//
//		if !task.IsStatus(statusPending) {
//			task.Reset()
//		}
//	}
//
//	s.prepareTask(task)
//
//	resp := s.deliverTx(task.Ctx, task.TxBytes)
//
//	task.SetStatus(statusExecuted)
//	task.Result = resp
//
//	for _, v := range task.TxExecStores {
//		//v.DebugPrint()
//		v.WriteToRwSetStore()
//	}
//}

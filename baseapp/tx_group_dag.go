package baseapp

type SimpleDag struct {
	txs [][]byte
	dag []int64
}

func (t *SimpleDag) GetTxs() [][]byte {
	return t.txs
}

func (t *SimpleDag) GetDag() []int64 {
	return t.dag
}

// buildGroup builds a simple DAG for the given transaction group.
func (app *BaseApp) buildGroup(txs [][]byte) (*SimpleDag, error) {
	txGroup, err := app.GroupByTxs(txs)
	if err == nil {
		return nil, err
	}
	txGroup.buildSampDag()

	return txGroup.txGroupDAG, nil
}

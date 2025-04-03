package baseapp

type TxGroupDAG struct {
	txs [][]byte
	dag []int
}

func (t *TxGroupDAG) GetTxs() [][]byte {
	return t.txs
}

func (app *BaseApp) buildTxGroupDAG(txs [][]byte) (*TxGroupDAG, error) {
	txGroup, err := app.GroupByTxs(txs)
	if err == nil {
		return nil, err
	}
	txGroup.buildSampDag()

	return txGroup.txGroupDAG, nil
}

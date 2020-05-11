package listappend

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

func TestPlotAnalysis(t *testing.T) {
	t1 := mustParseOp(`{:index 2, :type :invoke, :value [[:append x 1] [:r y [1]] [:r z [1 2]]]} `)
	t1p := mustParseOp(`{:index 3, :type :ok, :value [[:append x 1] [:r y [1]] [:r z [1 2]]]} `)
	t2 := mustParseOp(`{:index 4, :type :invoke, :value [[:append z 1]]}`)
	t2p := mustParseOp(`{:index 5, :type :ok, :value [[:append z 1]]}`)
	t3 := mustParseOp(`{:index 0, :type :invoke, :value [[:r x [1 2]] [:r z [1]]]}`)
	t3p := mustParseOp(`{:index 1, :type :ok, :value [[:r x [1 2]] [:r z [1]]]}`)
	t4 := mustParseOp(`{:index 6, :type :invoke, :value [[:append z 2] [:append y 1]]}`)
	t4p := mustParseOp(`{:index 7, :type :ok, :value [[:append z 2] [:append y 1]]}`)
	t5 := mustParseOp(`{:index 8, :type :invoke, :value [[:r z nil] [:append x 2]]}`)
	t5p := mustParseOp(`{:index 9, :type :ok, :value [[:r z nil] [:append x 2]]}`)
	h := []core.Op{t3, t3p, t1, t1p, t2, t2p, t4, t4p, t5, t5p}

	analyzer := core.Combine(graph, core.RealtimeGraph)
	checkResult := core.Check(analyzer, h)
	require.Equal(t, nil, plotAnalysis(checkResult, "/tmp"))
}

//func TestHugeSccPlotAnalysis(t *testing.T) {
//	content, err := ioutil.ReadFile("../histories/huge-scc.edn")
//	if err != nil {
//		t.Fail()
//	}
//	history, err := core.ParseHistory(string(content))
//	if err != nil {
//		t.Fail()
//	}
//	history = preProcessHistory(history)
//	analyzer := core.Combine(graph, core.RealtimeGraph)
//	checkResult := core.Check(analyzer, history)
//	require.Equal(t, nil, plotAnalysis(checkResult, "/tmp"))
//}

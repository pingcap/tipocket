package elle

//  There are several points in an analysis where we want to construct some
//  intermediate data, like a version graph, for later. However, it might be that
//  *during* the construction of that version graph, we discover the graph itself
//  contains anomalies. This doesn't *invalidate* the graph--perhaps we can still
//  use it to discover anomalies later, but it *is* information we need to pass
//  upstream to the user.
//
//  The Anomalies protocol lets the version graph signal
//  that additional anomalies were encountered, so whatever uses the version
//  graph can merge those into its upstream anomaly set.
//
//  This is kind of like an error monad, but it felt weird and less composable to
//  use exceptions for this.
type Anomalies interface {
	// Returns a map of anomaly types to anomalies.
	Anomalies() interface{}
}

// Merges n Anomaly objects together.
func MergeAnomalies([]Anomalies) Anomalies {
	panic("implement me")
}

//  The TransactionGrapher takes a history and produces a TransactionGraph,
//  optionally augmented with Anomalies discovered during the graph inference
//  process.
type TransactionGrapher interface {
	BuildTxnGraph(history *History) TransactionGraph
}

// A TransactionGraph represents a graph of dependencies between transactions
//  (really, operations), where edges are sets of tagged relationships, like :ww
//  or :realtime.
type TransactionGraph interface {
	// Returns a Bifurcan IDirectedGraph of dependencies between transactions (represented as completion operations)
	TxnGraph() DiGraph
}

// TODO: make it clear. Please not use it's interface now.
type DataExplainer interface {
	// Given a pair of operations a and b, explains why b depends on a, in the
	//    form of a data structure. Returns `nil` if b does not depend on a.
	ExplainPairData()
	// Given a pair of operations, and short names for them, explain why b
	//  depends on a, as a string. `nil` indicates that b does not depend on a.
	RenderExplanation()
}

func CombineExplainer([]DataExplainer) DataExplainer {
	panic("implement me")
}

//  A function which takes a history and returns a {:graph, :explainer, :anomalies} map; e.g. realtime-graph.
type Analyzer interface {
	Analyze(histories []History) (DiGraph, DataExplainer, Anomalies)
}

func Check(opts interface{}, histories []History) {
	panic("implement me")
}

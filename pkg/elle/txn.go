// Functions for cycle analysis over transactional workloads.

package elle

// Takes an options map, including a collection of expected consistency models
//  :consistency-models, a set of additional :anomalies, an analyzer function,
//  and a history. Analyzes the history and yields the analysis, plus an anomaly
//  map like {:G1c [...]}.
func Cycles(opts interface{}, analyzer Analyzer, history []History) Anomalies {
	panic("implement me")
}

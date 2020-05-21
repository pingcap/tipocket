package rwregister

import (
	"fmt"
	"log"
	"sort"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

// key -> v -> Op
type writeIdx map[string]map[int]core.Op

// key -> {v1 -> [Op1, Op2, Op3...], v2 -> [Op4, Op5...]}
type readIdx map[string]map[int][]core.Op

// GCaseTp type aliases []core.Anomaly
type GCaseTp []core.Anomaly

// InternalConflict records a internal conflict
type InternalConflict struct {
	Op       core.Op
	Mop      core.Mop
	Expected core.Mop
}

// IAnomaly ...
func (i InternalConflict) IAnomaly() {}

// String ...
func (i InternalConflict) String() string {
	return fmt.Sprintf("(InternalConflict) Op: %s, mop: %s, expected: %s", i.Op, i.Mop.String(), i.Expected)
}

func internalOp(op core.Op) core.Anomaly {
	dataMap := make(map[string]int)
	for _, mop := range *op.Value {
		if mop.IsWrite() {
			for k, v := range mop.M {
				vprt := v.(*int)
				if vprt == nil {
					panic("write value should not be nil")
				}
				dataMap[k] = *vprt
			}
		}
		if mop.IsRead() {
			for k, v := range mop.M {
				vprt := v.(*int)
				if vprt == nil {
					continue
				}
				if prev, ok := dataMap[k]; !ok {
					dataMap[k] = *vprt
				} else {
					if prev != *vprt {
						expected := mop.Copy()
						expected.M[k] = &prev
						return InternalConflict{
							Op:       op,
							Mop:      mop,
							Expected: expected,
						}
					}
				}
			}
		}
	}
	return nil
}

func internal(history core.History) GCaseTp {
	var tp GCaseTp
	okHistory := core.FilterOkHistory(history)
	for _, op := range okHistory {
		res := internalOp(op)
		if res != nil {
			tp = append(tp, res)
		}
	}
	return tp
}

// G1Conflict records a G1 conflict
type G1Conflict struct {
	Op      core.Op
	Mop     core.Mop
	Writer  core.Op
	Element string
}

// IAnomaly ...
func (g G1Conflict) IAnomaly() {}

// String ...
func (g G1Conflict) String() string {
	return fmt.Sprintf("(G1Conflict) Op: %s, mop: %s, writer: %s, element: %s", g.Op, g.Mop.String(), g.Writer.String(), g.Element)
}

func g1aCases(history core.History) GCaseTp {
	failedMap := make(map[core.KV]core.Op)
	var anomalies []core.Anomaly

	for _, op := range core.FilterFailedHistory(history) {
		for _, mop := range *op.Value {
			if mop.IsWrite() {
				for k, v := range mop.M {
					vprt := v.(*int)
					if vprt == nil {
						panic("write value should not be nil")
					}
					failedMap[core.KV{K: k, V: *vprt}] = op
				}
			}
		}
	}

	for _, op := range core.FilterOkHistory(history) {
		for _, mop := range *op.Value {
			if mop.IsRead() {
				for k, v := range mop.M {
					vprt := v.(*int)
					if vprt == nil {
						continue
					}
					if failedOp, ok := failedMap[core.KV{K: k, V: *vprt}]; ok {
						anomalies = append(anomalies, G1Conflict{
							Op:      op,
							Mop:     mop,
							Writer:  failedOp,
							Element: k,
						})
					}
				}
			}
		}
	}

	return anomalies
}

func g1bCases(history core.History) GCaseTp {
	interMap := make(map[core.KV]core.Op)
	var anomalies []core.Anomaly

	for _, op := range core.FilterOkHistory(history) {
		valMap := make(map[string]int)
		for _, mop := range *op.Value {
			if mop.IsWrite() {
				for k, v := range mop.M {
					vprt := v.(*int)
					if vprt == nil {
						panic("write value should not be nil")
					}
					if old, ok := valMap[k]; ok {
						interMap[core.KV{K: k, V: old}] = op
					}
					valMap[k] = *vprt
				}
			}
		}
	}

	for _, op := range core.FilterOkHistory(history) {
		for _, mop := range *op.Value {
			if mop.IsRead() {
				for k, v := range mop.M {
					vprt := v.(*int)
					if vprt == nil {
						continue
					}
					if interOp, ok := interMap[core.KV{K: k, V: *vprt}]; ok && op != interOp {
						anomalies = append(anomalies, G1Conflict{
							Op:      op,
							Mop:     mop,
							Writer:  interOp,
							Element: k,
						})
					}
				}
			}
		}
	}

	return anomalies
}

type wrExplainResult struct {
	Typ   core.DependType
	Key   string
	Value core.MopValueType
}

// WRExplainResult creates a wrExplainResult
func WRExplainResult(key string, value core.MopValueType) wrExplainResult {
	return wrExplainResult{
		Typ:   core.RWDepend,
		Key:   key,
		Value: value,
	}
}

func (w wrExplainResult) Type() core.DependType {
	return core.WRDepend
}

// wwExplainer explains write-write dependencies
type wrExplainer struct {
	writeIdx
	readIdx
}

func (w *wrExplainer) ExplainPairData(a, b core.PathType) core.ExplainResult {
	for _, bmop := range *b.Value {
		if !bmop.IsRead() {
			continue
		}
		for _, amop := range *b.Value {
			if !amop.IsWrite() {
				continue
			}
			for bk, bval := range bmop.M {
				bvptr := bval.(*int)
				if bvptr == nil {
					continue
				}
				for ak, aval := range amop.M {
					if ak != bk {
						continue
					}
					avptr := aval.(*int)
					if avptr == nil {
						continue
					}
					if *avptr == *bvptr {
						return WRExplainResult(
							bk,
							*avptr,
						)
					}
				}
			}
		}
	}
	return nil
}

func (w *wrExplainer) RenderExplanation(result core.ExplainResult, a, b string) string {
	if result.Type() != core.WRDepend {
		log.Fatalf("result type is not %s, type error", core.WWDepend)
	}
	er := result.(wrExplainResult)
	return fmt.Sprintf("%s read %v written by %s with value %s",
		b, er.Key, a, er.Value,
	)
	return ""
}

// wwGraph analyzes write-write dependencies
func wrGraph(history core.History) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	history, writeIdx, readIdx := preprocess(history)
	g := core.NewDirectedGraph()

	for _, op := range history {
		for _, mop := range *op.Value {
			if !mop.IsRead() {
				continue
			}
			deps := wrMopDep(writeIdx, op, mop)
			for dep := range deps {
				g.Link(core.Vertex{Value: dep}, core.Vertex{Value: op}, core.WR)
			}
		}
	}
	return nil, g, &wrExplainer{
		writeIdx: writeIdx,
		readIdx:  readIdx,
	}
}

type extKeyExplainer struct {
}

func (w *extKeyExplainer) ExplainPairData(a, b core.PathType) core.ExplainResult {
	return nil
}

func (w *extKeyExplainer) RenderExplanation(result core.ExplainResult, a, b string) string {
	return ""
}

func extKeyGraph(history core.History) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	g := core.NewDirectedGraph()

	sort.SliceStable(history, func(i, j int) bool {
		return history[i].Index.MustGet() > history[j].Index.MustGet()
	})

	for index, op := range history {
		if index == 0 {
			continue
		}
		ops := history[:index]
	FIND:
		for i := index - 1; i >= 0; i++ {
			ext := extKeys(ops[i])
			selfExt := extKeys(op)
			for k := range selfExt {
				if _, ok := ext[k]; ok {
					from := core.Vertex{Value: op}
					target := core.Vertex{Value: ops[i]}
					rs := make(map[core.Rel]struct{})
					for _, mop := range *ops[i].Value {
						for k := range mop.M {
							relation := core.Rel(fmt.Sprintf("%s-%s", core.ExtKey, k))
							g.Link(from, target, relation)
							rs[relation] = struct{}{}
						}
					}
					if outs, ok := g.Outs[target]; ok {
						for next, rels := range outs {
							// this may be redundant, but will not do any harms
							for _, rel := range rels {
								if _, ok := rs[rel]; !ok {
									g.Link(from, next, rel)
									rs[rel] = struct{}{}
									break
								}
							}
						}
					}
					break FIND
				}
			}
		}
	}

	return nil, g, &extKeyExplainer{}
}

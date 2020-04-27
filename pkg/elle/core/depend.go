package core

import (
	"fmt"
	"log"
	"strings"
)

// DataExplainer ...
type DataExplainer interface {
	// Given a pair of operations a and b, explains why b depends on a, in the
	//    form of a data structure. Returns `nil` if b does not depend on a.
	ExplainPairData(p1, p2 PathType) ExplainResult
	// Given a pair of operations, and short names for them, explain why b
	//  depends on a, as a string. `nil` indicates that b does not depend on a.
	RenderExplanation(result ExplainResult, preName, postName string) string
}

type DependType string

const (
	RealtimeDepend  DependType = "realtime"
	MonotonicDepend DependType = "monotonic"
	ProcessDepend   DependType = "process"
	WWDepend        DependType = "ww"
	WRDepend        DependType = "wr"
	RWDepend        DependType = "rw"

	// CombineDepend is a special form of DependType
	CombineDepend DependType = "combine"
)

type ExplainResult interface {
	Type() DependType
}

// CombinedExplainer struct
type CombinedExplainer struct {
	Explainers []DataExplainer
}

type CombineExplainResult struct {
	ex DataExplainer
	er ExplainResult
}

func (c CombineExplainResult) Type() DependType {
	return CombineDepend
}

// ExplainPairData find dependencies in a and b
func (c *CombinedExplainer) ExplainPairData(p1, p2 PathType) ExplainResult {
	for _, ex := range c.Explainers {
		er := ex.ExplainPairData(p1, p2)
		if er != nil {
			return CombineExplainResult{
				ex: ex,
				er: er,
			}
		}
	}
	return nil
}

// RenderExplanation render explanation result
func (c *CombinedExplainer) RenderExplanation(result ExplainResult, p1, p2 string) string {
	if result.Type() != CombineDepend {
		log.Fatalf("result type is not %s, type error", CombineDepend)
	}
	er := result.(CombineExplainResult)
	return er.ex.RenderExplanation(er.er, p1, p2)
}

// Combine composes multiple analyzers
func Combine(analyzers ...Analyzer) Analyzer {
	return func(history History) (anomalies Anomalies, graph *DirectedGraph, explainer DataExplainer) {
		ca := make(Anomalies)
		cg := NewDirectedGraph()
		var ce []DataExplainer

		for _, analyzer := range analyzers {
			a, g, e := analyzer(history)
			if g == nil {
				panic("got a nil graph")
			}
			ca.Merge(a)
			cg = DigraphUnion(cg, g)
			ce = append(ce, e)
		}

		return ca, cg, &CombinedExplainer{Explainers: ce}
	}
}

type ProcessDependent struct {
	Process int
}

func (_ ProcessDependent) Type() DependType {
	return ProcessDepend
}

// ProcessExplainer ...
type ProcessExplainer struct{}

// ExplainPairData explain pair data
func (e ProcessExplainer) ExplainPairData(p1, p2 PathType) ExplainResult {
	if !p1.Process.Present() || !p2.Process.Present() {
		return nil
	}
	if p1.Process.MustGet() == p2.Process.MustGet() && p1.Index < p2.Index {
		return ProcessDependent{Process: p1.Process.MustGet()}
	} else {
		return nil
	}
}

// RenderExplanation render explanation
func (e ProcessExplainer) RenderExplanation(result ExplainResult, preName, postName string) string {
	if result.Type() != ProcessDepend {
		log.Fatalf("result type is not %s, type error", ProcessDepend)
	}
	res := result.(ProcessDependent)
	return fmt.Sprintf("process %d excuted %s before %s", res.Process, preName, postName)
}

// RealtimeExplainer is Realtime order explainer
type RealtimeExplainer struct {
	historyReference History
	nextIndex        []int
}

// TODO: here it needs to get the preEnd and the postStart, it maybe complex and introduce another logic here.
func (r RealtimeExplainer) ExplainPairData(preEnd, postEnd PathType) ExplainResult {
	postStart := r.historyReference[r.nextIndex[postEnd.Index]]
	if preEnd.Index < postStart.Index {
		return RealtimeDependent{
			prefixEnd: &preEnd,
			postStart: &postStart,
		}
	}
	return nil
}

func (r RealtimeExplainer) RenderExplanation(result ExplainResult, preName, postName string) string {
	if result.Type() != RealtimeDepend {
		log.Fatalf("result type is not %s, type error", RealtimeDepend)
	}
	res := result.(RealtimeDependent)
	s := fmt.Sprintf("%s complete at index %d, ", preName, res.postStart.Index)

	if !res.postStart.Time.IsZero() && !res.prefixEnd.Time.IsZero() {
		t1, t2 := res.prefixEnd.Time, res.postStart.Time
		if t1.Before(t2) {
			delta := t2.Sub(t1)
			s += fmt.Sprintf("%v seconds just ", delta.Milliseconds())
		}
	}

	s += fmt.Sprintf("before the invocation of %s at index %d", postName, res.prefixEnd.Index)
	return s
}

type RealtimeDependent struct {
	prefixEnd *Op
	postStart *Op
}

func (_ RealtimeDependent) Type() DependType {
	return RealtimeDepend
}

// Note: MonotonicKey is used on rw_register, so I don't explain it currently.
// MonotonicKeyExplainer ...
type MonotonicKeyExplainer struct{}

func (e MonotonicKeyExplainer) ExplainPairData(p1, p2 PathType) ExplainResult {
	panic("implement me")
}

// RenderExplanation render explanation
func (e MonotonicKeyExplainer) RenderExplanation(result ExplainResult, preName, postName string) string {
	panic("impl me")
}

type ICycleExplainer interface {
	ExplainCycle(pairExplainer DataExplainer, circle Circle) (Circle, []Step)
	RenderCycleExplanation(explainer DataExplainer, circle Circle, steps []Step) string
}

// CycleExplainer provides the step-by-step explanation of the relationships between pairs of operations
type CycleExplainer struct{}

// Explain for the whole scc.
func (c *CycleExplainer) ExplainCycle(explainer DataExplainer, circle Circle) (Circle, []Step) {
	var steps []Step
	for i := 1; i < len(circle.Path); i++ {
		res := explainer.ExplainPairData(circle.Path[i-1], circle.Path[i])
		steps = append(steps, Step{Result: res})
	}
	return circle, steps
}

type OpBinding struct {
	Operation Op
	Name      string
}

func (c *CycleExplainer) RenderCycleExplanation(explainer DataExplainer, circle Circle, steps []Step) string {
	var bindings []OpBinding
	if len(bindings) < 2 {
		return ""
	}
	for i, v := range circle.Path[:len(circle.Path)-1] {
		bindings = append(bindings, OpBinding{
			Operation: v,
			Name:      fmt.Sprintf("T%d", i),
		})
	}
	bindingsExplain := explainBindings(bindings)
	stepsResult := explainCycleOps(explainer, bindings, steps)
	return bindingsExplain + "\n" + stepsResult
}

// Takes a seq of [name op] pairs, and constructs a string naming each op.
func explainBindings(bindings []OpBinding) string {
	var seq []string
	seq = []string{"Let:"}
	for _, v := range bindings {
		seq = append(seq, fmt.Sprintf(" %s = %s", v.Name, v.Operation.String()))
	}
	return strings.Join(seq, "\n")
}

func explainCycleOps(pairExplainer DataExplainer, bindings []OpBinding, steps []Step) string {
	var explainitions []string
	for i := 1; i < len(bindings); i++ {
		explainitions = append(explainitions, fmt.Sprintf("%s < %s, because %s", bindings[i-1].Name,
			bindings[i].Name, pairExplainer.RenderExplanation(steps[i-1].Result, bindings[i-1].Name, bindings[i].Name)))
	}
	// extra result
	explainitions = append(explainitions, fmt.Sprintf("%s < %s, because %s", bindings[len(bindings)-1].Name,
		bindings[0].Name, pairExplainer.RenderExplanation(steps[len(bindings)].Result, bindings[len(bindings)-1].Name, bindings[0].Name)))

	return strings.Join(explainitions, "\n")
}

func explainSCC(g *DirectedGraph, cycleExplainer CycleExplainer, pairExplainer DataExplainer, scc SCC) string {
	cycle := FindCycle(*g, scc)
	_, steps := cycleExplainer.ExplainCycle(pairExplainer, cycle)
	return cycleExplainer.RenderCycleExplanation(pairExplainer, cycle, steps)
}

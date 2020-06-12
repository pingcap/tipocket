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

// DependType records the depend type of a relation
type DependType string

const (
	// RealtimeDepend ...
	RealtimeDepend DependType = "realtime"
	// MonotonicDepend ...
	MonotonicDepend DependType = "monotonic"
	// ProcessDepend ...
	ProcessDepend DependType = "process"
	// WWDepend ...
	WWDepend DependType = "ww"
	// WRDepend ...
	WRDepend DependType = "wr"
	// RWDepend ...
	RWDepend DependType = "rw"
)

// ExplainResult is an interface, contains rwExplainerResult, wwExplainerResult wr ExplainerResult etc
type ExplainResult interface {
	Type() DependType
}

// CombinedExplainer struct
type CombinedExplainer struct {
	Explainers []DataExplainer
	store      map[ExplainResult]DataExplainer
}

// NewCombineExplainer builds a CombinedExplainer according to a explainer array
func NewCombineExplainer(explainers []DataExplainer) *CombinedExplainer {
	return &CombinedExplainer{
		Explainers: explainers,
		store:      make(map[ExplainResult]DataExplainer),
	}
}

// ExplainPairData find dependencies in a and b
func (c *CombinedExplainer) ExplainPairData(p1, p2 PathType) ExplainResult {
	for _, ex := range c.Explainers {
		er := ex.ExplainPairData(p1, p2)
		if er != nil {
			c.store[er] = ex
			return er
		}
	}
	return nil
}

// RenderExplanation render explanation result
func (c *CombinedExplainer) RenderExplanation(result ExplainResult, p1, p2 string) string {
	return c.store[result].RenderExplanation(result, p1, p2)
}

// Combine composes multiple analyzers
func Combine(analyzers ...Analyzer) Analyzer {
	return func(history History, opts ...interface{}) (anomalies Anomalies, graph *DirectedGraph, explainer DataExplainer) {
		ca := make(Anomalies)
		cg := NewDirectedGraph()
		var ce []DataExplainer

		for _, analyzer := range analyzers {
			a, g, e := analyzer(history, opts...)
			if g == nil {
				panic("got a nil graph")
			}
			ca.Merge(a)
			cg = DigraphUnion(cg, g)
			ce = append(ce, e)
		}

		return ca, cg, NewCombineExplainer(ce)
	}
}

// ProcessResult ...
type ProcessResult struct {
	Process int
}

// Type ...
func (ProcessResult) Type() DependType {
	return ProcessDepend
}

// ProcessExplainer ...
type ProcessExplainer struct{}

// ExplainPairData explain pair data
func (e ProcessExplainer) ExplainPairData(p1, p2 PathType) ExplainResult {
	if !p1.Process.Present() || !p2.Process.Present() {
		return nil
	}
	if p1.Process.MustGet() == p2.Process.MustGet() && p1.Index.MustGet() < p2.Index.MustGet() {
		return ProcessResult{Process: p1.Process.MustGet()}
	}
	return nil
}

// RenderExplanation render explanation
func (e ProcessExplainer) RenderExplanation(result ExplainResult, preName, postName string) string {
	if result.Type() != ProcessDepend {
		log.Fatalf("result type is not %s, type error", ProcessDepend)
	}
	res := result.(ProcessResult)
	return fmt.Sprintf("process %d excuted %s before %s", res.Process, preName, postName)
}

// RealtimeExplainer is Realtime order explainer
type RealtimeExplainer struct {
	pair map[Op]Op
}

// ExplainPairData ...
func (r RealtimeExplainer) ExplainPairData(preEnd, postEnd PathType) ExplainResult {
	postStart, ok := r.pair[postEnd]
	if !ok {
		log.Fatalf("cannot find the invocation of %s, the code may has bug", postEnd.String())
	}
	if preEnd.Index.MustGet() < postStart.Index.MustGet() {
		return RealtimeExplainResult{
			PreEnd:    preEnd,
			PostStart: postStart,
		}
	}
	return nil
}

// RenderExplanation ...
func (r RealtimeExplainer) RenderExplanation(result ExplainResult, preName, postName string) string {
	if result.Type() != RealtimeDepend {
		log.Fatalf("result type is not %s, type error", RealtimeDepend)
	}
	res := result.(RealtimeExplainResult)
	s := fmt.Sprintf("%s complete at index %d, ", preName, res.PreEnd.Index.MustGet())

	if !res.PostStart.Time.IsZero() && !res.PreEnd.Time.IsZero() {
		t1, t2 := res.PreEnd.Time, res.PostStart.Time
		if t1.Before(t2) {
			delta := t2.Sub(t1)
			s += fmt.Sprintf("%v seconds just ", delta.Seconds())
		}
	}

	s += fmt.Sprintf("before the invocation of %s at index %d", postName, res.PostStart.Index.MustGet())
	return s
}

// RealtimeExplainResult records a real time explain result
type RealtimeExplainResult struct {
	PreEnd    Op
	PostStart Op
}

// Type ...
func (RealtimeExplainResult) Type() DependType {
	return RealtimeDepend
}

// MonotonicKeyExplainer ...
// Note: MonotonicKey is used on rw_register, so I don't explain it currently.
type MonotonicKeyExplainer struct{}

// ExplainPairData ...
func (e MonotonicKeyExplainer) ExplainPairData(p1, p2 PathType) ExplainResult {
	panic("implement me")
}

// RenderExplanation render explanation
func (e MonotonicKeyExplainer) RenderExplanation(result ExplainResult, preName, postName string) string {
	panic("impl me")
}

// CycleExplainerResult impls Anomaly
type CycleExplainerResult struct {
	Circle Circle
	Steps  []Step
	Typ    string
}

// IAnomaly ...
func (c CycleExplainerResult) IAnomaly() {}

// String ...
func (c CycleExplainerResult) String() string {
	panic("implement me")
}

// Type ...
func (c CycleExplainerResult) Type() string {
	return c.Typ
}

// ICycleExplainer is an interface
type ICycleExplainer interface {
	ExplainCycle(pairExplainer DataExplainer, circle Circle) CycleExplainerResult
	RenderCycleExplanation(explainer DataExplainer, cr CycleExplainerResult) string
}

// CycleExplainer provides the step-by-step explanation of the relationships between pairs of operations
type CycleExplainer struct{}

// ExplainCycle for a circle
func (c *CycleExplainer) ExplainCycle(explainer DataExplainer, circle Circle) CycleExplainerResult {
	var steps []Step
	for i := 1; i < len(circle.Path); i++ {
		res := explainer.ExplainPairData(circle.Path[i-1], circle.Path[i])
		steps = append(steps, Step{Result: res})
	}
	return CycleExplainerResult{
		Circle: circle,
		Steps:  steps,
		// don't return type
		Typ: "",
	}
}

// OpBinding binds the operation with Txx name
type OpBinding struct {
	Operation Op
	Name      string
}

// RenderCycleExplanation ...
func (c *CycleExplainer) RenderCycleExplanation(explainer DataExplainer, cr CycleExplainerResult) string {
	var bindings []OpBinding
	for i, v := range cr.Circle.Path[:len(cr.Circle.Path)-1] {
		bindings = append(bindings, OpBinding{
			Operation: v,
			Name:      fmt.Sprintf("T%d", i+1),
		})
	}
	bindingsExplain := explainBindings(bindings)
	stepsResult := explainCycleOps(explainer, bindings, cr.Steps)
	return bindingsExplain + "\n\nThen:\n" + stepsResult
}

// Takes a seq of [name op] pairs, and constructs a string naming each op.
func explainBindings(bindings []OpBinding) string {
	var seq []string
	seq = []string{"Let:"}
	for _, v := range bindings {
		seq = append(seq, fmt.Sprintf("  %s = %s", v.Name, v.Operation.String()))
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
		bindings[0].Name, pairExplainer.RenderExplanation(steps[len(steps)-1].Result, bindings[len(bindings)-1].Name, bindings[0].Name)))

	for idx, ex := range explainitions {
		if idx == len(explainitions)-1 {
			explainitions[idx] = fmt.Sprintf("  - However, %s: a contradiction!", ex)
		} else {
			explainitions[idx] = fmt.Sprintf("  - %s.", ex)
		}
	}

	return strings.Join(explainitions, "\n")
}

func explainSCC(g *DirectedGraph, cycleExplainer CycleExplainer, pairExplainer DataExplainer, scc SCC) string {
	cycle := NewCircle(FindCycle(g, scc))
	if cycle == nil {
		panic("don't find a cycle, the code may has bug")
	}
	cr := cycleExplainer.ExplainCycle(pairExplainer, *cycle)
	return cycleExplainer.RenderCycleExplanation(pairExplainer, cr)
}

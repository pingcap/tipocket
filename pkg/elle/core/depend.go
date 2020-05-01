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
	return c.er.Type()
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
	if p1.Process.MustGet() == p2.Process.MustGet() && p1.Index.MustGet() < p2.Index.MustGet() {
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
	pair map[Op]Op
}

// TODO: here it needs to get the preEnd and the postStart, it maybe complex and introduce another logic here.
func (r RealtimeExplainer) ExplainPairData(preEnd, postEnd PathType) ExplainResult {
	postStart, ok := r.pair[postEnd]
	if !ok {
		log.Fatalf("cannot find the invocation of %s, the code may has bug", postEnd.String())
	}
	if preEnd.Index.MustGet() < postStart.Index.MustGet() {
		return RealtimeDependent{
			preEnd:    &preEnd,
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
	s := fmt.Sprintf("%s complete at index %d, ", preName, res.preEnd.Index.MustGet())

	if !res.postStart.Time.IsZero() && !res.preEnd.Time.IsZero() {
		t1, t2 := res.preEnd.Time, res.postStart.Time
		if t1.Before(t2) {
			delta := t2.Sub(t1)
			s += fmt.Sprintf("%v seconds just ", delta.Seconds())
		}
	}

	s += fmt.Sprintf("before the invocation of %s at index %d", postName, res.postStart.Index.MustGet())
	return s
}

type RealtimeDependent struct {
	preEnd    *Op
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

// CycleExplainerResult impls Anomaly
type CycleExplainerResult struct {
	Circle Circle
	Steps  []Step
	Typ    DependType
}

func (c CycleExplainerResult) String() string {
	panic("implement me")
}

func (c CycleExplainerResult) IAnomaly() string {
	return c.String()
}

func (c CycleExplainerResult) Type() DependType {
	return c.Typ
}

type ICycleExplainer interface {
	ExplainCycle(pairExplainer DataExplainer, circle Circle) CycleExplainerResult
	RenderCycleExplanation(explainer DataExplainer, cr CycleExplainerResult) string
}

// CycleExplainer provides the step-by-step explanation of the relationships between pairs of operations
type CycleExplainer struct{}

// Explain for the whole scc.
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

type OpBinding struct {
	Operation Op
	Name      string
}

func (c *CycleExplainer) RenderCycleExplanation(explainer DataExplainer, cr CycleExplainerResult) string {
	var bindings []OpBinding
	for i, v := range cr.Circle.Path[:len(cr.Circle.Path)-1] {
		bindings = append(bindings, OpBinding{
			Operation: v,
			Name:      fmt.Sprintf("T%d", i),
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

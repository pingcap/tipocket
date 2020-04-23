package core

import (
	"fmt"
	"log"
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

// CombinedExplainer struct
type CombinedExplainer struct {
	Explainers []DataExplainer
}

// ExplainPairData find dependencies in a and b
func (c *CombinedExplainer) ExplainPairData(p1, p2 PathType) ExplainResult {
	panic("implement me")
}

// RenderExplanation render explanation result
func (c *CombinedExplainer) RenderExplanation(result ExplainResult, preName, postName string) string {
	return ""
}

// CombineExplainer combines explainers into one
func CombineExplainer(explainers []DataExplainer) DataExplainer {
	return &CombinedExplainer{explainers}
}

// Combine composes multiple analyzers
func Combine(analyzers ...Analyzer) Analyzer {
	panic("implement me")
}

type DependType string

const (
	RealtimeDepend  DependType = "realtime"
	MonotonicDepend DependType = "monotonic"
	ProcessDepend   DependType = "process"
)

type ExplainResult interface {
	Type() DependType
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
		log.Fatalf("result type is not %s, type error", result.Type())
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
		log.Fatalf("result type is not %s, type error", result.Type())
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

// TODO: add panic & recover logic here.
func explainCyclePairData(pairExplainer DataExplainer, p1, p2 PathType) ExplainResult {
	return pairExplainer.ExplainPairData(p1, p2)
}

// CycleExplainer provides the step-by-step explanation of the relationships between pairs of operations
type CycleExplainer struct {}

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
	panic("implement me")
}

func explainBindings(bindings []OpBinding) string {
	panic("implement me")
}

func explainCycleOps(pairExplainer DataExplainer, binding OpBinding, steps []Step) {
	panic("implement me")
}

func explainSCC(graph DirectedGraph, pairExplainer DataExplainer, scc SCC) string {
	panic("implement me")
}

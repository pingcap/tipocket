package listappend

import (
	"fmt"
	"strings"

	"github.com/goccy/go-graphviz"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

type record struct {
	name      string
	label     string
	height    float32
	color     string
	fontColor string
}

func (r record) String() string {
	return fmt.Sprintf(`%s [height=%.2f,shape=record,label="%s",color="%s",fontcolor="%s"]`, r.name, r.height, r.label, r.color, r.fontColor)
}

type edge struct {
	from      string
	to        string
	label     string
	color     string
	fontColor string
}

func (e edge) String() string {
	return fmt.Sprintf(`%s -> %s [label="%s",fontcolor="%s",color="%s"]`, e.from, e.to, e.label, e.color, e.fontColor)
}

var typeColor = map[core.OpType]string{
	core.OpTypeOk:   "#0058AD",
	core.OpTypeInfo: "#AC6E00",
	core.OpTypeFail: "#A50053",
}

func relColor(rel core.DependType) string {
	switch rel {
	case core.WWDepend:
		return "#C02700"
	case core.WRDepend:
		return "#C000A5"
	case core.RWDepend:
		return "#5B00C0"
	case core.RealtimeDepend:
		return "#0050C0"
	case core.ProcessDepend:
		return "#00C0C0"
	default:
		return "#585858"
	}

}

func renderOp(nodeIdx map[core.Op]string, op core.Op) record {
	var labels []string
	for idx, mop := range *op.Value {
		labels = append(labels, fmt.Sprintf("<f%d> %s", idx, mop.String()))
	}
	return record{
		name:      nodeIdx[op],
		label:     strings.Join(labels, "|"),
		height:    0.4,
		color:     typeColor[op.Type],
		fontColor: typeColor[op.Type],
	}
}

func renderEdge(analysis core.CheckResult, nodeIdx map[core.Op]string, a, b core.Op) edge {
	explainer := analysis.Explainer
	ex := explainer.ExplainPairData(a, b)
	an := nodeIdx[a]
	bn := nodeIdx[b]
	switch t := ex.(type) {
	case rwExplainResult:
		ami := t.AMopIndex
		an = fmt.Sprintf("%s:f%d", an, ami)
		bmi := t.BMopIndex
		bn = fmt.Sprintf("%s:f%d", bn, bmi)
	case wwExplainResult:
		ami := t.AMopIndex
		an = fmt.Sprintf("%s:f%d", an, ami)
		bmi := t.BMopIndex
		bn = fmt.Sprintf("%s:f%d", bn, bmi)
	case wrExplainResult:
		ami := t.AMopIndex
		an = fmt.Sprintf("%s:f%d", an, ami)
		bmi := t.BMopIndex
		bn = fmt.Sprintf("%s:f%d", bn, bmi)
	}
	return edge{
		from:      an,
		to:        bn,
		label:     string(ex.Type()),
		color:     relColor(ex.Type()),
		fontColor: relColor(ex.Type()),
	}
}

func renderEdges(analysis core.CheckResult, scc core.SCC, nodeIdx map[core.Op]string, op core.Op) []edge {
	edges := make([]edge, 0)
	g := analysis.Graph

	sccSet := map[core.Op]struct{}{}
	for _, v := range scc.Vertices {
		sccSet[v.Value.(core.Op)] = struct{}{}
	}
	for _, nextOp := range g.Out(core.Vertex{Value: op}) {
		if _, e := sccSet[nextOp.Value.(core.Op)]; e {
			edges = append(edges, renderEdge(analysis, nodeIdx, op, nextOp.Value.(core.Op)))
		}
	}
	return edges
}

func renderScc(analysis core.CheckResult, scc core.SCC) string {
	var tpl = []string{"digraph g {"}
	nodeIdx := make(map[core.Op]string)
	for _, node := range scc.Vertices {
		op := node.Value.(core.Op)
		if op.Index.Present() {
			nodeIdx[op] = fmt.Sprintf("T%d", op.Index.MustGet())
		} else {
			nodeIdx[op] = fmt.Sprintf("n%d", op.Index.MustGet())
		}
	}

	var nodes []record
	var edges []edge

	for _, node := range scc.Vertices {
		op := node.Value.(core.Op)
		nodes = append(nodes, renderOp(nodeIdx, op))
		edges = append(edges, renderEdges(analysis, scc, nodeIdx, op)...)
	}

	for _, node := range nodes {
		tpl = append(tpl, fmt.Sprintf("    %s", node.String()))
	}

	tpl = append(tpl, "\n")

	for _, edge := range edges {
		tpl = append(tpl, fmt.Sprintf("    %s", edge.String()))
	}

	tpl = append(tpl, "}")
	return strings.Join(tpl, "\n")
}

func plotAnalysis(analysis core.CheckResult, directory string) error {
	g := graphviz.New()
	for i, scc := range analysis.Sccs {
		tpl := renderScc(analysis, scc)
		graph, err := graphviz.ParseBytes([]byte(tpl))
		if err != nil {
			return err
		}
		if err := g.RenderFilename(graph, graphviz.SVG, fmt.Sprintf("%s/%d.svg", directory, i)); err != nil {
			return err
		}
	}
	return nil
}

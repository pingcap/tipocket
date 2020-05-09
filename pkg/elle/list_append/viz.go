package list_append

import (
	"fmt"
	"github.com/goccy/go-graphviz"
	"github.com/pingcap/tipocket/pkg/elle/core"
	"strings"
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

func plotAnalysis(analysis core.CheckResult) error {
	g := graphviz.New()
	for i, scc := range analysis.Sccs {
		tpl := renderScc(analysis, scc)
		graph, err := graphviz.ParseBytes([]byte(tpl))
		if err != nil {
			return err
		}
		if err := g.RenderFilename(graph, graphviz.SVG, fmt.Sprintf("%d.svg", i)); err != nil {
			return err
		}
	}
	return nil
}

//func render(dg *core.DirectedGraph) error {
//	g := graphviz.New()
//	//graph, err := g.Graph()
//	//if err != nil {
//	//	log.Fatal(err)
//	//}
//	//defer func() {
//	//	if err := graph.Close(); err != nil {
//	//		log.Fatal(err)
//	//	}
//	//	g.Close()
//	//}()
//	//n, err := graph.CreateNode("n")
//	//n.SetHeight(0.4)
//	//n.SetShape("record")
//	//n.SetLabel("<f0> r :x [1 2]|<f1> r :z [1]")
//	//n.SetColor("#0058AD")
//	//n.SetFontColor("#0058AD")
//	//if err != nil {
//	//	log.Fatal(err)
//	//}
//	//m, err := graph.CreateNode("m")
//	//if err != nil {
//	//	log.Fatal(err)
//	//}
//	//m.SetHeight(0.4)
//	//m.SetShape("record")
//	//m.SetLabel("<f0> a :z 2|<f1> a :y 1")
//	//m.SetColor("#0058AD")
//	//m.SetFontColor("#0058AD")
//	//
//	//e, err := graph.CreateEdge("e", n, m)
//	//if err != nil {
//	//	log.Fatal(err)
//	//}
//	//e.SetLabel("rw")
//	//var buf bytes.Buffer
//	//if err := g.Render(graph, "dot", &buf); err != nil {
//	//	log.Fatal(err)
//	//}
//	//fmt.Println(buf.String())
//	//
//	//if err := g.RenderFilename(graph, graphviz.SVG, "graph.svg"); err != nil {
//	//	log.Fatal(err)
//	//}
//	//return nil
//
//	graph, _ := graphviz.ParseBytes([]byte(`
//digraph a_graph {
//    T1 [height=0.4,shape=record,label="<f0> r :x [1 2]|<f1> r :z [1]",color="#0058AD",fontcolor="#0058AD"]
//    T5 [height=0.4,shape=record,label="<f0> a :z 1",color="#0058AD",fontcolor="#0058AD"]
//    T7 [height=0.4,shape=record,label="<f0> a :z 2|<f1> a :y 1",color="#0058AD",fontcolor="#0058AD"]
//    T9 [height=0.4,shape=record,label="<f0> r :z nil|<f1> a :x 2",color="#0058AD",fontcolor="#0058AD"]
//    T3 [height=0.4,shape=record,label="<f0> a :x 1|<f1> r :y [1]|<f2> r :z [1 2]",color="#0058AD",fontcolor="#0058AD"]
//
//    T1 -> T3 [label="rt",fontcolor="#0050C0",color="#0050C0"]
//
//    T1:f1 -> T7:f0 [label="rw",fontcolor="#5B00C0",color="#5B00C0"]
//    T5:f0 -> T1:f1 [label="wr",fontcolor="#C000A5",color="#C000A5"]
//    T5:f0 -> T7:f0 [label="ww",fontcolor="#C02700",color="#C02700"]
//    T7:f1 -> T3:f1 [label="wr",fontcolor="#C000A5",color="#C000A5"]
//    T7 -> T9 [label="rt",fontcolor="#0050C0",color="#0050C0"]
//    T9:f1 -> T1:f0 [label="wr",fontcolor="#C000A5",color="#C000A5"]
//    T9:f0 -> T5:f0 [label="rw",fontcolor="#5B00C0",color="#5B00C0"]
//    T3 -> T5 [label="rt",fontcolor="#0050C0",color="#0050C0"]
//    T3:f0 -> T9:f1 [label="ww",fontcolor="#C02700",color="#C02700"]
//}
//`))
//	if err := g.RenderFilename(graph, graphviz.SVG, "graph.svg"); err != nil {
//		log.Fatal(err)
//	}
//	return nil
//}

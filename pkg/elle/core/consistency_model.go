package core

var impliedAnomalies = MapToDirectedGraph(map[Vertex][]Vertex{
	Vertex{"G0"}:                     {{"G1c"}},
	Vertex{"G0-process"}:             {{"G1c-process"}, {"G0-realtime"}},
	Vertex{"G0-realtime"}:            {{"G1c-realtime"}},
	Vertex{"G1a"}:                    {{"G1"}},
	Vertex{"G1b"}:                    {{"G1"}},
	Vertex{"G1c"}:                    {{"G1"}},
	Vertex{"G1c-process"}:            {{"G1-process"}, {"G1c-realtime"}},
	Vertex{"G-single"}:               {{"G-nonadjacent"}, {"GSIb"}},
	Vertex{"G-single-process"}:       {{"G-nonadjacent-process"}, {"G-single-realtime"}},
	Vertex{"G-single-realtime"}:      {{"G-nonadjacent-realtime"}},
	Vertex{"G-nonadjacent"}:          {{"G2"}},
	Vertex{"G-nonadjacent-process"}:  {{"G2-process"}, {"G-nonadjacent-realtime"}},
	Vertex{"G-nonadjacent-realtime"}: {{"G2-realtime"}},
	Vertex{"G2-item"}:                {{"G2"}},
	Vertex{"G2-item-process"}:        {{"G2-process"}, {"G2-item-realtime"}},
	Vertex{"G2-item-realtime"}:       {{"G2-realtime"}},
	Vertex{"G2-process"}:             {{"G2-realtime"}},
	Vertex{"GSIa"}:                   {{"GSI"}},
	Vertex{"GSIb"}:                   {{"GSI"}},
	Vertex{"incompatible-order"}:     {{"G1a"}},
	Vertex{"dirty-update"}:           {{"G1a"}},
})

// ConsistencyModelName defines the consistency model name
type ConsistencyModelName = string

var canonicalModelNames = map[ConsistencyModelName]string{
	"consistent-view":         "PL-2+",
	"conflict-serializable":   "PL-3",
	"cursor-stability":        "PL-CS",
	"forward-consistent-view": "PL-FCV",
	"monotonic-snapshot-read": "PL-MSR",
	"monotonic-view":          "PL-2L",
	"read-committed":          "PL-2",
	"read-uncommitted":        "PL-1",
	"repeatable-read":         "PL-2.99",
	"serializable":            "PL-3",
	"snapshot-isolation":      "PL-SI",
	"strict-serializable":     "PL-SS",
	"update-serializable":     "PL-3U",
}

// AllAnomaliesImplying yields a set of anomalies which would imply any of those anomalies
func AllAnomaliesImplying(anomalies []string) (res []string) {
	var initV []Vertex
	for _, anomaly := range anomalies {
		initV = append(initV, Vertex{Value: anomaly})
	}
	return set(stringSlice(impliedAnomalies.Bfs(initV, false)))
}

// AllImpliedAnomalies yields a set of anomalies implied by those
func AllImpliedAnomalies(anomalies []string) (res []string) {
	var initV []Vertex
	for _, anomaly := range anomalies {
		initV = append(initV, Vertex{Value: anomaly})
	}
	return set(stringSlice(impliedAnomalies.Bfs(initV, true)))
}

// canonicalModelName can be applied to DirectedGraph.MapVertices
func canonicalModelName(name interface{}) interface{} {
	if cname, ok := canonicalModelNames[name.(string)]; ok {
		return cname
	}
	return name
}

// friendlyModelName can be applied to DirectedGraph.MapVertices too
func friendlyModelName(name interface{}) interface{} {
	if name == "PL-3" {
		return "serializable"
	}
	for f, c := range canonicalModelNames {
		if c == name {
			return f
		}
	}
	return name
}

// Models sees https://jepsen.io/consistency for sources.
var Models = MapToDirectedGraph(map[Vertex][]Vertex{
	Vertex{"causal-cerone"}:                     {{"read-atomic"}},
	Vertex{"consistent-view"}:                   {{"cursor-stability"}, {"monotonic-view"}},
	Vertex{"conflict-serializable"}:             {{"view-serializable"}},
	Vertex{"cursor-stability"}:                  {{"read-committed"}, {"PL-2"}},
	Vertex{"forward-consistent-view"}:           {{"consistent-view"}, {"PL-1"}},
	Vertex{"PL-3"}:                              {{"repeatable-read"}, {"update-serializable"}},
	Vertex{"update-serializable"}:               {{"forward-consistent-view"}},
	Vertex{"monotonic-atomic-view"}:             {{"read-committed"}},
	Vertex{"monotonic-view"}:                    {{"PL-2"}},
	Vertex{"monotonic-snapshot-read"}:           {{"PL-2"}},
	Vertex{"parallel-snapshot-isolation"}:       {{"causal-cerone"}},
	Vertex{"prefix"}:                            {{"causal-cerone"}},
	Vertex{"read-committed"}:                    {{"read-uncommitted"}},
	Vertex{"repeatable-read"}:                   {{"cursor-stability"}, {"monotonic-atomic-view"}},
	Vertex{"serializable"}:                      {{"repeatable-read"}, {"snapshot-isolation"}, {"view-serializable"}},
	Vertex{"session-serializable"}:              {{"1SR"}},
	Vertex{"snapshot-isolation"}:                {{"forward-consistent-view"}, {"monotonic-atomic-view"}, {"monotonic-snapshot-read"}, {"parallel-snapshot-isolation"}, {"prefix"}},
	Vertex{"strict-serializable"}:               {{"PL-3"}, {"serializable"}, {"linearizable"}, {"snapshot-isolation"}, {"strong-session-serializable"}},
	Vertex{"strong-serializable"}:               {{"session-serializable"}},
	Vertex{"strong-session-serializable"}:       {{"serializable"}},
	Vertex{"strong-session-snapshot-isolation"}: {{"snapshot-isolation"}},
	Vertex{"strong-snapshot-isolation"}:         {{"strong-session-snapshot-isolation"}},
	Vertex{"linearizable"}:                      {{"sequential"}},
	Vertex{"causal"}:                            {{"writes-follow-reads"}},
	Vertex{"causal"}:                            {{"writes-follow-reads"}, {"PRAM"}},
	Vertex{"PRAM"}:                              {{"monotonic-reads"}, {"monotonic-writes"}, {"read-your-writes"}},
}).MapVertices(canonicalModelName)

// Takes a set of models, and expands it to a set of all models which are implied by any of those models
func allImpliedModels(models []string) (res []string) {
	var initV []Vertex
	for _, model := range models {
		initV = append(initV, Vertex{Value: model})
	}
	return set(stringSlice(Models.Bfs(initV, true)))
}

// Takes a set of models which are impossible, expands it to a set of all models which are also impossible.
func allImpossibleModels(models []string) (res []string) {
	var initV []Vertex
	for _, model := range models {
		initV = append(initV, Vertex{Value: model})
	}
	vertices := Models.Bfs(initV, false)
	for _, vertex := range vertices {
		res = append(res, vertex.Value.(string))
	}
	return set(res)
}

func mostModel(ms []string, isOut bool) (res []string) {
	var cnames []string
	for _, model := range ms {
		cnames = append(cnames, canonicalModelName(model).(string))
	}
	cnames = set(cnames)
	res = cnames[:]
	for _, model := range cnames {
		if hasCommon(slice(res, model), stringSlice(Models.Bfs([]Vertex{{model}}, isOut))) {
			res = slice(res, model)
		}
	}
	return res
}

func strongestModels(ms []string) []string {
	return mostModel(ms, false)
}

func weakestModels(ms []string) []string {
	return mostModel(ms, true)
}

var directProscribedAnomalies = MapToDirectedGraph(map[Vertex][]Vertex{
	Vertex{"causal-cerone"}:                     {{"internal"}, {"G1a"}},
	Vertex{"cursor-stability"}:                  {{"G1"}, {"G-cursor"}},
	Vertex{"monotonic-view"}:                    {{"G1"}, {"G-monotonic"}},
	Vertex{"monotonic-snapshot-read"}:           {{"G1"}, {"G-MSR"}},
	Vertex{"consistent-view"}:                   {{"G1"}, {"G-single"}},
	Vertex{"forward-consistent-view"}:           {{"G1"}, {"G-SIb"}},
	Vertex{"parallel-snapshot-isolation"}:       {{"internal"}, {"G1a"}},
	Vertex{"PL-3"}:                              {{"G1"}, {"G2"}},
	Vertex{"PL-2"}:                              {{"G1"}},
	Vertex{"PL-1"}:                              {{"G0"}, {"duplicate-elements"}, {"cyclic-versions"}},
	Vertex{"prefix"}:                            {{"internal"}, {"G1a"}},
	Vertex{"serializable"}:                      {{"internal"}},
	Vertex{"snapshot-isolation"}:                {{"internal"}, {"G1"}, {"G-SI"}},
	Vertex{"read-atomic"}:                       {{"internal"}, {"G1a"}},
	Vertex{"repeatable-read"}:                   {{"G1"}, {"G2-item"}},
	Vertex{"strict-serializable"}:               {{"G1"}, {"G1c-realtime"}, {"G2-realtime"}},
	Vertex{"strong-session-snapshot-isolation"}: {{"G-nonadjacent"}},
	Vertex{"strong-session-serializable"}:       {{"G1c-process"}, {"G2-process"}},
	Vertex{"update-serializable"}:               {{"G1"}, {"G-update"}},
}).MapVertices(canonicalModelName)

// AnomaliesProhibitedBy takes a collection of consistency models, and returns a set of anomalies
//  which can't be present if all of those models are to hold
func AnomaliesProhibitedBy(models []string) []string {
	var cnames []string
	var anomalies []string
	for _, model := range models {
		cnames = append(cnames, canonicalModelName(model).(string))
	}
	cnames = allImpliedModels(cnames)

	for _, model := range cnames {
		anomalies = append(anomalies, stringSlice(directProscribedAnomalies.Out(Vertex{model}))...)
	}
	return AllAnomaliesImplying(anomalies)
}

// anomaliesImpossibleModels takes a collection of anomalies, and returns a set of models which can't hold, given those anomalies are present
func anomaliesImpossibleModels(anomalies []string) []string {
	as := AllImpliedAnomalies(anomalies)
	var allAnomalies []string

	for _, anomaly := range as {
		allAnomalies = append(allAnomalies, stringSlice(directProscribedAnomalies.In(Vertex{anomaly}))...)
	}
	return allImpossibleModels(allAnomalies)
}

// FriendlyBoundary takes a set of anomalies, and yields not and alsoNot
// where not is the weakest set of consistency models invalidated by the given anomaly,
// and alsoNot is the remaining set of stronger models
func FriendlyBoundary(anomalies []string) (not []string, alsoNot []string) {
	impossible := anomaliesImpossibleModels(anomalies)
	not = weakestModels(impossible)
	alsoNot = set(impossible)
	for _, n := range not {
		alsoNot = slice(alsoNot, n)
	}
	return set(mapFunc(not, func(s string) string {
			return friendlyModelName(s).(string)
		})), set(mapFunc(alsoNot, func(s string) string {
			return friendlyModelName(s).(string)
		}))
}

// slice returns a new slice that removes the model
func slice(models []string, model string) []string {
	var res []string
	for _, m := range models {
		if m == model {
			continue
		}
		res = append(res, m)
	}
	return res
}

func exists(target string, set []string) bool {
	for _, s := range set {
		if s == target {
			return true
		}
	}
	return false
}

func hasCommon(s1 []string, s2 []string) bool {
	for _, s := range s1 {
		if exists(s, s2) {
			return true
		}
	}
	return false
}

func set(slice []string) []string {
	var res = make([]string, 0)
	for _, s := range slice {
		if !exists(s, res) {
			res = append(res, s)
		}
	}
	return res
}

// Set export set
func Set(slice []string) []string {
	return set(slice)
}

// stringSlice casts []Vertex to []string
func stringSlice(vs []Vertex) (res []string) {
	for _, v := range vs {
		res = append(res, v.Value.(string))
	}
	return res
}

func mapFunc(slice []string, f func(string) string) []string {
	var res []string
	for _, s := range slice {
		res = append(res, f(s))
	}
	return res
}

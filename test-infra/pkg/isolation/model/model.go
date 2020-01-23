package model

// Session object used in postgres testing.
// Each session is executed in its own connection. A session part consists
//  of three parts: setup, teardown and one or more "steps".
type Session struct {
	Name        string
	SetupSQL    string
	TeardownSQL string
	Steps       []Step
}

// Step will looks like:
// ```
// step "<name>" { <SQL> }
// ```
type Step struct {
	Name string
	SQL  string

	// The attributes to be used in test runtime

	Session  int
	Used     bool
	ErrorMsg string
}

type Permutation struct {
	NSteps    int
	StepNames []string
}

type TestSpec struct {
	SetupSQLs   []string
	TeardownSQL string
	Sessions    []Session

	Permutations []Permutation
	AllSteps     []Step
}

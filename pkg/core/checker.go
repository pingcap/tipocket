package core

// Checker checks a history of operations.
type Checker interface {
	// Check a series of operations with the given model.
	// Return false or error if operations do not satisfy the model.
	Check(m Model, ops []Operation) (bool, error)

	// Name returns the unique name for the checker.
	Name() string
}

// NoopChecker is a noop checker.
type NoopChecker struct{}

// Check impls Checker.
func (NoopChecker) Check(m Model, ops []Operation) (bool, error) {
	return true, nil
}

// Name impls Checker.
func (NoopChecker) Name() string {
	return "NoopChecker"
}

type multiChecker struct {
	checkers []Checker
	name     string
}

func (c multiChecker) Check(m Model, ops []Operation) (valid bool, err error) {
	for _, checker := range c.checkers {
		valid, err = checker.Check(m, ops)
		if !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

func (c multiChecker) Name() string {
	return c.name
}

// MultiChecker assembles multiple checkers
func MultiChecker(name string, checkers ...Checker) Checker {
	return multiChecker{
		checkers: checkers,
		name:     name,
	}
}

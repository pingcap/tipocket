package core

import (
	"encoding/json"
	"fmt"
)

// IntOptional is an optional int.
type IntOptional struct {
	value interface{}
}

// NewOptInt creates an optional.Int from a int.
func NewOptInt(v int) IntOptional {
	return IntOptional{
		value: v,
	}
}

// Set sets the int value.
func (i *IntOptional) Set(v int) {
	i.value = v
}

// MustGet gets values or panic if isn't present
func (i IntOptional) MustGet() int {
	return i.value.(int)
}

// GetOr gets the value of return the defv if not present
func (i IntOptional) GetOr(defv int) int {
	if i.Present() {
		return i.MustGet()
	}
	return defv
}

func (i IntOptional) String() string {
	if i.value == nil {
		return "nil"
	}
	return fmt.Sprintf("%d", i.value)
}

// Present returns whether or not the value is present.
func (i IntOptional) Present() bool {
	return i.value != nil
}

// MarshalJSON ...
func (i IntOptional) MarshalJSON() ([]byte, error) {
	if i.Present() {
		return json.Marshal(i.value)
	}
	return json.Marshal(nil)
}

// UnmarshalJSON ...
func (i *IntOptional) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		i.value = nil
		return nil
	}

	var value int

	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	i.value = value
	return nil
}

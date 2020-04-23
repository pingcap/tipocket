package core

import (
	"encoding/json"
)

// Int is an optional int.
type IntOptional struct {
	value interface{}
}

// NewInt creates an optional.Int from a int.
func NewOptInt(v int) IntOptional {
	return IntOptional{
		value: v,
	}
}

// Set sets the int value.
func (i *IntOptional) Set(v int) {
	i.value = v
}

func (i IntOptional) MustGet() int {
	return i.value.(int)
}

// Present returns whether or not the value is present.
func (i IntOptional) Present() bool {
	return i.value != nil
}

func (i IntOptional) MarshalJSON() ([]byte, error) {
	if i.Present() {
		return json.Marshal(i.value)
	}
	return json.Marshal(nil)
}

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

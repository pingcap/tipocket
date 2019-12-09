package executor

// Option struct
type Option struct {
	ID        int
	Clear     bool
	Log       string
	LogSuffix string
	Reproduce string
	Stable    bool
	Mute      bool
}

// Clone option
func (o *Option) Clone() *Option {
	o1 := *o
	return &o1
}

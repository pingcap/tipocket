package control

// Plugin defines a middleware
type Plugin interface {
	InitPlugin(control *Controller)
}

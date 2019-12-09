package builtin


type functionClass struct {
	name string
	minArg int
	maxArg int
	constArg bool
	mysql bool
	stable bool
}

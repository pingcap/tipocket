package builtin 


import "log"

var funcLists = [][]*functionClass{
	commonFunctions,
	mathFunctions,
	datetimeFunctions,
	stringFunctions,
	informationFunctions,
	controlFunctions,
	miscellaneousFunctions,
	encryptionFunctions,
	jsonFunctions,
	tidbFunctions,
}

func getFuncMap() map[string]*functionClass {
	funcs := make(map[string]*functionClass)
	for _, funcSet := range funcLists {
		for _, fn := range funcSet {
			if _, ok := funcs[fn.name]; ok {
				log.Fatalf("duplicated func name %s", fn.name)
			} else {
				funcs[fn.name] = fn
			}
		}
	}
	return funcs
}

package nemesis

import (
	"fmt"
	"math/rand"
)

type dynamicConfigType int

const (
	dynamicConfigTypeUnknown = iota
	dynamicConfigTypeEnum
	dynamicConfigTypeRandNum
	dynamicConfigTypeRandBool
)

type dynamicConfigItem struct {
	key       string
	template  string
	randTypes []dynamicConfigType
	randEmums [][]string
	randNums  [][2]int
}

func (d dynamicConfigItem) rand() string {
	var (
		randArgs  []interface{}
		enumIndex = 0
		numIndex  = 0
	)

	for _, rd := range d.randTypes {
		switch rd {
		case dynamicConfigTypeEnum:
			randArgs = append(randArgs, randFromEnum(d.randEmums[enumIndex]))
			enumIndex++
		case dynamicConfigTypeRandNum:
			randArgs = append(randArgs, randFromNum(d.randNums[numIndex]))
			numIndex++
		case dynamicConfigTypeRandBool:
			randArgs = append(randArgs, randBool())
		default:
			panic("unrechable")
		}
	}

	return fmt.Sprintf(d.template, randArgs...)
}

func randFromEnum(enums []string) string {
	if len(enums) == 0 {
		return ""
	}
	return enums[rand.Intn(len(enums))]
}

func randFromNum(num [2]int) int {
	if num[0] == num[1] {
		return num[0]
	}
	if num[0] > num[1] {
		num[0], num[1] = num[1], num[0]
	}

	return num[0] + rand.Intn(num[1]-num[0])
}

func randBool() bool {
	if rand.Intn(2) == 0 {
		return true
	}
	return false
}

var (
	tidbDynamicConfigItems = []dynamicConfigItem{}
	tikvDynamicConfigItems = []dynamicConfigItem{}
	pdDynamicConfigItems   = []dynamicConfigItem{
		{
			key:       "log.level",
			template:  "%s",
			randTypes: []dynamicConfigType{dynamicConfigTypeEnum},
			randEmums: [][]string{
				{"debug", "info", "warn", "error", "fatal"},
			},
		},
	}
)

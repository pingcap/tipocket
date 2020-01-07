// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package util


import (
	"fmt"
	"time"
	"strings"
	"github.com/pingcap/parser/mysql"
	tidbTypes "github.com/pingcap/tidb/types"
	"github.com/satori/go.uuid"
)

// GetUUID return uuid
func GetUUID() string {
	return strings.ToUpper(uuid.NewV4().String())
}

// GenerateRandDataItem rand data item with rand type
func GenerateRandDataItem() interface{} {
	switch Rd(6) {
	case 0:
		return GenerateDataItem("varchar")
	case 1:
		return GenerateDataItem("text")
	case 2:
		return GenerateDataItem("int")
	case 3:
		return GenerateDataItem("float")
	case 4:
		return GenerateDataItem("timestamp")
	case 5:
		return GenerateDataItem("datetime")
	}
	panic("unhandled switch")
}

// GenerateDataItemString rand data with given type
func GenerateDataItemString(columnType string) string {
	d := GenerateDataItem(columnType)
	switch c := d.(type) {
	case string:
		return c
	case int:
		return fmt.Sprintf("\"%d\"", c)
	case time.Time:
		return c.Format("2006-01-02 15:04:05")
	case tidbTypes.Time:
		return c.String()
	case float64:
		return fmt.Sprintf("%f", c)
	}
	return "not implement data transfer"
}

// GenerateDataItem rand data interface with given type
func GenerateDataItem(columnType string) interface{} {
	var res interface{}
	switch columnType {
	case "varchar":
		res = GenerateStringItem()
	case "text":
		res = GenerateStringItem()
	case "int":
		res = GenerateIntItem()
	case "timestamp", "datetime":
		res = GenerateTiDBDateItem()
	case "float":
		res = GenerateFloatItem()
	}
	return res
}

func GenerateStringItem() string {
	return strings.ToUpper(RdStringChar(Rd(100)))
}

func GenerateIntItem() int {
	return Rd(2147483647)
}

func GenerateFloatItem() float64 {
	return float64(Rd(100000)) * RdFloat64()
}

func GenerateDateItem() time.Time {
	t := RdDate()
	for ifDaylightTime(t) {
		t = RdDate()
	}
	return t
}

func GenerateTimestampItem() time.Time {
	t := RdTimestamp()
	for ifDaylightTime(t) {
		t = RdDate()
	}
	return t
}

func GenerateTiDBDateItem() tidbTypes.Time {
	return tidbTypes.Time{
		Time: tidbTypes.FromGoTime(GenerateDateItem()),
		Type: mysql.TypeDatetime,
	}
}

func ifDaylightTime(t time.Time) bool {
	if t.Year() < 1986 || t.Year() > 1991 {
		return false
	}
	if t.Month() < 4 || t.Month() > 9 {
		return false
	}
	return true
}

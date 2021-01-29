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

package builtin

import "github.com/pingcap/parser/ast"

var datetimeFunctions = []*functionClass{
	{ast.AddDate, 3, 3, false, true, false},
	{ast.DateAdd, 3, 3, false, true, false},
	{ast.SubDate, 3, 3, false, true, false},
	{ast.DateSub, 3, 3, false, true, false},
	{ast.AddTime, 2, 2, false, true, false},
	{ast.ConvertTz, 3, 3, false, true, false},
	// {ast.Curdate, 0, 0, false, true, false},
	// {ast.CurrentDate, 0, 0, false, true, false},
	// {ast.CurrentTime, 0, 1, false, true, false},
	// {ast.CurrentTimestamp, 0, 1, false, true, false},
	// {ast.Curtime, 0, 1, false, true, false},
	{ast.Date, 1, 1, false, true, false},
	{ast.DateLiteral, 1, 1, false, true, false},
	{ast.DateFormat, 2, 2, false, true, false},
	{ast.DateDiff, 2, 2, false, true, false},
	{ast.Day, 1, 1, false, true, false},
	{ast.DayName, 1, 1, false, true, false},
	{ast.DayOfMonth, 1, 1, false, true, false},
	{ast.DayOfWeek, 1, 1, false, true, false},
	{ast.DayOfYear, 1, 1, false, true, false},
	{ast.Extract, 2, 2, false, true, false},
	{ast.FromDays, 1, 1, false, true, false},
	{ast.FromUnixTime, 1, 2, false, true, false},
	{ast.GetFormat, 2, 2, false, true, false},
	{ast.Hour, 1, 1, false, true, false},
	// {ast.LocalTime, 0, 1, false, true, false},
	// {ast.LocalTimestamp, 0, 1, true, true, false},
	{ast.MakeDate, 2, 2, false, true, false},
	{ast.MakeTime, 3, 3, false, true, false},
	{ast.MicroSecond, 1, 1, false, true, false},
	{ast.Minute, 1, 1, false, true, false},
	{ast.Month, 1, 1, false, true, false},
	{ast.MonthName, 1, 1, false, true, false},
	// {ast.Now, 0, 1, false, true, false},
	{ast.PeriodAdd, 2, 2, false, true, false},
	// {ast.PeriodDiff, 2, 2, false, true, false},
	{ast.Quarter, 1, 1, false, true, false},
	{ast.SecToTime, 1, 1, false, true, false},
	{ast.Second, 1, 1, false, true, false},
	{ast.StrToDate, 2, 2, false, true, false},
	{ast.SubTime, 2, 2, false, true, false},
	// will make diff
	// {ast.Sysdate, 0, 1, false, true, false},
	{ast.Time, 1, 1, false, true, false},
	{ast.TimeLiteral, 1, 1, false, true, false},
	{ast.TimeFormat, 2, 2, false, true, false},
	{ast.TimeToSec, 1, 1, false, true, false},
	{ast.TimeDiff, 2, 2, false, true, false},
	{ast.Timestamp, 1, 2, false, true, false},
	{ast.TimestampLiteral, 1, 2, false, true, false},
	{ast.TimestampAdd, 3, 3, false, true, false},
	{ast.TimestampDiff, 3, 3, false, true, false},
	{ast.ToDays, 1, 1, false, true, false},
	{ast.ToSeconds, 1, 1, false, true, false},
	// {ast.UnixTimestamp, 0, 1, false, true, false},
	// {ast.UTCDate, 0, 0, false, true, false},
	// {ast.UTCTime, 0, 1, false, true, false},
	// {ast.UTCTimestamp, 0, 1, false, true, false},
	{ast.Week, 1, 2, false, true, false},
	{ast.Weekday, 1, 1, false, true, false},
	{ast.WeekOfYear, 1, 1, false, true, false},
	{ast.Year, 1, 1, false, true, false},
	{ast.YearWeek, 1, 2, false, true, false},
	{ast.LastDay, 1, 1, false, true, false},
}

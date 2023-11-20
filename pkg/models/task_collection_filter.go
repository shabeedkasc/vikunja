// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public Licensee as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public Licensee for more details.
//
// You should have received a copy of the GNU Affero General Public Licensee
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"code.vikunja.io/api/pkg/config"

	"github.com/iancoleman/strcase"
	"github.com/jszwedko/go-datemath"
	"xorm.io/xorm/schemas"
)

type taskFilterComparator string

const (
	taskFilterComparatorInvalid taskFilterComparator = "invalid"

	taskFilterComparatorEquals       taskFilterComparator = "="
	taskFilterComparatorGreater      taskFilterComparator = ">"
	taskFilterComparatorGreateEquals taskFilterComparator = ">="
	taskFilterComparatorLess         taskFilterComparator = "<"
	taskFilterComparatorLessEquals   taskFilterComparator = "<="
	taskFilterComparatorNotEquals    taskFilterComparator = "!="
	taskFilterComparatorLike         taskFilterComparator = "like"
	taskFilterComparatorIn           taskFilterComparator = "in"
)

// Guess what you get back if you ask Safari for a rfc 3339 formatted date?
const safariDateAndTime = "2006-01-02 15:04"
const safariDate = "2006-01-02"

type taskFilter struct {
	field      string
	value      interface{} // Needs to be an interface to be able to hold the field's native value
	comparator taskFilterComparator
	isNumeric  bool
}

func parseTimeFromUserInput(timeString string) (value time.Time, err error) {
	value, err = time.Parse(time.RFC3339, timeString)
	if err != nil {
		value, err = time.Parse(safariDateAndTime, timeString)
	}
	if err != nil {
		value, err = time.Parse(safariDate, timeString)
	}
	if err != nil {
		// Here we assume a date like 2022-11-1 and try to parse it manually
		parts := strings.Split(timeString, "-")
		if len(parts) < 3 {
			return
		}
		year, err := strconv.Atoi(parts[0])
		if err != nil {
			return value, err
		}
		month, err := strconv.Atoi(parts[1])
		if err != nil {
			return value, err
		}
		day, err := strconv.Atoi(parts[2])
		if err != nil {
			return value, err
		}
		value = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		return value.In(config.GetTimeZone()), nil
	}
	return value.In(config.GetTimeZone()), err
}

func getTaskFiltersByCollections(c *TaskCollection) (filters []*taskFilter, err error) {

	if len(c.FilterByArr) > 0 {
		c.FilterBy = append(c.FilterBy, c.FilterByArr...)
	}

	if len(c.FilterValueArr) > 0 {
		c.FilterValue = append(c.FilterValue, c.FilterValueArr...)
	}

	if len(c.FilterComparatorArr) > 0 {
		c.FilterComparator = append(c.FilterComparator, c.FilterComparatorArr...)
	}

	if c.FilterConcat != "" && c.FilterConcat != filterConcatAnd && c.FilterConcat != filterConcatOr {
		return nil, ErrInvalidTaskFilterConcatinator{
			Concatinator: taskFilterConcatinator(c.FilterConcat),
		}
	}

	filters = make([]*taskFilter, 0, len(c.FilterBy))
	for i, f := range c.FilterBy {
		filter := &taskFilter{
			field:      f,
			comparator: taskFilterComparatorEquals,
		}

		if len(c.FilterComparator) > i {
			filter.comparator, err = getFilterComparatorFromString(c.FilterComparator[i])
			if err != nil {
				return
			}
		}

		err = validateTaskFieldComparator(filter.comparator)
		if err != nil {
			return
		}

		// Cast the field value to its native type
		var reflectValue *reflect.StructField
		if len(c.FilterValue) > i {
			reflectValue, filter.value, err = getNativeValueForTaskField(filter.field, filter.comparator, c.FilterValue[i])
			if err != nil {
				return nil, ErrInvalidTaskFilterValue{
					Value: filter.field,
					Field: c.FilterValue[i],
				}
			}
		}
		if reflectValue != nil {
			filter.isNumeric = reflectValue.Type.Kind() == reflect.Int64
		}

		filters = append(filters, filter)
	}

	return
}

func validateTaskFieldComparator(comparator taskFilterComparator) error {
	switch comparator {
	case
		taskFilterComparatorEquals,
		taskFilterComparatorGreater,
		taskFilterComparatorGreateEquals,
		taskFilterComparatorLess,
		taskFilterComparatorLessEquals,
		taskFilterComparatorNotEquals,
		taskFilterComparatorLike,
		taskFilterComparatorIn:
		return nil
	case taskFilterComparatorInvalid:
		fallthrough
	default:
		return ErrInvalidTaskFilterComparator{Comparator: comparator}
	}
}

func getFilterComparatorFromString(comparator string) (taskFilterComparator, error) {
	switch comparator {
	case "equals":
		return taskFilterComparatorEquals, nil
	case "greater":
		return taskFilterComparatorGreater, nil
	case "greater_equals":
		return taskFilterComparatorGreateEquals, nil
	case "less":
		return taskFilterComparatorLess, nil
	case "less_equals":
		return taskFilterComparatorLessEquals, nil
	case "not_equals":
		return taskFilterComparatorNotEquals, nil
	case "like":
		return taskFilterComparatorLike, nil
	case "in":
		return taskFilterComparatorIn, nil
	default:
		return taskFilterComparatorInvalid, ErrInvalidTaskFilterComparator{Comparator: taskFilterComparator(comparator)}
	}
}

func getValueForField(field reflect.StructField, rawValue string) (value interface{}, err error) {
	switch field.Type.Kind() {
	case reflect.Int64:
		value, err = strconv.ParseInt(rawValue, 10, 64)
	case reflect.Float64:
		value, err = strconv.ParseFloat(rawValue, 64)
	case reflect.String:
		value = rawValue
	case reflect.Bool:
		value, err = strconv.ParseBool(rawValue)
	case reflect.Struct:
		if field.Type == schemas.TimeType {
			var t datemath.Expression
			t, err = datemath.Parse(rawValue)
			if err == nil {
				value = t.Time(datemath.WithLocation(config.GetTimeZone()))
			} else {
				value, err = parseTimeFromUserInput(rawValue)
			}
		}
	case reflect.Slice:
		// If this is a slice of pointers we're dealing with some property which is a relation
		// In that case we don't really care about what the actual type is, we just cast the value to an
		// int64 since we need the id - yes, this assumes we only ever have int64 IDs, but this is fine.
		if field.Type.Elem().Kind() == reflect.Ptr {
			value, err = strconv.ParseInt(rawValue, 10, 64)
			return
		}

		// There are probably better ways to do this - please let me know if you have one.
		if field.Type.Elem().String() == "time.Time" {
			value, err = time.Parse(time.RFC3339, rawValue)
			value = value.(time.Time).In(config.GetTimeZone())
			return
		}
		fallthrough
	default:
		panic(fmt.Errorf("unrecognized filter type %s for field %s, value %s", field.Type.String(), field.Name, value))
	}

	return
}

func getNativeValueForTaskField(fieldName string, comparator taskFilterComparator, value string) (reflectField *reflect.StructField, nativeValue interface{}, err error) {

	realFieldName := strings.ReplaceAll(strcase.ToCamel(fieldName), "Id", "ID")

	if realFieldName == "Assignees" {
		vals := strings.Split(value, ",")
		valueSlice := append([]string{}, vals...)
		return nil, valueSlice, nil
	}

	field, ok := reflect.TypeOf(&Task{}).Elem().FieldByName(realFieldName)
	if !ok {
		return nil, nil, ErrInvalidTaskField{TaskField: fieldName}
	}

	if realFieldName == "Reminders" {
		field, ok = reflect.TypeOf(&TaskReminder{}).Elem().FieldByName("Reminder")
		if !ok {
			return nil, nil, ErrInvalidTaskField{TaskField: fieldName}
		}
	}

	if comparator == taskFilterComparatorIn {
		vals := strings.Split(value, ",")
		valueSlice := []interface{}{}
		for _, val := range vals {
			v, err := getValueForField(field, val)
			if err != nil {
				return nil, nil, err
			}
			valueSlice = append(valueSlice, v)
		}
		return nil, valueSlice, nil
	}

	val, err := getValueForField(field, value)
	return &field, val, err
}

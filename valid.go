// Package govalid
// Author: Perry He
// Created on: 2025-11-27 08:34:55
package govalid

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const tagKey = "binding"

func Valid(s any, path ...string) error {
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)

	// if s is a pointer, unwrap(拆开、解开、取出里面的东西) it
	// reflection can't work with fields on a pointer directly(直接)
	// so we need call .Elem() go get the actual(真实、实际) struct behind(在……之后，背面) it
	if t.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}

		// https://pkg.go.dev/reflect#Value.Elem
		// .Elem() 的作用就是把 “指针类型” 解引用为 “指向的结构体类型”
		v = v.Elem() // get the value the pointer points to
		t = t.Elem() // get the type the pointer points to
	}
	if t.Kind() != reflect.Struct {
		return errors.New("only support sturcts types")
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// skip unexported fields (those starting with lowercase)
		if field.PkgPath != "" {
			continue
		}

		// build the full path for this field, including any nested structs
		fullPath := strings.Join(append(path, field.Name), ".")

		value := v.Field(i)

		switch value.Kind() {
		case reflect.Struct:
			// if this field is a struct, validate it recursively(递归地)
			if value.Type() == reflect.TypeOf(time.Time{}) {
				// not handle yet.
			} else {
				if err := Valid(value.Interface(), fullPath); err != nil {
					// attach the field name to the error
					return fmt.Errorf("%s.%w", field.Name, err)
				}
				continue
			}

		case reflect.Pointer:
			// if it's a pointer to a struct, unwrap it and validate recursively
			if !value.IsNil() && value.Elem().Kind() == reflect.Struct {
				if err := Valid(value.Elem().Interface(), fullPath); err != nil {
					// attach the field name to the error
					return fmt.Errorf("%s.%w", field.Name, err)
				}
				continue
			}
		}

		tag := field.Tag.Get(tagKey)
		if tag == "" {
			continue
		}
		// only look at this tag, ignore json/form tags
		hasOmitempty := strings.Contains(tag, "omitempty")

		// parse rules from tag
		rules := strings.SplitSeq(tag, ",")
		for rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "required" {
				// check if field is required
				if isEmpty(value) && hasOmitempty {
					continue
				}
				if isEmpty(value) {
					return fmt.Errorf("field %s is required", field.Name)
				}
				continue
			}
			if strings.Contains(rule, "=") {
				parts := strings.SplitN(rule, "=", 2)
				if len(parts) != 2 {
					continue
				}
				// Here's the issue:
				// For example, with `ID int 'binding:"min=1,max=100,omitempty'"`
				// If the field is empty and `omitempty` is set, it will skip all checks
				if isEmpty(value) && hasOmitempty {
					continue
				}
				// compare(比较) value against(针对、与……对比) rule parameter
				op := parts[0]
				param := parts[1]
				if err := compareValue(value, op, param, fullPath); err != nil {
					return err
				}
			}
		}

	}
	return nil
}

// isEmpty checks if a reflect.Value is empty or zero
func isEmpty(v reflect.Value) bool {
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	case reflect.Struct:
		// time.Time
		if v.Type() == reflect.TypeOf(time.Time{}) {
			return v.Interface().(time.Time).IsZero()
		}
	}
	return false
}

// compareValue compares a value against a rule and parameter
func compareValue(v reflect.Value, op, param, fieldName string) error {
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			v = reflect.Zero(v.Type().Elem())
		} else {
			v = v.Elem()
		}
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val := v.Int()
		num, err := strconv.ParseInt(param, 10, 64)
		if err != nil {
			return fmt.Errorf("%s has an invalid parameter for %s", fieldName, op)
		}

		switch op {
		case "gt":
			if val <= num {
				return fmt.Errorf("%s must be > %d", fieldName, num)
			}
		case "gte":
			if val < num {
				return fmt.Errorf("%s must be >= %d", fieldName, num)
			}
		case "lt":
			if val >= num {
				return fmt.Errorf("%s must be < %d", fieldName, num)
			}
		case "lte":
			if val > num {
				return fmt.Errorf("%s must be <= %d", fieldName, num)
			}
		case "min":
			if val < num {
				return fmt.Errorf("%s must be at least %d", fieldName, num)
			}
		case "max":
			if val > num {
				return fmt.Errorf("%s must be at most %d", fieldName, num)
			}
		}

	case reflect.Slice, reflect.Array, reflect.String, reflect.Map:
		length := v.Len()
		num, err := strconv.Atoi(param)
		if err != nil {
			return fmt.Errorf("%s has an invalid parameter for %s", fieldName, op)
		}

		switch op {
		case "min":
			if length < num {
				return fmt.Errorf("%s needs at least %d items/characters", fieldName, num)
			}
		case "max":
			if length > num {
				return fmt.Errorf("%s can have at most %d items/characters", fieldName, num)
			}
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			// convert time to the local time zone
			// use RFC3339 format
			t, err := time.Parse(time.RFC3339, param)
			if err != nil {
				return fmt.Errorf("%s has an invalid date format for %s", fieldName, op)
			}
			fieldTime := v.Interface().(time.Time)

			// convert both fieldTime and param time to local timezone
			fieldTime = fieldTime.In(time.Local)
			t = t.In(time.Local)

			switch op {
			case "min":
				if fieldTime.Before(t) {
					return fmt.Errorf("%s must be after %s", fieldName, t.Format(time.RFC3339))
				}
			case "max":
				if fieldTime.After(t) {
					return fmt.Errorf("%s must be before %s", fieldName, t.Format(time.RFC3339))
				}
			case "gte":
				if fieldTime.Before(t) {
					return fmt.Errorf("%s must be at or after %s", fieldName, t.Format(time.RFC3339))
				}
			case "lte":
				if fieldTime.After(t) {
					return fmt.Errorf("%s must be at or before %s", fieldName, t.Format(time.RFC3339))
				}
			}
		}

	default:
		return fmt.Errorf("%s: type not supported for validation", fieldName)
	}

	return nil
}

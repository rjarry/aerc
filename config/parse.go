package config

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"

	"git.sr.ht/~rjarry/aerc/lib/templates"
	"github.com/emersion/go-message/mail"
	"github.com/go-ini/ini"
)

func MapToStruct(s *ini.Section, v interface{}, useDefaults bool) error {
	typ := reflect.TypeOf(v)
	val := reflect.ValueOf(v)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	} else {
		panic("MapToStruct requires a pointer")
	}
	if typ.Kind() != reflect.Struct {
		panic("MapToStruct requires a pointer to a struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldType := typ.Field(i)

		name := fieldType.Tag.Get("ini")
		if name == "" || name == "-" {
			continue
		}
		key, err := s.GetKey(name)
		if err != nil {
			defValue, found := fieldType.Tag.Lookup("default")
			if useDefaults && found {
				key, _ = s.NewKey(name, defValue)
			} else {
				continue
			}
		}
		err = setField(s, key, reflect.ValueOf(v), fieldVal, fieldType)
		if err != nil {
			return fmt.Errorf("[%s].%s: %w", s.Name(), name, err)
		}
	}
	return nil
}

func setField(
	s *ini.Section, key *ini.Key, struc reflect.Value,
	fieldVal reflect.Value, fieldType reflect.StructField,
) error {
	var methodValue reflect.Value
	method := getParseMethod(s, key, struc, fieldType)
	if method.IsValid() {
		in := []reflect.Value{reflect.ValueOf(s), reflect.ValueOf(key)}
		out := method.Call(in)
		err, _ := out[1].Interface().(error)
		if err != nil {
			return err
		}
		methodValue = out[0]
	}

	ft := fieldType.Type

	switch ft.Kind() {
	case reflect.String:
		if method.IsValid() {
			fieldVal.SetString(methodValue.String())
		} else {
			fieldVal.SetString(key.String())
		}
	case reflect.Bool:
		if method.IsValid() {
			fieldVal.SetBool(methodValue.Bool())
		} else {
			boolVal, err := key.Bool()
			if err != nil {
				return err
			}
			fieldVal.SetBool(boolVal)
		}
	case reflect.Int32:
		// impossible to differentiate rune from int32, they are aliases
		// this is an ugly hack but there is no alternative...
		if fieldType.Tag.Get("type") == "rune" {
			if method.IsValid() {
				fieldVal.Set(methodValue)
			} else {
				runes := []rune(key.String())
				if len(runes) != 1 {
					return errors.New("value must be 1 character long")
				}
				fieldVal.Set(reflect.ValueOf(runes[0]))
			}
			return nil
		}
		fallthrough
	case reflect.Int64:
		// ParseDuration will not return err for `0`, so check the type name
		if ft.PkgPath() == "time" && ft.Name() == "Duration" {
			durationVal, err := key.Duration()
			if err != nil {
				return err
			}
			fieldVal.Set(reflect.ValueOf(durationVal))
			return nil
		}
		fallthrough
	case reflect.Int, reflect.Int8, reflect.Int16:
		if method.IsValid() {
			fieldVal.SetInt(methodValue.Int())
		} else {
			intVal, err := key.Int64()
			if err != nil {
				return err
			}
			fieldVal.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if method.IsValid() {
			fieldVal.SetUint(methodValue.Uint())
		} else {
			uintVal, err := key.Uint64()
			if err != nil {
				return err
			}
			fieldVal.SetUint(uintVal)
		}
	case reflect.Float32, reflect.Float64:
		if method.IsValid() {
			fieldVal.SetFloat(methodValue.Float())
		} else {
			floatVal, err := key.Float64()
			if err != nil {
				return err
			}
			fieldVal.SetFloat(floatVal)
		}
	case reflect.Slice, reflect.Array:
		switch {
		case method.IsValid():
			fieldVal.Set(methodValue)
		case ft.Elem().Kind() == reflect.Ptr &&
			typePath(ft.Elem().Elem()) == "net/mail.Address":
			addrs, err := mail.ParseAddressList(key.String())
			if err != nil {
				return err
			}
			fieldVal.Set(reflect.ValueOf(addrs))
		case ft.Elem().Kind() == reflect.String:
			delim := fieldType.Tag.Get("delim")
			fieldVal.Set(reflect.ValueOf(key.Strings(delim)))
		default:
			panic(fmt.Sprintf("unsupported type []%s", typePath(ft.Elem())))
		}
	case reflect.Struct:
		if method.IsValid() {
			fieldVal.Set(methodValue)
		} else {
			panic(fmt.Sprintf("unsupported type %s", typePath(ft)))
		}
	case reflect.Ptr:
		if method.IsValid() {
			fieldVal.Set(methodValue)
		} else {
			switch typePath(ft.Elem()) {
			case "net/mail.Address":
				addr, err := mail.ParseAddress(key.String())
				if err != nil {
					return err
				}
				fieldVal.Set(reflect.ValueOf(addr))
			case "regexp.Regexp":
				r, err := regexp.Compile(key.String())
				if err != nil {
					return err
				}
				fieldVal.Set(reflect.ValueOf(r))
			case "text/template.Template":
				t, err := templates.ParseTemplate(key.String(), key.String())
				if err != nil {
					return err
				}
				fieldVal.Set(reflect.ValueOf(t))
			default:
				panic(fmt.Sprintf("unsupported type %s", typePath(ft)))
			}
		}
	default:
		panic(fmt.Sprintf("unsupported type %s", typePath(ft)))
	}
	return nil
}

func getParseMethod(
	section *ini.Section, key *ini.Key,
	struc reflect.Value, typ reflect.StructField,
) reflect.Value {
	methodName, found := typ.Tag.Lookup("parse")
	if !found {
		return reflect.Value{}
	}
	method := struc.MethodByName(methodName)
	if !method.IsValid() {
		panic(fmt.Sprintf("(*%s).%s: method not found",
			struc, methodName))
	}

	if method.Type().NumIn() != 2 ||
		method.Type().In(0) != reflect.TypeOf(section) ||
		method.Type().In(1) != reflect.TypeOf(key) ||
		method.Type().NumOut() != 2 {
		panic(fmt.Sprintf("(*%s).%s: invalid signature, expected %s",
			struc.Elem().Type().Name(), methodName,
			"func(*ini.Section, *ini.Key) (any, error)"))
	}

	return method
}

func typePath(t reflect.Type) string {
	var prefix string
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		prefix = "*"
	}
	return fmt.Sprintf("%s%s.%s", prefix, t.PkgPath(), t.Name())
}

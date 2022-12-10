package glue

import (
	"errors"
	"github.com/mitchellh/mapstructure"
	lua "github.com/yuin/gopher-lua"
	"reflect"
)

const (
	// OptionsLenient is whether excess table fields should be ignored.
	OptionsLenient Options = 1 << iota
)

// Options are the options that can be passed to a mapper.
type Options int

// ErrTableExpected is an error about a mismatching LTable kind.
var ErrTableExpected = errors.New("expected LTable to be a table, got an array")

// ErrCannotConvert is an error about an inconvertible type.
var ErrCannotConvert = errors.New("could not convert type")

type Mapper struct {
	Options Options
}

// NewMapper creates a new mapper from options.
func NewMapper(options Options) *Mapper {
	return &Mapper{Options: options}
}

// Decode maps the Lua table to the given struct pointer.
func (m *Mapper) Decode(tbl *lua.LTable, st interface{}) error {
	lValue, err := m.AsGoValue(tbl)
	if err != nil {
		return err
	}
	mp, ok := lValue.(map[interface{}]interface{})
	if !ok {
		return ErrTableExpected
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           st,
		TagName:          "glue",
		ErrorUnused:      m.Options&OptionsLenient == 0,
	})
	if err != nil {
		return err
	}

	return decoder.Decode(mp)
}

// Encode maps the given struct to the given Lua table.
func (m *Mapper) Encode(st interface{}, tbl *lua.LTable) error {
	if tbl == nil {
		*tbl = lua.LTable{}
	}
	if tbl.MaxN() != 0 {
		return ErrTableExpected
	}

	val, err := m.FromGoValue(st)
	if err != nil {
		return err
	}
	val.(*lua.LTable).ForEach(func(key lua.LValue, value lua.LValue) {
		tbl.RawSet(key, value)
	})

	return nil
}

// AsGoValue converts the given lua.LValue to a Go object.
func (m *Mapper) AsGoValue(lv lua.LValue) (interface{}, error) {
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil, nil
	case lua.LBool:
		return bool(v), nil
	case lua.LString:
		return string(v), nil
	case lua.LNumber:
		return float64(v), nil
	case *lua.LTable:
		maxN := v.MaxN()
		if maxN == 0 { // table
			ret := make(map[interface{}]interface{})

			var err error
			v.ForEach(func(key, value lua.LValue) {
				if err != nil { // skip the rest if error is set
					return
				}

				lKey, err := m.AsGoValue(key)
				if err != nil {
					return
				}
				lValue, err := m.AsGoValue(value)
				if err != nil {
					return
				}
				ret[lKey] = lValue
			})
			return ret, err
		} else { // array
			ret := make([]interface{}, 0, maxN)
			for i := 1; i <= maxN; i++ {
				lItem, err := m.AsGoValue(v.RawGetInt(i))
				if err != nil {
					return nil, err
				}

				ret = append(ret, lItem)
			}
			return ret, nil
		}
	default:
		return nil, ErrCannotConvert
	}
}

// FromGoValue converts the given Go object to a LValue.
func (m *Mapper) FromGoValue(v interface{}) (lua.LValue, error) {
	switch t := v.(type) {
	case nil:
		return lua.LNil, nil
	case bool:
		return lua.LBool(t), nil
	case string:
		return lua.LString(t), nil
	case int:
		return lua.LNumber(t), nil
	case float64:
		return lua.LNumber(t), nil
	case map[interface{}]interface{}:
		table := &lua.LTable{}

		for key, value := range t {
			lKey, err := m.FromGoValue(key)
			if err != nil {
				return nil, err
			}
			lValue, err := m.FromGoValue(value)
			if err != nil {
				return nil, err
			}
			table.RawSet(lKey, lValue)
		}
		return table, nil
	default:
		kind := reflect.TypeOf(v).Kind()
		switch kind {
		case reflect.Struct:
			var mp map[interface{}]interface{}
			decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				WeaklyTypedInput: true,
				Result:           &mp,
				TagName:          "glue",
				ErrorUnused:      m.Options&OptionsLenient == 0,
			})
			if err != nil {
				return nil, err
			}

			if err := decoder.Decode(v); err != nil {
				return nil, err
			}
			return m.FromGoValue(mp)
		case reflect.Pointer:
			return m.FromGoValue(reflect.ValueOf(v).Elem().Interface())
		case reflect.Slice, reflect.Array:
			table := &lua.LTable{}

			value := reflect.ValueOf(v)
			for i := 0; i < value.Len(); i++ {
				lItem, err := m.FromGoValue(value.Index(i).Interface())
				if err != nil {
					return nil, err
				}
				table.Append(lItem)
			}
			return table, nil
		}
		return nil, ErrCannotConvert
	}
}

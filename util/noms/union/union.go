package union

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/aboodman/replicant/util/chk"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
)

func Marshal(st interface{}, noms types.ValueReadWriter) (types.Value, error) {
	t := reflect.TypeOf(st)
	v := reflect.ValueOf(st)
	chk.Equal(reflect.Struct, v.Kind())
	var r types.Value
	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i).Interface()
		chk.Equal(reflect.Struct, v.Kind())
		if reflect.DeepEqual(reflect.Zero(t.Field(i).Type).Interface(), fv) {
			continue
		}
		nfv, err := marshal.Marshal(noms, fv)
		if err != nil {
			return nil, fmt.Errorf("Could not marshal field %s: %v", t.Field(i).PkgPath, err)
		}
		if r != nil {
			return nil, errors.New("At most one field of a union may be set")
		}
		r = nfv
	}
	return r, nil
}

func Unmarshal(in types.Value, out interface{}) error {
	if in.Kind() != types.StructKind {
		return errors.New("Can only unmarshal Noms structs into unions")
	}
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr {
		return errors.New("Can only unmarshal unions into pointer to struct")
	}
	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return errors.New("Can only unmarshal unions into pointer to struct")
	}
	t := v.Type()
	fn := in.(types.Struct).Name()
	_, ok := t.FieldByName(fn)
	if !ok {
		return fmt.Errorf("Could not get field: %s", fn)
	}
	err := marshal.Unmarshal(in, v.FieldByName(fn).Addr().Interface())
	if err != nil {
		return fmt.Errorf("Cannot unmarshal noms value onto field %s: %v", fn, err)
	}
	return nil
}

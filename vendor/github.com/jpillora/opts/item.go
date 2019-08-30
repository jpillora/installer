package opts

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"time"
)

//item group represents a single "Options" block
//in the help text ouput
type itemGroup struct {
	name  string
	flags []*item
}

const defaultGroup = ""

//item is the structure representing a
//an opt item. it also implements flag.Value
//generically using reflect.
type item struct {
	val       reflect.Value
	mode      string
	name      string
	shortName string
	envName   string
	useEnv    bool
	help      string
	defstr    string
	slice     bool
	min, max  int //valid if slice
	noarg     bool
	completer Completer
	sets      int
}

func newItem(val reflect.Value) (*item, error) {
	if !val.IsValid() {
		return nil, fmt.Errorf("invalid value")
	}
	i := &item{}
	supported := false
	//take interface value V
	v := val.Interface()
	pv := interface{}(nil)
	if val.CanAddr() {
		pv = val.Addr().Interface()
	}
	//convert V or &V into a setter:
	for _, t := range []interface{}{v, pv} {
		if tm, ok := t.(encoding.TextUnmarshaler); ok {
			v = &textValue{tm}
		} else if bm, ok := t.(encoding.BinaryUnmarshaler); ok {
			v = &binaryValue{bm}
		} else if d, ok := t.(*time.Duration); ok {
			v = newDurationValue(d)
		} else if s, ok := t.(Setter); ok {
			v = s
		}
	}
	//implements setter (flag.Value)?
	if s, ok := v.(Setter); ok {
		supported = true
		//NOTE: replacing val removes our ability to set
		//the value, resolved by flag.Value handling all Set calls.
		val = reflect.ValueOf(s)
	}
	//implements completer?
	if c, ok := v.(Completer); ok {
		i.completer = c
	}
	//val must be concrete at this point
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	//lock in val
	i.val = val
	i.slice = val.Kind() == reflect.Slice
	//prevent defaults on slices (should vals be appended? should it be reset? how to display defaults?)
	if i.slice && val.Len() > 0 {
		return nil, fmt.Errorf("slices cannot have default values")
	}
	//type checks
	t := i.elemType()
	if t.Kind() == reflect.Ptr {
		return nil, fmt.Errorf("slice elem (%s) cannot be a pointer", t.Kind())
	} else if i.slice && t.Kind() == reflect.Bool {
		return nil, fmt.Errorf("slice of bools not supported")
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String, reflect.Bool:
		supported = true
	}
	//use the inner bool flag, if defined, otherwise if bool
	if bf, ok := v.(interface{ IsBoolFlag() bool }); ok {
		i.noarg = bf.IsBoolFlag()
	} else if t.Kind() == reflect.Bool {
		i.noarg = true
	}
	if !supported {
		return nil, fmt.Errorf("field type not supported: %s", t.Kind())
	}
	return i, nil
}

func (i *item) set() bool {
	return i.sets != 0
}

func (i *item) elemType() reflect.Type {
	t := i.val.Type()
	if i.slice {
		t = t.Elem()
	}
	return t
}

func (i *item) String() string {
	if !i.val.IsValid() {
		return ""
	}
	v := i.val.Interface()
	if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%v", v)
}

func (i *item) Set(s string) error {
	//can only set singles once
	if i.sets != 0 && !i.slice {
		return errors.New("already set")
	}
	//set has two modes, slice and inplace.
	// when slice, create a new zero value, scan into it, append to slice
	// when inplace, take pointer, scan into it
	var elem reflect.Value
	if i.slice {
		elem = reflect.New(i.elemType()) //ptr
	} else if i.val.CanAddr() {
		elem = i.val.Addr() //pointer to concrete type
	} else {
		elem = i.val //possibly interface type
	}
	v := elem.Interface()
	//convert string into value
	if fv, ok := v.(Setter); ok {
		//addr implements set
		if err := fv.Set(s); err != nil {
			return err
		}
	} else if elem.Kind() == reflect.Ptr {
		//magic set with scanf
		n, err := fmt.Sscanf(s, "%v", v)
		if err != nil {
			return err
		} else if n == 0 {
			return errors.New("could not be parsed")
		}
	} else {
		return errors.New("could not be set")
	}
	//slice? append!
	if i.slice {
		//no pointer elems
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		//append!
		i.val.Set(reflect.Append(i.val, elem))
	}
	//mark item as set!
	i.sets++
	//done
	return nil
}

//IsBoolFlag implements the hidden interface
//documented here https://golang.org/pkg/flag/#Value
func (i *item) IsBoolFlag() bool {
	return i.noarg
}

//noopValue defines a flag value which does nothing
var noopValue = noopValueType(0)

type noopValueType int

func (noopValueType) String() string {
	return ""
}

func (noopValueType) Set(s string) error {
	return nil
}

//textValue wraps marshaller into a setter
type textValue struct {
	encoding.TextUnmarshaler
}

func (t textValue) Set(s string) error {
	return t.UnmarshalText([]byte(s))
}

//binaryValue wraps marshaller into a setter
type binaryValue struct {
	encoding.BinaryUnmarshaler
}

func (t binaryValue) Set(s string) error {
	return t.UnmarshalBinary([]byte(s))
}

//borrowed from the stdlib :)
type durationValue time.Duration

func newDurationValue(p *time.Duration) *durationValue {
	return (*durationValue)(p)
}

func (d *durationValue) Set(s string) error {
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = durationValue(v)
	return nil
}

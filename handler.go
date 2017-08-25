package xing

import "reflect"

func Methods(v interface{}) (map[string]reflect.Value,
	map[string]reflect.Type, map[string]reflect.Type) {
	calls := make(map[string]reflect.Value)
	in := make(map[string]reflect.Type)
	out := make(map[string]reflect.Type)

	fooType := reflect.TypeOf(v)
	// val := reflect.Indirect(reflect.ValueOf(v))
	// tmp := strings.Split(val.Type().String(), ".")
	// typeName := tmp[len(tmp)-1]
	for i := 0; i < fooType.NumMethod(); i++ {
		method := fooType.Method(i)
		// We're assuming that RPC is like this
		// Hello(in) (out, err)
		calls[method.Name] = method.Func
		// method.Type holds the prototype of the method, which is a function
		itype := method.Type.In(2).Elem() // get the actual type, not the poionter
		otype := method.Type.In(3).Elem()
		in[method.Name] = itype
		out[method.Name] = otype
	}

	return calls, in, out
}

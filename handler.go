package xing

import "reflect"

// Methods ...
func Methods(service string, v interface{}) (map[string]reflect.Value, map[string]reflect.Type, map[string]reflect.Type) {
	name := func(n string) string {
		return fullName(service, n)
	}
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
		calls[name(method.Name)] = method.Func
		// method.Type holds the prototype of the method, which is a function
		itype := method.Type.In(2).Elem() // get the actual type, not the poionter
		otype := method.Type.In(3).Elem()
		in[name(method.Name)] = itype
		out[name(method.Name)] = otype
	}

	return calls, in, out
}

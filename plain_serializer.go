package xing

type PlainSerilizer struct {
	name string
}

func (s *PlainSerilizer) ContentType() string {
	return "text/plain"
}

func (s *PlainSerilizer) Marshal(data interface{}) ([]byte, error) {
	str := data.(string) // FIMXE: check type instead of assertion
	return []byte(str), nil
}

func (s *PlainSerilizer) Unmarshal(data []byte, output *interface{}) error {
	*output = string(data)
	return nil
}

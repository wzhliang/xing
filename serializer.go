package xing

// Serializer ...
type Serializer interface {
	ContentType() string
	Marshal(event string, data interface{}) ([]byte, error)
	Unmarshal(event string, data []byte) (interface{}, error)
	DefaultValue() interface{}
}

// PlainSerializer ...
type PlainSerializer struct {
	name string
}

// ContentType ...
func (s *PlainSerializer) ContentType() string {
	return "text/plain"
}

// Marshal ...
func (s *PlainSerializer) Marshal(event string, data interface{}) ([]byte, error) {
	str := data.(string) // FIMXE: check type instead of assertion
	return []byte(str), nil
}

// Unmarshal ...
func (s *PlainSerializer) Unmarshal(event string, data []byte) (interface{}, error) {
	return string(data), nil
}

// DefaultValue ...
func (s *PlainSerializer) DefaultValue() interface{} {
	return ""
}

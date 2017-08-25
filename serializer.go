package xing

import (
	"github.com/micro/protobuf/proto"

	"encoding/json"
)

// Serializer ...
type Serializer interface {
	ContentType() string
	Marshal(data interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
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
func (s *PlainSerializer) Marshal(data interface{}) ([]byte, error) {
	str := data.(string) // FIMXE: check type instead of assertion
	return []byte(str), nil
}

// Unmarshal ...
func (s *PlainSerializer) Unmarshal(data []byte, v interface{}) error {
	return nil
}

// DefaultValue ...
func (s *PlainSerializer) DefaultValue() interface{} {
	return ""
}

// ProtoSerializer ...
type ProtoSerializer struct {
	name string
}

// ContentType ...
func (s *ProtoSerializer) ContentType() string {
	return "application/protobuf"
}

// Marshal ...
func (s *ProtoSerializer) Marshal(data interface{}) ([]byte, error) {
	m := data.(proto.Message)
	return proto.Marshal(m)
}

// Unmarshal ...
func (s *ProtoSerializer) Unmarshal(data []byte, v interface{}) error {
	return proto.Unmarshal(data, v.(proto.Message))
}

// DefaultValue ...
func (s *ProtoSerializer) DefaultValue() interface{} {
	return ""
}

// JSONSerializer ...
type JSONSerializer struct {
	name string
}

// ContentType ...
func (s *JSONSerializer) ContentType() string {
	return "application/json"
}

// Marshal ...
func (s *JSONSerializer) Marshal(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// Unmarshal ...
func (s *JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// DefaultValue ...
func (s *JSONSerializer) DefaultValue() interface{} {
	return "{}"
}

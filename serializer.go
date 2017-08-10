package xing

type Serializer interface {
	ContentType() string
	Marshal() ([]byte, error)
	Unmarshal([]byte, *interface{}) error
}

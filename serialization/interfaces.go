package serialization

type Clonable[T any] interface {
	Clone() T
}

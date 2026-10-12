package domain

type Pinnable[T any] interface {
	Pin() *T
}

package predicates

func Must[T any](result T, err error) T {
	if err != nil {
		panic(err)
	}
	return result
}

func MustFunc[T any](fn func() (T, error)) func() T {
	return func() T {
		result, err := fn()
		if err != nil {
			panic(err)
		}
		return result
	}
}

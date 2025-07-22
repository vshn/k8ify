package util

func GetPointer[Type any](orig Type) *Type {
	return &orig
}

func Coalesce[T any](orig ...*T) *T {
	for _, t := range orig {
		if t != nil {
			return t
		}
	}
	return nil
}

func OrEmptyString(orig ...*string) string {
	strings := append(orig, GetPointer(""))
	return *Coalesce(strings...)
}

package validator

// Проверка соответствия строки на длину
func StrIsValid(str string, min int, max int) bool {
	l := len(str)
	return l >= min && l <= max
}

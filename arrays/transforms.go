package arrays

func Map[InputType, OutputType any](input []InputType, f func(InputType) OutputType) []OutputType {
	result := make([]OutputType, len(input))
	for i, v := range input {
		result[i] = f(v)
	}
	return result
}

func Filter[ArrayType any](input []ArrayType, f func(ArrayType) bool) []ArrayType {
	result := []ArrayType{}

	for _, v := range input {
		if f(v) {
			result = append(result, v)
		}
	}
	return result
}

func Reduce[InputType, OutputType any](input []InputType, f func(OutputType, InputType) OutputType, initial OutputType) OutputType {
	result := initial
	for _, v := range input {
		result = f(result, v)
	}
	return result
}

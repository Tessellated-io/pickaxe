package arrays

func Batch[T any](source []T, batchSize int) [][]T {
	var batches [][]T

	for batchSize < len(source) {
		source, batches = source[batchSize:], append(batches, source[0:batchSize:batchSize])
	}

	// Append the last batch if any items are left
	if len(source) > 0 {
		batches = append(batches, source)
	}

	return batches
}

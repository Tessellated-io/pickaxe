package arrays_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tessellated-io/pickaxe/arrays"
)

func TestBatch_EvenBatches(t *testing.T) {
	arr := []int{1, 2, 3, 4}
	batched := arrays.Batch(arr, 2)

	require.Equal(t, len(batched), 2)

	require.Equal(t, len(batched[0]), 2)
	require.Equal(t, len(batched[1]), 2)

	require.Equal(t, batched[0][0], 1)
	require.Equal(t, batched[0][1], 2)
	require.Equal(t, batched[1][0], 3)
	require.Equal(t, batched[1][1], 4)
}

func TestBatch_UnevenBatches(t *testing.T) {
	arr := []int{1, 2, 3}
	batched := arrays.Batch(arr, 2)

	require.Equal(t, len(batched), 2)

	require.Equal(t, len(batched[0]), 2)
	require.Equal(t, len(batched[1]), 1)

	require.Equal(t, batched[0][0], 1)
	require.Equal(t, batched[0][1], 2)
	require.Equal(t, batched[1][0], 3)
}

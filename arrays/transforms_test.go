package arrays_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tessellated-io/pickaxe/arrays"
)

func TestMap_Int(t *testing.T) {
	arr := []int{1, 2, 3}
	incrementFunc := func(input int) int { return input + 1 }

	transformed := arrays.Map(arr, incrementFunc)

	require.Equal(t, transformed[0], 2)
	require.Equal(t, transformed[1], 3)
	require.Equal(t, transformed[2], 4)
}

func TestMap_String(t *testing.T) {
	arr := []string{"ab", "cd", "ef"}
	incrementFunc := func(input string) string { return strings.ToUpper(input) }

	transformed := arrays.Map(arr, incrementFunc)

	require.Equal(t, transformed[0], "AB")
	require.Equal(t, transformed[1], "CD")
	require.Equal(t, transformed[2], "EF")
}

func TestFilter_Int(t *testing.T) {
	arr := []int{1, 2, 3, 4, 5}
	evenOnlyFunc := func(input int) bool { return input%2 == 0 }

	transformed := arrays.Filter(arr, evenOnlyFunc)

	require.Equal(t, len(transformed), 2)
	require.Equal(t, transformed[0], 2)
	require.Equal(t, transformed[1], 4)
}

func TestFilter_String(t *testing.T) {
	arr := []string{"abc", "de", "fgh", "ij", "klm"}
	evenOnlyStringLengthFunc := func(input string) bool { return len(input)%2 == 0 }

	transformed := arrays.Filter(arr, evenOnlyStringLengthFunc)

	require.Equal(t, len(transformed), 2)
	require.Equal(t, transformed[0], "de")
	require.Equal(t, transformed[1], "ij")
}

func TestReduce_Int(t *testing.T) {
	arr := []int{1, 2, 3}
	sumFunc := func(a, b int) int { return a + b }

	transformed := arrays.Reduce(arr, sumFunc, 0)

	require.Equal(t, transformed, 6)
}

func TestReduce_String(t *testing.T) {
	arr := []string{"ab", "cd", "ef"}
	appendStringFunc := func(a, b string) string { return fmt.Sprintf("%s%s", a, b) }

	transformed := arrays.Reduce(arr, appendStringFunc, "")

	require.Equal(t, transformed, "abcdef")
}

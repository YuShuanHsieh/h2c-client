package term

import "testing"

func TestParseLine(t *testing.T) {
	tests := []struct {
		data     string
		expected int
	}{
		{
			data:     "test data\r\n",
			expected: 2,
		},
		{
			data:     "",
			expected: 0,
		},
		{
			data:     "test data test2 \r\n",
			expected: 3,
		},
	}

	for i, test := range tests {
		result := parseLine([]byte(test.data))
		if len(result) != test.expected {
			t.Errorf("Test case %d failed: expected %d but get %v \n", i, test.expected, result)
		}
	}
}

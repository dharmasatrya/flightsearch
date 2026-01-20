package currency

import (
	"fmt"
	"math"
)

func FormatIDR(amount float64) string {
	rounded := math.Round(amount)

	negative := rounded < 0
	if negative {
		rounded = -rounded
	}

	intStr := fmt.Sprintf("%.0f", rounded)
	formatted := addThousandsSeparator(intStr, ".")

	result := "IDR " + formatted
	if negative {
		result = "-" + result
	}

	return result
}

func addThousandsSeparator(s string, sep string) string {
	n := len(s)
	if n <= 3 {
		return s
	}

	numSeps := (n - 1) / 3
	result := make([]byte, n+numSeps)

	j := len(result) - 1
	for i := n - 1; i >= 0; i-- {
		result[j] = s[i]
		j--

		pos := n - i
		if pos%3 == 0 && i > 0 {
			result[j] = sep[0]
			j--
		}
	}

	return string(result)
}

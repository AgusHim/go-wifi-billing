package utils

import (
	"fmt"
	"strings"
)

// Helper function to format currency
func FormatCurrency(amount int) string {
	// Simple Indonesian Rupiah formatting
	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		return str
	}

	var result string
	for i, digit := range str {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(digit)
	}
	return result
}

func NormalizeIDPhone(phone string) string {
	// remove spaces, dashes, parentheses
	replacer := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")
	phone = replacer.Replace(phone)

	// remove leading +
	phone = strings.TrimPrefix(phone, "+")

	// 08xxxx → 628xxxx
	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}

	// basic validation (Indonesia mobile usually 10–13 digits after 62)
	if !strings.HasPrefix(phone, "62") || len(phone) < 10 {
		return ""
	}

	return phone
}

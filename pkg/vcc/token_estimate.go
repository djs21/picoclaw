package vcc

// imageChars is the character weight assigned to a single image part
// (algo spec §3.7).
const imageChars = 4800

// defaultCharsPerToken is the calibration factor for char→token estimation,
// clamped to the [2, 6] range (algo spec §3.7).
const defaultCharsPerToken = 4

// clampCalibration restricts the chars-per-token calibration factor to [2, 6].
func clampCalibration(n int) int {
	if n < 2 {
		return 2
	}
	if n > 6 {
		return 6
	}
	return n
}

// estimateTokensFromChars estimates the token count for `chars` characters
// using the calibrated chars-per-token factor (algo spec §3.7).
func estimateTokensFromChars(chars int) int {
	if chars <= 0 {
		return 0
	}
	return chars / clampCalibration(defaultCharsPerToken)
}

// estimateMessageContentChars estimates the character count of message
// content, which may be a plain string or an array of parts. Each image part
// counts as imageChars (4800) characters (algo spec §3.7).
func estimateMessageContentChars(content interface{}) int {
	switch c := content.(type) {
	case string:
		return len(c)
	case []interface{}:
		total := 0
		for _, p := range c {
			m, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if _, ok := m["image"]; ok {
				total += imageChars
				continue
			}
			if t, ok := m["text"].(string); ok {
				total += len(t)
			}
		}
		return total
	default:
		return 0
	}
}

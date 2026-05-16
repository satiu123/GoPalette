package service

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	cjkCharsPerMinute     = 500
	englishWordsPerMinute = 225
)

func estimateReadingMinutes(content string, fallback string) int64 {
	text := strings.TrimSpace(content)
	if text == "" {
		text = strings.TrimSpace(fallback)
	}
	if text == "" {
		return 1
	}

	cjkChars, latinWords := countReadableUnits(text)
	cjkMinutes := float64(cjkChars) / cjkCharsPerMinute
	latinMinutes := float64(latinWords) / englishWordsPerMinute
	minutes := int64(cjkMinutes + latinMinutes)
	if cjkMinutes+latinMinutes > float64(minutes) {
		minutes++
	}
	if minutes < 1 {
		return 1
	}
	return minutes
}

func countReadableUnits(text string) (int, int) {
	cjkChars := 0
	latinWords := 0
	inWord := false

	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		text = text[size:]

		if isCJK(r) {
			cjkChars++
			inWord = false
			continue
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				latinWords++
				inWord = true
			}
			continue
		}

		inWord = false
	}

	return cjkChars, latinWords
}

func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) ||
		(r >= 0x3400 && r <= 0x4DBF) ||
		(r >= 0x20000 && r <= 0x2A6DF) ||
		(r >= 0x2A700 && r <= 0x2B73F) ||
		(r >= 0x2B740 && r <= 0x2B81F) ||
		(r >= 0x2B820 && r <= 0x2CEAF) ||
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0x2F800 && r <= 0x2FA1F)
}

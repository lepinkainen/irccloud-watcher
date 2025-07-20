package utils

import (
	"regexp"
	"strings"
)

var (
	// IRC color control codes pattern
	// Matches color codes like \x03NN,NN (foreground,background) or \x03NN (foreground only)
	// Also matches bold (\x02), underline (\x1F), italic (\x1D), strikethrough (\x1E), reset (\x0F)
	ircColorRegex = regexp.MustCompile(`\x03(?:\d{1,2}(?:,\d{1,2})?)?|\x02|\x1F|\x1D|\x1E|\x0F`)
)

// RemoveIRCColors removes IRC color and formatting control codes from a message
func RemoveIRCColors(message string) string {
	return ircColorRegex.ReplaceAllString(message, "")
}

// CleanIRCMessage removes IRC control codes and trims whitespace
func CleanIRCMessage(message string) string {
	cleaned := RemoveIRCColors(message)
	return strings.TrimSpace(cleaned)
}

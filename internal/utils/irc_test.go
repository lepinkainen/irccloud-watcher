package utils

import "testing"

func TestRemoveIRCColors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no color codes",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "foreground color only",
			input:    "\x0307Borg - Deduplicating Archiver with Compression and Encryption",
			expected: "Borg - Deduplicating Archiver with Compression and Encryption",
		},
		{
			name:     "foreground and background color",
			input:    "\x0307,01Text with colors",
			expected: "Text with colors",
		},
		{
			name:     "color reset",
			input:    "\x0307Colored text\x0FNormal text",
			expected: "Colored textNormal text",
		},
		{
			name:     "bold text",
			input:    "\x02Bold text\x02",
			expected: "Bold text",
		},
		{
			name:     "underline text",
			input:    "\x1FUnderlined text\x1F",
			expected: "Underlined text",
		},
		{
			name:     "italic text",
			input:    "\x1DItalic text\x1D",
			expected: "Italic text",
		},
		{
			name:     "strikethrough text",
			input:    "\x1EStrikethrough text\x1E",
			expected: "Strikethrough text",
		},
		{
			name:     "complex message with multiple formatting",
			input:    "\x0307Borg - Deduplicating Archiver with Compression and Encryption\x0399 \x0314[2 rubyn00bie]\x0399",
			expected: "Borg - Deduplicating Archiver with Compression and Encryption [2 rubyn00bie]",
		},
		{
			name:     "real IRC message example",
			input:    "07Borg - Deduplicating Archiver with Compression and Encryption99 14[2 rubyn00bie]99 https://www.borgbackup.org/ https://news.ycombinator.com/item?id=44621487",
			expected: "07Borg - Deduplicating Archiver with Compression and Encryption99 14[2 rubyn00bie]99 https://www.borgbackup.org/ https://news.ycombinator.com/item?id=44621487",
		},
		{
			name:     "real IRC message with control codes",
			input:    "\x0307Borg - Deduplicating Archiver with Compression and Encryption\x0399 \x0314[2 rubyn00bie]\x0399 https://www.borgbackup.org/ https://news.ycombinator.com/item?id=44621487",
			expected: "Borg - Deduplicating Archiver with Compression and Encryption [2 rubyn00bie] https://www.borgbackup.org/ https://news.ycombinator.com/item?id=44621487",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveIRCColors(tt.input)
			if result != tt.expected {
				t.Errorf("RemoveIRCColors(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCleanIRCMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean message with leading/trailing spaces",
			input:    "  \x0307Hello, world!\x03  ",
			expected: "Hello, world!",
		},
		{
			name:     "message with tabs and newlines",
			input:    "\t\x02Bold text\x02\n",
			expected: "Bold text",
		},
		{
			name:     "empty message with only control codes",
			input:    "\x0307\x03\x02\x02",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanIRCMessage(tt.input)
			if result != tt.expected {
				t.Errorf("CleanIRCMessage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

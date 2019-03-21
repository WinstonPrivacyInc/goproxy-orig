package goproxy

import (
	"testing"
)

func TestSanitizeCiper(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Starts with Dash",
			input:    "!hello world",
			expected: "hello-world",
		}, {
			name:     "Simple",
			input:    "Hello, world!",
			expected: "hello-world",
		}, {
			name:     "Samsung Galaxy S9",
			input:    "Mozilla/5.0 (Linux; Android 8.0.0; SM-G960F Build/R16NW) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.84 Mobile Safari/537.36",
			expected: "mozilla-5-0-linux-android",
		}, {
			name:     "Chrome-FreeBSD",
			input:    "Mozilla/5.0 (X11; FreeBSD amd64; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36",
			expected: "mozilla-5-0-x11-freebsd",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v := SanitizeCipherSignature(test.input)

			if v != test.expected {
				t.Fatalf("SanitizeCipherSignature(%q) returned %q; expected %q", test.input, v, test.expected)
			}
		})
	}
}

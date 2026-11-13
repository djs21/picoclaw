package vcc

import (
	"testing"
)

func TestTextOfString(t *testing.T) {
	got := textOf("hello world")
	if got != "hello world" {
		t.Errorf("textOf(string) = %q, want %q", got, "hello world")
	}
}

func TestTextOfEmptyString(t *testing.T) {
	got := textOf("")
	if got != "" {
		t.Errorf("textOf(empty) = %q, want %q", got, "")
	}
}

func TestTextOfArray(t *testing.T) {
	parts := []interface{}{
		map[string]interface{}{"text": "hello"},
		map[string]interface{}{"text": "world"},
	}
	got := textOf(parts)
	if got != "hello\nworld" {
		t.Errorf("textOf(array) = %q, want %q", got, "hello\nworld")
	}
}

func TestTextOfArraySkipsNonTextParts(t *testing.T) {
	parts := []interface{}{
		map[string]interface{}{"text": "hello"},
		map[string]interface{}{"type": "tool_call"},
		map[string]interface{}{"text": "world"},
	}
	got := textOf(parts)
	if got != "hello\nworld" {
		t.Errorf("textOf(array with non-text) = %q, want %q", got, "hello\nworld")
	}
}

func TestSanitizeCRLFtoLF(t *testing.T) {
	got := sanitize("a\r\nb\r c")
	if got != "a\nb\n c" {
		t.Errorf("sanitize(CRLF/CR) = %q, want %q", got, "a\nb\n c")
	}
}

func TestSanitizeStripsANSI(t *testing.T) {
	got := sanitize("\x1b[31mred\x1b[0m text")
	if got != "red text" {
		t.Errorf("sanitize(ANSI) = %q, want %q", got, "red text")
	}
}

func TestSanitizeStripsControlChars(t *testing.T) {
	got := sanitize("hello\x00world\x0b")
	if got != "helloworld" {
		t.Errorf("sanitize(control) = %q, want %q", got, "helloworld")
	}
}

func TestClipUnderLimit(t *testing.T) {
	got := clip("hi", 10)
	if got != "hi" {
		t.Errorf("clip(short) = %q, want %q", got, "hi")
	}
}

func TestClipOverLimit(t *testing.T) {
	got := clip("hello world", 5)
	if got != "hello" {
		t.Errorf("clip(over) = %q, want %q", got, "hello")
	}
}

func TestClipExactLimit(t *testing.T) {
	got := clip("hello", 5)
	if got != "hello" {
		t.Errorf("clip(exact) = %q, want %q", got, "hello")
	}
}

func TestClipSentenceUnderLimit(t *testing.T) {
	got := clipSentence("hi", 10)
	if got != "hi" {
		t.Errorf("clipSentence(short) = %q, want %q", got, "hi")
	}
}

func TestClipSentenceAtBoundary(t *testing.T) {
	got := clipSentence("Hello world. This is a test.", 12)
	if got != "Hello world." {
		t.Errorf("clipSentence(boundary) = %q, want %q", got, "Hello world.")
	}
}

func TestClipSentenceNoBoundary(t *testing.T) {
	got := clipSentence("Hello world this is a test", 5)
	if got != "Hello" {
		t.Errorf("clipSentence(no boundary) = %q, want %q", got, "Hello")
	}
}

func TestNonEmptyLines(t *testing.T) {
	got := nonEmptyLines("line1\n\nline3\n  \nline5")
	want := []string{"line1", "line3", "line5"}
	if len(got) != len(want) {
		t.Fatalf("nonEmptyLines = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("nonEmptyLines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNonEmptyLinesEmpty(t *testing.T) {
	got := nonEmptyLines("")
	if len(got) != 0 {
		t.Errorf("nonEmptyLines(empty) = %v, want empty", got)
	}
}

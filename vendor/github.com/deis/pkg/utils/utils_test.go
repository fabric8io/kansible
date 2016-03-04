package utils

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamOuptutSuccess(t *testing.T) {
	str := "abc\ndef\n"
	src := strings.NewReader(str)
	dest := bytes.NewBuffer(nil)
	out := bytes.NewBuffer(nil)

	err := streamOutput(src, dest, out)
	if err != nil {
		t.Fatal(err)
	}

	destStr := string(dest.Bytes())
	if destStr != str {
		t.Fatalf("dest was [%s], expected [%s]", destStr, str)
	}

	outStr := string(out.Bytes())
	if outStr != str {
		t.Fatalf("out was [%s], expected [%s]", outStr, str)
	}
}

func TestStreamOutputErr(t *testing.T) {
	// no ending newline
	str := "abc\ndef"
	src := strings.NewReader(str)
	dest := bytes.NewBuffer(nil)
	out := bytes.NewBuffer(nil)
	err := streamOutput(src, dest, out)
	if err == nil {
		t.Fatal("expected an error for a string that doesn't end with a newline")
	}
}

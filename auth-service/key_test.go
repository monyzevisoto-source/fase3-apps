package main

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	key, err := generateAPIKey()
	if err != nil {
		t.Fatalf("generateAPIKey() returned error: %v", err)
	}
	if !strings.HasPrefix(key, "tm_key_") {
		t.Fatalf("key prefix = %q, want tm_key_", key)
	}
	if len(key) != len("tm_key_")+64 {
		t.Fatalf("key length = %d, want %d", len(key), len("tm_key_")+64)
	}
}

func TestHashAPIKey(t *testing.T) {
	const key = "tm_key_example"
	const want = "33dc025d78e9d936c330595364f216b48b29fa6679dd1eec34cb3630f26e4013"

	if got := hashAPIKey(key); got != want {
		t.Fatalf("hashAPIKey(%q) = %q, want %q", key, got, want)
	}
}

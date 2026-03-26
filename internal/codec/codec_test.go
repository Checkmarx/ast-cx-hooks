package codec_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/codec"
)

func TestDecodeStdin(t *testing.T) {
	f, err := os.CreateTemp("", "codec-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	type Payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	want := Payload{Name: "alice", Age: 30}
	json.NewEncoder(f).Encode(want)
	f.Seek(0, 0)

	orig := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = orig }()

	var got Payload
	if err := codec.DecodeStdin(&got); err != nil {
		t.Fatalf("DecodeStdin: %v", err)
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestEncodeStdout(t *testing.T) {
	f, err := os.CreateTemp("", "codec-out-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	orig := os.Stdout
	os.Stdout = f
	defer func() { os.Stdout = orig }()

	type Payload struct {
		Status string `json:"status"`
	}
	if err := codec.EncodeStdout(Payload{Status: "ok"}); err != nil {
		t.Fatalf("EncodeStdout: %v", err)
	}

	f.Seek(0, 0)
	var got Payload
	if err := json.NewDecoder(f).Decode(&got); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if got.Status != "ok" {
		t.Fatalf("got %q, want ok", got.Status)
	}
}

func TestDecodeStdinInvalidJSON(t *testing.T) {
	f, err := os.CreateTemp("", "codec-bad-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("not json at all {{{")
	f.Seek(0, 0)

	orig := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = orig }()

	var v map[string]any
	if err := codec.DecodeStdin(&v); err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

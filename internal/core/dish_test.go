package core

import (
	"bytes"
	"testing"
)

func TestDishStringRoundTrip(t *testing.T) {
	d := NewDish([]byte("hello"), TypeString)
	if got := d.String(); got != "hello" {
		t.Fatalf("String() = %q, want %q", got, "hello")
	}
	if d.Type() != TypeString {
		t.Fatalf("Type() = %v, want %v", d.Type(), TypeString)
	}
}

func TestDishGetConversions(t *testing.T) {
	d := NewDish([]byte("AB"), TypeString)

	if got, err := d.Get(TypeString); err != nil || got.(string) != "AB" {
		t.Fatalf("Get(string) = %v, %v", got, err)
	}
	if got, err := d.Get(TypeByteArray); err != nil || !bytes.Equal(got.([]byte), []byte("AB")) {
		t.Fatalf("Get(byteArray) = %v, %v", got, err)
	}
	if got, err := d.Get(TypeArrayBuffer); err != nil || !bytes.Equal(got.([]byte), []byte("AB")) {
		t.Fatalf("Get(ArrayBuffer) = %v, %v", got, err)
	}
}

func TestDishGetNumber(t *testing.T) {
	d := NewDish([]byte("42"), TypeString)
	got, err := d.Get(TypeNumber)
	if err != nil {
		t.Fatalf("Get(number) error: %v", err)
	}
	if got.(float64) != 42 {
		t.Fatalf("Get(number) = %v, want 42", got)
	}
}

func TestDishBytes(t *testing.T) {
	raw := []byte{0x00, 0xff, 0x10}
	d := NewDish(raw, TypeByteArray)
	if !bytes.Equal(d.Bytes(), raw) {
		t.Fatalf("Bytes() = %v, want %v", d.Bytes(), raw)
	}
}

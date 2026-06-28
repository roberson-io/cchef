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

func TestDishSet(t *testing.T) {
	d := NewDish(nil, TypeString)

	if err := d.Set("hi", TypeString); err != nil || d.String() != "hi" {
		t.Fatalf("Set string: %v, %q", err, d.String())
	}
	if err := d.Set([]byte{1, 2}, TypeByteArray); err != nil || !bytes.Equal(d.Bytes(), []byte{1, 2}) {
		t.Fatalf("Set byteArray: %v, %v", err, d.Bytes())
	}
	if err := d.Set(float64(42), TypeNumber); err != nil || d.String() != "42" {
		t.Fatalf("Set number float: %v, %q", err, d.String())
	}
	if err := d.Set(7, TypeNumber); err != nil || d.String() != "7" {
		t.Fatalf("Set number int: %v, %q", err, d.String())
	}
	if d.Type() != TypeNumber {
		t.Fatalf("Type() = %v after Set number", d.Type())
	}

	// Type mismatches and unknown types must error.
	if err := d.Set(123, TypeString); err == nil {
		t.Fatal("expected error setting non-string as string")
	}
	if err := d.Set("x", TypeByteArray); err == nil {
		t.Fatal("expected error setting non-bytes as byteArray")
	}
	if err := d.Set("x", "bogus"); err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestDishBytes(t *testing.T) {
	raw := []byte{0x00, 0xff, 0x10}
	d := NewDish(raw, TypeByteArray)
	if !bytes.Equal(d.Bytes(), raw) {
		t.Fatalf("Bytes() = %v, want %v", d.Bytes(), raw)
	}
}

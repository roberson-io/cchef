package core

import (
	"fmt"
	"strconv"
)

// DishType identifies how the bytes in a Dish should be interpreted. It mirrors
// CyberChef's Dish types (src/core/Dish.mjs). All byte-backed types share the
// same canonical storage ([]byte); conversions only matter at the edges.
type DishType string

const (
	// TypeString treats the bytes as a UTF-8 string.
	TypeString DishType = "string"
	// TypeByteArray treats the bytes as a raw byte array.
	TypeByteArray DishType = "byteArray"
	// TypeArrayBuffer is CyberChef's binary hub type; identical storage to byteArray.
	TypeArrayBuffer DishType = "ArrayBuffer"
	// TypeNumber treats the bytes as the ASCII representation of a number.
	TypeNumber DishType = "number"
	// TypeJSON treats the bytes as JSON text.
	TypeJSON DishType = "JSON"
	// TypeBigNumber treats the bytes as the decimal text of an arbitrary-precision number.
	TypeBigNumber DishType = "BigNumber"
	// TypeFileList holds several named files rather than one byte string. It is
	// CyberChef's List<File>; such a dish is terminal — it cannot be converted
	// back to bytes and so cannot feed a following operation.
	TypeFileList DishType = "List<File>"
)

// NamedFile is one file in a TypeFileList dish.
type NamedFile struct {
	Name string
	Data []byte
}

// Dish is the data container passed between operations. It holds canonical
// []byte storage plus a type tag describing the current interpretation.
type Dish struct {
	data  []byte
	typ   DishType
	files []NamedFile
}

// NewDish builds a Dish from raw bytes and a type tag.
func NewDish(data []byte, typ DishType) *Dish {
	return &Dish{data: data, typ: typ}
}

// NewFileListDish builds a Dish holding several named files.
func NewFileListDish(files []NamedFile) *Dish {
	return &Dish{typ: TypeFileList, files: files}
}

// Files returns the named files of a TypeFileList dish (nil for any other type).
func (d *Dish) Files() []NamedFile { return d.files }

// Type returns the dish's current type tag.
func (d *Dish) Type() DishType { return d.typ }

// Bytes returns the canonical byte storage.
func (d *Dish) Bytes() []byte { return d.data }

// String returns the dish data as a UTF-8 string.
func (d *Dish) String() string { return string(d.data) }

// Get returns the dish value converted to the requested type. String,
// byteArray and ArrayBuffer are byte-backed and convert trivially; number is
// parsed from the ASCII representation.
func (d *Dish) Get(typ DishType) (any, error) {
	if d.typ == TypeFileList || typ == TypeFileList {
		if d.typ == TypeFileList && typ == TypeFileList {
			return d.files, nil
		}
		return nil, fmt.Errorf("cannot convert between %q and %q: a file list holds several files and cannot be chained", d.typ, typ)
	}
	switch typ {
	case TypeString, TypeJSON, TypeBigNumber:
		return string(d.data), nil
	case TypeByteArray, TypeArrayBuffer:
		return d.data, nil
	case TypeNumber:
		n, err := strconv.ParseFloat(string(d.data), 64)
		if err != nil {
			return nil, fmt.Errorf("cannot interpret dish as number: %w", err)
		}
		return n, nil
	default:
		return nil, fmt.Errorf("unknown dish type %q", typ)
	}
}

// Set replaces the dish value. Byte-backed values are stored as-is; numbers are
// rendered to their ASCII representation.
func (d *Dish) Set(value any, typ DishType) error {
	switch typ {
	case TypeString, TypeJSON, TypeBigNumber:
		switch v := value.(type) {
		case string:
			d.data = []byte(v)
		case []byte:
			d.data = v
		default:
			return fmt.Errorf("expected string or []byte for type %q, got %T", typ, value)
		}
	case TypeByteArray, TypeArrayBuffer:
		b, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("expected []byte for type %q, got %T", typ, value)
		}
		d.data = b
	case TypeNumber:
		switch n := value.(type) {
		case float64:
			d.data = []byte(strconv.FormatFloat(n, 'f', -1, 64))
		case int:
			d.data = []byte(strconv.Itoa(n))
		default:
			return fmt.Errorf("expected number for type %q, got %T", typ, value)
		}
	default:
		return fmt.Errorf("unknown dish type %q", typ)
	}
	d.typ = typ
	return nil
}

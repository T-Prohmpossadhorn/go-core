package logger

import (
	"time"
)

// FieldType represents the type of a field
type FieldType int

const (
	// StringType for string fields
	StringType FieldType = iota
	// IntType for integer fields
	IntType
	// BoolType for boolean fields
	BoolType
	// FloatType for float fields
	FloatType
	// ErrorType for error fields
	ErrorType
	// TimeType for time fields
	TimeType
	// DurationType for duration fields
	DurationType
	// ObjectType for arbitrary object fields
	ObjectType
)

// Field represents a log field
type Field struct {
	Key       string
	Type      FieldType
	String    string
	Int       int64
	Bool      bool
	Float     float64
	Error     error
	Time      time.Time
	Duration  time.Duration
	Interface interface{}
}

// String creates a string field
func String(key, value string) Field {
	return Field{
		Key:    key,
		Type:   StringType,
		String: value,
	}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{
		Key:  key,
		Type: IntType,
		Int:  int64(value),
	}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{
		Key:  key,
		Type: IntType,
		Int:  value,
	}
}

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return Field{
		Key:  key,
		Type: BoolType,
		Bool: value,
	}
}

// Float creates a float field
func Float(key string, value float64) Field {
	return Field{
		Key:   key,
		Type:  FloatType,
		Float: value,
	}
}

// Error creates an error field
func Errors(err error) Field {
	return Field{
		Key:   "error",
		Type:  ErrorType,
		Error: err,
	}
}

// Time creates a time field
func Time(key string, value time.Time) Field {
	return Field{
		Key:  key,
		Type: TimeType,
		Time: value,
	}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{
		Key:      key,
		Type:     DurationType,
		Duration: value,
	}
}

// Any creates a field with an arbitrary value
func Any(key string, value interface{}) Field {
	return Field{
		Key:       key,
		Type:      ObjectType,
		Interface: value,
	}
}

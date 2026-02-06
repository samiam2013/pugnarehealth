package main

import (
	"errors"
	"fmt"
	"strings"
)

type enum []string

func NewEnum(values []string) enum {
	return enum(values)
}

var ErrInvalidEnumValue = errors.New("invalid value for enum")

func (e *enum) CheckError(value string) error {
	if !e.Valid(value) {
		err := fmt.Errorf("%s is not a valid value for enum must be one of %v", value, *e)
		return errors.Join(ErrInvalidEnumValue, err)
	}
	return nil
}

// Valid performs a case-sensitive check for validity
func (e *enum) Valid(value string) bool {
	for _, v := range *e {
		if v == value {
			return true
		}
	}
	return false
}

// ValidCI performs a case-insensitive check for validity
func (e *enum) ValidCI(value string) bool {
	lowerValue := strings.ToLower(value)
	for _, v := range *e {
		if strings.ToLower(v) == lowerValue {
			return true
		}
	}
	return false
}

func (e *enum) MustValidate(value string) {
	if err := e.CheckError(value); err != nil {
		panic(err)
	}
}

package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Option represents a changeable internal parameter within an engine.
type Option struct {
	Name    string
	Type    int
	Default interface{}
	Min     int
	Max     int
	Vars    map[string]bool
}

const (
	none = iota
	name
	dtype
	defaultval
	min
	max
	variable
)

func stringToToken(s string) int {
	switch s {
	case "name":
		return name
	case "type":
		return dtype
	case "default":
		return defaultval
	case "min":
		return min
	case "max":
		return max
	case "var":
		return variable
	default:
		return none
	}
}

// Option Types
const (
	// Check means checkbox. An Option of type Check should
	// have a boolean default value i.e. "true" or "false"
	Check = iota
	// Spin means spinner. An Option of type Spinner should
	// have a minimum value, a maximum value and a default value.
	// All of which should be integers.
	Spin
	// Combo means combination box. An Option of type Combo should
	// have a set of variables and a default value which is
	// one of the variables.
	Combo
	// Button means button.
	Button
	// String means string. An Option of type String should have
	// a default value that is a none-empty string.
	String
)

func stringToType(s string) int {
	switch s {
	case "check":
		return Check
	case "spin":
		return Spin
	case "combo":
		return Combo
	case "button":
		return Button
	case "string":
		return String
	default:
		return -1
	}
}

// OptionFromCFP creates an Option object from an option command
// sent to the GUI from an engine.
//
// TODO: Clean this up!
func OptionFromCFP(args []string) (Option, error) {
	var err error
	result := Option{Vars: make(map[string]bool)}
	currentParameter := none
	currentIndex := 0
	for i, v := range args {
		token := stringToToken(v)
		if token == none && i != len(args)-1 {
			continue
		}
		if currentParameter != none {
			var value string
			if i == len(args)-1 {
				value = strings.Join(args[currentIndex+1:i+1], " ")
			} else {
				value = strings.Join(args[currentIndex+1:i], " ")
			}
			switch currentParameter {
			case name:
				result.Name = value
			case dtype:
				result.Type = stringToType(value)
			case defaultval:
				result.Default = value
			case min:
				result.Min, err = strconv.Atoi(value)
				if err != nil {
					return result, errors.Wrap(err, "couldn't parse min")
				}
			case max:
				result.Max, err = strconv.Atoi(value)
				if err != nil {
					return result, errors.Wrap(err, "couldn't parse max")
				}
			case variable:
				result.Vars[value] = true
			}
		}
		currentParameter = token
		currentIndex = i
	}
	return result, result.finalize()
}

// CFPSetOption creates a setoption command to send to an engine.
func (o Option) CFPSetOption(value interface{}) (string, error) {
	if o.Type == Button {
		return fmt.Sprintf("setoption name %s", o.Name), nil
	}
	if v, err := o.Vtos(value); err == nil {
		return fmt.Sprintf("setoption name %s value %s", o.Name, v), nil
	}
	return "", errors.New("invalid value")
}

// TODO: Clean this up!
func (o Option) finalize() error {
	if o.Name == "" {
		return errors.New("option must have name")
	}
	switch o.Type {
	case Check:
		switch o.Default {
		case "true":
			o.Default = true
		case "false":
			o.Default = false
		default:
			return errors.New("check default must be true or false")
		}
	case Spin:
		str, ok := o.Default.(string)
		if !ok {
			return errors.New("default is not string")
		}
		def, err := strconv.Atoi(str)
		if err != nil {
			return errors.New("spin default must be an integer")
		}
		o.Default = def
		if o.Min > o.Max {
			return errors.New("min cannot be greater than max")
		}
		if def < o.Min || def > o.Max {
			return errors.New("default out or range")
		}
	case Combo:
		str, ok := o.Default.(string)
		if !ok {
			return errors.New("default must be a string")
		}
		if _, ok = o.Vars[str]; !ok {
			return errors.New("default not in vars")
		}
	case Button:
	case String:
		if _, ok := o.Default.(string); !ok {
			return errors.New("default is not a string")
		}
	default:
		return errors.New("option must have type")
	}
	return nil
}

// Vtos converts an interface{} value to a string for a specific option.
// An error will be returned if the value is not of the correct type
// for the option.
func (o Option) Vtos(value interface{}) (string, error) {
	switch v := value.(type) {
	case nil:
		if o.Type != Button {
			return "", errors.New("value cannot be nil")
		}
	case bool:
		if o.Type != Check {
			return "", errors.New("incorrect type")
		}
	case int:
		if o.Type != Spin {
			return "", errors.New("incorrect type")
		}
	case string:
		if o.Type != String && o.Type != Combo {
			return "", errors.New("incorrect type")
		}
		if _, ok := o.Vars[v]; o.Type == Combo && !ok {
			return "", errors.New("invalid string")
		}
	default:
		return "", errors.New("invalid type")
	}
	return fmt.Sprintf("%v", value), nil
}

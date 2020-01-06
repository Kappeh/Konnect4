package main

// Option is an internal parameter of an engine
// that is changable from the gui
type Option interface {
	// OptionName returns the name of the option
	OptionName() string
}

// CheckBox is a true or false value
type CheckBox struct {
	Name  string
	Value bool
}

// OptionName returns the name of the CheckBox
func (c CheckBox) OptionName() string {
	return c.Name
}

// Spinner is an integer value within a spcified range
type Spinner struct {
	Name  string
	Min   int
	Max   int
	Value int
}

// OptionName returns the name of the Spinner
func (s Spinner) OptionName() string {
	return s.Name
}

// Button is a signal that an event has happened
// or that an event is to happen in the engine
type Button struct {
	Name string
}

// OptionName returns the name of the Button
func (b Button) OptionName() string {
	return b.Name
}

// ComboBox is a string value that can only be
// a set of values specified by the engine
type ComboBox struct {
	Name  string
	Vars  map[string]bool
	Value string
}

// OptionName returns the name of the ComboBox
func (c ComboBox) OptionName() string {
	return c.Name
}

// String is an unrestricted string value
type String struct {
	Name  string
	Value string
}

// OptionName returns the name of the String
func (s String) OptionName() string {
	return s.Name
}

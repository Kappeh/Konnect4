package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	// infoBufferSize is the maximum amount of messages from the
	// engine that will be stored within the communications channel
	infoBufferSize = 10
	// handshakeTimeout is the maximum amount of time in nanoseconds the
	// engine is allowed to perform the CFP handshake
	handshakeTimeout = 5.0 * time.Second
	// bestmoveTimeout is the maximum amount of time in nanoseconds the
	// engine is allowed to respond to a stop command with bestmove
	bestmoveTimeout = 5.0 * time.Second
	// readyokTimeout is the maximum amount of time in nanoseconds the
	// engine is allowed to respond to an isready command with readyok
	readyokTimeout = 5.0 * time.Second
)

// CFPCommunicator is an interface to an engine that
// supports CFP. It stores the input and output streams
// to the engine's process which are used to send and
// receive commands to and from the engine.
type CFPCommunicator struct {
	// Communication pipes
	stdin  io.WriteCloser
	stdout io.ReadCloser
	// Handshake channels
	name   chan string
	author chan string
	option chan Option
	cfpok  chan bool
	// Other communication channels
	readyok  chan bool
	bestmove chan int
	info     chan string
}

// NewCFPCommunicator creates a new Communicator that
// uses the CFP protocol to interact with an engine.
// cmd should be the engine's process.
// An error will be returned if the input and/or output pipes
// cannot be aquired.
func NewCFPCommunicator(cmd *exec.Cmd) (Communicator, error) {
	// Make new Communicator along with all channels used
	// for sending signals around the communicator
	result := CFPCommunicator{
		name:     make(chan string),
		author:   make(chan string),
		option:   make(chan Option),
		cfpok:    make(chan bool),
		readyok:  make(chan bool),
		bestmove: make(chan int),
		info:     make(chan string, infoBufferSize),
	}
	// Aquire stdin and stdout pipes
	var err error
	if result.stdin, err = cmd.StdinPipe(); err != nil {
		return nil, errors.Wrap(err, "couldn't aquire stdin pipe")
	}
	if result.stdout, err = cmd.StdoutPipe(); err != nil {
		return nil, errors.Wrap(err, "couldn't aquire stdout pipe")
	}
	// Return the result
	return Communicator(&result), nil
}

// Handshake performs the CFP. During which, the name, author and
// engine options will be aquired.
// If the engine doesn't support CFP, doesn't perform the handshake
// in time or doesn't provide required information, an error will
// be returned.
func (c *CFPCommunicator) Handshake(name, author *string, options *map[string]Option) error {
	// Starts listening for commands from engine
	go c.listenToEngine()
	// Send command to initialize handshake
	if _, err := c.stdin.Write([]byte("cfp\n")); err != nil {
		return errors.Wrap(err, "unable to send cfp command")
	}
	var (
		timeout   = time.After(handshakeTimeout)
		setName   = false
		setAuthor = false
	)
	// While the handshake is being performed
	for running := true; running; {
		// Check if engine has sent information
		select {
		case v := <-c.name:
			*name = v
			setName = true
		case v := <-c.author:
			*author = v
			setAuthor = true
		case v := <-c.option:
			(*options)[v.OptionName()] = v
		case <-c.cfpok:
			// Engine has signaled they are finished with
			// the CFP handshake
			running = false
		case <-timeout:
			// Engine took too long to perform handshake
			return errors.New("handshake timed out")
		}
	}
	// A name and author is required. If either is not
	// provided by the engine, the handshake fails
	if !setName || !setAuthor {
		return errors.New("name and/or author was not provided")
	}
	// The handshake was a success
	return nil
}

// Debug enables or disables debug mode for the engine
func (c *CFPCommunicator) Debug(enable bool) error {
	var cmd string
	if enable {
		cmd = "debug on\n"
	} else {
		cmd = "debug off\n"
	}
	if _, err := c.stdin.Write([]byte(cmd)); err != nil {
		return errors.Wrap(err, "couldn't send debug command")
	}
	return nil
}

// SetOption sets an internal parameter of the engine
// The options have been specified by the engine
// during the CFP handshake
func (c *CFPCommunicator) SetOption(o Option) error {
	// Getting the value of the option as a string
	var valueString string
	// The format of the command depends on the option type
	switch v := o.(type) {
	case CheckBox:
		valueString = fmt.Sprintf(" value %t", v.Value)
	case Spinner:
		valueString = fmt.Sprintf(" value %d", v.Value)
	case Button:
		valueString = ""
	case ComboBox:
		valueString = fmt.Sprintf(" value %s", v.Value)
	case String:
		valueString = fmt.Sprintf(" value %s", v.Value)
	default:
		// The CFP protocol doesn't support this option type
		return errors.New("unsupported option type")
	}
	// Telling engine to set the specified internal parameter
	cmd := fmt.Sprintf("setoption name %s%s\n", o.OptionName(), valueString)
	if _, err := c.stdin.Write([]byte(cmd)); err != nil {
		return errors.Wrap(err, "couldn't send setoption command")
	}
	// Command sent successfully
	return nil
}

// NewGame tells the engine that the next position it
// will receive is from a different game to the previous
// position it was sent
func (c *CFPCommunicator) NewGame() error {
	if err := c.waitForReady(); err != nil {
		return errors.Wrap(err, "engine not ready")
	}
	if _, err := c.stdin.Write([]byte("cfpnewgame\n")); err != nil {
		return errors.Wrap(err, "couldn't send cfpnewgame command")
	}
	return nil
}

// Position tells the engine to analyse a different
// position. Usually because of a game reset or a move
// has been made
func (c *CFPCommunicator) Position(s State) error {
	// Check that the engine is ready for new commands
	if err := c.waitForReady(); err != nil {
		return errors.Wrap(err, "engine not ready")
	}
	// Changing s into a string representation of the
	// position in compliance with the CFP protocol
	posRunes := [43]rune{}
	for i, v := range s.Tiles {
		switch v {
		case Empty:
			posRunes[i] = '0'
		case Player1:
			posRunes[i] = '1'
		case Player2:
			posRunes[i] = '2'
		}
	}
	switch s.Player {
	case Player1:
		posRunes[42] = '1'
	case Player2:
		posRunes[42] = '2'
	}
	// Sending command
	cmd := fmt.Sprintf("position %s\n", string(posRunes[:]))
	if _, err := c.stdin.Write([]byte(cmd)); err != nil {
		return errors.Wrap(err, "couldn't send position command")
	}
	// Command successfully sent
	return nil
}

// Go Tells the engine that it should start analysing the
// last position it was sent. In addition to this,
// if moveTime is positive, the engine will be told to
// complete it's move within the given time.
func (c *CFPCommunicator) Go(moveTime float32) error {
	// Check engine is ready for commands
	if err := c.waitForReady(); err != nil {
		return errors.Wrap(err, "engine not ready")
	}
	// Generating command to send
	var cmd string
	if moveTime <= 0.0 {
		cmd = "go\n"
	} else {
		cmd = fmt.Sprintf("go movetime %f\n", moveTime)
	}
	// Sending command
	if _, err := c.stdin.Write([]byte(cmd)); err != nil {
		return errors.Wrap(err, "couldn't send go command")
	}
	// Command successfully sent
	return nil
}

// Stop tells the engine to stop analysing it's position
// and return the best move that it found
// If the engine doesn't provide a best move, an
// error will be returned
func (c *CFPCommunicator) Stop() (int, error) {
	// Check engine is ready for commands
	if err := c.waitForReady(); err != nil {
		return 0, errors.Wrap(err, "engine not ready")
	}
	// Send stop command
	if _, err := c.stdin.Write([]byte("stop\n")); err != nil {
		return 0, errors.Wrap(err, "couldn't send stop command")
	}
	// Wait on bestmove command from engine
	select {
	case v := <-c.bestmove:
		// Return the best move
		return v, nil
	case <-time.After(bestmoveTimeout):
		// Engine didn't send best move in time
		return 0, errors.New("bestmove timed out")
	}
}

// Quit tells the engine to quit as soon as possible and
// closes the stdin and stdout pipes used to communicate
// to the engine's process
func (c *CFPCommunicator) Quit() error {
	// Check engine is ready for commands
	if err := c.waitForReady(); err != nil {
		return errors.Wrap(err, "engine not ready")
	}
	// Send quit command
	if _, err := c.stdin.Write([]byte("quit\n")); err != nil {
		return errors.Wrap(err, "couldn't send quit command")
	}
	// Close stdin and stdout pipes
	if err := c.stdin.Close(); err != nil {
		return errors.Wrap(err, "couldn't close stdin pipe")
	}
	if err := c.stdout.Close(); err != nil {
		return errors.Wrap(err, "couldn't close stdout pipe")
	}
	// return successfully
	return nil
}

// InfoChannel returns a channel that get's populated
// with the information received from the engine
// via info commands
func (c *CFPCommunicator) InfoChannel() <-chan string {
	return c.info
}

// listenToEngine listens out for commands
// sent by the engine via stdout
func (c *CFPCommunicator) listenToEngine() {
	// Parsing commands on one goroutine
	// to maintain order or commands and
	// avoid race conditions
	scanner := bufio.NewScanner(c.stdout)
	for scanner.Scan() {
		c.receivedCommand(scanner.Text())
	}
}

// waitForReady sends an isready command to the engine
// and waits until the engine responds with a readyok command
// If the engine takes too long, an error will be returned
func (c *CFPCommunicator) waitForReady() error {
	// Send isready command
	if _, err := c.stdin.Write([]byte("isready\n")); err != nil {
		return errors.Wrap(err, "unable to send isready command")
	}
	// Wait for response or timeout
	select {
	case <-time.After(readyokTimeout):
		// Engine took too long to respond
		return errors.New("engine took too long to respond")
	case <-c.readyok:
		// Engine responded
		return nil
	}
}

// receivedCommand is ran whenever a command is sent
// from the engine. Either an event is triggered
// of the command string is sent to another function
// to be parsed and handled fully
func (c *CFPCommunicator) receivedCommand(msg string) {
	args := strings.Split(msg, " ")
	if len(args) == 0 {
		return
	}
	switch strings.ToLower(args[0]) {
	case "id":
		c.receivedIDCommand(args[1:])
	case "cfpok":
		c.cfpok <- true
	case "readyok":
		c.readyok <- true
	case "bestmove":
		c.receivedBestMoveCommand(args[1:])
	case "info":
		c.receivedInfoCommand(args[1:])
	case "option":
		c.receivedOptionCommand(args[1:])
	}
}

// receivedIDCommand is called when an id command is received
// from the engine
func (c *CFPCommunicator) receivedIDCommand(args []string) {
	if len(args) < 2 {
		return
	}
	switch strings.ToLower(args[0]) {
	case "name":
		c.name <- strings.Join(args[1:], " ")
	case "author":
		c.author <- strings.Join(args[1:], " ")
	}
}

// receivedIDCommand is called when a bestmove command is received
// from the engine
func (c *CFPCommunicator) receivedBestMoveCommand(args []string) {
	if len(args) < 1 {
		return
	}
	move, err := strconv.Atoi(args[0])
	if err != nil {
		return
	}
	c.bestmove <- move
}

// receivedIDCommand is called when an info command is received
// from the engine
func (c *CFPCommunicator) receivedInfoCommand(args []string) {
	if len(args) < 1 {
		return
	}
	c.info <- strings.Join(args, " ")
}

// sliceIndex finds the indx of the first element of a slice that
// holds to a predicate function
func sliceIndex(limit int, predicate func(int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}

// receivedOptionCommand is called whenever the engine
// has specified an internal parameter that can be changed
// The command arguments are used to determine which option type
// is being specified and then calls the respective parsing function
// Note: As specified in the CFP protocol, if the command
// cannot be parsed, it is ignored.
func (c *CFPCommunicator) receivedOptionCommand(args []string) {
	// Getting index of type identifier
	typeIndex := sliceIndex(len(args), func(i int) bool {
		return strings.ToLower(args[i]) == "type"
	})
	if typeIndex == -1 {
		return
	}
	var (
		option Option
		err    error
	)
	// Calling parsing function depending on type
	switch strings.ToLower(args[typeIndex+1]) {
	case "check":
		option, err = c.checkOption(args[:])
	case "spin":
		option, err = c.spinOption(args[:])
	case "button":
		option, err = c.buttonOption(args[:])
	case "combo":
		option, err = c.comboOption(args[:])
	case "string":
		option, err = c.stringOption(args[:])
	default:
		return
	}
	// If an error occured, ignore the command
	if err != nil {
		return
	}
	// Otherwise, send parsed option to be handled
	c.option <- option
}

// Parameter is used to group keywords into
// name, value pairs
type Parameter struct {
	name  string
	value string
}

// extractParameters identifies keywords in an argument list
// and following sections of the string which are to be interpreted as
// the parameters' values
func (c *CFPCommunicator) extractParameters(args []string) []Parameter {
	// Map to quickly identify keywords
	identifiers := map[string]bool{
		"name":    true,
		"type":    true,
		"default": true,
		"min":     true,
		"max":     true,
		"var":     true,
	}
	// Counting parameters
	parameterCount := 0
	for _, v := range args {
		if _, ok := identifiers[strings.ToLower(v)]; ok {
			parameterCount++
		}
	}
	// Making space for results
	indexes := make([]int, parameterCount)
	parameters := make([]Parameter, parameterCount)
	// Extracting keyword indexes
	index := 0
	for i, v := range args {
		if _, ok := identifiers[strings.ToLower(v)]; ok {
			indexes[index] = i
			index++
		}
	}
	// Extracting keywords and values
	for i := 0; i < parameterCount; i++ {
		endIndex := len(args)
		if i != parameterCount-1 {
			endIndex = indexes[i+1]
		}
		parameters[i] = Parameter{
			name:  strings.ToLower(args[indexes[i]]),
			value: strings.Join(args[indexes[i]+1:endIndex], " "),
		}
	}
	// Returning results
	return parameters
}

// checkOption creates a CheckBox from an argument list
func (c *CFPCommunicator) checkOption(args []string) (Option, error) {
	var (
		// The list of extracted parameters
		parameters = c.extractParameters(args)
		// The resulting CheckBox
		result = CheckBox{}
		// Variables to check required information is provided
		nameSet  = false
		valueSet = false
	)
	// For each parameter that is extracted
	for _, v := range parameters {
		// If we case about it, put it into the result and set
		// the flags checking for the respective value
		switch v.name {
		case "name":
			result.Name = v.value
			nameSet = true
		case "default":
			if v.value != "true" && v.value != "false" {
				return nil, fmt.Errorf("%s is not a valid value for checkbox", v.value)
			}
			result.Value = v.value == "true"
			valueSet = true
		}
	}
	// Check required information has been provided
	if !nameSet || !valueSet {
		return nil, errors.New("checkbox requires more information")
	}
	// Return the result
	return result, nil
}

// spinOption creates a Spinner from an argument list
func (c *CFPCommunicator) spinOption(args []string) (Option, error) {
	var (
		// The list of extracted parameters
		parameters = c.extractParameters(args)
		// The resulting Spinner
		result = Spinner{}
		// Variables to check required information is provided
		nameSet  = false
		minSet   = false
		maxSet   = false
		valueSet = false
	)
	// For each parameter that is extracted
	for _, v := range parameters {
		// If we case about it, put it into the result and set
		// the flags checking for the respective value
		switch v.name {
		case "name":
			result.Name = v.value
			nameSet = true
		case "min":
			num, err := strconv.Atoi(v.value)
			if err != nil {
				return nil, errors.Wrap(err, "invalid value for min")
			}
			result.Min = num
			minSet = true
		case "max":
			num, err := strconv.Atoi(v.value)
			if err != nil {
				return nil, errors.Wrap(err, "invalid value for max")
			}
			result.Max = num
			maxSet = true
		case "default":
			num, err := strconv.Atoi(v.value)
			if err != nil {
				return nil, errors.Wrap(err, "invalid value for default")
			}
			result.Value = num
			valueSet = true
		}
	}
	// Check required information has been provided
	if !nameSet || !minSet || !maxSet || !valueSet {
		return nil, errors.New("spinner requires more information")
	}
	// Return the result
	return result, nil
}

// buttonOption creates a Button from an argument list
func (c *CFPCommunicator) buttonOption(args []string) (Option, error) {
	var (
		// The list of extracted parameters
		parameters = c.extractParameters(args)
		// The resulting Button
		result = Button{}
		// Variables to check required information is provided
		nameSet = false
	)
	// For each parameter that is extracted
	for _, v := range parameters {
		// If we case about it, put it into the result and set
		// the flags checking for the respective value
		switch v.name {
		case "name":
			result.Name = v.value
			nameSet = true
		}
	}
	// Check required information has been provided
	if !nameSet {
		return nil, errors.New("button requires more information")
	}
	// Return the result
	return result, nil
}

// comboOption creates a ComboBox from an argument list
func (c *CFPCommunicator) comboOption(args []string) (Option, error) {
	var (
		// The list of extracted parameters
		parameters = c.extractParameters(args)
		// The resulting ComboBox
		result = ComboBox{Vars: make(map[string]bool)}
		// Variables to check required information is provided
		nameSet  = false
		valueSet = false
		varSet   = 0
	)
	// For each parameter that is extracted
	for _, v := range parameters {
		// If we case about it, put it into the result and set
		// the flags checking for the respective value
		switch v.name {
		case "name":
			result.Name = v.value
			nameSet = true
		case "default":
			result.Value = v.value
			valueSet = true
		case "var":
			result.Vars[v.value] = true
			varSet++
		}
	}
	// Check required information has been provided
	if !nameSet || !valueSet || varSet < 2 {
		return nil, errors.New("combobox requires more information")
	}
	// Check the value is within the vars
	if _, ok := result.Vars[result.Value]; !ok {
		return nil, errors.New("value not in combobox vars")
	}
	// Return the result
	return result, nil
}

// stringOption creates a String from an argument list
func (c *CFPCommunicator) stringOption(args []string) (Option, error) {
	var (
		// The list of extracted parameters
		parameters = c.extractParameters(args)
		// The resulting String
		result = String{}
		// Variables to check required information is provided
		nameSet  = false
		valueSet = false
	)
	// For each parameter that is extracted
	for _, v := range parameters {
		// If we case about it, put it into the result and set
		// the flags checking for the respective value
		switch v.name {
		case "name":
			result.Name = v.value
			nameSet = true
		case "default":
			result.Value = v.value
			valueSet = true
		}
	}
	// Check required information has been provided
	if !nameSet || !valueSet {
		return nil, errors.New("string option requires more information")
	}
	// Return the result
	return result, nil
}

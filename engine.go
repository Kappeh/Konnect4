package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// Engine is a process that is to be provided with connect four
// positions to analyse and provide best moves for according to
// it's evaluation function(s)
// An engine can have internal parameters which can be changed
// from an external source. In order to communicate with the engine
// protocol agnostically, communications are handled through a
// communicator interface where specific implimentations impliment
// specific protocols
type Engine struct {
	// Used for interacting with the engine
	Path         string
	cmd          *exec.Cmd
	communicator Protocol
	// Information provided by the engine
	Name    string
	Author  string
	Options map[string]Option
	// Current engine state
	ready    bool
	thinking bool
}

// NewEngine creates a new engine, esablishes a connection with it
// and performs a handshake with it to make sure it supports the
// protocol the communicator impliments.
// During the handshake, engine specific information should
// be provided by the engine and extracted into the datastructure
// If: the engine is not found; a connection couldn't be established
// or the protocol handshake failed, an error will be returned
func NewEngine(path string, protocol func(*exec.Cmd) (Protocol, error)) (*Engine, error) {
	// Checking if the engine file exists
	path = filepath.Join(EngineDirectory, path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "couldn't find engine")
	} else if err != nil {
		return nil, errors.Wrap(err, "couldn't find engine")
	}
	// Making engine struct
	engine := Engine{
		Path:    path,
		cmd:     exec.Command(path),
		Options: make(map[string]Option),
	}
	// Establishing connection to engine
	var err error
	engine.communicator, err = protocol(engine.cmd)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create communicator")
	}
	return &engine, nil
}

// Load starts the engine process and performs a handshake
// using the protocol implimentation of the communicator
func (e *Engine) Load() error {
	// Starting engine
	if err := e.cmd.Start(); err != nil {
		return errors.Wrap(err, "couldn't start engine")
	}
	// Performing protocol handshake
	err := e.communicator.Handshake(
		&e.Name,
		&e.Author,
		&e.Options,
	)
	if err != nil {
		return errors.Wrap(err, "protocol handshake failed")
	}
	// Engine started successfully
	e.ready = true
	return nil
}

// Debug enables and disables the engine's debug mode
func (e *Engine) Debug(enable bool) error {
	if !e.ready {
		return errors.New("engine is not ready")
	}
	return e.communicator.Debug(enable)
}

// SetOption sets an internal parameter of the engine
func (e *Engine) SetOption(o Option) error {
	if !e.ready {
		return errors.New("engine is not ready")
	}
	if _, ok := e.Options[o.OptionName()]; !ok {
		return errors.New("option not specified by engine")
	}
	e.Options[o.OptionName()] = o
	return e.communicator.SetOption(o)
}

// NewGame tells the engine that the next position it
// will receive is from a different game to the
// previous position it was provided
func (e *Engine) NewGame() error {
	if !e.ready {
		return errors.New("engine is not ready")
	}
	return e.communicator.NewGame()
}

// Position gives the engine a new position to analyse
func (e *Engine) Position(s State) error {
	if !e.ready {
		return errors.New("engine is not ready")
	}
	return e.communicator.Position(s)
}

// Go tells the engine to start analysing the last position
// it was provided
// If moveTime is positive, the engine will be told that it
// has moveTime seconds to analyse the position before it
// will be asked to stop and provide its best move
func (e *Engine) Go(moveTime time.Duration) error {
	if !e.ready {
		return errors.New("engine is not ready")
	}
	if e.thinking {
		return errors.New("engine is thinking")
	}
	e.thinking = true
	return e.communicator.Go(moveTime)
}

// Stop tells the engine to stop analysing the position
// as soon as posible and to provide a best move
func (e *Engine) Stop() (int, error) {
	if !e.ready {
		return 0, errors.New("engine is not ready")
	}
	if !e.thinking {
		return 0, errors.New("engine is not thinking")
	}
	e.thinking = false
	bestMove, err := e.communicator.Stop()
	return bestMove, err
}

// Quit tells the engine to exit as soon as possible
// then terminates the process
// If the engine doesn't quit by itself, the program
// will hang here. As killing the process seems
// a little excessive and possibly dangerous
// I may change this after a little more research
func (e *Engine) Quit() error {
	if !e.ready {
		return errors.New("engine is not ready")
	}
	e.ready = false
	err := e.communicator.Quit()
	if err != nil {
		return errors.Wrap(err, "couldn't stop engine communicator")
	}
	return e.cmd.Wait() // goroutine may hang here
}

// NotifyInfo sets the channel in which any information
// from the engine should be sent to
func (e *Engine) NotifyInfo(channel chan<- string) {
	e.communicator.NotifyInfo(channel)
}

// NotifyComm sets the channel in which communications between
// the protocol implimentation and the engine should be send to
func (e *Engine) NotifyComm(channel chan<- Communication) {
	e.communicator.NotifyComm(channel)
}

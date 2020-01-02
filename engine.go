package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	// CfpOkTimeout is the time the GUI will wait for a cfpok command
	CfpOkTimeout = 10.0
	// ReadyOkTimeout is the time the GUI will wait for a readyok command
	ReadyOkTimeout = 10.0
	// BestMoveTimeout is the time the GUI will wait for a bestmove command
	BestMoveTimeout = 10.0
)

// CFPEngine is a connection to an engine process that supports the CFP protocol
type CFPEngine struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	ready  bool

	name      string
	author    string
	searching bool

	cfpok        chan bool
	readyok      chan bool
	bestmove     chan int
	infoCallback func(string)

	options map[string]Option
}

// NewCFPEngine creates a new CFPEngine object and starts running the binary at the path.
// An error will be returned if the file doesn't exist, if it's not executable or the
// engine doesn't support the cfp protocol i.e. doesn't perform the handshake.
func NewCFPEngine(path string) (*CFPEngine, error) {
	var err error
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "couldn't find engine")
	} else if err != nil {
		return nil, errors.Wrap(err, "couldn't find engine")
	}
	engine := new(CFPEngine)
	engine.cmd = exec.Command(path)
	if engine.stdin, err = engine.cmd.StdinPipe(); err != nil {
		return nil, errors.Wrap(err, "couldn't open stdin pipe")
	}
	if engine.stdout, err = engine.cmd.StdoutPipe(); err != nil {
		return nil, errors.Wrap(err, "couldn't open stdout pipe")
	}
	if err := engine.cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "couldn't start engine")
	}
	engine.startListening()
	engine.options = make(map[string]Option)
	if err := engine.handshake(); err != nil {
		return nil, errors.Wrap(err, "couldn't perform cfp handshake")
	}
	engine.ready = true
	return engine, nil
}

func (engine *CFPEngine) startListening() {
	engine.cfpok = make(chan bool, 1)
	engine.readyok = make(chan bool, 1)
	engine.bestmove = make(chan int, 1)
	go func() {
		scanner := bufio.NewScanner(engine.stdout)
		for scanner.Scan() {
			engine.command(strings.ToLower(scanner.Text()))
		}
	}()
}

// I know, I know. I'll clean this up later!
func (engine *CFPEngine) command(command string) {
	args := strings.Split(command, " ")
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "id":
		if len(args) < 3 {
			return
		}
		value := strings.Join(args[2:], " ")
		switch args[1] {
		case "name":
			engine.name = value
		case "author":
			engine.author = value
		}
	case "cfpok":
		engine.cfpok <- true
	case "readyok":
		engine.readyok <- true
	case "bestmove":
		if len(args) < 2 {
			return
		}
		number, err := strconv.Atoi(args[1])
		if err != nil {
			return
		}
		engine.bestmove <- number
	case "info":
		if engine.infoCallback == nil {
			return
		}
		value := strings.Join(args[1:], " ")
		engine.infoCallback(value)
	case "option":
		option, err := OptionFromCFP(args[1:])
		if err != nil {
			fmt.Println("unable to add option", err)
			return
		}
		err = engine.addOption(option)
		if err != nil {
			fmt.Println("unable to add option", err)
			return
		}
	}
}

func (engine *CFPEngine) waitForCFPOk() error {
	select {
	case <-engine.cfpok:
		return nil
	case <-time.After(CfpOkTimeout * time.Second):
		return errors.New("cfpok took too long")
	}
}

func (engine *CFPEngine) waitForReadyOk() error {
	select {
	case <-engine.readyok:
		return nil
	case <-time.After(ReadyOkTimeout * time.Second):
		return errors.New("readyok took too long")
	}
}

func (engine *CFPEngine) waitForBestMove() (int, error) {
	select {
	case v := <-engine.bestmove:
		return v, nil
	case <-time.After(BestMoveTimeout * time.Second):
		return 0, errors.New("readyok took too long")
	}
}

func (engine *CFPEngine) handshake() error {
	if _, err := engine.stdin.Write([]byte("cfp\n")); err != nil {
		return errors.Wrap(err, "unable to send command")
	}
	if err := engine.waitForCFPOk(); err != nil {
		return errors.Wrap(err, "engine doesn't support cfp")
	}
	if engine.name == "" || engine.author == "" {
		return errors.New("name and/or author not provided")
	}
	return nil
}

func (engine *CFPEngine) addOption(option Option) error {
	if _, ok := engine.options[option.Name]; ok {
		return errors.New("option already exists")
	}
	engine.options[option.Name] = option
	return nil
}

// SetInfoCallback allows you to specify which function should be called when
// the engine sends an info command.
func (engine *CFPEngine) SetInfoCallback(callback func(string)) {
	engine.infoCallback = callback
}

// EnableDebug sends a debug on command to the engine.
func (engine *CFPEngine) EnableDebug() error {
	if !engine.ready {
		return errors.New("engine is not loaded")
	}
	if _, err := engine.stdin.Write([]byte("debug on\n")); err != nil {
		return errors.Wrap(err, "unable to send command")
	}
	return nil
}

// DisableDebug sends a debug off command to the engine.
func (engine *CFPEngine) DisableDebug() error {
	if !engine.ready {
		return errors.New("engine is not loaded")
	}
	if _, err := engine.stdin.Write([]byte("debug off\n")); err != nil {
		return errors.Wrap(err, "unable to send command")
	}
	return nil
}

// Ping checks if the connection to the engine is still alive
func (engine *CFPEngine) Ping() error {
	if !engine.ready {
		return errors.New("engine is not loaded")
	}
	if _, err := engine.stdin.Write([]byte("isready\n")); err != nil {
		return errors.Wrap(err, "unable to send command")
	}
	if err := engine.waitForReadyOk(); err != nil {
		return errors.Wrap(err, "error communicating with engine")
	}
	return nil
}

// SetOption sends a setoption command to the engine.
// An error will be returned if the engine didn't identify the
// option you're try to set or if the value is of the wrong type.
func (engine *CFPEngine) SetOption(name string, value interface{}) error {
	if !engine.ready {
		return errors.New("engine is not loaded")
	}
	option, ok := engine.options[name]
	if !ok {
		return errors.New("option doesn't exist")
	}
	command, err := option.CFPSetOption(value)
	if err != nil {
		return errors.Wrap(err, "unable to make setoption command")
	}
	if _, err := engine.stdin.Write([]byte(command + "\n")); err != nil {
		return errors.Wrap(err, "unable to send command")
	}
	return nil
}

// NewGame sends a cfpnewgame command to the engine.
func (engine *CFPEngine) NewGame() error {
	if !engine.ready {
		return errors.New("engine is not loaded")
	}
	if _, err := engine.stdin.Write([]byte("cfpnewgame\n")); err != nil {
		return errors.Wrap(err, "unable to send command")
	}
	return nil
}

// UpdatePosition sends a position command to the engine.
func (engine *CFPEngine) UpdatePosition(position, moves string) error {
	if !engine.ready {
		return errors.New("engine is not loaded")
	}
	if moves != "" {
		position += " " + moves
	}
	if _, err := engine.stdin.Write([]byte("position " + position + "\n")); err != nil {
		return errors.Wrap(err, "unable to send command")
	}
	return engine.Ping()
}

// Go sends a go command to the engine. This signifies it should start thinking.
// moveTime is the amount of time in miliseconds that the engine has to think.
// If moveTime is -1, the engine should keep thinking until a stop command is sent.
func (engine *CFPEngine) Go(moveTime int) error {
	if !engine.ready {
		return errors.New("engine is not loaded")
	}
	if engine.searching {
		return errors.New("engine is already searching")
	}
	message := "go"
	if moveTime > -1 {
		message += fmt.Sprintf("movetime %d", moveTime)
	}
	engine.searching = true
	if _, err := engine.stdin.Write([]byte(message + "\n")); err != nil {
		return errors.Wrap(err, "unable to send command")
	}
	return nil
}

// Stop sends a stop command to the engine. The engine should stop thinking
// as soon as possible and send a bestmove command back to the GUI.
func (engine *CFPEngine) Stop() (int, error) {
	if !engine.ready {
		return 0, errors.New("engine is not loaded")
	}
	if !engine.searching {
		return 0, errors.New("engine is not searching")
	}
	engine.searching = false
	if _, err := engine.stdin.Write([]byte("stop\n")); err != nil {
		return 0, errors.Wrap(err, "unable to send command")
	}
	result, err := engine.waitForBestMove()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't get best move")
	}
	return result, nil
}

// Close stops the engine and closes the process.
func (engine *CFPEngine) Close() error {
	if engine.searching {
		engine.Stop()
	}
	engine.ready = false
	engine.stdin.Write([]byte("quit\n"))
	return engine.cmd.Wait()
}

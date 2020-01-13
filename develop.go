package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Develop is a frontend which contains a single game
// The user can load different engines and play two of
// them against each other. The interface is a web application
// served via Develop.server
type Develop struct {
	// engines is a map containing all of the loaded engines
	engines map[int]*Engine
	// nextEngineID is the id allocated for the next engine
	// that is loaded
	nextEngineID int
	// player1EngineID is the id of the engine currently
	// selected to be player1 in the game
	player1EngineID int
	// player2EngineID is the id of the engine currently
	// selected to be player2 in the game
	player2EngineID int
	// game is the game which is being played
	game *Game
	// server is used to serve the user with the frontend
	server *Server
}

// NewDevelop creates a new Develop struct which is
// ready to serve the user with a frontend
func NewDevelop() (*Develop, error) {
	// Creating a new server
	s, err := NewServer("develop")
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make server")
	}
	// Adding the result of the features to the result
	return &Develop{
		engines:         make(map[int]*Engine),
		nextEngineID:    0,
		player1EngineID: -1,
		player2EngineID: -1,
		game:            NewGame(),
		server:          s,
	}, nil
}

// Start tells the Develop to start serving content
// Start is not expected to exit unless the process is killed
// or an error occurs, thus it always returns an error
func (d *Develop) Start() error {
	// Set up event listeners
	go d.listenToClients()
	go d.listenToGame()
	// Start the server
	return d.server.Start()
}

// listenToEngineInfo handles any info
// events sent from an engine
func (d *Develop) listenToEngineInfo(e *Engine) {
	// Make channel to receive events
	channel := make(chan string)
	e.NotifyInfo(channel)
	for {
		// Get info from channel
		info, ok := <-channel
		if !ok {
			return
		}
		// Output it to all clients
		d.server.TriggerEvent(ServerEvent{
			WSCommand: fmt.Sprintf(
				"output time %s sender %s message %s",
				FormatTime(time.Now()), e.Name, info,
			),
		})
	}
}

// listenToEngineComm handles any communications
// between an engine and the gui
func (d *Develop) listenToEngineComm(e *Engine) {
	// Make channel to receive events
	channel := make(chan Communication)
	e.NotifyComm(channel)
	for {
		// Get communication from channel
		comm, ok := <-channel
		if !ok {
			return
		}
		// Output it to all clients
		d.server.TriggerEvent(ServerEvent{
			WSCommand: fmt.Sprintf(
				"communication time %s engine %s toengine %t message %s",
				FormatTime(comm.Time), e.Name, comm.ToEngine, comm.Message,
			),
		})
	}
}

// listenToGame handles any game events that
// happen while the game is running
func (d *Develop) listenToGame() {
	// Make channel to receive game events
	channel := make(chan GameEvent)
	d.game.NotifyEvents(channel)
	for {
		// Get game event
		evt, ok := <-channel
		if !ok {
			return
		}
		// Figure out which type of event occured
		switch v := evt.(type) {
		case GameOverEvent:
			// If the game is over, tell each client
			d.server.TriggerEvent(ServerEvent{
				WSCommand: fmt.Sprintf("gameover winner %d", v.Winner),
			})
			// Send output command
			d.server.TriggerEvent(ServerEvent{WSCommand: fmt.Sprintf(
				"output time %s sender %s message %s",
				FormatTime(time.Now()), "INFO", "Game has finished",
			)})
		case NewStateEvent:
			// If there is a new position that has been reached,
			// tell each of the clients
			d.server.TriggerEvent(ServerEvent{
				WSCommand: fmt.Sprintf("position %s", v.State.CFPString()),
			})
		case ErrorEvent:
			// If there has been an error, tell each client
			d.server.TriggerEvent(ServerEvent{
				WSCommand: fmt.Sprintf(
					"output time %s sender %s message %s",
					FormatTime(time.Now()), "ERROR", v.Error.Error(),
				),
			})
		}
	}
}

// listenToClients handles any incoming commands from
// any of the connected clients
func (d *Develop) listenToClients() {
	// Make channel to receive client events
	channel := make(chan ClientEvent)
	d.server.NotifyClientEvents(channel)
	for {
		// Get the event
		evt, ok := <-channel
		if !ok {
			return
		}
		// Seperate the event command into its arguments
		args := strings.Split(evt.WsCommand, " ")
		// Exit early if there are no arguments
		if len(args) == 0 {
			continue
		}
		// Figure out which type of command has been received
		// and execute the respective function
		switch strings.ToLower(args[0]) {
		case "init":
			d.initRequest(evt)
		case "newgame":
			d.newGameRequest(evt)
		case "setplayers":
			d.setPlayersRequest(evt, args[1:])
		case "play":
			d.playRequest(evt)
		case "pause":
			d.pauseRequest(evt)
		case "enginepaths":
			d.enginePathsRequest(evt)
		case "engine":
			d.engineEventRequest(evt, args[1:])
		}
	}
}

// initRequest handles any init commands sent from a client
func (d *Develop) initRequest(evt ClientEvent) {
	// Send engine load commands
	for k, v := range d.engines {
		d.server.Respond(evt, fmt.Sprintf(
			"engine load id %d name %s author %s",
			k, v.Name, v.Author,
		))
	}
	// Send player set command
	d.server.Respond(evt, fmt.Sprintf(
		"players player1 %d player2 %d",
		d.player1EngineID, d.player2EngineID,
	))
	// Send game history commands
	d.server.Respond(evt, "newgame")
	for i := 0; i <= d.game.HistoryIndex; i++ {
		d.server.Respond(evt, "position "+d.game.History[i].CFPString())
	}
	// Send output command
	d.server.Respond(evt, fmt.Sprintf(
		"output time %s sender %s message %s",
		FormatTime(time.Now()), "INFO", "Connected successfully",
	))
}

// newGameRequest handles any newgame commands sent from a client
func (d *Develop) newGameRequest(evt ClientEvent) {
	// Try to execute newgame
	err := d.newGame()
	// If there is an error, respond to the client
	if err != nil {
		d.server.Respond(evt, fmt.Sprintf(
			"output time %s sender %s message %s",
			FormatTime(time.Now()), "ERROR", err.Error(),
		))
	}
}

// setPlayerRequest handles any setplayer commands sent from clients
// args is a slice of arguments that follow setplayers in the command
func (d *Develop) setPlayersRequest(evt ClientEvent, args []string) {
	// Find the index of the string 'player1'
	player1Index := SliceIndex(len(args), func(i int) bool {
		return strings.ToLower(args[i]) == "player1"
	})
	// If it's not found, respond with an error
	if player1Index == -1 {
		d.respondError(evt, errors.New("couldn't find player1 in command string"))
		return
	}
	// Find the index of the string 'player2'
	player2Index := SliceIndex(len(args), func(i int) bool {
		return strings.ToLower(args[i]) == "player2"
	})
	// If it's not found, respond with an error
	if player2Index == -1 {
		d.respondError(evt, errors.New("couldn't find player2 in command string"))
		return
	}
	// Get the string values of the parameters
	player1Str := strings.Join(args[player1Index+1:player2Index], " ")
	player2Str := strings.Join(args[player2Index+1:len(args)], " ")
	// Attempt to convert the player1 parameter to an integer
	player1, err := strconv.Atoi(player1Str)
	// If conversion fails, respond with an error
	if err != nil {
		d.respondError(evt, errors.Wrap(err, "couldn't get player1"))
		return
	}
	// Attempt to convert the player2 parameter to an integer
	player2, err := strconv.Atoi(player2Str)
	// If conversion fails, respond with an error
	if err != nil {
		d.respondError(evt, errors.Wrap(err, "couldn't get player2"))
		return
	}
	// Try to set the players
	err = d.setPlayers(player1, player2)
	// If an error occurs, respond with that error
	if err != nil {
		d.respondError(evt, errors.Wrap(err, "couldn't set players"))
		return
	}
}

// playRequest handles any play commands sent from clients
func (d *Develop) playRequest(evt ClientEvent) {
	// Try to play the game
	err := d.play()
	// If that fails, respond with an error
	if err != nil {
		d.respondError(evt, errors.Wrap(err, "couldn't play game"))
	}
}

// pauseRequest handles any pause command sent from clients
func (d *Develop) pauseRequest(evt ClientEvent) {
	// Try to pause the game
	err := d.pause()
	// If that fails, respond with an error
	if err != nil {
		d.respondError(evt, errors.Wrap(err, "couldn't pause game"))
	}
}

// enginePathsRequest handles and enginepaths commands sent from clients
func (d *Develop) enginePathsRequest(evt ClientEvent) {
	// Get paths to all files within engine directory
	files, err := FilesAt(EngineDirectory)
	if err != nil {
		d.server.Respond(evt, "noenginepaths")
		d.respondError(evt, errors.Wrap(err, "couldn't get engine paths"))
		return
	}
	// Remove any file paths to engines that are already loaded
OUTER:
	for i := len(files) - 1; i >= 0; i-- {
		v := filepath.Join(EngineDirectory, files[i])
		for _, e := range d.engines {
			if e.Path == v {
				files[i] = files[len(files)-1]
				files = files[:len(files)-1]
				continue OUTER
			}
		}
	}
	// Send response to client
	if len(files) == 0 {
		d.server.Respond(evt, "noenginepaths")
		d.respondError(evt, errors.New("no engines in engines directory"))
	} else {
		d.server.Respond(evt, "enginepaths path "+strings.Join(files, " path "))
	}
}

// engineEventRequest handles any engine operation commands sent from clients
func (d *Develop) engineEventRequest(evt ClientEvent, args []string) {
	// If there are no arguments, forget about it
	if len(args) == 0 {
		return
	}
	// Figure out if this is a load or unload operation
	switch strings.ToLower(args[0]) {
	case "load":
		d.engineLoadRequest(evt, args[1:])
	case "unload":
		d.engineUnloadRequest(evt, args[1:])
	}
}

// engineLoadRequest handles any engine load command sent from clients
func (d *Develop) engineLoadRequest(evt ClientEvent, args []string) {
	// If there are no arguments, forget about it
	if len(args) == 0 {
		return
	}
	// Find the index of the string 'path' in args
	pathIndex := SliceIndex(len(args), func(i int) bool {
		return args[i] == "path"
	})
	// If it isn't found, respond with an error
	if pathIndex == -1 {
		d.respondError(evt, errors.New("couldn't find path in command string"))
		return
	}
	// Create path and attempt to load the engine
	path := strings.Join(args[pathIndex+1:len(args)], " ")
	err := d.loadEngine(path)
	if err != nil {
		d.respondError(evt, errors.Wrap(err, "couldn't load engine"))
	}
}

// engineUnloadRequest handles any engine unload operations sent from clients
func (d *Develop) engineUnloadRequest(evt ClientEvent, args []string) {
	// If there are no arguments, forget about it
	if len(args) == 0 {
		return
	}
	// Find the index of the string 'id' in args
	idIndex := SliceIndex(len(args), func(i int) bool {
		return args[i] == "id"
	})
	// If it isn't found, respond with an error
	if idIndex == -1 {
		d.respondError(evt, errors.New("couldn't find id in command string"))
		return
	}
	// Try to convert the parameter into an integer
	idString := strings.Join(args[idIndex+1:len(args)], " ")
	id, err := strconv.Atoi(idString)
	if err != nil {
		d.respondError(evt, errors.Wrap(err, "couldn't convert id into integer"))
		return
	}
	// Try to unload the engine with specified id
	err = d.unloadEngine(id)
	if err != nil {
		d.respondError(evt, errors.Wrap(err, "couldn't unload engine"))
	}
}

// newGame starts a new game
func (d *Develop) newGame() error {
	// Try to reset the game
	err := d.game.Reset()
	if err != nil {
		return errors.Wrap(err, "couldn't start new game")
	}
	// Send server events to all clients
	d.server.TriggerEvent(ServerEvent{WSCommand: "newgame"})
	d.server.TriggerEvent(ServerEvent{WSCommand: "position " + d.game.State.CFPString()})
	// Send output command
	d.server.TriggerEvent(ServerEvent{WSCommand: fmt.Sprintf(
		"output time %s sender %s message %s",
		FormatTime(time.Now()), "INFO", "Game has been reset",
	)})
	return nil
}

// setPlayers sets the players which are to play the game
func (d *Develop) setPlayers(player1, player2 int) error {
	// Space to store values
	var (
		engine1 *Engine
		engine2 *Engine
		ok      bool
	)
	// Get the engine that is to be player1
	if player1 != -1 {
		engine1, ok = d.engines[player1]
		if !ok {
			return errors.New("no engine with that id")
		}
	}
	// Get the engine that is to be player2
	if player2 != -1 {
		engine2, ok = d.engines[player2]
		if !ok {
			return errors.New("no engine with that id")
		}
	}
	var err error
	// Try to set player1
	if engine1 != nil {
		err = d.game.SetPlayer1(engine1)
		if err != nil {
			return errors.Wrap(err, "couldn't set player1")
		}
		d.player1EngineID = player1
	}
	// Try to set player2
	if engine2 != nil {
		err = d.game.SetPlayer2(engine2)
		if err != nil {
			return errors.Wrap(err, "couldn't set player2")
		}
		d.player2EngineID = player1
	}
	// If this operation updated anything, send update to all clients
	if engine1 != nil || engine2 != nil {
		d.server.TriggerEvent(ServerEvent{WSCommand: fmt.Sprintf(
			"players player1 %d player2 %d",
			player1, player2,
		)})
		// Send output command
		d.server.TriggerEvent(ServerEvent{WSCommand: fmt.Sprintf(
			"output time %s sender %s message %s",
			FormatTime(time.Now()), "INFO", "New players have been set",
		)})
	}
	return nil
}

// play starts the game playing
func (d *Develop) play() error {
	// Attempt to set the game playing
	err := d.game.Play()
	if err != nil {
		return errors.Wrap(err, "couldn't play game")
	}
	// Tell the clients that the game is going
	d.server.TriggerEvent(ServerEvent{WSCommand: "play"})
	// Send output command
	d.server.TriggerEvent(ServerEvent{WSCommand: fmt.Sprintf(
		"output time %s sender %s message %s",
		FormatTime(time.Now()), "INFO", "Started playing game",
	)})
	return nil
}

// pause pauses the game mid play
func (d *Develop) pause() error {
	// Attempt to pause the game
	err := d.game.Pause()
	if err != nil {
		return errors.Wrap(err, "couldn't pause game")
	}
	// Tell the clients that the game is paused
	d.server.TriggerEvent(ServerEvent{WSCommand: "pause"})
	// Send output command
	d.server.TriggerEvent(ServerEvent{WSCommand: fmt.Sprintf(
		"output time %s sender %s message %s",
		FormatTime(time.Now()), "INFO", "Paused game",
	)})
	return nil
}

// loadEngine loads an engine with a specified path
// Note: the path is RELATIVE to the EngineDirectory in config.go
func (d *Develop) loadEngine(path string) error {
	// Try to create engine
	engine, err := NewEngine(path, CFP)
	if err != nil {
		return errors.Wrap(err, "couldn't create engine")
	}
	// Set up engine event handlers
	go d.listenToEngineInfo(engine)
	go d.listenToEngineComm(engine)
	// Load the engine
	err = engine.Load()
	if err != nil {
		return errors.Wrap(err, "couldn't start engine")
	}
	// Store the engine in the loaded engines map
	d.engines[d.nextEngineID] = engine
	// Tell clients that engine is loaded
	d.server.TriggerEvent(ServerEvent{WSCommand: fmt.Sprintf(
		"engine load id %d name %s author %s",
		d.nextEngineID, engine.Name, engine.Author,
	)})
	// Send output command
	d.server.TriggerEvent(ServerEvent{WSCommand: fmt.Sprintf(
		"output time %s sender %s message %s",
		FormatTime(time.Now()), "INFO", "Engine loaded successfully",
	)})
	d.nextEngineID++
	return nil
}

// unloadEngine unloads a loaded engine with a specified id
func (d *Develop) unloadEngine(id int) error {
	// If the engine is player1, set player1 to nil
	if d.player1EngineID == id {
		err := d.game.SetPlayer1(nil)
		if err != nil {
			return errors.Wrap(err, "couldn't disable player1 for engine")
		}
	}
	// If the engine is player2, set player2 to nil
	if d.player2EngineID == id {
		err := d.game.SetPlayer2(nil)
		if err != nil {
			return errors.Wrap(err, "couldn't disable player2 for engine")
		}
	}
	// Get the engine. ok will be false if the engine isn't loaded
	engine, ok := d.engines[id]
	if !ok {
		return errors.New("no engine with that id")
	}
	// Tell the engine to quit
	err := engine.Quit()
	if err != nil {
		return errors.Wrap(err, "couldn't make engine quit")
	}
	// Delete the engine from the loaded engines map
	delete(d.engines, id)
	// Tell the clients the engine has been unloaded
	d.server.TriggerEvent(ServerEvent{WSCommand: fmt.Sprintf(
		"engine unload id %d", id,
	)})
	// Send output command
	d.server.TriggerEvent(ServerEvent{WSCommand: fmt.Sprintf(
		"output time %s sender %s message %s",
		FormatTime(time.Now()), "INFO", "Engine has been disconnected",
	)})
	return nil
}

// respondError responds to a client event with an error
func (d *Develop) respondError(evt ClientEvent, err error) {
	d.server.Respond(evt, fmt.Sprintf(
		"output time %s sender %s message %s",
		FormatTime(time.Now()), "ERROR", err.Error(),
	))
}

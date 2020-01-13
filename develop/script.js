"use strict"

// Protocol objects
let socket;                 // The websocket connection to the GUI
let state;                  // The current state of the GUI

// GUI object
let gui;

// Constants for state
const EMPTY     = 0;
const PLAYER_1  = 1;
const PLAYER_2  = 2;
const TIE       = 3;

// Constants for controls (Button IDs)
const LOAD_ENGINE_BUTTON    = 0;
const NEW_GAME_BUTTON       = 1;
const SETUP_BOARD_BUTTON    = 2;
const START_BUTTON          = 3;
const PREVIOUS_BUTTON       = 4;
const PLAY_PAUSE_BUTTON     = 5;
const NEXT_BUTTON           = 6;
const END_BUTTON            = 7;
const GO_BACK_BUTTON        = 8;

// Constants for engine specific controls
const ENGINE_BUTTONS_START      = 9;
const ENGINE_BUTTONS_STRIDE     = 3;
const ENGINE_PLAYER1_BUTTON     = 0;
const ENGINE_PLAYER2_BUTTON     = 1;
const ENGINE_DISCONNECT_BUTTON  = 2;

// GUI State
class State {
    constructor() {
        this.engines        = {};
        this.player1ID      = -1;
        this.player2ID      = -1;

        this.playing        = false;
    
        this.history        = [];
        this.gameOver       = false;
        this.winner         = null;

        this.historyIndex   = 0;
    }

    loadEngine(engine) {
        this.engines["engine"+engine.id] = engine;
    }

    unloadEngine(engineID) {
        delete this.engines["engine"+engine.id];
        if (this.player1ID == engineID)
            this.player1ID = -1;
        if (this.player2ID == engineID)
            this.player2ID = -1;
    }

    updatePlayers(engineID1, engineID2) {
        this.player1ID = engineID1;
        this.player2ID = engineID2;
    }

    newGame() {
        this.history        = [];
        this.historyIndex   = 0;
        this.gameOver       = false;
        this.winner         = null;
    }

    updatePosition(position) {
        if (this.historyIndex == this.history.length - 1) {
            this.historyIndex++;
        }
        this.history.push(position);
    }

    gameOver(winner) {
        this.playing    = false;
        this.gameOver   = true;
        this.winner     = winner;
    }

    play() {
        this.playing = true;
    }

    pause() {
        this.playing = false;
    }
}

function buttonClick() {
    if (this.classList.contains("disabled"))
        return;
    switch (this.buttonId) {
    case LOAD_ENGINE_BUTTON:
        gui.showOverlay();
        requestEnginePaths();
        return;
    case NEW_GAME_BUTTON:
        requestNewGame();
        return;
    case SETUP_BOARD_BUTTON:
        // Not Implimented
        return;
    case START_BUTTON:
        state.historyIndex = 0;
        gui.updateButtons();
        return;
    case PREVIOUS_BUTTON:
        if (state.historyIndex > 0) {
            state.historyIndex--;
            gui.updateButtons();
        }
        return;
    case PLAY_PAUSE_BUTTON:
        if (state.playing) {
            requestPause();
        } else {
            requestPlay();
        }
        return;
    case NEXT_BUTTON:
        if (state.historyIndex < state.history.length-1) {
            state.historyIndex++;
            gui.updateButtons();
        }
        return;
    case END_BUTTON:
        if (state.historyIndex < state.history.length-1) {
            state.historyIndex = state.history.length-1;
            gui.updateButtons();
        }
        return;
    case GO_BACK_BUTTON:
        gui.hideOverLay();
        return;
    }
    // If we reach this point, it's en engine specific button
    let tmp         = this.buttonId - ENGINE_BUTTONS_START;
    let button      = tmp % ENGINE_BUTTONS_STRIDE;
    let engineId    = Math.floor(tmp / ENGINE_BUTTONS_STRIDE);
    requestEngineOperation(engineId, button);
}   

// Controls all of the visuals of the GUI
// Will rely on `state` heavily.
// Make sure `state` is set correctly before calling any methods!
class GUI {
    constructor() {
        // Space to store the engine DOM elements
        this.engines            = {};
        
        // Getting DOM elements
        this.loadOverlay        = document.getElementById("load-engine-overlay");
        this.overlayLoader      = document.getElementsByClassName("loader")[0];
        this.loadList           = document.getElementById("load-engine-list");
        this.goBackButton       = document.getElementsByClassName("go-back-button")[0];

        this.engineList         = document.getElementById("engine-list");
        
        this.loadEngineButton   = document.getElementById("load-engine-button");
        
        this.newGameButton      = document.getElementById("new-game");
        this.setupBoardButton   = document.getElementById("setup-board");
        
        this.canvas             = document.getElementById("game-screen-canvas");

        this.startButton        = document.getElementById("start");
        this.previousButton     = document.getElementById("previous");
        this.playPauseButton    = document.getElementById("play-pause");
        this.nextButton         = document.getElementById("next");
        this.endButton          = document.getElementById("end");
        
        this.outputTerminal         = document.getElementById("output-terminal").getElementsByTagName("p")[0];
        this.communicationTerminal  = document.getElementById("communications-terminal").getElementsByTagName("p")[0];

        // Getting canvas drawing context
        this.canvas.width   = 800;
        this.canvas.height  = 500;
        this.ctx            = this.canvas.getContext("2d");

        // Speaks for itself
        this.addEventHandlers();
    }

    addEventHandlers() {
        this.loadEngineButton.buttonId  = LOAD_ENGINE_BUTTON;
        this.newGameButton.buttonId     = NEW_GAME_BUTTON;
        this.setupBoardButton.buttonId  = SETUP_BOARD_BUTTON;
        this.startButton.buttonId       = START_BUTTON;
        this.previousButton.buttonId    = PREVIOUS_BUTTON;
        this.playPauseButton.buttonId   = PLAY_PAUSE_BUTTON;
        this.nextButton.buttonId        = NEXT_BUTTON;
        this.endButton.buttonId         = END_BUTTON;
        this.goBackButton.buttonId      = GO_BACK_BUTTON;

        this.loadEngineButton.addEventListener("click", buttonClick, false);
        this.newGameButton.addEventListener("click", buttonClick, false);
        this.setupBoardButton.addEventListener("click", buttonClick, false);
        this.startButton.addEventListener("click", buttonClick, false);
        this.previousButton.addEventListener("click", buttonClick, false);
        this.playPauseButton.addEventListener("click", buttonClick, false);
        this.nextButton.addEventListener("click", buttonClick, false);
        this.endButton.addEventListener("click", buttonClick, false);
        this.goBackButton.addEventListener("click", buttonClick, false);
    }

    showOverlay() {
        let filepaths = this.loadList.getElementsByClassName("file-path");
        for (let i = filepaths.length-1; i >= 0; i--) {
            filepaths[i].remove();
        }
        this.goBackButton.style.display     = "none";
        this.overlayLoader.style.display    = "block";
        this.loadOverlay.style.display      = "flex";
    }
    
    hidePathLoader() {
        this.goBackButton.style.display     = "block";
        this.overlayLoader.style.display    = "none";

    }

    showFilepath(name) {
        let filepath = document.createElement("li");
        filepath.classList.add("file-path");
        let fileinfo = document.createElement("section");
        fileinfo.classList.add("file-info");
        let pathtext = document.createElement("p");
        pathtext.innerHTML = "<span class=\"prefix\">engines/</span>" + name;
        fileinfo.appendChild(pathtext);
        filepath.appendChild(fileinfo);
        let filebutton = document.createElement("section");
        filebutton.classList.add("file-button");
        filebutton.innerHTML = "<p>Load</p>";
        filepath.appendChild(filebutton);
        this.loadList.insertBefore(filepath, this.goBackButton);
        filebutton.addEventListener("click", () => {
            requestEngineLoad(name);
            gui.hideOverLay();
        });
    }

    hideOverLay() {
        this.loadOverlay.style.display = "none";
    }

    loadEngine(engine) {
        if (this.engines["engine"+engine.id] != null) {
            this.engines["engine"+engine.id].remove();
            delete this.engines["engine"+engine.id];
        }
        let index = ENGINE_BUTTONS_START + engine.id * ENGINE_BUTTONS_STRIDE;
        this.engines["engine"+engine.id] = document.createElement("section");
        this.engines["engine"+engine.id].classList.add("engine");
        let engineInfo = document.createElement("div");
        engineInfo.classList.add("engine-info");
        let center = document.createElement("div");
        center.classList.add("center");
        let engineName = document.createElement("p");
        engineName.classList.add("engine-name");
        engineName.innerHTML = engine.name;
        let engineAuthor = document.createElement("p");
        engineAuthor.classList.add("engine-author");
        engineAuthor.innerHTML = "by " + engine.author;
        center.appendChild(engineName);
        center.appendChild(engineAuthor);
        engineInfo.appendChild(center);
        let player1 = document.createElement("div");
        player1.classList.add("engine-player1");
        player1.innerHTML = "<h3>P1</h3>";
        player1.buttonId = index + ENGINE_PLAYER1_BUTTON;
        player1.addEventListener("click", buttonClick, false);
        let player2 = document.createElement("div");
        player2.classList.add("engine-player2");
        player2.innerHTML = "<h3>P2</h3>";
        player2.buttonId = index + ENGINE_PLAYER2_BUTTON;
        player2.addEventListener("click", buttonClick, false);
        let disconnect = document.createElement("div");
        disconnect.classList.add("engine-disconnect");
        disconnect.innerHTML = "<h3>DC</h3>";
        disconnect.buttonId = index + ENGINE_DISCONNECT_BUTTON;
        disconnect.addEventListener("click", buttonClick, false);
        this.engines["engine"+engine.id].appendChild(engineInfo);
        this.engines["engine"+engine.id].appendChild(player1);
        this.engines["engine"+engine.id].appendChild(player2);
        this.engines["engine"+engine.id].appendChild(disconnect);
        this.engineList.insertBefore(this.engines["engine"+engine.id], this.loadEngineButton); 
    }

    removeEngine(engineID) {
        if (this.engines["engine"+engineID] == null) return;
        this.engines["engine"+engineID].remove();
        delete this.engines["engine"+engineID];
    }

    updatePlayers() {
        for (let key in this.engines) {
            this.engines[key].getElementsByClassName("engine-player1")[0]
                .classList.remove("active");
            this.engines[key].getElementsByClassName("engine-player2")[0]
                .classList.remove("active");
        }
        this.engines["engine"+state.player1ID].getElementsByClassName("engine-player1")[0]
            .classList.add("active");
        this.engines["engine"+state.player2ID].getElementsByClassName("engine-player2")[0]
            .classList.add("active");
    }

    updateButtons() {
        // New Game Button
        if (state.playing) {
            this.newGameButton.classList.add("disabled");
        } else {
            this.newGameButton.classList.remove("disabled");
        }
        // Setup Board Button
        // For now just keeping it disabled
        this.setupBoardButton.classList.add("disabled");
        // Start Button and Previous Button
        if (state.running || state.historyIndex == 0) {
            this.startButton.classList.add("disabled");
            this.previousButton.classList.add("disabled");
        } else {
            this.startButton.classList.remove("disabled");
            this.previousButton.classList.remove("disabled");
        }
        // Play Pause Button
        if (state.gameOver || state.player1ID == -1 || state.player2ID == -1) {
            this.playPauseButton.classList.add("disabled");
        } else {
            this.playPauseButton.classList.remove("disabled");
        }
        if (state.playing) {
            this.playPauseButton.getElementsByTagName("a")[0].innerHTML = "Pause";
        } else {
            this.playPauseButton.getElementsByTagName("a")[0].innerHTML = "Play";
        }
        // Next Button and End Button
        if (state.running || state.historyIndex >= state.history.length-1) {
            this.nextButton.classList.add("disabled");
            this.endButton.classList.add("disabled");
        } else {
            this.nextButton.classList.remove("disabled");
            this.endButton.classList.remove("disabled");
        }
    }

    drawloop() {
        this.draw();
        requestAnimationFrame(this.drawloop);
    }

    draw() {
        // Draw something to this.canvas with this.ctx
    }

    output(time, sender, message) {
        if (this.outputTerminal.innerHTML != "")
            this.outputTerminal.innerHTML += "<br>";
        this.outputTerminal.innerHTML += "["+time+"]["+sender+"]: " + message;
    }

    communication(time, sender, receiver, message) {
        if (this.communicationTerminal.innerHTML != "")
            this.communicationTerminal.innerHTML += "<br>";
        this.communicationTerminal.innerHTML += "["+time+"]["+sender+" -> "+receiver+"]: " + message;
    }
}

// A Board state
class Position {
    constructor(posString) {
        // posString is a CFP representation of a position
        this.tiles  = [];
        this.player = null;
        // Setting the tiles
        for (let i = 0; i < 42; i++) {
        switch (posString[i]) {
        case "0":
            this.tiles.push(EMPTY);
            break;
        case "1":
            this.tiles.push(PLAYER_1);
            break;
        case "2":
            this.tiles.push(PLAYER_2);
            break;
        }}
        // Setting the player
        switch (posString[42]) {
        case "1":
            this.player = PLAYER_1;
            break;
        case "2":
            this.player = PLAYER_2;
            break;
        }
    }
}

class Engine {
    constructor(engineString) {
        let args = engineString.split(" ");
        let idIndex = args.indexOf("id");
        let nameIndex = args.indexOf("name");
        let authorIndex = args.indexOf("author");
        this.id = args.slice(idIndex+1, nameIndex).join(" ");
        this.id = parseInt(this.id);
        this.name = args.slice(nameIndex+1, authorIndex).join(" ");
        this.author = args.slice(authorIndex+1, args.length).join(" ");
    }
}

function command(msg) {
    args = msg.split(" ");
    switch (args.shift()) {
    case "enginepaths":
        showPaths(args);
    case "engines":
        switch (args.shift()) {
        case "load":
            loadEngines(args);
            break;
        case "unload":
            unloadEngines(args);
            break;
        }
        break;
    case "engine":
        switch (args.shift()) {
        case "load":
            loadEngine(args);
            break;
        case "unload":
            unloadEngine(args);
            break;
        }
        break;
    case "players":
        players(args);
        break;
    case "newgame":
        newGame();
        break;
    case "position":
        position(args);
        break;
    case "gameover":
        gameOver(args);
        break;
    case "history":
        history(args);
        break;
    case "play":
        play();
        break;
    case "pause":
        pause();
        break;
    case "output":
        output(args);
        break;
    case "communication":
        communication(args);
        break;
    }
}

function showPaths(args) {
    gui.hidePathLoader();
    let lastPathIndex = null;
    for (let i = 0; i < args.length; i++) {
        if (args[i] == "path" && lastPathIndex != null) {
            gui.showFilepath(args.slice(lastPathIndex+1, i).join(" "));
        }
        if (args[i] == "path") {
            lastPathIndex = i;
        }
    }
    gui.showFilepath(args.slice(lastPathIndex+1, args.length).join(" "));
}

function loadEngines(args) {
    let lastEngineIndex = null;
    for (let i = 0; i < args.length; i++) {
        if (args[i] == "engine" && lastEngineIndex != null) {
            loadEngine(args.slice(lastEngineIndex+1, i));
        }
        if (args[i] == "engine") {
            lastEngineIndex = i;
        }
    }
    loadEngine(args.slice(lastEngineIndex+1, args.length));
}

function unloadEngines(args) {
    let lastEngineIndex = null;
    for (let i = 0; i < args.length; i++) {
        if (args[i] == "engine" || lastEngineIndex != null) {
            unloadEngine(args.slice(lastEngineIndex+1, i));
        }
        if (args[i] == "engine") {
            lastEngineIndex = i;
        }
    }
    unloadEngine(args.slice(lastEngineIndex+1, args.length));
}

function loadEngine(args) {
    let engine = new Engine(args.join(" "));
    state.loadEngine(engine);
    gui.loadEngine(engine);
}

function unloadEngine(args) {
    let id = parseInt(args[args.length-1]);
    state.unloadEngine(id);
    gui.unloadEngine(id);
}

function players(args) {
    let player1Index = args.indexOf("player1");
    let player2Index = args.indexOf("player2");
    let player1 = parseInt(args.slice(player1Index+1, player2Index).join(" "));
    let player2 = parseInt(args.slice(player2Index+1, args.length).join(" "));
    state.updatePlayers(player1, player2);
    gui.updatePlayers(player1, player2);
}

function newGame() {
    state.newGame();
    gui.updateButtons();
}

function position(args) {
    let position = new Position(args[args.length - 1]);
    state.updatePosition(position);
    gui.updateButtons();
}

function gameOver(args) {
    state.gameOver(parseInt(args[args.length-1]));
    gui.updateButtons();
}

function history(args) {
    newGame();
    for (let i = 0; i < args.length; i++)
        if (args[i] != "position")
            position(args[i]);
}   

function play() {
    state.play();
    gui.updateButtons();
}

function pause() {
    state.pause();
    gui.updateButtons();
}

function output(args) {
    let timeIndex       = args.indexOf("time");
    let senderIndex     = args.indexOf("sender");
    let messageIndex    = args.indexOf("message");
    
    let time    = args.slice(timeIndex+1, senderIndex).join(" ");
    let sender  = args.slice(senderIndex+1, messageIndex).join(" ");
    let message = args.slice(messageIndex+1, args.length).join(" ");

    gui.output(time, sender, message);
}

function communication(args) {
    let timeIndex       = args.indexOf("time");
    let senderIndex     = args.indexOf("sender");
    let receiverIndex   = args.indexOf("receiver");
    let messageIndex    = args.indexOf("message");

    let time        = args.slice(timeIndex+1, senderIndex).join(" ");
    let sender      = args.slice(senderIndex+1, receiverIndex).join(" ");
    let receiver    = args.slice(receiverIndex+1, messageIndex).join(" ");
    let message     = args.slice(messageIndex+1, args.length).join(" ");

    gui.communication(time, sender, receiver, message);
}

function requestNewGame() {
    socket.send("newgame");
}

function requestPause() {
    if (state.running) socket.send("pause");
}

function requestPlay() {
    if (!state.running) socket.send("play");
}

function requestEngineOperation(engineId, button) {
    switch (button) {
    case ENGINE_PLAYER1_BUTTON:
        requestPlayers(engineId, state.player2ID);
        break;
    case ENGINE_PLAYER2_BUTTON:
        requestPlayers(state.player1ID, engineId);
        break;
    case ENGINE_DISCONNECT_BUTTON:
        requestEngineUnload(engineId);
        break;
    }
}

function requestPlayers(player1, player2) {
    socket.send("setplayers player1 "+player1+" player2 "+player2);
}

function requestEnginePaths() {
    socket.send("enginepaths");
}

function requestEngineLoad(path) {
    socket.send("engine load path "+path);
}

function requestEngineUnload(engineId) {
    socket.send("engine unload id "+engineId);
}

window.onload = () => {
    // Load GUI elements into js...
    gui = new GUI();
    
    // Setting up connection to backend...
    socket = new WebSocket("ws://localhost:8080/ws");
    console.log("Attempting Web Socket Connection");
    
    socket.onopen = () => {
        console.log("Successfully Connected");
        state = new State();
        gui.updateButtons();
        //gui.drawloop();

        // socket.send("init");
    }
    
    socket.onclose = (evt) => {
        console.log("Socket Closed Connection: ", evt);
    }
    
    socket.onerror = (err) => {
        console.log("Socket Error: ", err);
    }
    
    socket.onmessage = (msg) => {
        command(msg.data);
    }
}
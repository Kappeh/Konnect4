"use strict"

// Protocol objects
let socket;                 // The websocket connection to the GUI
let state;                  // The current state of the GUI

// GUI object
let gui;

// Constants for graphics
const BORDER_STYLE  = "#ffffff";
const BORDER_WIDTH  = 1;
const X_LENGTH      = 20;
const X_WIDTH       = 5;
const X_COLOUR      = "#ff0000";
const O_RADIUS      = 20;
const O_WIDTH       = 5;
const O_COLOUR      = "#0000ff";

// Constants for state
const EMPTY     = 0;
const PLAYER_1  = 1;
const PLAYER_2  = 2;
const TIE       = 3;

// Constants for engine settings types
const SETTING_CHECK     = 0;
const SETTING_SPIN      = 1;
const SETTING_COMBO     = 2;
const SETTING_BUTTON    = 3;
const SETTING_STRING    = 4;

// Constants for controls (Button IDs)
const LOAD_ENGINE_BUTTON            = 0;
const NEW_GAME_BUTTON               = 1;
const SETUP_BOARD_BUTTON            = 2;
const START_BUTTON                  = 3;
const PREVIOUS_BUTTON               = 4;
const PLAY_PAUSE_BUTTON             = 5;
const NEXT_BUTTON                   = 6;
const END_BUTTON                    = 7;
const ENGINE_LIST_GO_BACK_BUTTON    = 8;
const SETTINGS_GO_BACK_BUTTON       = 9;

// Constants for engine specific controls
const ENGINE_BUTTONS_START      = 10;
const ENGINE_BUTTONS_STRIDE     = 4;
const ENGINE_PLAYER1_BUTTON     = 0;
const ENGINE_PLAYER2_BUTTON     = 1;
const ENGINE_SETTINGS_BUTTON    = 2;
const ENGINE_DISCONNECT_BUTTON  = 3;

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
        delete this.engines["engine"+engineID];
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
        this.history.push(position);
        this.historyIndex = this.history.length - 1;
    }

    setGameOver(winner) {
        this.playing    = false;
        this.gameOver   = true;
        this.winner     = winner;
    }

    play() {
        this.historyIndex = this.history.length - 1;
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
        gui.showLoadOverlay();
        requestEnginePaths();
        break;
    case NEW_GAME_BUTTON:
        requestNewGame();
        break;
    case SETUP_BOARD_BUTTON:
        // Not Implimented
        break;
    case START_BUTTON:
        state.historyIndex = 0;
        break;
    case PREVIOUS_BUTTON:
        if (state.historyIndex > 0) {
            state.historyIndex--;
        }
        break;
    case PLAY_PAUSE_BUTTON:
        if (state.playing) {
            requestPause();
        } else {
            requestPlay();
        }
        break;
    case NEXT_BUTTON:
        if (state.historyIndex < state.history.length-1) {
            state.historyIndex++;
        }
        break;
    case END_BUTTON:
        if (state.historyIndex < state.history.length-1) {
            state.historyIndex = state.history.length-1;
        }
        break;
    case ENGINE_LIST_GO_BACK_BUTTON:
        gui.hideLoadOverlay();
        break;
    case SETTINGS_GO_BACK_BUTTON:
        gui.hideSettingsOverlay();
        break;
    }
    if (this.buttonId >= ENGINE_BUTTONS_START) {
        // If we reach this point, it's en engine specific button
        let tmp         = this.buttonId - ENGINE_BUTTONS_START;
        let button      = tmp % ENGINE_BUTTONS_STRIDE;
        let engineId    = Math.floor(tmp / ENGINE_BUTTONS_STRIDE);
        requestEngineOperation(engineId, button);
    }
    gui.updateButtons();
}   

// Controls all of the visuals of the GUI
// Will rely on `state` heavily.
// Make sure `state` is set correctly before calling any methods!
class GUI {
    constructor() {
        // Space to store the engine and settings DOM elements
        this.engines                = {};
        this.settings               = {};

        // Getting DOM elements
        this.loadOverlay            = document.getElementById("load-engine-overlay");
        this.loadOverlayLoader      = document.getElementsByClassName("loader")[0];
        this.loadList               = document.getElementById("load-engine-list");        
        this.engineListGoBackButton = document.getElementsByClassName("go-back-button")[0];
        this.engineList             = document.getElementById("engine-list");
        this.loadEngineButton       = document.getElementById("load-engine-button");
        
        this.settingsOverlay        = document.getElementById("engine-settings-overlay");
        this.settingsList           = document.getElementById("engine-settings");
        this.settingsOverlayLoader  = document.getElementsByClassName("loader")[1];
        this.settingsGoBackButton   = document.getElementsByClassName("go-back-button")[1];

        this.newGameButton          = document.getElementById("new-game");
        this.setupBoardButton       = document.getElementById("setup-board");
        
        this.canvas                 = document.getElementById("game-screen-canvas");

        this.startButton            = document.getElementById("start");
        this.previousButton         = document.getElementById("previous");
        this.playPauseButton        = document.getElementById("play-pause");
        this.nextButton             = document.getElementById("next");
        this.endButton              = document.getElementById("end");
        
        this.outputTerminal         = document.getElementById("output-terminal").getElementsByTagName("p")[0];
        this.communicationTerminal  = document.getElementById("communications-terminal").getElementsByTagName("p")[0];

        // Getting canvas drawing context
        this.canvas.width   = 700;
        this.canvas.height  = 600;
        this.ctx            = this.canvas.getContext("2d");

        // Speaks for itself
        this.addEventHandlers();
    }

    addEventHandlers() {
        this.loadEngineButton.buttonId          = LOAD_ENGINE_BUTTON;
        this.newGameButton.buttonId             = NEW_GAME_BUTTON;
        this.setupBoardButton.buttonId          = SETUP_BOARD_BUTTON;
        this.startButton.buttonId               = START_BUTTON;
        this.previousButton.buttonId            = PREVIOUS_BUTTON;
        this.playPauseButton.buttonId           = PLAY_PAUSE_BUTTON;
        this.nextButton.buttonId                = NEXT_BUTTON;
        this.endButton.buttonId                 = END_BUTTON;
        this.engineListGoBackButton.buttonId    = ENGINE_LIST_GO_BACK_BUTTON;
        this.settingsGoBackButton.buttonId      = SETTINGS_GO_BACK_BUTTON;

        this.loadEngineButton.addEventListener("click", buttonClick, false);
        this.newGameButton.addEventListener("click", buttonClick, false);
        this.setupBoardButton.addEventListener("click", buttonClick, false);
        this.startButton.addEventListener("click", buttonClick, false);
        this.previousButton.addEventListener("click", buttonClick, false);
        this.playPauseButton.addEventListener("click", buttonClick, false);
        this.nextButton.addEventListener("click", buttonClick, false);
        this.endButton.addEventListener("click", buttonClick, false);
        this.engineListGoBackButton.addEventListener("click", buttonClick, false);
        this.settingsGoBackButton.addEventListener("click", buttonClick, false);
    }

    showLoadOverlay() {
        let filepaths = this.loadList.getElementsByClassName("file-path");
        for (let i = filepaths.length-1; i >= 0; i--) {
            filepaths[i].remove();
        }
        this.engineListGoBackButton.style.display   = "none";
        this.loadOverlayLoader.style.display        = "block";
        this.loadOverlay.style.display              = "flex";
    }
    
    hidePathLoader() {
        this.engineListGoBackButton.style.display   = "block";
        this.loadOverlayLoader.style.display        = "none";
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
        this.loadList.insertBefore(filepath, this.engineListGoBackButton);
        filebutton.addEventListener("click", () => {
            requestEngineLoad(name);
            gui.hideLoadOverlay();
        });
    }

    hideLoadOverlay() {
        this.loadOverlay.style.display = "none";
    }

    showSettingsOverlay() {
        let settings = this.settingsList.getElementsByClassName("engine-setting");
        for (let i = settings.length-1; i >= 0; i--) {
            let setting = settings[i];
            settings[i].remove();
            delete this.settings["engineid " + setting.engineID + " name " + setting.name];
        }
        this.settingsGoBackButton.style.display     = "none";
        this.settingsOverlayLoader.style.display    = "block";
        this.settingsOverlay.style.display          = "flex";
    }

    hideSettingsLoader() {
        this.settingsGoBackButton.style.display     = "block";
        this.settingsOverlayLoader.style.display    = "none"
    }

    showSetting(setting) {
        this.settings["engineid " + setting.engineID + " name " + setting.name] = setting;
        setting.generateDOM();
        this.settingsList.insertBefore(
            setting.DOM,
            this.settingsGoBackButton
        );
    }

    updateSetting(engineID, name, value) {
        let setting = this.settings["engineid " + engineID + " name " + name];
        // Settings updates will be sent to each client
        // some clients might not have the setting that's being
        // updated visible on the screen
        if (setting == undefined) {
            return;
        }
        setting.update(value);
        setting.updateDOM();
    }

    hideSettingsOverlay() {
        this.settingsOverlay.style.display = "none";
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
        let settings = document.createElement("div");
        settings.classList.add("engine-settings");
        settings.innerHTML = "<h3>SE</h3>";
        settings.buttonId = index + ENGINE_SETTINGS_BUTTON;
        settings.addEventListener("click", buttonClick, false);
        let disconnect = document.createElement("div");
        disconnect.classList.add("engine-disconnect");
        disconnect.innerHTML = "<h3>DC</h3>";
        disconnect.buttonId = index + ENGINE_DISCONNECT_BUTTON;
        disconnect.addEventListener("click", buttonClick, false);
        this.engines["engine"+engine.id].appendChild(engineInfo);
        this.engines["engine"+engine.id].appendChild(player1);
        this.engines["engine"+engine.id].appendChild(player2);
        this.engines["engine"+engine.id].appendChild(settings);
        this.engines["engine"+engine.id].appendChild(disconnect);
        this.engineList.insertBefore(this.engines["engine"+engine.id], this.loadEngineButton); 
    }

    unloadEngine(engineID) {
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
        if (state.player1ID != -1)
            this.engines["engine"+state.player1ID].getElementsByClassName("engine-player1")[0]
                .classList.add("active");
        if (state.player2ID != -1) 
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
        if (state.playing || state.historyIndex == 0) {
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
        if (state.playing || state.historyIndex >= state.history.length-1) {
            this.nextButton.classList.add("disabled");
            this.endButton.classList.add("disabled");
        } else {
            this.nextButton.classList.remove("disabled");
            this.endButton.classList.remove("disabled");
        }
        // Engine List
        if (state.playing) {
            this.engineList.classList.add("disabled");
        } else {
            this.engineList.classList.remove("disabled");
        }
    }

    draw() {
        // Clearing canvas
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        // Drawing borders of board
        this.ctx.strokeStyle    = BORDER_STYLE;
        this.ctx.lineWidth      = BORDER_WIDTH;
        for (let i = 1; i < 7; i++) {
            let xpos = i * this.canvas.width / 7;
            this.ctx.beginPath();
            this.ctx.moveTo(xpos, 0);
            this.ctx.lineTo(xpos, this.canvas.height);
            this.ctx.stroke();
        }
        for (let i = 1; i < 6; i++) {
            let ypos = i * this.canvas.height / 6;
            this.ctx.beginPath();
            this.ctx.moveTo(0, ypos);
            this.ctx.lineTo(this.canvas.width, ypos);
            this.ctx.stroke();
        }
        // Drawing peices
        if (state.history.length == 0) return;
        for (let i = 0; i < 42; i++) {
            let tile = state.history[state.historyIndex].tiles[i];
            if (tile == EMPTY)
                continue;
            let x = i % 7;
            let y = (i - x) / 7;
            let xcenter = (x + 0.5) * this.canvas.width / 7;
            let ycenter = (y + 0.5) * this.canvas.height / 6;
            if (tile == PLAYER_1)
                this.drawX(xcenter, ycenter);
            else
                this.drawO(xcenter, ycenter);
        }
    }

    drawX(xcenter, ycenter) {
        this.ctx.strokeStyle    = X_COLOUR;
        this.ctx.lineWidth      = X_WIDTH;
        this.ctx.beginPath();
        this.ctx.moveTo(xcenter - X_LENGTH, ycenter - X_LENGTH);
        this.ctx.lineTo(xcenter + X_LENGTH, ycenter + X_LENGTH);
        this.ctx.moveTo(xcenter - X_LENGTH, ycenter + X_LENGTH);
        this.ctx.lineTo(xcenter + X_LENGTH, ycenter - X_LENGTH);
        this.ctx.stroke();
    }

    drawO(xcenter, ycenter) {
        this.ctx.strokeStyle    = O_COLOUR;
        this.ctx.lineWidth      = O_WIDTH;
        this.ctx.beginPath();
        this.ctx.arc(xcenter, ycenter, O_RADIUS, 0, Math.PI * 2);
        this.ctx.stroke();
    }

    output(time, sender, message) {
        if (this.outputTerminal.innerHTML != "")
            this.outputTerminal.innerHTML += "<br>";
        this.outputTerminal.innerHTML += "["+time+"]["+sender+"]: " + message;
        this.outputTerminal.scrollTo(0, this.outputTerminal.scrollHeight);
    }

    communication(time, engine, toengine, message) {
        let prefix = "<--" + engine;
        if (toengine) prefix = "-->" + engine;
        
        if (this.communicationTerminal.innerHTML != "")
            this.communicationTerminal.innerHTML += "<br>";
        this.communicationTerminal.innerHTML += "["+time+"]["+prefix+"]: " + message;
        this.communicationTerminal.scrollTo(0, this.communicationTerminal.scrollHeight);
    }
}

class Setting {
    constructor(engineID, name, type, value, min, max, vars) {
        this.engineID   = engineID;
        this.name       = name;
        this.type       = type;

        // Only spinners have a min and max
        if (type == SETTING_SPIN) {
            this.min = min;
            this.max = max;
        } else {
            this.min = null;
            this.max = null;
        }

        // Only buttons don't have a value
        if (type != SETTING_BUTTON) {
            this.value = value;
        } else {
            this.value = null;
        }

        // Only comboboxes have vars
        if (type == SETTING_COMBO) {
            this.vars = vars;
        } else {
            this.vars = null;
        }

        this.DOM            = null;
        this.requestValue   = null;
    }

    update(value) {
        this.value = value;
    }

    updateDOM() {
        switch (this.type) {
        case SETTING_CHECK:
            let checkbox = this.DOM.getElementsByClassName("setting-checkbox")[0];
            if (this.value == "true") {
                checkbox.classList.add("checked");
            } else {
                checkbox.classList.remove("checked");
            }
            break;
        case SETTING_SPIN:
            let spinner = this.DOM.getElementsByTagName("input")[0];
            spinner.value = parseInt(this.value);
            break;
        case SETTING_COMBO:
            let combo = this.DOM.getElementsByClassName("selected")[0];
            combo.innerHTML = this.value;
            break;
        case SETTING_BUTTON:
            break;
        case SETTING_STRING:
            let textbox = this.DOM.getElementsByTagName("input")[0];
            textbox.value = this.value;
            break;
        }
    }

    generateDOM() {
        // Common settings elements
        let result = document.createElement("li");
        result.classList.add("engine-setting");
        let name = document.createElement("div");
        name.classList.add("engine-setting-name");
        name.innerHTML = "<p>" + this.name + "</p>";
        let control = document.createElement("div");
        control.classList.add("engine-setting-control");
        let updateButton = document.createElement("div");
        updateButton.classList.add("setting-update-button");
        updateButton.innerHTML = "Update";
        updateButton.addEventListener("click", () => {
            if (!updateButton.classList.contains("clickable")) {
                return;
            }
            updateButton.classList.remove("clickable");
            requestSetOption(this);
        }, false);

        // Setting specific elements
        switch (this.type) {
        case SETTING_CHECK:
            result.classList.add("engine-setting-check");
            let checkbox = document.createElement("div");
            checkbox.classList.add("setting-checkbox");
            if (this.value) {
                checkbox.classList.add("checked");
            }
            checkbox.addEventListener("click", () => {
                checkbox.classList.toggle("checked");
                updateButton.classList.add("clickable");
                this.requestValue = checkbox.classList.contains("checked");
            }, false);
            control.appendChild(checkbox);
            control.appendChild(updateButton);
            break;
        case SETTING_SPIN:
            result.classList.add("engine-setting-spin");
            let spin = document.createElement("div");
            spin.classList.add("setting-spin");
            let numberInput = document.createElement("input");
            numberInput.type = "number";
            numberInput.value = this.value;
            numberInput.addEventListener("input", () => {
                updateButton.classList.add("clickable");
                this.requestValue = numberInput.value;
            }, false);
            let arrows = document.createElement("div");
            arrows.classList.add("spin-arrows");
            let up = document.createElement("div");
            up.classList.add("spin-up");
            let down = document.createElement("div");
            down.classList.add("spin-down");
            arrows.appendChild(up);
            arrows.appendChild(down);
            spin.appendChild(numberInput);
            spin.appendChild(arrows);
            control.appendChild(spin);
            control.appendChild(updateButton);
            break;
        case SETTING_COMBO:
            result.classList.add("engine-setting-combo");
            let combo = document.createElement("div");
            combo.classList.add("setting-combo");
            let selectedOption = document.createElement("div");
            selectedOption.classList.add("option");
            let selected = document.createElement("p");
            selected.classList.add("selected");
            selected.innerHTML = this.value;
            selectedOption.appendChild(selected);
            let options = document.createElement("ul");
            options.classList.add("options");
            for (let i = 0; i < this.vars.length; i++) {
                let option = document.createElement("li");
                option.classList.add("option");
                option.innerHTML = "<p>" + this.vars[i] + "</p>";
                option.addEventListener("click", () => {
                    selected.innerHTML = this.vars[i];
                    updateButton.classList.add("clickable");
                    this.requestValue = this.vars[i];
                }, false);
                options.appendChild(option);
            }
            combo.appendChild(selectedOption);
            combo.appendChild(options);
            control.appendChild(combo);
            control.appendChild(updateButton);
            break;
        case SETTING_BUTTON:
            let button = document.createElement("div");
            button.classList.add("setting-button");
            button.innerHTML = "Trigger";
            button.addEventListener("click", () => {
                requestSetOption(this);
            }, false);
            control.appendChild(button);
            break;
        case SETTING_STRING:
            let string = document.createElement("div");
            string.classList.add("setting-string");
            let stringInput = document.createElement("input");
            stringInput.type = "text";
            stringInput.value = this.value;
            stringInput.addEventListener("input", () => {
                updateButton.classList.add("clickable");
                this.requestValue = stringInput.value;
            }, false);
            string.appendChild(stringInput);
            control.appendChild(string);
            control.appendChild(updateButton);
            break;
        default:
            return null;
        }

        // Common settings elements
        result.appendChild(name);
        result.appendChild(control);
        this.DOM = result;
    }

    wsSetOptionCommand() {
        if (this.type == SETTING_BUTTON) {
            return "setoption engineid "+this.engineID+
                " name "+this.name;
        } else {
            return "setoption engineid "+this.engineID+
                " name "+this.name+" value "+this.requestValue;
        }
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
    let args = msg.split(" ");
    switch (args.shift()) {
    case "enginepaths":
        showPaths(args);
        break;
    case "noenginepaths":
        gui.hideLoadOverlay();
        break;
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
    case "option":
        option(args);
        break;
    case "nooptions":
        gui.hideSettingsOverlay();
        break;
    case "updateoption":
        updateOption(args);
        break;
    case "output":
        output(args);
        break;
    case "communication":
        communication(args);
        break;
    }
    gui.updateButtons();
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
}

function position(args) {
    let position = new Position(args[args.length - 1]);
    state.updatePosition(position);
}

function gameOver(args) {
    state.setGameOver(parseInt(args[args.length-1]));
}

function history(args) {
    newGame();
    for (let i = 0; i < args.length; i++)
        if (args[i] != "position")
            position(args[i]);
}   

function play() {
    state.play();
}

function pause() {
    state.pause();
}

function option(args) {
    gui.hideSettingsLoader();

    let engineIdIndex   = args.indexOf("engineid");
    let nameIndex       = args.indexOf("name");
    let typeIndex       = args.indexOf("type");

    let engineId    = parseInt(args[engineIdIndex+1]);
    let name        = args.slice(nameIndex+1, typeIndex).join(" ");
    let type        = args[typeIndex+1];

    switch (type) {
    case "check":
        checkboxOption(engineId, name, args.slice(typeIndex+2, args.length));
        break;
    case "spin":
        spinnerOption(engineId, name, args.slice(typeIndex+2, args.length));
        break;
    case "combo":
        comboboxOption(engineId, name, args.slice(typeIndex+2, args.length));
        break;
    case "button":
        buttonOption(engineId, name);
        break;
    case "string":
        stringOption(engineId, name, args.slice(typeIndex+2, args.length));
        break;
    }
}

function checkboxOption(engineId, name, args) {
    let valueIndex = args.indexOf("value");
    let value = args[valueIndex+1];
    if (value == "true") {
        gui.showSetting(new Setting(engineId, name, SETTING_CHECK, true, null, null, null));
    } else {
        gui.showSetting(new Setting(engineId, name, SETTING_CHECK, false, null, null, null));
    }
}

function spinnerOption(engineId, name, args) {
    let minIndex    = args.indexOf("min");
    let maxIndex    = args.indexOf("max");
    let valueIndex  = args.indexOf("value");
    
    let min     = parseInt(args[minIndex+1]);
    let max     = parseInt(args[maxIndex+1]);
    let value   = parseInt(args[valueIndex+1]);

    gui.showSetting(new Setting(engineId, name, SETTING_SPIN, value, min, max, null));
}

function comboboxOption(engineId, name, args) {
    let valueIndex = args.indexOf("value");
    let varIndexes = getAllIndexes(args, "var");
    
    if (varIndexes.length == 0) {
        return
    }
    
    let value   = args.slice(valueIndex+1, varIndexes[0]).join(" ");
    let vars    = [];
    
    varIndexes.push(args.length);
    for (let i = 0; i < varIndexes.length-1; i++) {
        vars.push(args.slice(varIndexes[i]+1, varIndexes[i+1]).join(" "));
    }

    gui.showSetting(new Setting(engineId, name, SETTING_COMBO, value, null, null, vars));
}

function buttonOption(engineId, name) {
    gui.showSetting(new Setting(engineId, name, SETTING_BUTTON, null, null, null, null));
}

function stringOption(engineId, name, args) {
    let valueIndex = args.indexOf("value");
    let value = args.slice(valueIndex+1, args.length).join(" ");

    gui.showSetting(new Setting(engineId, name, SETTING_STRING, value, null, null, null));
}

function updateOption(args) {
    let engineIdIndex   = args.indexOf("engineid");
    let nameIndex       = args.indexOf("name");
    let valueIndex      = args.indexOf("value");

    let engineId    = parseInt(args[engineIdIndex+1]);
    let name        = args.slice(nameIndex+1, valueIndex).join(" ");
    let value       = args.slice(valueIndex+1, args.length).join(" ");

    gui.updateSetting(engineId, name, value);
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
    let engineIndex     = args.indexOf("engine");
    let toengineIndex   = args.indexOf("toengine");
    let messageIndex    = args.indexOf("message");

    let time        = args.slice(timeIndex+1, engineIndex).join(" ");
    let engine      = args.slice(engineIndex+1, toengineIndex).join(" ");
    let toengine    = args.slice(toengineIndex+1, messageIndex).join(" ");
    let message     = args.slice(messageIndex+1, args.length).join(" ");

    toengine = toengine.toLowerCase() == "true";
    gui.communication(time, engine, toengine, message);
}

function requestNewGame() {
    socket.send("newgame");
}

function requestPause() {
    if (state.playing) socket.send("pause");
}

function requestPlay() {
    if (!state.playing) socket.send("play");
}

function requestEngineOperation(engineId, button) {
    switch (button) {
    case ENGINE_PLAYER1_BUTTON:
        requestPlayers(engineId, state.player2ID);
        break;
    case ENGINE_PLAYER2_BUTTON:
        requestPlayers(state.player1ID, engineId);
        break;
    case ENGINE_SETTINGS_BUTTON:
        gui.showSettingsOverlay();
        requestEngineSettings(engineId);
        break
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

function requestEngineSettings(engineId) {
    socket.send("options engineid " + engineId);
}

function requestSetOption(setting) {
    socket.send(setting.wsSetOptionCommand());
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
        drawloop();
        socket.send("init");
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

function drawloop() {
    gui.draw();
    requestAnimationFrame(drawloop);
}

function getAllIndexes(arr, val) {
    var indexes = [], i = -1;
    while ((i = arr.indexOf(val, i+1)) != -1){
        indexes.push(i);
    }
    return indexes;
}
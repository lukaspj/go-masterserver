import './app';
import {Lobby} from "./lib/models";
import {LobbyConnection} from "./lib/lobbyConnection";

declare global {
    let lobbyInput: LobbyInput;
    let lobbies: Array<Lobby>;
}

interface LobbyInput {
    name: string;
}

async function ListLobbies() {
    const resp = await fetch("http://localhost:3000/lobby", {
        method: "GET",
    })
    if (resp.status !== 200) {
        console.log(`Create lobby failed: Unexpected HTTP Status ${resp.status} ${resp.statusText}`);
        return;
    }

    let _lobbies = await resp.json() as Array<Lobby>;
    _lobbies.sort((l1, l2) => l1.Id.localeCompare(l2.Id));
    lobbies.splice(0, lobbies.length);
    _lobbies.forEach((l) => lobbies.push(l));
}

async function createLobby() {
    let lobbyConnection = new LobbyConnection('', lobbyInput.name);
    await lobbyConnection.create();
    await ListLobbies();
    lobbyInput.name = '';
}

async function deleteLobby(id: string) {
    let lobbyConnection = new LobbyConnection(id);
    await lobbyConnection.delete();
    await ListLobbies();
}

window["lobbyInput"] = Alpine.reactive({name: ''} as LobbyInput);
window["lobbies"] = Alpine.reactive(Array<Lobby>());
window["createLobby"] = createLobby;
window["deleteLobby"] = deleteLobby;

setTimeout(ListLobbies);

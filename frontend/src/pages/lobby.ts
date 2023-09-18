import '../lib/lobbyConnection'
import Alpine from "alpinejs";
import '../app';
import {LobbyConnection} from "../lib/lobbyConnection";
import {LogMessage} from "../lib/models";

declare global {
    let lobbyConnection: LobbyConnection;
}

window["lobbyConnection"] = Alpine.reactive(new LobbyConnection());
window["logMessages"] = Alpine.reactive(Array<LogMessage>());

const publishForm = document.getElementById("tmp-publish-form");
const messageInput = document.getElementById("tmp-message-input") as HTMLInputElement;

// onsubmit publishes the message from the user when the form is submitted.
publishForm.onsubmit = async ev => {
    ev.preventDefault()

    if (!lobbyConnection.id) {
        return
    }

    const msg = messageInput.value
    if (msg === "") {
        return
    }
    messageInput.value = ""

    try {
        const resp = await fetch(`http://localhost:3000/lobby/${lobbyConnection.id}`, {
            method: "POST",
            body: msg,
        })
        if (resp.status !== 202) {
            throw new Error(`Unexpected HTTP Status ${resp.status} ${resp.statusText}`)
        }
    } catch (err) {
        //appendLog(`Publish failed: ${err.message}`, true)
    }
}

let url = new URL(window.location.href);
let lobbyId = url.searchParams.get("lobbyId");
lobbyConnection.id = lobbyId;
lobbyConnection.join().then(r => console.log('joined lobby'));

//appendLog("Submit a message to get started!");

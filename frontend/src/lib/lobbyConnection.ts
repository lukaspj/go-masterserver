import './models';
import {LobbyMessage, LogMessage} from "./models";

export class LobbyConnection {
    public id: string;
    public name: string;
    private websocketConnection: WebSocket = null;
    public logMessages: Array<LogMessage> = new Array<LogMessage>();

    constructor(id = '', name = '') {
        this.id = id;
        this.name = name;
    }

    public async create() {
        const resp = await fetch("http://localhost:3000/lobby", {
            method: "POST",
            body: JSON.stringify({name: this.name}),
            headers: new Headers({
                "Content-Type": "application/json"
            })
        })
        if (resp.status !== 200) {
            this.appendLog(`Create lobby failed: Unexpected HTTP Status ${resp.status} ${resp.statusText}`, true);
            return;
        }
        this.id = await resp.text();
    }

    public async delete() {
        const resp = await fetch(`http://localhost:3000/lobby/${this.id}`, {
            method: "DELETE",
        })
        if (resp.status !== 200) {
            this.appendLog(`Delete lobby failed: Unexpected HTTP Status ${resp.status} ${resp.statusText}`, true);
            return;
        }
        this.id = await resp.text();
    }

    public async join() {
        if (this.websocketConnection !== null) {
            this.websocketConnection.close();
        }
        this.websocketConnection = new WebSocket(`ws://localhost:3000/lobby/${this.id}`)

        this.websocketConnection.addEventListener("close", ev => {
            this.appendLog(`WebSocket Disconnected code: ${ev.code}, reason: ${ev.reason}`, true)
            if (ev.code !== 1001) {
                this.appendLog("Reconnecting in 1s", true)
                setTimeout(this.join, 1000, this.id)
            }
            this.appendLog("websocket disconnected", true)
        });

        this.websocketConnection.addEventListener("open", ev => {
            console.info("websocket connected")
        });

        // This is where we handle messages received.
        this.websocketConnection.addEventListener("message", ev => {
            if (typeof ev.data !== "string") {
                console.error("unexpected message type", typeof ev.data)
                return
            }

            let message: LobbyMessage = JSON.parse(ev.data);

            switch (message.type) {
                case "text":
                    console.log("text message:", message);
                    this.appendLog(message.text.content, false, new Date(message.text.created));
                    break;
                case "meta":
                    console.log("meta message:", message);
                    this.name = message.meta.name;
                    this.id = message.meta.id;
                    break;
                default:
                    console.error('unhandled message type', message);
            }
        });
        this.appendLog(`Joined ${this.id}`);
    }

    // appendLog appends the passed text to messageLog.
    private appendLog(text: string, error?: boolean, time: Date = new Date()) {
        let msg = new LogMessage();
        msg.time = time;
        msg.message = text;
        msg.error = error;
        this.logMessages.push(msg);
        this.logMessages.sort((m1, m2) => m1.time.getTime() - m2.time.getTime())
    }
}
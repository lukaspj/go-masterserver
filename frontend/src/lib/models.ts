export interface Lobby {
    Id: string;
    Name: string;
}

export class LogMessage {
    public time: Date;
    public error: boolean;
    public message: string;
}

export class LobbyText {
    public content: string;
    public created: string;
}

export class LobbyMeta {
    public name: string;
    public id: string;
    public subscribers: number;
}

export class LobbyMessage {
    public type: "text" | "meta";
    public text: LobbyText;
    public meta: LobbyMeta;
}
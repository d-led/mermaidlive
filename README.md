# Mermaid Diagram Live Update Demo

## Running

```bash
go run .
```

&darr;

[http://localhost:8080/ui/](http://localhost:8080/ui/)

![screencast](./docs/img/live_state.gif)

### Embedded Resources

to only generate UI resources from [ui-src](./ui-src), run:

```shell
go run . -transpile
```

to build a binary with embedded UI:

```shell
go build --tags=embed .
```

## Architecture

```mermaid
flowchart LR
    Server --serves---> UI
    StateMachine --runs on --> Server
    UI --posts commands to --> Server
    Server --forwards commands to -->StateMachine
    Server --publishes events to -->PubSub
    StateMachine --publishes events to -->PubSub
    Server --subscribes each connected client to -->PubSub
    Server --streams events to --> UI
```

## Testing

- WIP
- "test-after" as a spike in itself
- the demo revolves around reusing the [specification](./features/)

### Unit

```shell
go test -v ./...
```

## Approach

- [Mermaid API](https://mermaid.js.org/config/setup/modules/mermaidAPI.html)
- [JSON Streaming](https://en.wikipedia.org/wiki/JSON_streaming)

## Deployment

- currently deployed on [fly.io](https://fly.io/) &rarr; [mermaidlive.fly.dev](https://mermaidlive.fly.dev/)

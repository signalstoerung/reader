# Reader

Reader is a simple web-based RSS reader written in Go.

It polls RSS feeds and displays them in a "news ticker" style: A timestamp, a tag identifying the source, and a headline. 

![](docs/tickr.png)

## Breaking news

New in this version is headline scoring. This is using gpt-3.5-turbo through the OpenAI API. The model is prompted to identify high-priority breaking news through the system prompt. The system prompt is tailored in two ways:

1. Reading the broader news context from a text file (default: `db/newscontext.txt`). This can be used to describe the big ongoing stories in the media that the model is unlikely to be aware of due to its content cutoff date.
2. The last 10 top-scoring headlines are retrieved from the database in real time and appended to the system prompt to avoid duplication of recent headlines.

To ensure we get valid JSON back from the model, we use the `response_format` option ([see API docs](https://platform.openai.com/docs/api-reference/chat/create#chat-create-response_format)). This requires using the latest version of the 3.5-turbo model, gpt-3.5-turbo-1106. 

(Using gpt-4 did not appear to meaningfully improve performance, so we went with the cheaper option.)

## Installation

I have Reader running in production behind a NGINX reverse proxy. 

### Production

Clone repo from git. Build with go.

Create a small script that will start reader (setting certain command-line flags, for instance).

Then, create a file `reader.service` and save it into `~/.config/systemd/user`:

```systemd
[Unit]
Description=Reader, a simple RSS reader
After=network.target

[Service]
ExecStart=/home/nimi/reader/reader.sh
Type=Simple

[Install]
WantedBy=default.target
RequiredBy=network.target
```

system.d should pick it up from there.

> [!WARNING]
> When running `start/stop/restart/enable` etc., you need to use `systemctl --user`. Without the `--user` option, systemctl will not find the service file.



### Dev / testing

For **local testing**, create a directory `./db/` in the work directory (Reader will store the sqlite database there), copy the sample config file into the same directory, rename it `config.yaml` and edit it as appropriate. Then you can run Reader with `go run .` and access it in a browser at `localhost:8000`.

- The first time that Reader runs, it will allow anyone to create an account. Follow the link from the homepage to create an account. Once you've done this, registrations will automatically close and nobody else can create an account. (I'm assuming that this is a single-user instance.)
- *Troubleshooting:* Account creation is only open when Reader starts up and does not find a database. So if you started Reader and stopped it again without creating an account, registration will be closed when you restart, because Reader will have created the database on the first startup. Solution: database in the `./db/` directory. 

## Command-line flags

```
  -config string
    	File path to a yaml config file (default "./db/config.yaml")
  -context string
    	File path to a text file describing the news context (default "./db/newscontext.txt")
  -db string
    	File path to sqlite database (default "./db/reader.db")
  -debug
    	Activate debug options and logging
```

## Reading the news

This is going to be self-explanatory, I hope! All links open in a new tab.


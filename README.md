# Reader

Reader is a simple web-based RSS reader written in Go.

It polls RSS feeds and displays them in a "news ticker" style: A timestamp, a tag identifying the source, and a headline. 

![](docs/tickr.png)

## Installation

I have Reader running in production behind a NGINX reverse proxy. 

For **local testing**, create a directory `./db/` in the work directory (Reader will store the sqlite database there), copy the sample config file into the same directory, rename it `config.yaml` and edit it as appropriate. Then you can run Reader with `go run .` and access it in a browser at `localhost:8000`.

- The first time that Reader runs, it will allow anyone to create an account. Follow the link from the homepage to create an account. Once you've done this, registrations will automatically close and nobody else can create an account. (I'm assuming that this is a single-user instance.)
- *Troubleshooting:* Account creation is only open when Reader starts up and does not find a database. So if you started Reader and stopped it again without creating an account, registration will be closed when you restart, because Reader will have created the database on the first startup. Solution: database in the `./db/` directory. 


## Reading the news

This is going to be self-explanatory, I hope! All links open in a new tab.


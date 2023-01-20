# Reader

Reader is a simple web-based RSS reader written in Go.

It polls RSS feeds and displays them in a "news ticker" style: A timestamp, a tag identifying the source, and a headline. 

![](docs/tickr.png)

## Installation

I have Reader deployed in a docker container and behind a NGINX reverse proxy. It currently runs on an EC2 micro instance. (It could probably run on a nano instance, too, but docker ran out of memory compiling the container.)

Installation steps (assuming that you also want to use Docker):
- Install nginx and certificates from Let's Encrypt
- Clone the git repository
- Create a Docker volume (to store the sqlite database and the config file, so that they are persistent if you upgrade the container to a new version)
- Build the container image
- Run a container from the image, mounting the volume to /app/db of the container with `-v [volumename]:/app/db`
- Reader will panic because the config file is missing, but you can't copy the file into the Docker volume if no container is running?! 
- Copy the sample config file into `config.yaml`, edit it as appropriate and copy it with `docker cp` to the directory /app/db of the container.
- Restart the container, be sure to mount the volume as above and to expose port 8000 with `-p 8000:8000`.
- Configure nginx as a reverse proxy to 127.0.0.1:8000. 
- The first time that Reader runs, it will allow anyone to create an account. Follow the link from the homepage to `/register/` and create an account. Once you've done this, registrations will automatically close and nobody else can create an account. (I'm assuming that this is a single-user instance.)
- *Troubleshooting:* Account creation is only open when Reader starts up and does not find a database. So if you started a container and stopped it again without creating an account, registration will be closed when you restart the container, because Reader will have created the database on the first startup. Solution: ssh into the container with `docker exec -it <container name> /bin/sh` and delete the database in the `/app/db/` directory. Then restart the container.
- Log in and go to `/feeds/` to add your RSS feeds (and delete the sample feeds that Reader has automatically created at first startup).

## Reading the news

This is going to be self-explanatory, I hope! All links open in a new tab.


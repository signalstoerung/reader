# Reader

Reader is a simple web-based RSS reader written in Go.

It polls RSS feeds and displays them in a "news ticker" style: A timestamp, a tag identifying the source, and a headline. Clicking on the headline opens the article in a new window.

I got used to this style of following the news in about a decade working as a wire service journalist, and it still works for me. A quick scan of the headlines to know what's happening, and clicking on a story if I want to know more.

In the wild, reader is deployed in a docker container and behind a NGINX reverse proxy. It currently runs on an EC2 micro instance. (It could probably run on a nano instance, too, but docker ran out of memory compiling the container.)

# ap-5r

AP-5R is a simple Discord bot that interacts with https://swgoh.gg/.

The main pourpose of this bot is to help playeres of
Star Wars Galaxy of Heroes to have a rich discussion
about their collection, characters and ships,
without leaving the Discord chat.

The bot came alive in the Galactic Empire discord server,
and slowly became a good Guild tool as well.

Join the discussion in our Discord chat: https://discord.gg/4GJ8Ty2

Support this bot with a donation: http://bit.ly/support-ap-5r

# Running the Bot

The bot can be used in any Discord server, either using a version
hosted by the authors, or self-hosting your own copy.

Regardless of what method you choose, you also need to meet the
Discord server requirements and settings.

## Using the hosted version

Just follow this link on your PC and add the hosted version to your server:
https://discordapp.com/oauth2/authorize?client_id=355873395164053512&scope=bot&permissions=511040

After adding the bot, check the *Discord server requirements and settings*
section of this page.

## Discord server requirements and settings

* You must be a server admin in order to add AP-5R to your server
* AP-5R requires some API permissions, leave all checkboxes in the link
  above ON.
* AP-5R uses a channel to load his server-restrict configuration. As of now,
  a channel with the name #swgoh-gg need to be fed with profile links.
  Each player profile link is connected by AP-5R to the user who posts it,
  or to the user mentioned in the link text.

**Tip**: you can restrict where AP-5R can read/write messages by
changing the permissions of the bot role that Discord
adds automatically: `SWGoH Bot`.

## Self-hosting

The bot is composed by three main components: the main bot program,
the PageRender program and an API proxy for the https://swgoh.gg website.

The API proxy runs independently and is a hosted cloud service.
Learn more about this service here: https://swgoh-api.appspot.com/ and
here https://github.com/ronoaldo/swgohapi.

You will need to run two programs in your computer, and both can
be easily run using Docker. First install Docker CE on your
computer: https://docs.docker.com/engine/installation/

You also need to be familiar on how to setup the Bot on
your Discord developer account. This page can be handy:
https://github.com/reactiflux/discord-irc/wiki/Creating-a-discord-bot-&-getting-a-token

### Running PageRender

First, run the PageRender program from Docker hub:

    docker run --name pagerender --rm ronoaldo/pagerender:latest

PageRender is stateless and the image is a bit large due to
the dependencies required. 

### Running AP-5R

Second, run the AP-5R program from Docker hub, linking it to PageRender:

    docker run --link pagerender --name ap5r --rm -e BOT_TOKEN=your-token-here ronoaldo/ap-5r:latest

If all goes well, you should have the two containers running in the background,
and AP-5R is ready to be added to your Discord server!

### Building your own modified AP-R5 - for contributing code

#### Requirements: 
 * Go 1.7 or higher (https://golang.org/doc/install)

#### Setup:

1. Create a folder to put all of your AP-5R files, and create a src sub-directory.  For example:

```
$ mkdir /Users/USERNAME/ap-5r/src
```

2. Clone this project into that folder:

```
$ cd /Users/USERNAME/ap-5r/src
$ git clone https://github.com/ronoaldo/ap-5r.git
$ ls
ap-5r
```

3. Export the GOPATH environment variable:

```
$ export GOPATH="/Users/USERNAME/ap-5r"
````

4. Create the file .token as a one-line file that just has your bot's token in it.

```
$ vim /Users/USERNAME/ap-5r/src/ap-5r/.token
``` 

Now, go ahead and make your edits to the bots source code.  

When you're ready to start your bot with your modified code, just run: 

```
$ cd /Users/USERNAME/ap-5r/src/ap-5r
$ make run

rm ap-5r && GOARCH=amd64 GOOS=linux go build -o ap-5r
docker build \
		-t ronoaldo/ap-5r:latest \
		--build-arg GIT_HASH=$(git rev-parse --short HEAD) .
~~ Snip ~~
Bot is now running.  Press CTRL-C to exit.
```

If all goes well, you'll see the final line above.  Your updated code has been successfully built, your local container updated and it's running on your local system.
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

[![Add to server](images/add_to_server.png)](https://discordapp.com/oauth2/authorize?client_id=355873395164053512&scope=bot&permissions=511040)

After adding the bot, check the *Discord server requirements and settings*
section of this page.

## Discord server requirements and settings

* You must be a **server admin** in order to add AP-5R to your server
* AP-5R requires some API permissions, leave **all checkboxes in the link
  above ON**.
* AP-5R uses a channel to load his server-restrict configuration. As of now,
  a channel with the name #swgoh-gg need to be filled with profile links.
  **Each player profile link is connected by AP-5R to the user who posts it,
  or to the user mentioned in the link text**.

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

    docker run --name pagerender --rm --it ronoaldo/pagerender:latest

PageRender is stateless and the image is a bit large due to
the dependencies required. 

### Running AP-5R

Second, run the AP-5R program from Docker hub, linking it to PageRender:

    docker run --link pagerender --name ap5r --rm -e BOT_TOKEN=your-token-here --it ronoaldo/ap-5r:latest

If all goes well, you should have the two containers running in the background,
and AP-5R is ready to be added to your Discord server!

## Building your own modified AP-R5

In some cases, specially when you want to contribute code,
it may be helpful to build the bot from source.

### Requirements

The same requirements bellow for self-hosting and:

 * Go 1.7 or higher (https://golang.org/doc/install)
 * Git (https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)

### Setup

In the following step-by-step, we assume that you are familiar
with command line tools for Linux/Mac or Windows. While these
should work on Windows, none of the contributors made any
testing so far in that platform.

1. After you install Go, make sure to set your `GOPATH`
   environment variable. This way you can fetch the source
   using the Go tools. If you are new to Go and
   unsure about $GOPATH, set it to $HOME:

```
export GOPATH=$HOME
go get github.com/ronoaldo/ap-5r
```

*The command bellow will put all source code in the expected location
that the Go compiler will look for. You should see your code in
`$GOPATH/src/github.com/ronoaldo/ap-5r`*

2. Follow the [steps here](https://github.com/reactiflux/discord-irc/wiki/Creating-a-discord-bot-&-getting-a-token) to setup your Bot token and save the token
   as a one-line file in the `ap-5r` root source folder:

```
echo "YourLongTokenString" >  $GOPATH/src/github.com/ronoaldo/ap-5r/.token
``` 

3. Start the PageRender container:

```
docker run --name pagerender --rm --it ronoaldo/pagerender:latest
```

4. Start the bot running into development mode:

```
cd $GOPATH/src/github.com/ronoaldo/ap-5r
make run
```

You should see an output like this:

```
rm ap-5r && GOARCH=amd64 GOOS=linux go build -o ap-5r
docker build \
	-t ronoaldo/ap-5r:latest \
	--build-arg GIT_HASH=$(git rev-parse --short HEAD) .
~~ Snip ~~
Bot is now running.  Press CTRL-C to exit.
```

If all goes well, you'll see the final line above.  Your updated code has been successfully built, your local container updated and it's running on your local system.

Note: after you make changes, there is no live-reloading of the bot
yet. You need to press CTRL-C to kill the bot container and call
`make run` again

# License

Copyright 2017-2018 Ronoaldo JLP

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
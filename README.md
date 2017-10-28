# ap-5r

Discord bot that interacts with https://swgoh.gg/.
The main pourpose of this bot is to help playeres of
Star Wars Galaxy of Heroes to have a rich discussion
about their collection, characters and ships,
without leaving the Discord chat.

The bot came alive in the Empire Players discord server,
and slowly became a good Guild tool as well.

## Design

AP-5R is designed to be as stateless as possible,
with the minimum maintanance burden.
AP uses Google Cloud Platform components to run
smoothly: App Engine, Compute Engine and Containe Engine.

Website parsing and caching is done by the @ronoaldo/swgohapi
project, hosted on App Engine. Parsed profile data is stored
in the Cloud Datastore and Memcache, updated every 6 hours.

Image rendering is perfomred with a Docker container running
@ronoaldo/pagerender, in the same host that AP runs, but can
be configured to run on different machine.

Bot main code that interacts with the Discord websocket
runs on Compute Engine VM, also as a Docker container.

To run and restart automatically both containers in the
event of a failure, the VM is a Container OS machine,
and it runs `kubelet` in standalone mode.

The fact that the bot does not need a DB is good because
bot can run on multiple guilds without the need to build
specific database schemas or namespaces. The API automatically
stores new player profiles on-demand, as they are requested.
A channel is used to parse and configure the profiles that
will be used by player commands. It may be used in the future
to host other configurations as well.
This is by design and imposes some limitations, such as the inability
to change command prefix or customize the bot runtime.

## Using the hosted version

You can use this link to add the hosted version to your server:
https://discordapp.com/oauth2/authorize?client_id=355873395164053512&scope=bot&permissions=511040

This link will as you to which server the bot will be in. Bot
permissions are required to be active or it may not work.
Bot will parse the #swgho-gg channel (magick name, sorry!)
automatically and link the mentioned user or the user posting
a profile like to that profile. This way, one can easily
invoke the bot.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	token    = flag.String("token", os.Getenv("BOT_TOKEN"), "Token to connect to the discord api.")
	devMode  = flag.Bool("dev", asBool(os.Getenv("USE_DEV")), "Use development mode.")
	swggUser = flag.String("swgoh-user", os.Getenv("SWGOHGG_USER"), "Username to be used to contact swgoh.gg.")
	swggPass = flag.String("swgoh-pass", os.Getenv("SWGOHGG_PASS"), "Password to be used to contact swgoh.gg.")

	cmdPrefix  = flag.String("cmd-prefix", "/", "The command `prefix` to be used by the bot")
	guildCache = make(map[string]*Cache)
	apiCache   = NewAPICache()

	logger = &Logger{Guild: "~MAIN~"}

	renderPageHost = "http://localhost:8080"

	dispatcher = NewDispatcher()
)

// init is called before main, after var block is defined.
// This function will setup the commands to be used in the bot
// main program.
func init() {
	dispatcher.Handle("help", CmdFunc(cmdHelp))
	dispatcher.Handle("arena", CmdFunc(cmdArena))
	dispatcher.Handle("stats", CmdFunc(cmdStats))
	dispatcher.Handle("info", CmdFunc(cmdStats))
	dispatcher.Handle("mods", CmdFunc(cmdMods))
	dispatcher.Handle("faction", CmdFunc(cmdFaction))
	dispatcher.Handle("lookup", CmdFunc(cmdLookup))
	dispatcher.Handle("server-info", cmdDisabled(
		"~~i was doing a DDoS~~ the command was consuming too many resources;"+
			" it will be back soon")) // CmdFunc(cmdServerInfo))
	dispatcher.Handle("share-this-bot", CmdFunc(cmdShareThisBot))

	// Undocumented on pourpose
	dispatcher.Handle("guilds-i-am-running", CmdFunc(cmdBotStats))
	dispatcher.Handle("reload-profiles", CmdFunc(cmdReloadProfiles))
	dispatcher.Handle("leave-guild", CmdFunc(cmdLeaveGuild))
}

// main runs the main loop of our bot application.
func main() {
	flag.Parse()
	// Check we have a token
	if *token == "" {
		logger.Fatalf("Error initializing bot: missing token")
	}
	// When using linked docker containers, lookup for pagerender addr
	renderContainer := os.Getenv("PAGERENDER_PORT_8080_TCP_ADDR")
	if renderContainer != "" {
		renderPageHost = fmt.Sprintf("http://%s:8080", renderContainer)
	}
	logger.Printf("Using rendering service at %v", renderPageHost)

	// Start the websocket listener
	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		logger.Fatalf("Error initializing: %v", err)
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		logger.Fatalf("Error opening websocket: %v", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill, syscall.SIGKILL)
	<-sc
	fmt.Println("Trying to close sessions...")
	dg.Close()
}

// ready handles the event of a sucessfull startup and Discord API login as bot.
func ready(s *discordgo.Session, event *discordgo.Ready) {
	version := os.Getenv("BOT_VERSION")
	name := fmt.Sprintf("AP-5R Protocol Droid")
	if *devMode {
		name = name + " Beta"
	}
	s.UpdateStatus(0, "/help (Version: "+version+")")
	if u, err := s.UserUpdate("", "", name, "", ""); err != nil {
		logger.Errorf("Could not update profile: %v", err)
	} else {
		logger.Infof("Profile updated: %v", u)
	}
	listMyGuilds(s)
}

// onGuildJoin handles the event of joining a guild.
func onGuildJoin(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		return
	}
	logger.Printf("JOIN: new guild: %v", event.Name)
	logger.Printf("Channels: ")
	for _, channel := range event.Guild.Channels {
		logger.Printf("> #%v: %v", channel.Name, channel.ID)
	}
}

// messageCreate handles the Discord event of a new message in a channel.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if err := dispatcher.Dispatch(s, m); err != nil {
		logger.Errorf("unable to handle command: %v", err)
	}
}

// copyrightFooter is a reusable embed footer.
var copyrightFooter = &discordgo.MessageEmbedFooter{
	IconURL: "https://swgoh.gg/static/logos/swgohgg-logo-twitter-profile.png",
	Text:    "(C) https://swgoh.gg/",
}

// embedColor is the default color for embeds.
var embedColor = 0x00d1db

// send is a helper function that formats a text message and send to the target channel.
func send(s *discordgo.Session, channelID, message string, args ...interface{}) (*discordgo.Message, error) {
	m, err := s.ChannelMessageSend(channelID, fmt.Sprintf(message, args...))
	return m, err
}

// cleanup attempts to delete a posted message, if existent.
// Used to remove "i am loading stuff", temporary messages the bot issues.
func cleanup(s *discordgo.Session, m *discordgo.Message) {
	if s == nil || m == nil {
		logger.Infof("Skipped message clean up (%v, %v)", s, m)
		return
	}
	err := s.ChannelMessageDelete(m.ChannelID, m.ID)
	if err != nil {
		logger.Errorf("Unable to delete message#%v: %v", m.ID, err)
	}
}

// askForProfile explains to the user how to provide profile information.
func askForProfile(s *discordgo.Session, m *discordgo.MessageCreate, cmd string) {
	msg := "%s, not sure if I told you before, but you can setup your" +
		" profile at #swgoh-gg so I know where to look at. Otherwise, tell" +
		" me a profile name in [], like: /%s [ronoaldo] ..."
	send(s, m.ChannelID, msg, m.Author.Mention(), cmd)
}

// newAttachment creates a new attachment for the provided image, using the specified name.
func newAttachment(b []byte, name string) []*discordgo.File {
	return []*discordgo.File{
		&discordgo.File{
			Name:        name,
			ContentType: "image/jpg",
			Reader:      bytes.NewBuffer(b),
		},
	}
}

// prefetch downloads and discards an URL. It is intended to fetch and to let server
// cache data.
func download(logger *Logger, url string) ([]byte, error) {
	resp, err := http.Get(url)
	logger.Printf("PREF: %s prefetched (resp %v)", url, resp)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 299 {
		return b, fmt.Errorf("download: error downloading image: %v: %v", resp.StatusCode, string(b))
	}
	return b, err
}

// listMyGuilds list all my guilds currently working on.
func listMyGuilds(s *discordgo.Session) int {
	last := ""
	count := 0
	pageSize := 100
	for {
		guilds, err := s.UserGuilds(100, "", last)
		if err != nil {
			logger.Errorf("Unable to list guilds: %v", err)
			return -1
		}
		for _, g := range guilds {
			logger.Printf("+ Watching guild '%v' (%v)", g.Name, g.ID)
			count++
			last = g.ID
		}
		if len(guilds) < pageSize {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return count
}

// renderImageAt calls the pageRender server and returns the image bytes using download().
func renderImageAt(logger *Logger, targetURL, querySelector, click, size string) ([]byte, error) {
	renderURL := fmt.Sprintf("%s/pageRender?url=%s&querySelector=%s&clickSelector=%s&size=%s&ts=%d",
		renderPageHost, esc(targetURL), querySelector, click, size, time.Now().UnixNano())
	return download(logger, renderURL)
}

// logJSON takes a value and serializes it to the log stream  as a JSON
// object. This is intended for debugging only.
func logJSON(m string, v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		logger.Errorf("Could not json.Marshal %v: %v", v, err)
		return
	}
	logger.Printf(" %s: %s", m, string(b))
}

// asBool is an error-safe parse bool function.
// returns false if unable to parse the input as boolan.
func asBool(src string) bool {
	res, err := strconv.ParseBool(src)
	if err != nil {
		return false
	}
	return res
}

// esc is a shorthand for url.QueryEscape.
func esc(src string) string {
	return url.QueryEscape(src)
}

// unquote attempts to url.QueryUnescape the provided value.
// returns the original string if unable to escape.
func unquote(src string) string {
	if dst, err := url.QueryUnescape(src); err == nil {
		return dst
	}
	return src
}

var nonDigits = regexp.MustCompile("[^0-9]")

func isAllyCode(src string) (string, bool) {
	dst := nonDigits.ReplaceAllString(src, "")
	if len(dst) == 9 {
		return dst, true
	}
	return src, false
}

func factionName(src string) string {
	switch strings.ToLower(src) {
	case "imperial troopers":
		return "Imperial Troopers"
	case "bh", "bounty hunter", "bounty hunters":
		return "Bounty Hunters"
	}

	return strings.Title(src)
}

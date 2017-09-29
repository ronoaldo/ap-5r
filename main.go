package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"ronoaldo.gopkg.net/swgoh/swgohgg"

	"github.com/bwmarrin/discordgo"
)

var (
	tokenProd = "MzU1ODczMzk1MTY0MDUzNTEy.DKNDaw.5z1RFro_lwhNxeWAXEgkLCZze8k"
	tokenDev  = "MzYwNTUxMzQyODgxNjM2MzU1.DKXNJA.dt-WP50VAfItRGHQZgpgoje_Y10"
	useDev    = flag.Bool("dev", asBool(os.Getenv("USE_DEV")), "Use development mode")

	guildCache = make(map[string]*Cache)

	logger = &Logger{Guild: "~MAIN~"}
)

// main runs the main loop of our bot application.
func main() {
	flag.Parse()
	var token = tokenProd
	if *useDev {
		token = tokenDev
	}

	dg, err := discordgo.New("Bot " + token)
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

var helpMessage = `Hi %s, I'm AP-5R and I'm the Empire protocol droid unit that survivied the Death Star destruction. While I understand many languages, please use the following commands to contact me in this secure channel:

**/mods** *character*: if you want me to deliver an image of your mods on a character
**/stats** *character*: if you want me to display your current character stats
**/faction** *faction*: if you want me to display an image of your characters in a faction (*USE SINGULAR* not plural: droid and not droids)
**/server-info** *character*: if you want me to do some number chrunch and display server-wide stats about a character
**/reload-profiles**: this can be used to instruct me to do a reload of profiles. You don't need to, but just in case.

I'll assume that all users shared their profile at the #swgoh-gg channel. Please ask your server admin to create one. This is a requirement for me to properly function here.`

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Skip messages from self or non-command messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Load data from cache to prepare for command parsing.
	channel, err := s.Channel(m.ChannelID)
	if err != nil && strings.HasPrefix("/") {
		logger.Errorf("Error loading channel: %v", err)
		send(s, m.ChannelID, "Oh, no. This should not happen. Unable to identify channel for this message!")
		return
	}
	guild, err := s.Guild(channel.GuildID)
	if err != nil && strings.HasPrefix("/") {
		logger.Errorf("Error loading channel: %v", err)
		send(s, m.ChannelID, "Oh, no. This should not happen. Unable to identify server for this message!")
		return
	}
	logger := &Logger{Guild: guild.Name}
	cache, ok := guildCache[channel.GuildID]
	if !ok {
		//TODO: Lock cache for write?
		logger.Printf("No cache for guild ID %s, initializing one", channel.GuildID)
		// Initialize new cache and build guild profile cache
		cache = NewCache(channel.GuildID, guild.Name)
		cache.ReloadProfiles(s)
		guildCache[channel.GuildID] = cache
	}
	// If message is from swgoh-gg, reload profiles. if not, discard
	if channel.Name == "swgoh-gg" {
		cache.ReloadProfiles(s)
		if strings.HasPrefix(m.Content, "/") {
			send(s, m.ChannelID, "Sorry, let's keep this channel for profile links only!")
		}
		return
	}
	if !strings.HasPrefix(m.Content, "/") {
		return
	}
	logger.Printf("RECV: (#%v) %v: %v", channel.Name, m.Author, m.Content)
	if strings.HasPrefix(m.Content, "/help") {
		send(s, m.ChannelID, helpMessage, m.Author.Mention())
	} else if strings.HasPrefix(m.Content, "/mods") {
		args := strings.Fields(m.Content)[1:]
		profile, ok := cache.UserProfile(m.Author.String())
		if !ok {
			send(s, m.ChannelID, "Oh, interesintg. %s, it looks like you forgot to setup your profile at #swgoh-gg.", m.Author.Mention())
			return
		}
		char := strings.TrimSpace(strings.Join(args, " "))
		if char == "" {
			send(s, m.ChannelID, "%s, use this command with a character name. Try this: /mods tfp", m.Author.Mention())
			return
		}
		send(s, m.ChannelID, "Roger Roger! Let me check mods for '%s' on '%s' profile...", char, profile)
		targetUrl := fmt.Sprintf("https://swgoh.gg/u/%s/collection/%s/", profile, swgohgg.CharSlug(swgohgg.CharName(char)))
		querySelector := ".list-group.media-list.media-list-stream:nth-child(2)"
		renderUrl := renderImageAt(logger, targetUrl, querySelector)
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s's %s mods", unquote(profile), swgohgg.CharName(char)),
			Description: "Here is the thing you asked " + m.Author.Mention(),
			Image: &discordgo.MessageEmbedImage{
				URL: renderUrl,
			},
		})
	} else if strings.HasPrefix(m.Content, "/info") || strings.HasPrefix(m.Content, "/stats") {
		args := strings.Fields(m.Content)[1:]
		profile, ok := cache.UserProfile(m.Author.String())
		if !ok {
			send(s, m.ChannelID, "%s, not sure if I told you before, but you forgot to setup your profile at #swgoh-gg", m.Author.Mention())
			return
		}
		char := strings.TrimSpace(strings.Join(args, " "))
		if char == "" {
			send(s, m.ChannelID, "Good, you are learning! But you need to provide a character name. Try /info tfp")
			return
		}
		c := swgohgg.NewClient(profile)
		stats, err := c.CharacterStats(char)
		if err != nil {
			send(s, m.ChannelID, "Oops, that did not worked as expected: %v. I hope nothing is broken ....", err.Error())
			return
		}
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title: fmt.Sprintf("%s stats for %s", unquote(profile), swgohgg.CharName(char)),
			Fields: []*discordgo.MessageEmbedField{
				{"Basic", fmt.Sprintf("%d* G%d", stats.Stars, stats.GearLevel), true},
				{"Health", strconv.FormatInt(stats.Health, 10), true},
				{"Protection", strconv.FormatInt(stats.Protection, 10), true},
				{"Speed", strconv.FormatInt(stats.Speed, 10), true},
				{"Potency", fmt.Sprintf("%.02f%%", stats.Potency), true},
				{"Tenacity", fmt.Sprintf("%.02f%%", stats.Tenacity), true},
				{"Critical Damage", fmt.Sprintf("%d%%", stats.CriticalDamage), true},
				{"Physical Damage", fmt.Sprintf("%d", stats.PhysicalDamage), true},
				{"Special Damage", fmt.Sprintf("%d", stats.SpecialDamate), true},
			},
		})
	} else if strings.HasPrefix(m.Content, "/faction") {
		args := strings.Fields(m.Content)[1:]
		profile, ok := cache.UserProfile(m.Author.String())
		if !ok {
			send(s, m.ChannelID, "%s, not sure if I told you before, but you forgot to setup your profile at #swgoh-gg", m.Author.Mention())
			return
		}
		filter := strings.TrimSpace(strings.Join(args, " "))
		if filter == "" {
			send(s, m.ChannelID, "Please provide a faction! Try /faction Empire")
			return
		}
		send(s, m.ChannelID, "Checking %s characters on %s faction ... This may take some time.", unquote(profile), filter)
		filter = strings.Replace(strings.ToLower(filter), " ", "-", -1)
		if filter == "rebel-scum" || filter == "terrorists" {
			filter = "rebel"
		}
		targetUrl := fmt.Sprintf("https://swgoh.gg/u/%s/collection/?f=%s", profile, filter)
		querySelector := ".collection-char-list"
		renderUrl := renderImageAt(logger, targetUrl, querySelector)
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s's characters from %s", unquote(profile), filter),
			Description: "There we go " + m.Author.Mention(),
			Image: &discordgo.MessageEmbedImage{
				URL: renderUrl,
			},
		})
	} else if strings.HasPrefix(m.Content, "/reload-profiles") {
		send(s, m.ChannelID, "Copy that. I'll scan the channel #swgoh-gg again...")
		count, err := cache.ReloadProfiles(s)
		if err != nil {
			send(s, m.ChannelID, "Oh no! We're doomed! (err=%v)", err)
			return
		}
		send(s, m.ChannelID, "Reloaded profiles for the server. I found %d valid links.", count)
	} else if strings.HasPrefix(m.Content, "/server-info") {
		args := strings.Fields(m.Content)[1:]
		char := strings.TrimSpace(strings.Join(args, " "))
		if char == "" {
			send(s, m.ChannelID, "Oh, there we go again. You need to provide me a character name. Try /server-info tfp")
			return
		}
		guildProfiles := cache.ListProfiles()
		send(s, m.ChannelID, "Loading %d profiles in the server. This may take a while. You can take some tea.", len(guildProfiles))
		stars := make(map[int]int)
		gear := make(map[int]int)
		zetaCount := make(map[string]int)

		total := 0
		gg := swgohgg.NewClient("")
		allZetas, err := gg.Zetas()
		if err != nil {
			send(s, m.ChannelID, "Warning: I'll be skipping zetas as I could not load them. Something is wrong probably. (err=%v)", err)
		}
		zetas := make([]swgohgg.Ability, 0)
		for _, zeta := range allZetas {
			if strings.ToLower(zeta.Character) == strings.ToLower(swgohgg.CharName(char)) {
				zetas = append(zetas, zeta)
			}
		}
		errCount := 0

		var maxSpeed, avgSpeed, minSpeed int64
		minSpeed = 99999
		for _, profile := range guildProfiles {
			// Fetch char info for each profile
			gg.Profile(profile)
			stats, err := gg.CharacterStats(char)
			time.Sleep(100 * time.Millisecond)
			if err != nil {
				// if 404, the player just does not have him active?
				//send(s, m.ChannelID, "Oops, stopped at %d: %v", i, err.Error())
				logger.Errorf("Unable to fetch character %s for %s: %v", char, profile, err)
				errCount++
				continue
			}
			stars[int(stats.Stars)]++
			gear[int(stats.GearLevel)]++
			if stats.Speed > maxSpeed {
				maxSpeed = stats.Speed
			}
			if stats.Speed < minSpeed && stats.Speed > 0 {
				minSpeed = stats.Speed
			}
			avgSpeed += stats.Speed
			for _, skill := range stats.Skills {
				for _, zeta := range zetas {
					if strings.ToLower(skill.Name) == strings.ToLower(zeta.Name) && skill.Level == 8 {
						zetaCount[zeta.Name]++
					}
				}
			}
			total++
		}
		var msg bytes.Buffer
		fmt.Fprintf(&msg, "From %d %s players, %d have %s\n", len(guildProfiles), guild.Name, total, swgohgg.CharName(char))
		fmt.Fprintf(&msg, "\n*Stars:*\n")
		for i := 7; i >= 1; i-- {
			count, ok := stars[i]
			if !ok {
				continue
			}
			fmt.Fprintf(&msg, "**%d** at %d stars\n", count, i)
		}
		fmt.Fprintf(&msg, "\n*Gear:*\n")
		for i := 12; i >= 1; i-- {
			count, ok := gear[i]
			if !ok {
				continue
			}
			fmt.Fprintf(&msg, "**%d** at G%d\n", count, i)
		}
		fmt.Fprintf(&msg, "\n*Zetas:*\n")
		for zeta, count := range zetaCount {
			fmt.Fprintf(&msg, "**%d** zetas on *%s*\n", count, zeta)
		}
		if len(zetaCount) == 0 {
			fmt.Fprintf(&msg, "No one was brave enough! Or the caracter has no zetas. I'm not sure...\n")
		}
		fmt.Fprintf(&msg, "\n*Fun fact*\n")
		fmt.Fprintf(&msg, "Average speed is %.02f, with the "+
			"faster at %d and the slower at %d", float64(avgSpeed)/float64(total), maxSpeed, minSpeed)
		logger.Printf("INFO: %d profiles seems to be down. Need to improve error detection.", errCount)
		send(s, m.ChannelID, msg.String())
	} else if strings.HasPrefix(m.Content, "/share-this-bot") {
		msg := "AP-5R protocol droid is able to join other servers, but you need to follow this instructions:\n" +
			"> Join the Bot Users Playground at https://discord.gg/4GJ8Ty2\n" +
			"> Be a nice person\n" +
			"> Follow instructions in the #info channel on that server\n"
		send(s, m.ChannelID, msg)
		return
	} else if strings.HasPrefix(m.Content, "/guilds-i-am-running") {
		desc, quant := listMyGuilds(s)
		send(s, m.ChannelID, "Running on %d guilds:", quant)
		if err := send(s, m.ChannelID, desc); err != nil {
			logger.Errorf("Unable to send message %v", err)
		}
	} else if strings.HasPrefix(m.Content, "/leave-guild") {
		args := strings.Fields(m.Content)[1:]
		if len(args) < 1 {
			send(s, m.ChannelID, "Please inform the guild ID to leave")
			return
		}
		if err := s.GuildLeave(args[0]); err != nil {
			send(s, m.ChannelID, "Error leaving guild %s", args[0])
			logger.Errorf("Error leaving guild: %v", err)
			return
		}
		send(s, m.ChannelID, "Left guild.")
		listMyGuilds(s)
	}
}

// send is a helper function that formats a text message and send to the target channel.
func send(s *discordgo.Session, channelID, message string, args ...interface{}) error {
	_, err := s.ChannelMessageSend(channelID, fmt.Sprintf(message, args...))
	return err
}

// prefetch downloads and discards an URL. It is intended to fetch and to let server
// cache data.
func prefetch(logger *Logger, url string) error {
	resp, err := http.Head(url)
	logger.Printf("PREF: %s prefetched (resp %v)", url, resp)
	if err != nil {
		defer resp.Body.Close()
	}
	return err
}

// onGuildJoin is currently responsible to log new guilds. We will be adding
// some welcome message on the first channel, or to the person that adds the
// bot to the channel.
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

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {
	version := os.Getenv("BOT_VERSION")
	name := fmt.Sprintf("RA-7 Protocol Droid")
	if *useDev {
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

// listMyGuilds list all my guilds currently working on.
func listMyGuilds(s *discordgo.Session) (string, int) {
	var buff bytes.Buffer
	guilds, err := s.UserGuilds(100, "", "")
	if err != nil {
		logger.Errorf("Unable to list guilds: %v", err)
		return err.Error(), 0
	}
	count := 0
	for _, g := range guilds {
		fmt.Fprintf(&buff, "+ Watching guild '%v' (%v)\n", g.Name, g.ID)
		logger.Printf("+ Watching guild '%v' (%v)", g.Name, g.ID)
		count++
	}
	return buff.String(), count
}

func renderImageAt(logger *Logger, targetUrl, querySelector string) string {
	renderPageHost := "https://us-central1-ronoaldoconsulting.cloudfunctions.net"
	renderUrl := fmt.Sprintf("%s/pageRender?url=%s&querySelector=%s&ts=%d",
		renderPageHost, url.QueryEscape(targetUrl), querySelector, time.Now().UnixNano())
	prefetch(logger, renderUrl)
	return renderUrl
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

func asBool(src string) bool {
	res, err := strconv.ParseBool(src)
	if err != nil {
		return false
	}
	return res
}

func unquote(src string) string {
	if dst, err := url.QueryUnescape(src); err == nil {
		return dst
	}
	return src
}

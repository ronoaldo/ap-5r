package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
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
		log.Fatal("Error initializing: " + err.Error())
	}

	dg.AddHandler(ready)
	dg.AddHandler(messageCreate)

	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening: " + err.Error())
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill, syscall.SIGKILL)
	<-sc
	fmt.Println("Trying to close sessions...")
	dg.Close()
}

var helpMessage = `You can use the following commands:

	/mods character : display mods on a character
	/info character : display character basic stats
	/server-info character : display server-wide character stats

You need to setup your profile at the #swgoh-gg channel`

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Skip messages from self or non-command messages
	if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "/") {
		return
	}

	// Load data from cache to prepare for command parsing.
	channel, err := s.Channel(m.ChannelID)
	if err != nil {
		send(s, m.ChannelID, "Be-boooh-bo... unable to identify channel for this message")
		return
	}
	cache, ok := guildCache[channel.GuildID]
	if !ok {
		log.Printf("INFO: no cache for guild ID %s, initializing one", channel.GuildID)
		// Initialize new cache and build guild profile cache
		cache = NewCache(channel.GuildID)
		cache.ReloadProfiles(s)
	}
	log.Printf("INFO: parsing command on guild %s. Cached profiles ready")

	log.Printf("RECV: #%v %v: %v", m.ChannelID, m.Author, m.Content)
	if strings.HasPrefix(m.Content, "/help") {
		send(s, m.ChannelID, helpMessage)
	} else if strings.HasPrefix(m.Content, "/mods") {
		args := strings.Fields(m.Content)[1:]
		cache.ReloadProfiles(s)
		profile, ok := cache.UserProfile(m.Author.String())
		if !ok {
			send(s, m.ChannelID, "Be-booh-bo! @%s, it looks like you forgot to setup your profile at #swgoh-gg", m.Author.String())
			return
		}
		char := strings.TrimSpace(strings.Join(args, " "))
		if char == "" {
			send(s, m.ChannelID, "Be-booh-bo... you need to provide a character name. Try /mods tfp")
			return
		}
		send(s, m.ChannelID, "Be-boop! Let me check mods for '%s' on '%s' profile...", char, profile)
		targetUrl := fmt.Sprintf("https://swgoh.gg/u/%s/collection/%s/", profile, swgohgg.CharSlug(swgohgg.CharName(char)))
		querySelector := ".list-group.media-list.media-list-stream:nth-child(2)"
		renderPageHost := "https://us-central1-ronoaldoconsulting.cloudfunctions.net"
		renderUrl := fmt.Sprintf("%s/pageRender?url=%s&querySelector=%s&ts=%d", renderPageHost, targetUrl, querySelector, time.Now().UnixNano())
		prefetch(renderUrl)
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s's %s mods", profile, swgohgg.CharName(char)),
			Description: "@" + m.Author.String(),
			Image: &discordgo.MessageEmbedImage{
				URL: renderUrl,
			},
		})
	} else if strings.HasPrefix(m.Content, "/reload-profiles") {
		cache.ReloadProfiles(s)
		send(s, m.ChannelID, "Reloaded profiles for the server.")
	} else if strings.HasPrefix(m.Content, "/info") {
		args := strings.Fields(m.Content)[1:]
		cache.ReloadProfiles(s)
		profile, ok := cache.UserProfile(m.Author.String())
		if !ok {
			send(s, m.ChannelID, "Be-booh-bo! @%s, it looks like you forgot to setup your profile at #swgoh-gg", m.Author.String())
			return
		}
		char := strings.TrimSpace(strings.Join(args, " "))
		if char == "" {
			send(s, m.ChannelID, "Be-booh-bo... you need to provide a character name. Try /info tfp")
			return
		}
		c := swgohgg.NewClient(profile)
		stats, err := c.CharacterStats(char)
		if err != nil {
			send(s, m.ChannelID, "Oops, that did not worked: "+err.Error())
			return
		}
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title: fmt.Sprintf("%s stats for %s", profile, swgohgg.CharName(char)),
			Fields: []*discordgo.MessageEmbedField{
				{"Health", strconv.FormatInt(stats.Health, 10), true},
				{"Protection", strconv.FormatInt(stats.Protection, 10), true},
				{"Speed", strconv.FormatInt(stats.Speed, 10), true},
				{"Potency", fmt.Sprintf("%.02f%%", stats.Potency), true},
				{"Tenacity", fmt.Sprintf("%.02f%%", stats.Tenacity), true},
				{"Critical Damage", fmt.Sprintf("%d%%", stats.CriticalDamage), true},
			},
		})
	} else if strings.HasPrefix(m.Content, "/server-info") {
		args := strings.Fields(m.Content)[1:]
		char := strings.TrimSpace(strings.Join(args, " "))
		if char == "" {
			send(s, m.ChannelID, "Be-booh-bo... you need to provide a character name. Try /server-info tfp")
			return
		}
		guildProfiles := cache.ListProfiles()
		send(s, m.ChannelID, "Be-boop! Loading %d profiles in the server. This may take a while...", len(guildProfiles))
		stars := make(map[int]int)
		gear := make(map[int]int)
		zetaCount := make(map[string]int)

		total := 0
		gg := swgohgg.NewClient("")
		allZetas, err := gg.Zetas()
		if err != nil {
			send(s, m.ChannelID, "Warning: I'll be skipping zetas as I could not load them")
		}
		zetas := make([]swgohgg.Ability, 0)
		for _, zeta := range allZetas {
			if strings.ToLower(zeta.Character) == strings.ToLower(swgohgg.CharName(char)) {
				zetas = append(zetas, zeta)
			}
		}
		for _, profile := range guildProfiles {
			// Fetch char info for each profile
			gg.Profile(profile)
			stats, err := gg.CharacterStats(char)
			time.Sleep(100 * time.Millisecond)
			if err != nil {
				// if 404, the player just does not have him active?
				//send(s, m.ChannelID, "Oops, stopped at %d: %v", i, err.Error())
				log.Printf("ERROR: %v", err)
				continue
			}
			stars[int(stats.Stars)]++
			gear[int(stats.GearLevel)]++
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
		fmt.Fprintf(&msg, "From %d confirmed players, %d have %s\n", len(guildProfiles), total, swgohgg.CharName(char))
		fmt.Fprintf(&msg, "\n*Stars:*\n")
		for i := 7; i >= 1; i-- {
			count, ok := stars[i]
			if !ok {
				continue
			}
			fmt.Fprintf(&msg, "**%d** at %d stars\n", count, i)
		}
		fmt.Fprintf(&msg, "\n*Gear:*\n")
		for i := 12; i > 1; i-- {
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
			fmt.Fprintf(&msg, "No one was brave enough")
		}
		send(s, m.ChannelID, msg.String())
	} else if strings.HasPrefix(m.Content, "/share-this-bot") {
		if *useDev {
			send(s, m.ChannelID, "https://discordapp.com/oauth2/authorize?client_id=360551342881636355&scope=bot&permissions=511040")
			return
		}
		send(s, m.ChannelID, "https://discordapp.com/oauth2/authorize?client_id=355873395164053512&scope=bot&permissions=511040")
		return
	}
}

// send is a helper function that formats a text message and send to the target channel.
func send(s *discordgo.Session, channelID, message string, args ...interface{}) error {
	_, err := s.ChannelMessageSend(channelID, fmt.Sprintf(message, args...))
	return err
}

// prefetch downloads and discards an URL. It is intended to fetch and to let server
// cache data.
func prefetch(url string) error {
	resp, err := http.Head(url)
	log.Printf("PREF: %s prefetched (resp %v)", url, resp)
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
	log.Printf("JOIN: new guild: %v", event.Name)
	log.Printf("Channels: ")
	for _, channel := range event.Guild.Channels {
		log.Printf("> #%v: %v", channel.Name, channel.ID)
	}
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {
	version := os.Getenv("BOT_VERSION")
	s.UpdateStatus(0, "Defeating the rebel scum at Arena")
	s.UserUpdate("", "", fmt.Sprintf("RA-7 Protocol Droid (%s)", version), "", "")
}

// logJSON takes a value and serializes it to the log stream  as a JSON
// object. This is intended for debugging only.
func logJSON(m string, v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("ERR: Could not json.Marshal %v: %v", v, err)
		return
	}
	log.Printf(" %s: %s", m, string(b))
}

func asBool(src string) bool {
	res, err := strconv.ParseBool(src)
	if err != nil {
		return false
	}
	return res
}

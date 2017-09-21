package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"regexp"

	"ronoaldo.gopkg.net/swgoh/swgohgg"

	"github.com/bwmarrin/discordgo"
)

var token = "MzU1ODczMzk1MTY0MDUzNTEy.DKNDaw.5z1RFro_lwhNxeWAXEgkLCZze8k"

func main() {
	if token == "" {
		log.Fatal("You need to specify the token")
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
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

var helpMessage = `You can use the following commands:

	/mods character : display mods on a character
	/reload-profiles : read again the #swgoh-gg channel links
	/info character : display character basic stats

You need to setup your profile at the #swgoh-gg channel`

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Printf("RECV: #%v %v: %v", m.ChannelID, m.Author, m.Content)
	if m.Author.ID == s.State.User.ID {
		return
	}
	if strings.HasPrefix(m.Content, "/help") {
		send(s, m.ChannelID, helpMessage)
	} else if strings.HasPrefix(m.Content, "/mods") {
		args := strings.Fields(m.Content)[1:]
		profile, ok := profiles[m.Author.String()]
		if !ok {
			send(s, m.ChannelID, "Be-booh-bo! @%s, it looks like you forgot to setup your profile at #swgoh-gg", m.Author.String())
			loadProfiles(s)
			return
		}
		char := strings.TrimSpace(strings.Join(args, " "))
		if char == "" {
			send(s, m.ChannelID, "Be-booh-bo... you need to provide a character name. Try /mods tfp")
			return
		}
		send(s, m.ChannelID, "Be-boop! Let me check your mods for '%s' @ '%s' ...", char, profile)
		targetUrl := fmt.Sprintf("https://swgoh.gg/u/%s/collection/%s/", profile, swgohgg.CharSlug(swgohgg.CharName(char)))
		querySelector := ".list-group.media-list.media-list-stream:nth-child(2)"
		renderPageHost := "https://us-central1-ronoaldoconsulting.cloudfunctions.net"
		renderUrl := fmt.Sprintf("%s/pageRender?url=%s&querySelector=%s", renderPageHost, targetUrl, querySelector)
		prefetch(renderUrl)
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("%s mods on %s", profile, swgohgg.CharName(char)),
			Description: "@" + m.Author.String(),
			Image: &discordgo.MessageEmbedImage{
				URL: renderUrl,
			},
		})
	} else if strings.HasPrefix(m.Content, "/reload-profiles") {
		loadProfiles(s)
	} else if strings.HasPrefix(m.Content, "/info") {
		args := strings.Fields(m.Content)[1:]
		profile, ok := profiles[m.Author.String()]
		if !ok {
			send(s, m.ChannelID, "Be-booh-bo! @%s, it looks like you forgot to setup your profile at #swgoh-gg", m.Author.String())
			loadProfiles(s)
			return
		}
		char := strings.TrimSpace(strings.Join(args, " "))
		if char == "" {
			send(s, m.ChannelID, "Be-booh-bo... you need to provide a character name. Try /mods tfp")
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
	}
}

func send(s *discordgo.Session, channelID, message string, args ...interface{}) error {
	_, err := s.ChannelMessageSend(channelID, fmt.Sprintf(message, args...))
	return err
}

func prefetch(url string) error {
	resp, err := http.Head(url)
	log.Printf("PREF: %s prefetched (resp %v)", url, resp)
	if err != nil {
		defer resp.Body.Close()
	}
	return err
}

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

var (
	profiles    = make(map[string]string)
	SWGoHChanId = "350342664006270976"
)

var profileRe = regexp.MustCompile("https?://swgoh.gg/u/([^/]+)/?")

func extractProfile(src string) string {
	results := profileRe.FindAllStringSubmatch(src, -1)
	if len(results) == 0 {
		return ""
	}
	if len(results[0]) == 0 {
		return ""
	}
	return results[0][1]
}

func loadProfiles(s *discordgo.Session) {
	guilds, err := s.UserGuilds(0, "", "")
	if err != nil {
		log.Printf("Error loading my guilds...")
		return
	}
	for _, guild := range guilds {
		log.Printf("> Looking up #swgoh-gg channel for guild %s", guild.Name)
		// Lookup the #swgoh-gg channel to parse profiles
		channels, err := s.GuildChannels(guild.ID)
		if err != nil {
			log.Printf("Error loading channels. Skipping this guild (%v)", err)
			continue
		}
		chanID := ""
		for _, ch := range channels {
			if ch.Name == "swgoh-gg" {
				chanID = ch.ID
				break
			}
		}
		if chanID == "" {
			log.Printf("No channel ID found with name #swgoh-gg. Skipping this guild.")
			continue
		}
		after := ""
		for {
			log.Printf("Loading messages after %s", after)
			messages, err := s.ChannelMessages(chanID, 100, "", after, "")
			if err != nil {
				log.Printf("ERR: loading messages from #swgoh-gg channel: %v", err)
			}
			log.Printf("Currently with %d", len(messages))
			for _, m := range messages {
				after = m.ID
				profile := extractProfile(m.Content)
				if profile == "" {
					log.Printf("Not a valid profile %v", m.Content)
					continue
				}
				profiles[m.Author.String()] = profile
				log.Printf("Detected %v: %v", m.Author, profile)
			}
			if len(messages) <= 100 {
				break
			}
			log.Printf("Waiting a bit to avoid doing shit ...")
			time.Sleep(10 * time.Second)
		}
	}
	log.Printf("Full profile list loaded %d", len(profiles))
}

// This function will be called (due to AddHandler above) when the bot receives
// the "ready" event from Discord.
func ready(s *discordgo.Session, event *discordgo.Ready) {
	s.UpdateStatus(0, "Galactic War")
	loadProfiles(s)
}

func logJSON(m string, v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("ERR: Could not json.Marshal %v: %v", v, err)
		return
	}
	log.Printf(" %s: %s", m, string(b))
}

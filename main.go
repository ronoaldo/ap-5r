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
	// TODO: move this tokens out of source code bro!
	tokenProd = "MzU1ODczMzk1MTY0MDUzNTEy.DKNDaw.5z1RFro_lwhNxeWAXEgkLCZze8k"
	tokenDev  = "MzYwNTUxMzQyODgxNjM2MzU1.DKXNJA.dt-WP50VAfItRGHQZgpgoje_Y10"
	useDev    = flag.Bool("dev", asBool(os.Getenv("USE_DEV")), "Use development mode")

	guildCache = make(map[string]*Cache)
	apiCache   = NewAPICache()

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

var helpMessage = `Hi %s, I'm AP-5R and I'm the Empire protocol droid unit that survived the Death Star destruction. While I understand many languages, please use the following commands to contact me in this secure channel:

**/mods** *character*: display the mods you have on a character.
**/stats** *character*: display character basic stats.
**/arena**: display your current arena basic stats.
**/faction**: display an image of your characters in the given faction. Add +ships, +ship or +s to get ship info.
**/server-info** *character*: if you want me to do some number crunch and display server-wide stats about a character.
**/reload-profiles**: this can be used to instruct me to do a reload of profiles. You don't need to, but just in case.
**/share-this-bot**: if you want my help in a galaxy far, far away...

I'll assume that all users shared their profile at the #swgoh-gg channel. Please ask your server admin to create one. This is a important for me to properly function here. Alternatively, you can use [profile] at the end of /mods and /stats in order to get info from another profile than yours.`

var copyrightFooter = &discordgo.MessageEmbedFooter{
	IconURL: "https://swgoh.gg/static/logos/swgohgg-logo-twitter-profile.png",
	Text:    "(C) https://swgoh.gg/",
}

var embedColor = 0x00d1db

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Skip messages from self or non-command messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Load data from cache to prepare for command parsing.
	channel, err := apiCache.GetChannel(s, m.ChannelID)
	if err != nil && strings.HasPrefix(m.Content, "/") {
		logger.Errorf("Error loading channel: %v", err)
		send(s, m.ChannelID, "Oh, no. This should not happen. Unable to identify channel for this message!")
		return
	}
	guild, err := apiCache.GetGuild(s, channel.GuildID)
	if err != nil && strings.HasPrefix(m.Content, "/") {
		logger.Errorf("Error loading channel: %v", err)
		send(s, m.ChannelID, "Oh, no. This should not happen. Unable to identify server for this message!")
		return
	}
	if guild == nil {
		logger.Errorf("Unexpected error: %v (guild=%v)", err, guild)
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
		send(s, m.ChannelID, helpMessage, m.Author.Username)
	} else if strings.HasPrefix(m.Content, "/mods") {
		args := ParseArgs(m.Content)
		profile, ok := cache.UserProfileIfEmpty(args.Profile, m.Author.ID)
		if len(m.Mentions) > 0 {
			logger.Infof("Using mentioned profile %v", m.Mentions[0])
			// Lookup mentioned profile
			profile, ok = cache.UserProfileIfEmpty(args.Profile, m.Mentions[0].ID)
		}
		if !ok {
			askForProfile(s, m, "stats")
			return
		}
		char := args.Name
		if char == "" {
			send(s, m.ChannelID, "%s, use this command with a character name. Try this: /mods tfp", m.Author.Mention())
			return
		}
		sent, _ := send(s, m.ChannelID, "This might take a while. Let me check mods for '%s' on '%s' profile...", char, profile)
		defer cleanup(s, sent)
		targetUrl := fmt.Sprintf("https://swgoh.gg/u/%s/collection/%s/", profile, swgohgg.CharSlug(swgohgg.CharName(char)))
		querySelector := ".list-group.media-list.media-list-stream:nth-child(2)"
		clickSelector := ".icon.icon-chevron-down.pull-left"
		renderUrl := renderImageAt(logger, targetUrl, querySelector, clickSelector, "desktop")
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: "Here is the thing you asked " + m.Author.Mention(),
			Embed: &discordgo.MessageEmbed{
				Title: fmt.Sprintf("%s's %s mods", unquote(profile), swgohgg.CharName(char)),
				Image: &discordgo.MessageEmbedImage{
					URL: renderUrl,
				},
				Color:  embedColor,
				Footer: copyrightFooter,
			},
		})
	} else if strings.HasPrefix(m.Content, "/info") || strings.HasPrefix(m.Content, "/stats") {
		args := ParseArgs(m.Content)
		profile, ok := cache.UserProfileIfEmpty(args.Profile, m.Author.ID)
		if len(m.Mentions) > 0 {
			logger.Infof("Using mentioned profile %v", m.Mentions[0])
			// Lookup mentioned profile
			profile, ok = cache.UserProfileIfEmpty(args.Profile, m.Mentions[0].ID)
		}
		if !ok {
			askForProfile(s, m, "stats")
			return
		}
		char := args.Name
		if char == "" {
			send(s, m.ChannelID, "Good, you are learning! But you need to provide a character name. Try /info tfp")
			return
		}
		c := swgohgg.NewClient(profile).UseCache(true)
		collection, err := c.Collection()
		if err != nil {
			send(s, m.ChannelID, "Oops, that did not worked as expected: %v. I hope nothing is broken ....", err.Error())
			return
		}
		if !collection.Contains(swgohgg.CharName(char)) {
			send(s, m.ChannelID, "%s, looks like you don't have %s activated yet, do you?", m.Author.Mention(), char)
		}
		stats, err := c.CharacterStats(char)
		if err != nil {
			send(s, m.ChannelID, "Oops, that did not worked as expected: %v. I hope nothing is broken ....", err.Error())
			return
		}
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: fmt.Sprintf("Wow, nice stats %s", m.Author.Mention()),
			Embed: &discordgo.MessageEmbed{
				Title: fmt.Sprintf("%s stats for %s", unquote(profile), swgohgg.CharName(char)),
				Fields: []*discordgo.MessageEmbedField{
					{"Basic", fmt.Sprintf("%d* G%d Lvl %d", stats.Stars, stats.GearLevel, stats.Level), true},
					{"Health", strconv.Itoa(stats.Health), true},
					{"Protection", strconv.Itoa(stats.Protection), true},
					{"Speed", strconv.Itoa(stats.Speed), true},
					{"Potency", fmt.Sprintf("%.02f%%", stats.Potency), true},
					{"Tenacity", fmt.Sprintf("%.02f%%", stats.Tenacity), true},
					{"Critical Damage", fmt.Sprintf("%.02f%%", stats.CriticalDamage), true},
					{"Physical Damage", fmt.Sprintf("%d", stats.PhysicalDamage), true},
					{"Special Damage", fmt.Sprintf("%d", stats.SpecialDamate), true},
				},
				Color:  embedColor,
				Footer: copyrightFooter,
			},
		})
	} else if strings.HasPrefix(m.Content, "/arena") {
		args := ParseArgs(m.Content)
		profile, ok := cache.UserProfileIfEmpty(args.Profile, m.Author.ID)
		if len(m.Mentions) > 0 {
			logger.Infof("Using mentioned profile %v", m.Mentions[0])
			profile, ok = cache.UserProfileIfEmpty(args.Profile, m.Mentions[0].ID)
		}
		if !ok {
			askForProfile(s, m, "/arena")
			return
		}
		sent, _ := send(s, m.ChannelID, "OK, let me check your profile...")
		defer cleanup(s, sent)
		url := fmt.Sprintf("https://swgoh.gg/u/%s/", profile)
		querySelector := ".chart-arena"
		renderUrl := renderImageAt(logger, url, querySelector, "", "")
		gg := swgohgg.NewClient(profile).UseCache(true)
		team, update, err := gg.Arena()
		if err != nil {
			logger.Errorf("Unable to fetch your arena team: %v", err)
			send(s, m.ChannelID, "Oh no! I was unable to fetch your profile named '%s'. Please make sure the information is correct ", profile)
		}
		embed := &discordgo.MessageEmbed{
			Image: &discordgo.MessageEmbedImage{
				URL: renderUrl,
			},
			Title:       fmt.Sprintf("%s current arena team", profile),
			Description: fmt.Sprintf("*Updated at %v", update.Format(time.Stamp)),
			Color:       embedColor,
			Footer:      copyrightFooter,
		}
		for _, char := range team {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("%d* %s G%d", char.Stars, char.Name, char.GearLevel),
				Value:  fmt.Sprintf("%d Speed, %d HP, %d Prot", char.Speed, char.Health, char.Protection),
				Inline: false,
			})
		}
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: fmt.Sprintf("So, there is your team %v", m.Author.Mention()),
			Embed:   embed,
		})
	} else if strings.HasPrefix(m.Content, "/faction") {
		args := ParseArgs(m.Content)
		profile, ok := cache.UserProfileIfEmpty(args.Profile, m.Author.ID)
		if len(m.Mentions) > 0 {
			logger.Infof("Using mentioned profile %v", m.Mentions[0])
			// Lookup mentioned profile
			profile, ok = cache.UserProfileIfEmpty(args.Profile, m.Mentions[0].ID)
		}
		if !ok {
			send(s, m.ChannelID, "%s, not sure if I told you before, but you forgot to setup your profile at #swgoh-gg", m.Author.Mention())
			return
		}
		filter := strings.ToLower(strings.TrimSpace(args.Name))
		if filter == "" {
			send(s, m.ChannelID, "Please provide a faction! Try /faction Empire")
			return
		}
		filter = strings.TrimSuffix(filter, "s")
		displayName := filter
		if displayName == "rebel" {
			displayName = "rebel scum"
		} else if displayName == "imperial trooper" {
			displayName = "empire's finest"
		} else if displayName == "resistance" {
			displayName = "tank raid kings"
		}
		sent, _ := send(s, m.ChannelID, "Checking %s units on %s faction ... This may take some time.", unquote(profile), displayName)
		defer cleanup(s, sent)
		filter = strings.Replace(strings.ToLower(filter), " ", "-", -1)
		if filter == "rebel-scum" || filter == "terrorists" {
			filter = "rebel"
		}
		targetUrl := fmt.Sprintf("https://swgoh.gg/u/%s/collection/?f=%s", profile, filter)
		querySelector := ".collection-char-list"
		if args.ContainsFlag("+ships", "+ship", "+s") {
			targetUrl = fmt.Sprintf("https://swgoh.gg/u/%s/ships/?f=%s", profile, filter)
		}
		renderUrl := renderImageAt(logger, targetUrl, querySelector, "", "desktop")
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: "There we go " + m.Author.Mention(),
			Embed: &discordgo.MessageEmbed{
				Title: fmt.Sprintf("%s's characters tagged '%s'", unquote(profile), displayName),
				Image: &discordgo.MessageEmbedImage{
					URL: renderUrl,
				},
				Color:  embedColor,
				Footer: copyrightFooter,
			},
		})
	} else if strings.HasPrefix(m.Content, "/lookup") {
		args := ParseArgs(m.Content)
		char := swgohgg.CharName(args.Name)
		guildProfiles := cache.ListProfiles()

		minStar := 0
		minGear := 0
		errCount := 0
		hasActive := 0
		resultCount := 0

		cmp := func(a, b int) bool {
			return a >= b
		}
		for _, flag := range args.Flags {
			flag = strings.ToLower(flag)
			if strings.HasSuffix(flag, "star") {
				v, _ := strconv.Atoi(strings.Replace(flag, "star", "", -1))
				if v >= 0 {
					minStar = v
				}
			} else if strings.HasPrefix(flag, "+g") {
				v, _ := strconv.Atoi(strings.Replace(flag, "+g", "", -1))
				if v >= 0 {
					minGear = v
				}
			} else if flag == "+exact" {
				cmp = func(a, b int) bool {
					return a == b
				}
			} else {
				logger.Infof("Unknown flag: %v", flag)
			}
		}
		msg := fmt.Sprintf("Looking for profiles that have %s, ", char)
		if minStar > 0 {
			msg += fmt.Sprintf(" at %d stars, ", minStar)
		}
		if minGear > 0 {
			msg += fmt.Sprintf(" at gear level %d, ", minGear)
		}
		msg += " in the whole server. It takes a long while until I warm up the servers."
		send(s, m.ChannelID, "%s", msg)
		lines := make([]string, 0)
		for i := 0; i < len(guildProfiles); i++ {
			user := guildProfiles[i]
			logger.Infof("Parsing user #%d (%s)", i, user)
			profile, err := GetProfile(user)
			if err != nil {
				logger.Infof("> Error: %v", err)
				errCount++
				continue
			}
			c := profile.Character(swgohgg.CharName(char))
			if c == nil {
				continue
			}
			// Player has the character, lets see if he has the requested flags
			hasActive++
			logger.Infof("> Player has the character")
			ok := false
			switch {
			case minStar > 0 && minGear > 0:
				// Both filters profided
				ok = cmp(c.Stars, minStar) && cmp(c.Gear, minGear)
			case minStar > 0:
				ok = cmp(c.Stars, minStar)
			case minGear > 0:
				ok = cmp(c.Gear, minGear)
			}
			if ok {
				logger.Infof("> Matches all filters!")
				resultCount++
				lines = append(lines, fmt.Sprintf("**%s**", user))
			}
		}
		msg = fmt.Sprintf("%d players have **%s** %v of a total of %d who have it unlocked.", resultCount, char, args.Flags, hasActive)
		if errCount > 0 {
			msg += fmt.Sprintf(" (Unable to parse %d profiles)", errCount)
		}
		send(s, m.ChannelID, "%s", msg)
		// Outputs at most 10 profiles at a time.
		var buff bytes.Buffer
		count := 0
		for i := range lines {
			buff.WriteString(lines[i] + "\n")
			count++
			if count > 100 {
				send(s, m.ChannelID, "%s", buff.String())
				count = 0
				buff.Reset()
			}
		}
		if count > 0 {
			send(s, m.ChannelID, "%s", buff.String())
		}
	} else if strings.HasPrefix(m.Content, "/reload-profiles") {
		send(s, m.ChannelID, "Copy that. I'll scan the channel #swgoh-gg again...")
		count, err := cache.ReloadProfiles(s)
		if err != nil {
			logger.Errorf("Error loading profiles: %v", err)
			send(s, m.ChannelID, "Oh no! We're doomed! Unable to read profiles!")
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
		gg := swgohgg.NewClient("").UseCache(true)
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

		var maxSpeed, avgSpeed, minSpeed int
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
			fmt.Fprintf(&msg, "No one was brave enough! Or the character has no zetas. I'm not sure...\n")
		}
		fmt.Fprintf(&msg, "\n*Fun fact*\n")
		fmt.Fprintf(&msg, "Average speed is %.02f, with the "+
			"fastest at %d and the slowest at %d", float64(avgSpeed)/float64(total), maxSpeed, minSpeed)
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
		quant := listMyGuilds(s)
		send(s, m.ChannelID, "Running on %d guilds:", quant)
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
		" me a profile name too, like in /%s tfp [ronoaldo]"
	send(s, m.ChannelID, msg, m.Author.Mention(), cmd)
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

// ready is the registered callback for when the bot starts.
func ready(s *discordgo.Session, event *discordgo.Ready) {
	version := os.Getenv("BOT_VERSION")
	name := fmt.Sprintf("AP-5R Protocol Droid")
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
func listMyGuilds(s *discordgo.Session) int {
	last := ""
	count := 0
	pageSize := 100
	for {
		guilds, err := s.UserGuilds(100, last, "")
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

// renderImageAt pre-renders and cache the image in the cloud function.
func renderImageAt(logger *Logger, targetUrl, querySelector, click, size string) string {
	renderPageHost := "https://us-central1-ronoaldoconsulting.cloudfunctions.net"
	renderUrl := fmt.Sprintf("%s/pageRender?url=%s&querySelector=%s&click=%s&size=%s&ts=%d",
		renderPageHost, esc(targetUrl), querySelector, click, size, time.Now().UnixNano())
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

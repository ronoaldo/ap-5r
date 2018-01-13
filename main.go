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
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ronoaldo/swgoh/swgohgg"
)

var (
	token   = flag.String("token", os.Getenv("BOT_TOKEN"), "Token to connect to the discord api")
	devMode = flag.Bool("dev", asBool(os.Getenv("USE_DEV")), "Use development mode")

	guildCache = make(map[string]*Cache)
	apiCache   = NewAPICache()

	logger = &Logger{Guild: "~MAIN~"}

	renderPageHost = "http://localhost:8080"
)

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

var helpMessage = `Hi %s, I'm AP-5R and I'm the Empire protocol droid unit that survived the Death Star destruction. While I understand many languages, please use the following commands to contact me in this secure channel:

**/mods** *character*: display the mods you have on a character.
**/stats** *character*: display character basic stats.
**/arena**: display your current arena basic stats. Use +more to get more stats.
**/faction**: display an image of your characters in the given faction. Add +ships, +ship or +s to get ship info.
**/server-info** *character*: if you want me to do some number crunch and display server-wide stats about a character.
**/lookup** *character*: to search and see who has a specific character. +1star .. +7star to filter by star level, and +g1 .. +g12 to filter by gear level.
**/reload-profiles**: this can be used to instruct me to do a reload of profiles. You don't need to, but just in case.
**/share-this-bot**: if you want my help in a galaxy far, far away...

I'll assume that all users shared their profile at the #swgoh-gg channel. Please ask your server admin to create one. This is a important for me to properly function here. Alternatively, you can use [profile] at the end of /mods and /stats in order to get info from another profile than yours.`

// copyrightFooter is a reusable embed footer.
var copyrightFooter = &discordgo.MessageEmbedFooter{
	IconURL: "https://swgoh.gg/static/logos/swgohgg-logo-twitter-profile.png",
	Text:    "(C) https://swgoh.gg/",
}

// embedColor is the default color for embeds.
var embedColor = 0x00d1db

// messageCreate is called by discordgo API when a new message is created
// on any channel the bot is listening to.
//
// This method parses the message and call the apropriate methods that
// reply to the users.
//
// TODO(ronoaldo): it worth refactoring this method to make it define CommandHandlers.
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
		sent, _ := send(s, m.ChannelID, "Command received! Let me check mods for **%s** on **%s**'s profile... :clock130:", char, profile)
		defer cleanup(s, sent)
		targetUrl := fmt.Sprintf("https://swgoh.gg/u/%s/collection/%s/", profile, swgohgg.CharSlug(swgohgg.CharName(char)))
		querySelector := ".list-group.media-list.media-list-stream:nth-child(2)"
		clickSelector := ".icon.icon-chevron-down.pull-left"
		b, err := renderImageAt(logger, targetUrl, querySelector, clickSelector, "desktop")
		if err != nil {
			logger.Errorf("Unable to render image: %v", err)
			send(s, m.ChannelID, "Oh, no! I was unable to create the image :(")
			return
		}
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: "Here is the thing you asked " + m.Author.Mention(),
			Embed: &discordgo.MessageEmbed{
				Title: fmt.Sprintf("%s's %s mods", unquote(profile), swgohgg.CharName(char)),
				URL:   targetUrl,
				Image: &discordgo.MessageEmbedImage{
					URL: "attachment://image.jpg",
				},
				Color:  embedColor,
				Footer: copyrightFooter,
			},
			Files: newAttachment(b, "image.jpg"),
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
		c := swgohgg.NewClient(profile).UseCache(false)
		collection, err := c.Collection()
		if err != nil {
			send(s, m.ChannelID, "Oops, that did not work as expected: %v. I hope nothing is broken ....", err.Error())
			return
		}
		if !collection.Contains(swgohgg.CharName(char)) {
			send(s, m.ChannelID, "It looks like **%s** is not activated, is it %s?", char, m.Author.Mention())
		}
		stats, err := c.CharacterStats(char)
		if err != nil {
			send(s, m.ChannelID, "Oops, that did not work as expected: %v. I hope nothing is broken ....", err.Error())
			return
		}
		char = swgohgg.CharName(char)
		funCharTitle := char
		switch strings.ToLower(swgohgg.CharName(char)) {
		case "finn":
			funCharTitle += " Traitor!!!"
		case "sith assassin":
			funCharTitle = "Darth Nox"
		case "clone sergeant - phase i":
			funCharTitle = "Hevy"
		}
		funComment := " When I grow up I'll have one like this :eyes:"
		if stats.GearLevel < 9 {
			funComment = " But you need some more gear here hun? :unamused:"
		} else if stats.Speed < 150 {
			funComment = " Oh wait, is this a turtle? Give it some speeeeed :rolling_eyes:"
		}
		embedUrl := fmt.Sprintf("https://swgoh.gg/u/%s/collection/%s/", profile, swgohgg.CharSlug(char))
		logger.Infof("Sending embed URL=%v", embedUrl)
		_, err = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: fmt.Sprintf("Wow, nice stats %s!%s", m.Author.Mention(), funComment),
			Embed: &discordgo.MessageEmbed{
				Title: fmt.Sprintf("%s stats for %s", unquote(profile), funCharTitle),
				URL:   embedUrl,
				Fields: []*discordgo.MessageEmbedField{
					{"Power", fmt.Sprintf("%d", stats.GalacticPower), true},
					{"Basic", fmt.Sprintf("%d* G%d Lvl %d", stats.Stars, stats.GearLevel, stats.Level), true},
					{"Health", strconv.Itoa(stats.Health), true},
					{"Protection", strconv.Itoa(stats.Protection), true},
					{"Speed", strconv.Itoa(stats.Speed), true},
					{"Potency", fmt.Sprintf("%.02f%%", stats.Potency), true},
					{"Tenacity", fmt.Sprintf("%.02f%%", stats.Tenacity), true},
					{"Critical Damage", fmt.Sprintf("%.02f%%", stats.CriticalDamage), true},
					{"Physical Damage", fmt.Sprintf("%d", stats.PhysicalDamage), true},
					{"Physical Crit. Chan.", fmt.Sprintf("%.02f%%", stats.PhysicalCritChance), true},
					{"Special Damage", fmt.Sprintf("%d", stats.SpecialDamage), true},
					{"Special Crit. Chan.", fmt.Sprintf("%.02f%%", stats.SpecialCritChance), true},
				},
				Color:  embedColor,
				Footer: copyrightFooter,
			},
		})
		if err != nil {
			logger.Errorf("Unable to send message: %v", err)
		}
	} else if strings.HasPrefix(m.Content, "/arena") {
		args := ParseArgs(m.Content)
		profile, ok := cache.UserProfileIfEmpty(args.Profile, m.Author.ID)
		if len(m.Mentions) > 0 {
			logger.Infof("Using mentioned profile %v", m.Mentions[0])
			profile, ok = cache.UserProfileIfEmpty(args.Profile, m.Mentions[0].ID)
		}
		if !ok {
			askForProfile(s, m, "arena")
			return
		}
		sent, _ := send(s, m.ChannelID, "OK, let me check your profile... :clock130:")
		defer cleanup(s, sent)
		url := fmt.Sprintf("https://swgoh.gg/u/%s/", profile)
		querySelector := ".chart-arena"
		b, err := renderImageAt(logger, url, querySelector, "", "ipad")
		if err != nil {
			logger.Errorf("Unable to render image %v", err)
			send(s, m.ChannelID, "Oh no! I was unable to render the image :O")
			return
		}
		gg := swgohgg.NewClient(profile).UseCache(false)
		team, update, err := gg.Arena()
		if err != nil {
			logger.Errorf("Unable to fetch your arena team: %v", err)
			send(s, m.ChannelID, "Oh no! I was unable to fetch your profile named '%s'. Please make sure the information is correct ", profile)
		}
		embed := &discordgo.MessageEmbed{
			URL: fmt.Sprintf("https://swgoh.gg/u/%s", profile),
			Image: &discordgo.MessageEmbedImage{
				URL: "attachment://image.jpg",
			},
			Title:       fmt.Sprintf("%s current arena team", profile),
			Description: fmt.Sprintf("*Updated at %v*", update.Format(time.Stamp)),
			Color:       embedColor,
			Footer:      copyrightFooter,
		}
		var moreMessage string
		if !args.ContainsFlag("+more") {
			moreMessage = "\nTo see more stats just ask!  Add +more to your command."
		}
		var leaderIndicator string
		var inline bool
		for index, char := range team {
			// Most arena team members get no leader indicator and are inline.
			leaderIndicator = ""
			inline = true

			// But index 0 is the leader.. so setup the indicator and dont go inline.
			if index == 0 {
				leaderIndicator = "Leader - "
				inline = false
			}

			var value string

			// Are they looking for the expanded, "more" display?
			if args.ContainsFlag("+more") {
				value = fmt.Sprintf("Speed: %d\n", char.Speed)
				value += fmt.Sprintf("Health: %d\n", char.Health)
				value += fmt.Sprintf("Protection: %d (%d Total)\n", char.Protection, char.Health+char.Protection)
				value += fmt.Sprintf("Crit Dmg: %.1f%%\n", char.CriticalDamage)
				value += fmt.Sprintf("Crit Chance: %.1f%%\n", char.PhysicalCritChance)
			} else {
				value = fmt.Sprintf("%d *Spd*, %d *HP*, %d *Prot*", char.Speed, char.Health, char.Protection)
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("%s (%s%d* G%d)", char.Name, leaderIndicator, char.Stars, char.GearLevel),
				Value:  value,
				Inline: inline,
			})
		}
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: fmt.Sprintf("So, here is the team you asked for, %v. %s", m.Author.Mention(), moreMessage),
			Embed:   embed,
			Files:   newAttachment(b, "image.jpg"),
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
			askForProfile(s, m, "/faction")
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
		sent, _ := send(s, m.ChannelID, "Checking **%s** units tagged **%s** ... This may take some time :clock130:", unquote(profile), displayName)
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
		b, err := renderImageAt(logger, targetUrl, querySelector, "", "desktop")
		if err != nil {
			logger.Errorf("Error rendering image: %v", err)
			send(s, m.ChannelID, "Oh no! That is not good. Could not render image :-/")
			return
		}
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: "There we go " + m.Author.Mention(),
			Embed: &discordgo.MessageEmbed{
				Title: fmt.Sprintf("%s's characters tagged '%s'", unquote(profile), displayName),
				URL:   targetUrl,
				Image: &discordgo.MessageEmbedImage{
					URL: "attachment://image.jpg",
				},
				Color:  embedColor,
				Footer: copyrightFooter,
			},
			Files: newAttachment(b, "image.jpg"),
		})
	} else if strings.HasPrefix(m.Content, "/lookup") {
		// TODO(ronoaldo): this could be better if we make it so that
		// the bot has a db for each hosted server and we can then
		// query stuff here instead of this api calls.
		args := ParseArgs(m.Content)
		ships := args.ContainsFlag("+ships", "+ship", "+s")

		unit := swgohgg.CharName(args.Name)
		if ships {
			unit = swgohgg.ShipForCrew(unit)
		}
		guildProfiles := cache.ListProfiles()

		minStar := 0
		minGear := 0
		errCount := 0
		resultCount := 0
		loadingCount := 0

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
			} else if strings.HasSuffix(flag, "stars") {
				v, _ := strconv.Atoi(strings.Replace(flag, "stars", "", -1))
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
		msg := fmt.Sprintf("Looking for profiles that have **%s**,", unit)
		if minStar > 0 {
			msg += fmt.Sprintf(" at **%d stars**,", minStar)
		}
		if minGear > 0 {
			msg += fmt.Sprintf(" at **gear level %d**,", minGear)
		}
		msg += " in the whole server. It takes a long while until I get this data."
		msg += " Well, why don't you grab some oil for me? :clock10: :clock10: :clock10:"
		sent, _ := send(s, m.ChannelID, "%s", msg)
		defer cleanup(s, sent)
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
			if profile == nil {
				logger.Infof("*** Loading in background: %v***", err)
				loadingCount++
				continue
			}
			unitStars, unitGear := 0, 0
			if ships {
				s := profile.Ship(unit)
				if s == nil {
					continue
				}
				logger.Infof("> Unit: %v", s)
				unitStars, unitGear = s.Stars, 12
			} else {
				c := profile.Character(unit)
				if c == nil {
					continue
				}
				logger.Infof("> Unit: %v", c)
				unitStars, unitGear = c.Stars, c.Gear
			}
			ok := false
			switch {
			case minStar > 0 && minGear > 0:
				// Both filters provided
				ok = cmp(unitStars, minStar) && cmp(unitGear, minGear)
			case minStar > 0:
				ok = cmp(unitStars, minStar)
			case minGear > 0:
				ok = cmp(unitGear, minGear)
			default:
				ok = unitStars > 0 && unitGear > 0
			}
			if ok {
				logger.Infof("> Player has the unit")
				resultCount++
				lines = append(lines, fmt.Sprintf("**%s**", unquote(user)))
			}
		}
		msg = fmt.Sprintf("%d players have **%s** %v.", resultCount, unit, args.Flags)
		send(s, m.ChannelID, "%s", msg)
		// Outputs at most 100 profiles at a time.
		var buff bytes.Buffer
		count := 0
		sort.Strings(lines)
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
		if errCount > 0 {
			send(s, m.ChannelID, "I was unable to parse %d profiles. :cry:", errCount)
		}
		if loadingCount > 0 {
			send(s, m.ChannelID, "And I'm still analysing %d profiles. Please try again in 10min :grin:. "+
				"If you keep receiving this, some profiles on https://swgoh.gg/ may be outdated and need to do "+
				"a manual sync.", loadingCount)
		}
	} else if strings.HasPrefix(m.Content, "/reload-profiles") {
		args := ParseArgs(m.Content)
		send(s, m.ChannelID, "Copy that. I'll scan the channel #swgoh-gg again...")
		count, invalid, err := cache.ReloadProfiles(s)
		if err != nil {
			logger.Errorf("Error loading profiles: %v", err)
			send(s, m.ChannelID, "Oh no! We're doomed! Unable to read profiles!")
			return
		}
		send(s, m.ChannelID, "Reloaded profiles for the server. I found %d valid links.", count)
		if invalid != "" && args.ContainsFlag("+v", "+verbose") {
			send(s, m.ChannelID, "These are invalid profiles:\n%v", invalid)
		}
	} else if strings.HasPrefix(m.Content, "/server-info") {
		args := ParseArgs(m.Content)
		char := args.Name
		if char == "" {
			send(s, m.ChannelID, "Oh, there we go again. You need to provide me a character name. Try /server-info tfp")
			return
		}
		guildProfiles := cache.ListProfiles()
		sent, err := send(s, m.ChannelID, "Loading %d profiles in the server. This may take a while. "+
			"Take some tea and bring me some oil please. :clock10:", len(guildProfiles))
		defer cleanup(s, sent)
		stars := make(map[int]int)
		gear := make(map[int]int)
		zetaCount := make(map[string]int)

		total := 0
		gg := swgohgg.NewClient("").UseCache(false)
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
func renderImageAt(logger *Logger, targetUrl, querySelector, click, size string) ([]byte, error) {
	renderUrl := fmt.Sprintf("%s/pageRender?url=%s&querySelector=%s&clickSelector=%s&size=%s&ts=%d",
		renderPageHost, esc(targetUrl), querySelector, click, size, time.Now().UnixNano())
	return download(logger, renderUrl)
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

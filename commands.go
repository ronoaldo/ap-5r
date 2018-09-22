package main

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ronoaldo/swgoh/swgohgg"
)

// cmdDisabled returns a message saying the command was disabled.
func cmdDisabled(reason string) CmdHandler {
	return CmdFunc(func(r CmdRequest) (err error) {
		_, err = send(r.s, r.m.ChannelID, "Oh no! I am so sorry but **%s** command was disabled. "+
			"It was not working properly because %s. :cry:", r.args.Command, reason)
		return
	})
}

// cmdMods display mods equiped on a character.
func cmdMods(r CmdRequest) (err error) {
	if !r.profileOk {
		return errProfileRequered
	}
	char := r.args.Name
	if char == "" {
		send(r.s, r.m.ChannelID, "%s, use this command with a character name. Try this: /mods tfp", r.m.Author.Mention())
		return nil
	}
	targetURL := fmt.Sprintf("https://swgoh.gg/u/%s/collection/%s/", r.profile, swgohgg.CharSlug(swgohgg.CharName(char)))
	querySelector := ".list-group.media-list.media-list-stream:nth-child(2)"
	clickSelector := ".icon.icon-chevron-down.pull-left"
	b, err := renderImageAt(r.l, targetURL, querySelector, clickSelector, "desktop")
	if err != nil {
		send(r.s, r.m.ChannelID, "Oh, no! I was unable to create the image :(")
		return err
	}
	_, err = r.s.ChannelMessageSendComplex(r.m.ChannelID, &discordgo.MessageSend{
		Content: "Here is the thing you asked " + r.m.Author.Mention(),
		Embed: &discordgo.MessageEmbed{
			Title: fmt.Sprintf("%s's %s mods", unquote(r.profile), swgohgg.CharName(char)),
			URL:   targetURL,
			Image: &discordgo.MessageEmbedImage{
				URL: "attachment://image.jpg",
			},
			Color:  embedColor,
			Footer: copyrightFooter,
		},
		Files: newAttachment(b, "image.jpg"),
	})
	return err
}

// cmdStats display character statistics.
func cmdStats(r CmdRequest) (err error) {
	if !r.profileOk {
		return errProfileRequered
	}
	char := r.args.Name
	if char == "" {
		send(r.s, r.m.ChannelID, "Good, you are learning! But you need to provide a character name. Try /info tfp")
		return nil
	}
	c := swgohgg.NewClient(r.profile)
	gg := swgohgg.NewClient(r.profile)
	if *swggUser != "" && *swggPass != "" {
		if err := gg.Login(*swggUser, *swggPass); err != nil {
			r.l.Errorf("Error logging into website: %v:", err)
		}
	} else {
		r.l.Infof("Using non-auth client")
	}

	collection, err := c.Collection()
	if err != nil {
		send(r.s, r.m.ChannelID, "Oops, that did not work as expected: %v. I hope nothing is broken ....", err.Error())
		return
	}
	if !collection.Contains(swgohgg.CharName(char)) {
		send(r.s, r.m.ChannelID, "It looks like **%s** is not activated, is it %s?", char, r.m.Author.Mention())
		return
	}
	stats, err := c.CharacterStats(char)
	if err != nil {
		send(r.s, r.m.ChannelID, "Oops, that did not work as expected: %v. I hope nothing is broken ....", err.Error())
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
	embedURL := fmt.Sprintf("https://swgoh.gg/u/%s/collection/%s/", r.profile, swgohgg.CharSlug(char))
	logger.Infof("Sending embed URL=%v", embedURL)
	_, err = r.s.ChannelMessageSendComplex(r.m.ChannelID, &discordgo.MessageSend{
		Content: fmt.Sprintf("Wow, nice stats %s!%s", r.m.Author.Mention(), funComment),
		Embed: &discordgo.MessageEmbed{
			Title: fmt.Sprintf("%s stats for %s", unquote(r.profile), funCharTitle),
			URL:   embedURL,
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
	return err
}

// cmdArena display your arena team, statistics and chart.
func cmdArena(r CmdRequest) (err error) {
	if !r.profileOk {
		return errProfileRequered
	}
	url := fmt.Sprintf("https://swgoh.gg/u/%s/", r.profile)
	querySelector := ".chart-arena"
	b, err := renderImageAt(logger, url, querySelector, "", "ipad")
	if err != nil {
		logger.Errorf("Unable to render image %v", err)
		send(r.s, r.m.ChannelID, "Oh no! I was unable to render the image :O")
		return
	}
	gg := swgohgg.NewClient(r.profile)
	if *swggUser != "" && *swggPass != "" {
		if err := gg.Login(*swggUser, *swggPass); err != nil {
			r.l.Errorf("Error logging into website: %v:", err)
		}
	} else {
		r.l.Infof("Using non-auth client")
	}
	team, update, err := gg.Arena()
	if err != nil {
		logger.Errorf("Unable to fetch your arena team: %v", err)
		send(r.s, r.m.ChannelID, "Oh no! I was unable to fetch your profile named '%s'. Please make sure the information is correct ", r.profile)
	}
	embed := &discordgo.MessageEmbed{
		URL: fmt.Sprintf("https://swgoh.gg/u/%s", r.profile),
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://image.jpg",
		},
		Title:       fmt.Sprintf("%s current arena team", r.profile),
		Description: fmt.Sprintf("*Updated at %v*", update.Format(time.Stamp)),
		Color:       embedColor,
		Footer:      copyrightFooter,
	}
	var moreMessage string
	if !r.args.ContainsFlag("+more") {
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
		if r.args.ContainsFlag("+more") {
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
	_, err = r.s.ChannelMessageSendComplex(r.m.ChannelID, &discordgo.MessageSend{
		Content: fmt.Sprintf("So, here is the team you asked for, %v. %s", r.m.Author.Mention(), moreMessage),
		Embed:   embed,
		Files:   newAttachment(b, "image.jpg"),
	})
	return err
}

// cmdFaction display a faction of a player collection.
func cmdFaction(r CmdRequest) (err error) {
	if !r.profileOk {
		return errProfileRequered
	}
	filter := strings.ToLower(strings.TrimSpace(r.args.Name))
	if filter == "" {
		send(r.s, r.m.ChannelID, "Please provide a faction! Try /faction Empire")
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
	sent, _ := send(r.s, r.m.ChannelID, "Checking **%s** units tagged **%s** ... This may take some time :clock130:", unquote(r.profile), displayName)
	defer cleanup(r.s, sent)
	filter = strings.Replace(strings.ToLower(filter), " ", "-", -1)
	if filter == "rebel-scum" || filter == "terrorists" {
		filter = "rebel"
	}
	if filter == "dark-side" || filter == "light-side" {
		filter = strings.Replace(filter, "-", "%20", -1)
	}
	targetURL := fmt.Sprintf("https://swgoh.gg/u/%s/collection/?f=%s", r.profile, filter)
	querySelector := ".collection-char-list"
	if r.args.ContainsFlag("+ships", "+ship", "+s") {
		targetURL = fmt.Sprintf("https://swgoh.gg/u/%s/ships/?f=%s", r.profile, filter)
	}
	b, err := renderImageAt(logger, targetURL, querySelector, "", "desktop")
	if err != nil {
		logger.Errorf("Error rendering image: %v", err)
		send(r.s, r.m.ChannelID, "Oh no! That is not good. Could not render image :-/")
		return
	}
	_, err = r.s.ChannelMessageSendComplex(r.m.ChannelID, &discordgo.MessageSend{
		Content: "There we go " + r.m.Author.Mention(),
		Embed: &discordgo.MessageEmbed{
			Title: fmt.Sprintf("%s's characters tagged '%s'", unquote(r.profile), displayName),
			URL:   targetURL,
			Image: &discordgo.MessageEmbedImage{
				URL: "attachment://image.jpg",
			},
			Color:  embedColor,
			Footer: copyrightFooter,
		},
		Files: newAttachment(b, "image.jpg"),
	})
	return err
}

// cmdServerInfo performs server-wide statistics
func cmdServerInfo(r CmdRequest) (err error) {
	char := r.args.Name
	if char == "" {
		send(r.s, r.m.ChannelID, "Oh, there we go again. You need to provide me a character name. Try /server-info tfp")
		return
	}
	guildProfiles := r.cache.ListProfiles()
	sent, err := send(r.s, r.m.ChannelID, "Loading %d profiles in the server. This may take a while. "+
		"Take some tea and bring me some oil please. :clock10:", len(guildProfiles))
	defer cleanup(r.s, sent)
	stars := make(map[int]int)
	gear := make(map[int]int)
	zetaCount := make(map[string]int)

	total := 0
	gg := swgohgg.NewClient("")
	allZetas, err := gg.Zetas()
	if err != nil {
		send(r.s, r.m.ChannelID, "Warning: I'll be skipping zetas as I could not load them. Something is wrong probably. (err=%v)", err)
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
	fmt.Fprintf(&msg, "From %d %s players, %d have %s\n", len(guildProfiles), r.guild.Name, total, swgohgg.CharName(char))
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
	_, err = send(r.s, r.m.ChannelID, msg.String())
	return err
}

// cmdLookup performs server-wide character lookup.
// Usefull for platoon assignments.
func cmdLookup(r CmdRequest) (err error) {
	// TODO(ronoaldo): this could be better if we make it so that
	// the bot has a db for each hosted server and we can then
	// query stuff here instead of this api calls.
	ships := r.args.ContainsFlag("+ships", "+ship", "+s")

	unit := swgohgg.CharName(r.args.Name)
	if ships {
		unit = swgohgg.ShipForCrew(unit)
	}
	guildProfiles := r.cache.ListProfiles()

	minStar := 0
	minGear := 0
	errCount := 0
	resultCount := 0
	loadingCount := 0

	cmp := func(a, b int) bool {
		return a >= b
	}
	for _, flag := range r.args.Flags {
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
	msg += " Well, why don't you grab some oil for me?"
	sent, _ := send(r.s, r.m.ChannelID, "%s", msg)
	defer cleanup(r.s, sent)
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
	msg = fmt.Sprintf("%d players have **%s** %v.", resultCount, unit, r.args.Flags)
	send(r.s, r.m.ChannelID, "%s", msg)
	// Outputs at most 100 profiles at a time.
	var buff bytes.Buffer
	count := 0
	sort.Strings(lines)
	for i := range lines {
		buff.WriteString(lines[i] + "\n")
		count++
		if count > 100 {
			send(r.s, r.m.ChannelID, "%s", buff.String())
			count = 0
			buff.Reset()
		}
	}
	if count > 0 {
		send(r.s, r.m.ChannelID, "%s", buff.String())
	}
	if errCount > 0 {
		send(r.s, r.m.ChannelID, "I was unable to parse %d profiles. :cry:", errCount)
	}
	if loadingCount > 0 {
		send(r.s, r.m.ChannelID, "And I'm still analysing %d profiles. Please try again in 10min :grin:. "+
			"If you keep receiving this, some profiles on *swgoh.gg* may be outdated and need to do "+
			"a manual sync.", loadingCount)
	}
	return nil
}

// cmdReloadProfiles read all profiles from the metadata channel.
func cmdReloadProfiles(r CmdRequest) (err error) {
	count, invalid, err := r.cache.ReloadProfiles(r.s)
	if err != nil {
		logger.Errorf("Error parsing profiles: %v", err)
		send(r.s, r.m.ChannelID, "Oh no! We're doomed! Unable to read profiles!")
		return
	}
	send(r.s, r.m.ChannelID, "Parsed profiles for the server. I found %d valid links.", count)
	if invalid != "" && r.args.ContainsFlag("+v", "+verbose") {
		send(r.s, r.m.ChannelID, "These are invalid profiles:\n%v", invalid)
	}
	return nil
}

// cmdshareThisBot displays information on how to share the bot.
func cmdShareThisBot(r CmdRequest) (err error) {
	msg := "AP-5R protocol droid is able to join other servers, but you need to follow this instructions:\n" +
		"> Join the Bot Users Playground at https://discord.gg/4GJ8Ty2\n" +
		"> Be a nice person\n" +
		"> Follow instructions in the #info channel on that server\n"
	send(r.s, r.m.ChannelID, msg)
	return nil
}

// cmdBotStats returns statistics about bot Guilds.
// Not very useful, should be replaced once we use a database.
func cmdBotStats(r CmdRequest) (err error) {
	quant := listMyGuilds(r.s)
	_, err = send(r.s, r.m.ChannelID, "Running on **%d** guilds", quant)
	return err
}

// cmdLeaveGuild is an admin command to allow the bot to leave a guild.
// Disabled by default, activate for development pourposes or maintanance.
func cmdLeaveGuild(r CmdRequest) (err error) {
	if err = r.s.GuildLeave(r.args.Name); err != nil {
		send(r.s, r.m.ChannelID, "Error leaving guild %s", r.args.Name)
		return err
	}
	_, err = send(r.s, r.m.ChannelID, "Left guild.")
	listMyGuilds(r.s)
	return err
}

// cmdHelp displays the help message.
func cmdHelp(req CmdRequest) (err error) {
	m := "Hi **%s**, I'm AP-5R and I'm the Empire protocol droid unit that survived the Death Star destruction."
	m += " While I understand many languages, please use the following commands to contact me in this secure channel:\n\n"

	m += "**/arena**: display your current arena basic stats. Use +more to get more stats.\n"
	m += "**/stats** *character*: display character basic stats.\n"
	m += "**/mods** *character*: display the mods you have on a character.\n"
	m += "**/faction**: display an image of your characters in the given faction." +
		" *Add +ships, +ship or +s to get ship info.*\n\n"

	m += "**/server-info** *character*: if you want me to do some number crunch and display server-wide stats about a character." +
		" *Add +ships, +ship or +s to get ship info.*\n"
	m += "**/lookup** *character*: to search and see who has a specific character." +
		" *Add +1star .. +7star to filter by star level, and +g1 .. +g12 to filter by gear level.*" +
		" *Add +ships, +ship or +s to get ship info.*\n\n"

	m += "**/share-this-bot**: if you want my help in a galaxy far, far away...\n\n"

	m += "I'll assume that all users shared their profile at the #swgoh-gg channel." +
		" Please ask your server admin to create one." +
		" This is important for me to properly function here, as I'll link the message author with the profile." +
		" You can also share a profile on behalf of a shard-mate by @mentioning that player after the link." +
		" Alternatively, you can use [profile] syntax at the end of /mods, /stats, /faction and /arena" +
		" in order to get info from another profile than yours."
	_, err = send(req.s, req.m.ChannelID, m, req.m.Author.Username)
	return
}

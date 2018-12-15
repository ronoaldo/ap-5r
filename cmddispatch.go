package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ronoaldo/swgoh/swgohgg"
)

var (
	errProfileRequered = errors.New("ap-5r: profile required for this command")
	allyCodeRe         = regexp.MustCompile("^[0-9]{3,3}-?[0-9]{3,3}-?[0-9]{3,3}$")
)

// CmdRequest holds parsed data from the context of a MessageCreate event.
type CmdRequest struct {
	s       *discordgo.Session
	m       *discordgo.MessageCreate
	l       *Logger
	guild   *discordgo.Guild
	channel *discordgo.Channel
	cache   *Cache
	args    *Args
	// profile   string
	// profileOk bool
	allyCode   string
	allyCodeOk bool
}

// CmdHandler defines a handler to handle commands from user
type CmdHandler interface {
	HandleCommand(c CmdRequest) error
}

// CmdFunc is a type adapter to allow the use of functions as a CmdHandler.
type CmdFunc func(c CmdRequest) error

// HandleCommand calls f(c).
func (f CmdFunc) HandleCommand(c CmdRequest) error {
	return f(c)
}

// CmdDispatcher parses a MessageCreate and dispatches the request to the target command.
type CmdDispatcher struct {
	prefix string
	cmds   map[string]CmdHandler
}

// NewDispatcher creates a new command dispatcher.
func NewDispatcher() *CmdDispatcher {
	return &CmdDispatcher{
		cmds: make(map[string]CmdHandler),
	}
}

// Handle maps a command name to a command handler
func (d *CmdDispatcher) Handle(cmd string, handler CmdHandler) {
	d.cmds[cmd] = handler
}

// Dispatch parses the message and if a command is found, forwards the command to the handler.
// If no handler is mapped, returns an error. If no command is detected, discards the event.
func (d *CmdDispatcher) Dispatch(s *discordgo.Session, m *discordgo.MessageCreate) error {
	// Skip messages from self or non-command messages
	if m.Author.ID == s.State.User.ID {
		return nil
	}

	// Load data from cache to prepare for command parsing.
	channel, err := apiCache.GetChannel(s, m.ChannelID)
	if err != nil && strings.HasPrefix(m.Content, *cmdPrefix) {
		logger.Errorf("Error loading channel: %v", err)
		send(s, m.ChannelID, "Oh, no. This should not happen. Unable to identify channel for this message!")
		return nil
	}
	if channel == nil {
		logger.Errorf("Unexpected error loading channel for message %v: %v (channel=%v)", m, err, channel)
		return nil
	}

	guild, err := apiCache.GetGuild(s, channel.GuildID)
	if err != nil && strings.HasPrefix(m.Content, *cmdPrefix) {
		logger.Errorf("Error loading channel: %v", err)
		send(s, m.ChannelID, "Oh, no. This should not happen. Unable to identify server for this message!")
		return nil
	}
	if guild == nil {
		logger.Errorf("Unexpected error loading guild for message %v: %v (guild=%v)", m, err, guild)
		return nil
	}
	logger := &Logger{Guild: guild.Name}
	cache, ok := guildCache[channel.GuildID]
	if !ok {
		logger.Printf("No cache for guild ID %s, initializing one", channel.GuildID)
		// Initialize new cache and build guild profile cache
		cache = NewCache(channel.GuildID, guild.Name)
		cache.ReloadProfiles(s)
		guildCache[channel.GuildID] = cache
	}
	// If message is from swgoh-gg, reload profiles.
	if channel.Name == "swgoh-gg" {
		cache.ReloadProfiles(s)
		if strings.HasPrefix(m.Content, *cmdPrefix) {

			send(s, m.ChannelID, "Sorry, let's keep this channel for profile links only!")
		}
		return nil
	}
	// Discard non-commands
	if !strings.HasPrefix(m.Content, *cmdPrefix) {
		return nil
	}

	// Log RECV and react to command
	logger.Printf("RECV: (#%v) %v: %v", channel.Name, m.Author, m.Content)
	s.MessageReactionAdd(m.ChannelID, m.ID, emojiHourGlassNotDone)

	// Build the CmdRequest
	args := ParseArgs(m.Content)

	var allyCode string
	var allyCodeOk bool
	// If we got an ally code, use it
	if args.Profile != "" {
		// User passed explicitly. Check if it is ally code or not
		if allyCodeRe.MatchString(args.Profile) {
			// Use the provided ally code
			allyCode = args.Profile
			allyCodeOk = true
		} else {
			// Lookup the ally code from the profile
			allyCode = swgohgg.NewClient(args.Profile).AllyCode()
			allyCodeOk = allyCode != ""
		}
	} else {
		// User passed implicitly. Check if we had discovered ally code yet
		discordUserID := m.Author.ID
		if len(m.Mentions) > 0 {
			discordUserID = m.Mentions[0].ID
		}
		allyCode, allyCodeOk = cache.AllyCode(discordUserID)
	}

	req := CmdRequest{
		s:          s,
		m:          m,
		l:          logger,
		guild:      guild,
		channel:    channel,
		cache:      cache,
		args:       args,
		allyCode:   allyCode,
		allyCodeOk: allyCodeOk,
	}

	// Call the CmdHandler
	h, ok := d.cmds[args.Command]
	if !ok {
		s.MessageReactionAdd(m.ChannelID, m.ID, emojiQuestionMark)
		return fmt.Errorf("dispatcher: no command mapped to %v", args.Command)
	}

	// Hadle command and react with result
	logger.Infof("Dispatching command %#v", req)
	err = h.HandleCommand(req)
	result := emojiCheckMark
	if err == errProfileRequered {
		s.MessageReactionAdd(m.ChannelID, m.ID, emojiFacePalm)
		askForProfile(s, m, args.Command)
		return nil
	} else if err != nil {
		result = emojiCrossMark
	}
	s.MessageReactionAdd(m.ChannelID, m.ID, result)
	return err
}

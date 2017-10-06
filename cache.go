package main

import (
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Cache holds guild-based cache information.
type Cache struct {
	guildID       string
	profiles      map[string]string
	profilesMutex sync.Mutex
	logger        *Logger
}

// NewCache creates a new cache for the given guild ID.
func NewCache(guildID, guildName string) *Cache {
	return &Cache{
		guildID:  guildID,
		profiles: make(map[string]string),
		logger:   &Logger{Guild: guildName},
	}
}

// SetUserProfile stores the user profile in cache
func (c *Cache) SetUserProfile(user, profile string) {
	c.profilesMutex.Lock()
	defer c.profilesMutex.Unlock()
	c.profiles[user] = profile
	// prefetch(c.logger, fmt.Sprintf("https://swgoh-api.appspot.com/v1/profile/%s", profile))
}

func (c *Cache) UserProfileIfEmpty(profile, user string) (string, bool) {
	if profile != "" {
		return profile, true
	}
	return c.UserProfile(user)
}

func (c *Cache) UserProfile(user string) (string, bool) {
	profile, ok := c.profiles[user]
	return profile, ok
}

func (c *Cache) ListProfiles() (res []string) {
	for _, v := range c.profiles {
		res = append(res, v)
	}
	return res
}

func (c *Cache) RemoveAllProfiles() {
	// Cleanup all profiles of the given guild
	c.profilesMutex.Lock()
	defer c.profilesMutex.Unlock()
	c.profiles = make(map[string]string)
}

func (c *Cache) ReloadProfiles(s *discordgo.Session) (int, error) {
	guild, err := s.Guild(c.guildID)
	if err != nil {
		return 0, err
	}
	c.RemoveAllProfiles()
	c.logger.Printf("> Reloading profiles for guild %s#%s", guild.Name, guild.ID)

	c.logger.Printf("> Looking up #swgoh-gg channel for guild %s", guild.Name)
	channels, err := s.GuildChannels(guild.ID)
	if err != nil {
		c.logger.Errorf("Loading channels. Skipping this guild (%v)", err)
		return 0, err
	}
	chanID := ""
	for _, ch := range channels {
		if ch.Name == "swgoh-gg" {
			chanID = ch.ID
			break
		}
	}
	if chanID == "" {
		c.logger.Errorf("No channel ID found with name #swgoh-gg. Skipping this guild.")
		return 0, err
	}
	pageSize := 100
	first := ""
	last := ""
	for {
		c.logger.Printf("Loading messages, last  '%s'", last)
		messages, err := s.ChannelMessages(chanID, pageSize, last, "", "")
		if err != nil {
			send(s, chanID, "Oh, wait. I could not read messages on %v's #swgoh-gg channel."+
				" Try to remove me and add me again to the server, and make sure I have all"+
				" requested permissions! If your #swgoh-gg channel is restricted by a tag/role,"+
				" I need that tag/role too.", guild.Name)
			c.logger.Errorf("Loading messages from #swgoh-gg channel: %v", err)
			return 0, err
		}
		c.logger.Printf("> Currently with %d", len(messages))
		for _, m := range messages {
			if first == "" {
				first = m.ID + ": " + m.Content
			}
			last = m.ID
			profile := extractProfile(m.Content)
			if profile == "" {
				continue
			}
			// Let's try to fix some weird names, right?
			aux, err := url.QueryUnescape(profile)
			if err == nil {
				// We could decode, so let's encode again in a better way.
				profile = strings.Replace(url.QueryEscape(aux), "+", "%20", -1)
			}
			c.SetUserProfile(m.Author.ID, profile)
			c.logger.Printf("> Detected %v[%v]: %v", m.Author, m.Author.ID, profile)
		}
		if len(messages) < pageSize {
			break
		}
		c.logger.Printf("> Waiting a bit to avoid doing a server overload...")
		time.Sleep(1 * time.Second)
	}
	c.logger.Printf("Full profile list loaded %d", len(c.profiles))
	return len(c.profiles), nil
}

// profileRe is a regular expressions that allows one to extract the
// user profile nickname on the SWGoH.GG website.
var profileRe = regexp.MustCompile("https?://swgoh.gg/u/([^/]+)/?")

// extractProfile returns the user profile from the provided
// link text. Profile is extracted using profileRe.
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

// APICache holds some in-memory cached data.
type APICache struct {
	channels   map[string]*discordgo.Channel
	channelsMu sync.Mutex

	guilds   map[string]*discordgo.Guild
	guildsMu sync.Mutex
}

// NewAPICache creates a new API Cache in-memory.
func NewAPICache() *APICache {
	return &APICache{
		channels: make(map[string]*discordgo.Channel),
		guilds:   make(map[string]*discordgo.Guild),
	}
}

func (a *APICache) GetGuild(s *discordgo.Session, guildID string) (*discordgo.Guild, error) {
	a.guildsMu.Lock()
	defer a.guildsMu.Unlock()
	if g, ok := a.guilds[guildID]; ok {
		return g, nil
	}
	g, err := s.Guild(guildID)
	if g != nil {
		a.guilds[guildID] = g
	}
	return g, err
}

func (a *APICache) GetChannel(s *discordgo.Session, channelID string) (*discordgo.Channel, error) {
	a.channelsMu.Lock()
	defer a.channelsMu.Unlock()
	if c, ok := a.channels[channelID]; ok {
		return c, nil
	}
	c, err := s.Channel(channelID)
	if c == nil {
		a.channels[channelID] = c
	}
	return c, err
}

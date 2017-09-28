package main

import (
	"regexp"
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
	guilds, err := s.UserGuilds(0, "", "")
	if err != nil {
		return 0, err
	}
	c.RemoveAllProfiles()

	for _, guild := range guilds {
		if guild.ID != c.guildID {
			continue
		}
		c.logger.Printf("> Looking up #swgoh-gg channel for guild %s", guild.Name)
		channels, err := s.GuildChannels(guild.ID)
		if err != nil {
			c.logger.Errorf("Loading channels. Skipping this guild (%v)", err)
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
			c.logger.Errorf("No channel ID found with name #swgoh-gg. Skipping this guild.")
			continue
		}
		after := ""
		for {
			c.logger.Printf("Loading messages after id '%s'", after)
			messages, err := s.ChannelMessages(chanID, 100, "", after, "")
			if err != nil {
				send(s, chanID, "Oh, wait. I could not read messages on %v's #swgoh-gg channel."+
					" Try to remove me and add me again to the server, and make sure I have all"+
					" requested permissions! If your #swgoh-gg channel is restricted by a tag/role,"+
					" I need that tag/role too.", guild.Name)
				c.logger.Errorf("Loading messages from #swgoh-gg channel: %v", err)
				continue
			}
			c.logger.Printf("> Currently with %d", len(messages))
			for _, m := range messages {
				after = m.ID
				profile := extractProfile(m.Content)
				if profile == "" {
					continue
				}
				c.SetUserProfile(m.Author.String(), profile)
				c.logger.Printf("> Detected %v: %v", m.Author, profile)
			}
			if len(messages) <= 100 {
				break
			}
			c.logger.Printf("> Waiting a bit to avoid doing a server overload...")
			time.Sleep(10 * time.Second)
		}
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

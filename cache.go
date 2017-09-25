package main

import (
	"log"
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
}

// NewCache creates a new cache for the given guild ID.
func NewCache(guildID string) *Cache {
	return &Cache{
		guildID:  guildID,
		profiles: make(map[string]string),
	}
}

// SetUserProfile stores the user profile in cache
func (c *Cache) SetUserProfile(user, profile string) {
	c.profilesMutex.Lock()
	defer c.pprofilesMutex.Unlock()
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

func (c *Cache) ReloadProfiles(s *discordgo.Session) (int, error) {
	guilds, err := s.UserGuilds(0, "", "")
	if err != nil {
		return 0, err
	}
	// Cleanup all profiles of the given guild
	c.profilesMutex.Lock()
	defer c.pprofilesMutex.Unlock()
	c.profiles = make(map[string]string)

	for _, guild := range guilds {
		if guild.ID != c.guildID {
			continue
		}
		log.Printf("> Looking up #swgoh-gg channel for guild %s", guild.Name)
		channels, err := s.GuildChannels(guild.ID)
		if err != nil {
			log.Printf("ERROR: loading channels. Skipping this guild (%v)", err)
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
			log.Printf("ERROR: No channel ID found with name #swgoh-gg. Skipping this guild.")
			continue
		}
		after := ""
		for {
			log.Printf("ERROR: Loading messages after id '%s'", after)
			messages, err := s.ChannelMessages(chanID, 100, "", after, "")
			if err != nil {
				send(s, chanID, "Oh, wait. I could not read messages on %v#%v", guild.Name, ch.Name)
				log.Printf("ERROR: loading messages from #swgoh-gg channel: %v", err)
				continue
			}
			log.Printf("> Currently with %d", len(messages))
			for _, m := range messages {
				after = m.ID
				profile := extractProfile(m.Content)
				if profile == "" {
					continue
				}
				c.SetUserProfile(m.Author.String(), profile)
				log.Printf("> Detected %v: %v", m.Author, profile)
			}
			if len(messages) <= 100 {
				break
			}
			log.Printf("> Waiting a bit to avoid doing a server overload...")
			time.Sleep(10 * time.Second)
		}
	}
	log.Printf("INFO: Full profile list loaded %d", len(c.profiles))
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

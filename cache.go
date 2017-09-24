package main

import (
	"log"
	"regexp"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Cache holds guild-based cache information.
type Cache struct {
	guildID  string
	profiles map[string]string
}

func NewCache(guildID string) *Cache {
	return &Cache{
		guildID:  guildID,
		profiles: make(map[string]string),
	}
}

func (c *Cache) SetUserProfile(user, profile string) {
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
	c.profiles = make(map[string]string)

	for _, guild := range guilds {
		if guild.ID != c.guildID {
			continue
		}
		log.Printf("> Looking up #swgoh-gg channel for guild %s", guild.Name)
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
				c.SetUserProfile(m.Author.String(), profile)
				log.Printf("Detected %v: %v", m.Author, profile)
			}
			if len(messages) <= 100 {
				break
			}
			log.Printf("Waiting a bit to avoid doing shit ...")
			time.Sleep(10 * time.Second)
		}
	}
	log.Printf("Full profile list loaded %d", len(c.profiles))
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

// loadProfiles load and fill all profiles from the #swgoh-gg channel.
// Currently, the profiles are not fully loaded from the swgoh.gg site,
// they are just cached and linked to usernames so commands don't need
// to explicitly specify a nickname.
func loadProfiles(s *discordgo.Session, guildID string) {
}

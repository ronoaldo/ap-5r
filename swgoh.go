package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ronoaldo.gopkg.net/swgoh/swgohgg"
)

// Profile is an entity that saves user data from the website
type Profile struct {
	LastUpdate time.Time
	Collection swgohgg.Collection
	Ships      swgohgg.Ships
	Arena      []*swgohgg.CharacterStats
	Stats      []*swgohgg.CharacterStats
}

// Character looks for the character name in the profile collection.
// Returns nil if the player does not have the character.
func (p *Profile) Character(char string) *swgohgg.Char {
	for i := range p.Collection {
		c := p.Collection[i]
		if strings.ToLower(c.Name) == strings.ToLower(char) {
			return c
		}
	}
	return nil
}

// CharacterStats looks for the character name in the player collection.
// Returns nil if the player does not have the character.
func (p *Profile) CharacterStats(char string) *swgohgg.CharacterStats {
	for i := range p.Stats {
		stat := p.Stats[i]
		if strings.ToLower(stat.Name) == strings.ToLower(char) {
			return stat
		}
	}
	return nil
}

// Ship returns the ship for the provided name, if the player has it.
func (p *Profile) Ship(ship string) *swgohgg.Ship {
	for i := range p.Ships {
		s := p.Ships[i]
		if strings.ToLower(ship) == strings.ToLower(s.Name) {
			return s
		}
	}
	return nil
}

// GetProfile returns the profile for the player from a cached API.
func GetProfile(user string) (*Profile, error) {
	resp, err := http.Get(fmt.Sprintf("https://swgoh-api.appspot.com/v1/profile/%s", user))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("Error loading profile: %v %v", resp.Status, resp.StatusCode)
	}
	profile := &Profile{}
	err = json.NewDecoder(resp.Body).Decode(profile)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

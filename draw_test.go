package main

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ronoaldo/swgoh/swgohhelp"
)

var testPlayer = swgohhelp.Player{
	Name: "Ronoaldo",
	Titles: swgohhelp.PlayerTitle{
		Selected: "Dark Side Historian",
	},
}

func TestDrawCharacterStats(t *testing.T) {
	units := []swgohhelp.Unit{
		{
			Name:   "Emperor Palpatine",
			Rarity: 7, Gear: 12, Level: 85,
			Stats: &swgohhelp.UnitStats{
				Final: swgohhelp.UnitStatItems{
					Health:                 27748,
					Protection:             27887,
					Speed:                  230,
					CriticalDamage:         1.5,
					Potency:                0.728,
					Tenacity:               0.448,
					PhysicalDamage:         2903,
					SpecialDamage:          6434,
					PhysicalCriticalChance: 0.2308,
					SpecialCriticalChance:  0.2858,
					Armor:                     0.2066,
					Resistance:                0.2817,
					PhysicalCriticalAvoidance: 0,
					SpecialCriticalAvoidance:  0,
				},
				FromMods: swgohhelp.UnitStatItems{
					Health:                 2749,
					Protection:             2749,
					Speed:                  108,
					Potency:                0.258,
					Tenacity:               0.1048,
					PhysicalDamage:         540,
					SpecialDamage:          1134,
					PhysicalCriticalChance: 0.0312,
					SpecialCriticalChance:  0.0312,
					Armor:      0.0244,
					Resistance: 0.0255,
				},
			},
			Skills: []swgohhelp.UnitSkill{
				{IsZeta: true, Tier: 8, Name: "Emperor of the Galactic Empire"},
				{IsZeta: true, Tier: 8, Name: "Crackling Doom"},
				{IsZeta: false, Tier: 8, Name: "Let the hate flow"},
			},
		},
	}

	d := drawer{player: testPlayer}
	for _, unit := range units {
		if b, err := d.DrawCharacterStats(&unit); err != nil {
			t.Fatalf("Unexpected error %v", err)
		} else {
			ioutil.WriteFile(fmt.Sprintf("/tmp/assets/%s.png", unit.Name), b, 0644)
		}
	}
}

func TestDrawUnitList(t *testing.T) {
	teams := map[string][]swgohhelp.Unit{
		"arena": []swgohhelp.Unit{
			{
				Name:   "Gar Saxon",
				Rarity: 7, Gear: 12, Level: 85,
			},
			{
				Name:   "Stormtrooper",
				Rarity: 7, Gear: 12, Level: 85,
			},
			{
				Name:   "General Veers",
				Rarity: 7, Gear: 11, Level: 85,
			},
			{
				Name:   "Colonel Starck",
				Rarity: 7, Gear: 11, Level: 85,
			},
			{
				Name:   "Range Trooper",
				Rarity: 7, Gear: 11, Level: 85,
			},
		},
	}

	for team, units := range teams {
		d := drawer{player: testPlayer}
		b, err := d.DrawUnitList(units)
		if err != nil {
			t.Fatalf("Unexpected error drawing units: %v", err)
		} else {
			ioutil.WriteFile(fmt.Sprintf("/tmp/assets/team-%s.png", team), b, 0644)
		}
	}
}

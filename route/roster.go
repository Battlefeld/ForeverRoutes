package route

import (
	"sort"
	"strings"
)

// SupportedCharacters is the shipped Forever roster. Skyborne factions are separate contexts.
// Return a fresh slice so callers cannot change the roster used by audits and release checks.
func SupportedCharacters() []Character {
	roster := map[string][]string{
		"Human":     {"Hunter", "Mage", "Paladin", "Priest", "Rogue", "Warlock", "Warrior"},
		"Orc":       {"Hunter", "Mage", "Rogue", "Shaman", "Warlock", "Warrior"},
		"Dwarf":     {"Hunter", "Paladin", "Priest", "Rogue", "Shaman", "Warrior"},
		"NightElf":  {"Druid", "Hunter", "Priest", "Rogue", "Warrior"},
		"Undead":    {"Mage", "Paladin", "Priest", "Rogue", "Warlock", "Warrior"},
		"Tauren":    {"Druid", "Hunter", "Shaman", "Warrior"},
		"Gnome":     {"Mage", "Priest", "Rogue", "Warlock", "Warrior"},
		"Troll":     {"Hunter", "Mage", "Priest", "Rogue", "Shaman", "Warlock", "Warrior"},
		"SkyborneA": {"Druid", "Hunter", "Mage", "Rogue", "Warrior"},
		"SkyborneH": {"Druid", "Hunter", "Rogue", "Shaman", "Warrior"},
	}
	var out []Character
	for race, classes := range roster {
		for _, class := range classes {
			c := Character{Race: race, Class: class}
			if strings.HasPrefix(race, "Skyborne") {
				c.Race = "Skyborne"
				c.Faction = strings.TrimPrefix(race, "Skyborne")
			}
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RouteKey() < out[j].RouteKey() })
	return out
}

// RouteKey matches shipped file names, including the faction for Skyborne.
func (c Character) RouteKey() string {
	race := c.Race
	if race == "Scourge" {
		race = "Undead"
	}
	if race == "Skyborne" {
		side, _, _ := c.side()
		race += "-" + side
	}
	return strings.ToLower(race + "-" + c.Class)
}

func SameCharacter(a, b Character) bool {
	af, ar, aok := a.side()
	bf, br, bok := b.side()
	return aok && bok && af == bf && ar == br && a.Class == b.Class
}

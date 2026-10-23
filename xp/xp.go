// Package xp holds the experience rules of the Classic client WoW Forever is built on: the XP to reach the next level,
// the reduced quest XP above the quest's level, and the XP of a kill (cmangos-classic formula, solo, no rested bonus).
package xp

import "math"

// ClassicLevelXP[i] is the XP from level i+1 to i+2 (levels 1..59).
var ClassicLevelXP = []int{400, 900, 1400, 2100, 2800, 3600, 4500, 5400, 6500, 7600, 8800, 10100, 11400, 12900, 14400,
	16000, 17700, 19400, 21300, 23200, 25200, 27300, 29400, 31700, 34000, 36400, 38900, 41400, 44300, 47400, 50800, 54500,
	58600, 62800, 67100, 71600, 76100, 80800, 85700, 90700, 95800, 101000, 106300, 111800, 117500, 123200, 129100, 135100,
	141200, 147500, 153900, 160400, 167100, 173900, 180800, 187900, 195000, 202300, 209800}

// QuestFactor is the share of a quest's XP a player of playerLevel gets for a quest of questLevel.
func QuestFactor(playerLevel, questLevel int) float64 {
	if questLevel <= 0 {
		return 1
	}
	switch d := playerLevel - questLevel; {
	case d <= 5:
		return 1
	case d == 6:
		return 0.8
	case d == 7:
		return 0.6
	case d == 8:
		return 0.4
	case d == 9:
		return 0.2
	default:
		return 0.1
	}
}

// Quest is the XP of a quest worth fullXP for a player of playerLevel.
func Quest(fullXP, playerLevel, questLevel int) int {
	f := QuestFactor(playerLevel, questLevel)
	if f >= 1 {
		return fullXP
	}
	return int(math.Ceil(float64(fullXP) * f))
}

// QuestFromMoney is the full XP of a quest from the money it pays at the level cap instead (Classic: XP = money / 0.6).
func QuestFromMoney(moneyAtMaxLevel int) int {
	if moneyAtMaxLevel <= 0 {
		return 0
	}
	return int(math.Round(float64(moneyAtMaxLevel) / 0.6))
}

// GrayLevel is the highest mob level that gives no XP to a player of level pl.
func GrayLevel(pl int) int {
	switch {
	case pl <= 5:
		return 0
	case pl <= 39:
		return pl - 5 - pl/10
	case pl <= 59:
		return pl - 1 - pl/5
	default:
		return pl - 9
	}
}

func zeroDifference(pl int) int {
	switch {
	case pl < 8:
		return 5
	case pl < 10:
		return 6
	case pl < 12:
		return 7
	case pl < 16:
		return 8
	case pl < 20:
		return 9
	case pl < 30:
		return 11
	case pl < 40:
		return 12
	case pl < 45:
		return 13
	case pl < 50:
		return 14
	case pl < 55:
		return 15
	case pl < 60:
		return 16
	default:
		return 17
	}
}

// Kill is the XP of killing one mob of level ml for a player of level pl; elite mobs give twice as much.
func Kill(pl, ml int, elite bool) int {
	base := 0.0
	if ml >= pl {
		d := ml - pl
		if d > 4 {
			d = 4
		}
		base = float64(pl*5+45) * (1 + 0.05*float64(d))
	} else {
		if ml <= GrayLevel(pl) {
			return 0
		}
		zd := float64(zeroDifference(pl))
		base = float64(pl*5+45) * (1 - float64(pl-ml)/zd)
	}
	if elite {
		base *= 2
	}
	if base < 0 {
		return 0
	}
	return int(math.Round(base))
}

// Level is the level and the XP into it for a character with total XP, using table (XP per level step).
func Level(table []int, total int) (level, into int) {
	level, into = 1, total
	for level-1 < len(table) && into >= table[level-1] {
		into -= table[level-1]
		level++
	}
	return level, into
}

// Total is the XP from level 1 to the start of level.
func Total(table []int, level int) int {
	sum := 0
	for i := 0; i < level-1 && i < len(table); i++ {
		sum += table[i]
	}
	return sum
}

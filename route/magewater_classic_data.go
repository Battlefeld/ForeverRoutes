package route

// ClassicMageWaterReference contains published Classic item tooltip values.
// Forever-specific overrides must be verified against Forever client data
// before these values are treated as authoritative for that build.
// RequiredLevel is the drink's usable level, not the mage's conjure spell level.
var ClassicMageWaterReference = []MageWaterRank{
 {ItemID:5350, RequiredLevel:1, RestoreMana:151.2, DurationSeconds:18},
 {ItemID:2288, RequiredLevel:5, RestoreMana:436.8, DurationSeconds:21},
 {ItemID:2136, RequiredLevel:15, RestoreMana:835.2, DurationSeconds:24},
 {ItemID:3772, RequiredLevel:25, RestoreMana:1344.6, DurationSeconds:27},
 {ItemID:8077, RequiredLevel:35, RestoreMana:1992, DurationSeconds:30},
 {ItemID:8078, RequiredLevel:45, RestoreMana:2934, DurationSeconds:30},
 {ItemID:8079, RequiredLevel:55, RestoreMana:4200, DurationSeconds:30},
}

// Reference: Warcraft Wiki Conjured Water table and Wowhead Classic item pages.
// https://warcraft.wiki.gg/wiki/Conjured_Water/table
// https://www.wowhead.com/classic/item=3772/conjured-spring-water

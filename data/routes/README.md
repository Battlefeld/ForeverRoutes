# Routes 1–20

The best route ForeverRoutes knows for each race and class of WoW Forever, from level 1 to 20, for a character with
its starter weapon and no other gear or talents (the class's kill profile in `data/profiles`). ForeverCore follows
the character's route from wherever it stands (`foreverroutes next -route`); without ForeverCore.exe it is the route
to follow by hand.

Each file was made by `foreverroutes improve`: the RestedXP guide for the race and class and our planner's own route,
each improved by local edits that reach level 20 sooner under the model (leaving out a quest that costs more time than
it brings, putting in one the route passes by). The planned route is kept only when it is at least 3 % faster than the improved guide, else the guide's (`winner`), with the hours of all four
in `hours`. So a route is never slower under the model than the RestedXP guide it was measured against.

`events` are the steps in order: accept, complete or turn in a quest, or reach a level.

`classQuests` lists every class quest chain of the race and class below level 20 (`route/classquests.go`): `required`
chains (class letter, imp and voidwalker, taming, bear form, earth and fire totem, defensive stance, lockpicking) are in
every route; `recommended` (green weapons and robes, priest racials) and `optional` ones are options. `minutes` is what
a chain costs the route: for one in the route the time it adds (negative: its XP make the route faster), for an option
the time it adds when put in; `level` is the level the route hands it in at.

**Credits.** Routes whose `winner` is `guide` are adapted from the RestedXP Forever guides
(https://github.com/RestedXP/RXPGuides, © RestedXP, CC BY-NC-SA 4.0): their quest order, improved here. These files
are shared under the same licence, CC BY-NC-SA 4.0 (https://creativecommons.org/licenses/by-nc-sa/4.0/), non-commercial,
with this credit. Routes whose `winner` is `plan` are ForeverRoutes' own.

**Limits.** The hours are the model's (fight times from MythicSim for a naked character, walking on the navigation
grid of Shortest Path Forever, flights, boats, lifts and deathskips); in game they are checked by the recordings after the release. Quests the dataset cannot model yet (items
no source places) are left out of every route alike.

**Hours to level 20** (model, dataset of this commit):

| Race | Class | RestedXP | RestedXP improved | Planned | Planned improved | Kept |
|---|---|---|---|---|---|---|
| Dwarf | Hunter | 19.9 | 16.7 | 16.2 | 16.9 | guide |
| Dwarf | Paladin | 18.4 | 15.6 | 15.1 | 14.8 | plan |
| Dwarf | Priest | 22.3 | 17.6 | 18.6 | 18.4 | guide |
| Dwarf | Rogue | 16.1 | 13.8 | 14.4 | 13.7 | guide |
| Dwarf | Shaman | 19.4 | 17.5 | 15.9 | 17.7 | guide |
| Dwarf | Warrior | 16.1 | 14.0 | 14.4 | 14.2 | guide |
| Gnome | Mage | 23.0 | 19.1 | 18.1 | 17.6 | plan |
| Gnome | Priest | 22.3 | 18.4 | 18.7 | 18.3 | guide |
| Gnome | Rogue | 16.1 | 13.8 | 14.4 | 13.7 | guide |
| Gnome | Warlock | 21.8 | 16.7 | 16.5 | 16.3 | guide |
| Gnome | Warrior | 16.1 | 14.0 | 14.4 | 14.2 | guide |
| Human | Hunter | 22.2 | 17.1 | 16.3 | 15.9 | plan |
| Human | Mage | 21.3 | 18.4 | 18.0 | 17.2 | plan |
| Human | Paladin | 19.5 | 15.1 | 16.0 | 15.6 | guide |
| Human | Priest | 20.8 | 17.6 | 17.3 | 17.2 | guide |
| Human | Rogue | 15.8 | 14.0 | 13.2 | 13.3 | plan |
| Human | Warlock | 22.2 | 16.1 | 15.3 | 15.8 | guide |
| Human | Warrior | 15.2 | 13.6 | 13.8 | 13.8 | guide |
| NightElf | Druid | 22.9 | 19.5 | 17.9 | 19.0 | guide |
| NightElf | Hunter | 19.2 | 16.8 | 17.3 | 19.0 | guide |
| NightElf | Priest | 20.8 | 18.6 | 17.6 | 17.5 | plan |
| NightElf | Rogue | 16.0 | 14.6 | 14.0 | 13.6 | plan |
| NightElf | Warrior | 16.1 | 14.3 | 14.4 | 14.4 | guide |
| Orc | Hunter | 19.1 | 18.6 | 15.9 | 16.2 | plan |
| Orc | Mage | 16.4 | 15.1 | 15.4 | 15.6 | guide |
| Orc | Rogue | 13.5 | 12.8 | 12.2 | 12.8 | guide |
| Orc | Shaman | 17.4 | 17.2 | 15.3 | 15.9 | plan |
| Orc | Warlock | 14.9 | 13.9 | 14.6 | 14.7 | guide |
| Orc | Warrior | 12.9 | 11.9 | 12.8 | 13.4 | guide |
| SkyborneA | Druid | 22.8 | 19.5 | 18.4 | 18.4 | plan |
| SkyborneA | Hunter | 21.2 | 17.9 | 17.2 | 17.9 | guide |
| SkyborneA | Mage | 23.6 | 20.4 | 18.7 | 18.5 | plan |
| SkyborneA | Rogue | 17.5 | 16.2 | 12.8 | 13.3 | plan |
| SkyborneA | Warrior | 16.9 | 15.3 | 14.8 | 14.6 | plan |
| SkyborneH | Druid | 22.0 | 15.5 | 18.5 | 18.9 | guide |
| SkyborneH | Hunter | 19.8 | 16.5 | 18.4 | 19.6 | guide |
| SkyborneH | Rogue | 14.8 | 13.5 | 13.0 | 14.6 | guide |
| SkyborneH | Shaman | 18.2 | 16.2 | 17.2 | 18.1 | guide |
| SkyborneH | Warrior | 14.7 | 12.8 | 13.1 | 13.5 | guide |
| Tauren | Druid | 20.9 | 19.4 | 15.4 | 18.1 | plan |
| Tauren | Hunter | 18.8 | 15.4 | 16.3 | 17.0 | guide |
| Tauren | Shaman | 18.6 | 16.7 | 16.2 | 17.4 | guide |
| Tauren | Warrior | 16.1 | 14.1 | 12.9 | 13.2 | plan |
| Troll | Hunter | 19.1 | 18.6 | 15.9 | 16.2 | plan |
| Troll | Mage | 16.4 | 15.1 | 16.2 | 16.1 | guide |
| Troll | Priest | 16.3 | 14.6 | 16.2 | 16.2 | guide |
| Troll | Rogue | 13.3 | 12.9 | 12.2 | 12.8 | guide |
| Troll | Shaman | 17.4 | 17.2 | 15.3 | 15.9 | plan |
| Troll | Warlock | 14.9 | 13.9 | 14.6 | 14.7 | guide |
| Troll | Warrior | 12.9 | 11.9 | 12.8 | 13.3 | guide |
| Undead | Mage | 16.5 | 15.2 | 16.7 | 16.7 | guide |
| Undead | Paladin | 14.4 | 13.4 | 13.6 | 14.0 | guide |
| Undead | Priest | 15.7 | 14.6 | 18.3 | 18.0 | guide |
| Undead | Rogue | 13.2 | 12.8 | 12.3 | 13.4 | guide |
| Undead | Warlock | 15.2 | 15.6 | 15.6 | 16.6 | guide |
| Undead | Warrior | 12.6 | 11.9 | 12.0 | 12.1 | guide |

Average over the 56 combinations: RestedXP 17.88 h, the kept route 15.28 h (14.5 % faster). A planned route stays at home until the first class quests are done and crosses between Kalimdor and the Eastern Kingdoms at most twice up to level 20.

## Solo with dungeons

`<race>-<class>-dungeons.json` ("Solo mit Dungeon"): the same, with dungeon runs (a group only for the run itself) (`foreverroutes improve -dungeons`). The route takes a
dungeon's quests on the way (from three levels below it, within ten minutes), runs it with a group and hands the quests
in, across the sea if need be (Ragefire's quest from Undercity). The addon shows it after `/fr dungeon` (`/fr solo` goes back to the route without dungeons, which takes no dungeon quests). Run times are
estimates (`data/sources/dungeons.json`); better gear from the runs is not counted yet, so the routes with dungeons are rather
faster in game than here.

Average over the 56 combinations: 13.50 h to level 20 (without dungeons 15.28 h).

| Race | Class | Hours | Kept |
|---|---|---|---|
| Dwarf | Hunter | 14.0 | plan |
| Dwarf | Paladin | 12.7 | plan |
| Dwarf | Priest | 14.7 | plan |
| Dwarf | Rogue | 12.4 | plan |
| Dwarf | Shaman | 15.5 | guide |
| Dwarf | Warrior | 13.0 | guide |
| Gnome | Mage | 15.3 | plan |
| Gnome | Priest | 14.0 | plan |
| Gnome | Rogue | 12.4 | plan |
| Gnome | Warlock | 14.2 | plan |
| Gnome | Warrior | 13.0 | guide |
| Human | Hunter | 13.9 | plan |
| Human | Mage | 15.1 | plan |
| Human | Paladin | 12.0 | plan |
| Human | Priest | 14.9 | guide |
| Human | Rogue | 11.4 | plan |
| Human | Warlock | 13.2 | plan |
| Human | Warrior | 11.0 | plan |
| NightElf | Druid | 16.2 | guide |
| NightElf | Hunter | 14.2 | guide |
| NightElf | Priest | 14.4 | plan |
| NightElf | Rogue | 13.0 | plan |
| NightElf | Warrior | 12.5 | plan |
| Orc | Hunter | 13.1 | plan |
| Orc | Mage | 13.8 | guide |
| Orc | Rogue | 12.8 | guide |
| Orc | Shaman | 14.8 | plan |
| Orc | Warlock | 13.0 | guide |
| Orc | Warrior | 11.3 | guide |
| SkyborneA | Druid | 15.3 | plan |
| SkyborneA | Hunter | 15.6 | guide |
| SkyborneA | Mage | 15.6 | plan |
| SkyborneA | Rogue | 13.1 | plan |
| SkyborneA | Warrior | 12.8 | plan |
| SkyborneH | Druid | 13.7 | plan |
| SkyborneH | Hunter | 14.7 | guide |
| SkyborneH | Rogue | 12.9 | plan |
| SkyborneH | Shaman | 14.5 | guide |
| SkyborneH | Warrior | 12.7 | guide |
| Tauren | Druid | 15.9 | guide |
| Tauren | Hunter | 13.7 | plan |
| Tauren | Shaman | 13.4 | plan |
| Tauren | Warrior | 11.0 | plan |
| Troll | Hunter | 13.1 | plan |
| Troll | Mage | 13.7 | plan |
| Troll | Priest | 13.1 | plan |
| Troll | Rogue | 12.9 | guide |
| Troll | Shaman | 14.8 | plan |
| Troll | Warlock | 13.0 | guide |
| Troll | Warrior | 11.3 | guide |
| Undead | Mage | 13.2 | plan |
| Undead | Paladin | 12.1 | plan |
| Undead | Priest | 12.7 | plan |
| Undead | Rogue | 12.8 | plan |
| Undead | Warlock | 14.8 | plan |
| Undead | Warrior | 11.0 | plan |

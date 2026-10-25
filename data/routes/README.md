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

**Credits.** Routes whose `winner` is `guide` are adapted from the RestedXP Forever guides
(https://github.com/RestedXP/RXPGuides, © RestedXP, CC BY-NC-SA 4.0): their quest order, improved here. These files
are shared under the same licence, CC BY-NC-SA 4.0 (https://creativecommons.org/licenses/by-nc-sa/4.0/), non-commercial,
with this credit. Routes whose `winner` is `plan` are ForeverRoutes' own.

**Limits.** The hours are the model's (fight times from MythicSim for a naked character, walking on the navigation
grid of Shortest Path Forever, flights, boats, lifts and deathskips); in game they are checked by the recordings after the release. Quests the dataset cannot model yet (items
no source places) are left out of every route alike.

**Hours to level 20** (model, dataset of this commit):

| Race | Class | RestedXP | RestedXP improved | Plan | Plan improved | Kept |
|---|---|---|---|---|---|---|
| Dwarf | Hunter | 18.7 | 17.0 | 18.4 | 16.9 | guide |
| Dwarf | Paladin | 17.7 | 16.0 | 17.1 | 16.6 | guide |
| Dwarf | Priest | 19.1 | 18.1 | 18.8 | 18.4 | guide |
| Dwarf | Rogue | 16.3 | 14.8 | 15.1 | 14.6 | guide |
| Dwarf | Shaman | 19.1 | 17.4 | 16.2 | 16.4 | plan |
| Dwarf | Warrior | 16.1 | 14.7 | 15.2 | 14.9 | guide |
| Gnome | Mage | 20.0 | 18.5 | 18.3 | 17.8 | plan |
| Gnome | Priest | 19.1 | 17.9 | 19.2 | 18.9 | guide |
| Gnome | Rogue | 16.3 | 14.8 | 15.1 | 14.6 | guide |
| Gnome | Warlock | 17.1 | 16.4 | 16.4 | 16.4 | guide |
| Gnome | Warrior | 16.1 | 14.7 | 15.2 | 14.9 | guide |
| Human | Hunter | 21.7 | 16.9 | 16.2 | 16.0 | plan |
| Human | Mage | 19.2 | 18.3 | 17.9 | 17.3 | plan |
| Human | Paladin | 17.9 | 15.8 | 15.7 | 15.6 | guide |
| Human | Priest | 18.1 | 17.4 | 17.2 | 17.4 | guide |
| Human | Rogue | 16.3 | 14.9 | 14.3 | 14.8 | guide |
| Human | Warlock | 19.7 | 15.7 | 15.4 | 15.0 | plan |
| Human | Warrior | 15.6 | 14.3 | 14.8 | 14.8 | guide |
| NightElf | Druid | 23.8 | 19.6 | 17.8 | 17.8 | plan |
| NightElf | Hunter | 18.7 | 16.8 | 17.1 | 17.1 | guide |
| NightElf | Priest | 21.2 | 18.4 | 17.6 | 17.4 | plan |
| NightElf | Rogue | 18.0 | 15.9 | 15.2 | 15.4 | plan |
| NightElf | Warrior | 18.0 | 15.5 | 14.9 | 14.7 | plan |
| Orc | Hunter | 19.4 | 19.1 | 15.4 | 15.6 | plan |
| Orc | Mage | 16.6 | 15.5 | 15.9 | 15.9 | guide |
| Orc | Rogue | 16.1 | 15.3 | 15.1 | 15.5 | guide |
| Orc | Shaman | 18.5 | 18.2 | 15.7 | 15.8 | plan |
| Orc | Warlock | 15.1 | 14.5 | 14.3 | 14.3 | guide |
| Orc | Warrior | 15.6 | 13.4 | 14.4 | 14.7 | guide |
| Skyborne (A) | Druid | 20.7 | 19.2 | 17.9 | 17.7 | plan |
| Skyborne (A) | Hunter | 19.1 | 17.7 | 17.0 | 16.7 | plan |
| Skyborne (A) | Mage | 20.6 | 19.1 | 18.3 | 18.0 | plan |
| Skyborne (A) | Rogue | 19.4 | 17.2 | 14.8 | 14.5 | plan |
| Skyborne (A) | Warrior | 19.2 | 16.4 | 15.0 | 15.1 | plan |
| Skyborne (H) | Druid | 18.3 | 16.4 | 19.1 | 19.5 | guide |
| Skyborne (H) | Hunter | 17.0 | 15.7 | 19.0 | 19.0 | guide |
| Skyborne (H) | Rogue | 17.9 | 15.0 | 15.2 | 14.8 | guide |
| Skyborne (H) | Shaman | 19.1 | 18.2 | 16.7 | 16.8 | plan |
| Skyborne (H) | Warrior | 17.3 | 14.2 | 14.4 | 14.4 | guide |
| Tauren | Druid | 20.1 | 17.8 | 18.8 | 19.1 | guide |
| Tauren | Hunter | 17.7 | 16.5 | 16.1 | 15.7 | plan |
| Tauren | Shaman | 18.5 | 16.5 | 16.0 | 16.7 | guide |
| Tauren | Warrior | 17.5 | 15.9 | 14.2 | 14.4 | plan |
| Troll | Hunter | 19.4 | 19.1 | 15.4 | 15.6 | plan |
| Troll | Mage | 16.6 | 15.5 | 15.8 | 16.5 | guide |
| Troll | Priest | 16.4 | 15.0 | 16.0 | 15.5 | guide |
| Troll | Rogue | 15.7 | 15.2 | 14.8 | 15.0 | guide |
| Troll | Shaman | 18.5 | 18.2 | 15.7 | 15.8 | plan |
| Troll | Warlock | 15.1 | 14.5 | 14.3 | 14.3 | guide |
| Troll | Warrior | 15.6 | 13.4 | 14.4 | 14.7 | guide |
| Undead | Mage | 16.2 | 14.8 | 16.0 | 16.4 | guide |
| Undead | Paladin | 16.8 | 14.9 | 16.4 | 16.5 | guide |
| Undead | Priest | 15.3 | 14.7 | 18.3 | 18.0 | guide |
| Undead | Rogue | 15.4 | 13.5 | 14.0 | 14.2 | guide |
| Undead | Warlock | 15.1 | 14.6 | 15.2 | 15.5 | guide |
| Undead | Warrior | 15.3 | 13.1 | 13.2 | 13.8 | guide |

Average over the 56 combinations: RestedXP 17.84 h, the kept route 15.62 h (12.4 % faster). A planned route stays at home until the first class quests are done and crosses between Kalimdor and the Eastern Kingdoms at most twice up to level 20.

## Solo with dungeons

`<race>-<class>-dungeons.json` ("Solo mit Dungeon"): the same, with dungeon runs (a group only for the run itself) (`foreverroutes improve -dungeons`). The route takes a
dungeon's quests on the way (from three levels below it, within ten minutes), runs it with a group and hands the quests
in, across the sea if need be (Ragefire's quest from Undercity). The addon shows it after `/fr dungeon` (`/fr solo` goes back to the route without dungeons, which takes no dungeon quests). Run times are
estimates (`data/sources/dungeons.json`); better gear from the runs is not counted yet, so the routes with dungeons are rather
faster in game than here.

Average over the 56 combinations: 13.48 h to level 20 (without dungeons 15.62 h).

| Race | Class | Hours | Basis |
|---|---|---|---|
| Dwarf | Hunter | 13.5 | plan |
| Dwarf | Paladin | 13.7 | plan |
| Dwarf | Priest | 14.8 | plan |
| Dwarf | Rogue | 12.2 | plan |
| Dwarf | Shaman | 14.2 | plan |
| Dwarf | Warrior | 13.1 | plan |
| Gnome | Mage | 14.5 | plan |
| Gnome | Priest | 14.7 | plan |
| Gnome | Rogue | 12.2 | plan |
| Gnome | Warlock | 13.6 | plan |
| Gnome | Warrior | 13.1 | plan |
| Human | Hunter | 13.8 | plan |
| Human | Mage | 14.5 | plan |
| Human | Paladin | 13.9 | plan |
| Human | Priest | 14.5 | plan |
| Human | Rogue | 12.9 | plan |
| Human | Warlock | 14.0 | plan |
| Human | Warrior | 12.3 | plan |
| NightElf | Druid | 15.6 | plan |
| NightElf | Hunter | 14.2 | plan |
| NightElf | Priest | 15.2 | plan |
| NightElf | Rogue | 12.9 | plan |
| NightElf | Warrior | 13.9 | plan |
| Orc | Hunter | 13.9 | plan |
| Orc | Mage | 14.9 | plan |
| Orc | Rogue | 12.8 | plan |
| Orc | Shaman | 15.0 | plan |
| Orc | Warlock | 12.9 | plan |
| Orc | Warrior | 10.2 | plan |
| Skyborne (A) | Druid | 15.4 | plan |
| Skyborne (A) | Hunter | 14.3 | plan |
| Skyborne (A) | Mage | 15.3 | plan |
| Skyborne (A) | Rogue | 13.4 | plan |
| Skyborne (A) | Warrior | 13.5 | plan |
| Skyborne (H) | Druid | 14.8 | plan |
| Skyborne (H) | Hunter | 14.7 | plan |
| Skyborne (H) | Rogue | 12.9 | plan |
| Skyborne (H) | Shaman | 12.1 | plan |
| Skyborne (H) | Warrior | 12.5 | plan |
| Tauren | Druid | 14.6 | plan |
| Tauren | Hunter | 14.6 | plan |
| Tauren | Shaman | 12.1 | plan |
| Tauren | Warrior | 10.0 | plan |
| Troll | Hunter | 13.9 | plan |
| Troll | Mage | 14.9 | plan |
| Troll | Priest | 13.3 | plan |
| Troll | Rogue | 12.8 | plan |
| Troll | Shaman | 15.0 | plan |
| Troll | Warlock | 12.9 | plan |
| Troll | Warrior | 10.2 | plan |
| Undead | Mage | 12.6 | plan |
| Undead | Paladin | 13.4 | plan |
| Undead | Priest | 14.8 | plan |
| Undead | Rogue | 11.8 | plan |
| Undead | Warlock | 10.5 | plan |
| Undead | Warrior | 11.7 | plan |

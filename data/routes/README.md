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
| Dwarf | Hunter | 18.7 | 16.9 | 17.3 | 17.1 | guide |
| Dwarf | Paladin | 17.7 | 15.7 | 16.6 | 16.2 | guide |
| Dwarf | Priest | 19.1 | 18.1 | 17.5 | 17.4 | plan |
| Dwarf | Rogue | 16.1 | 14.8 | 15.1 | 14.8 | guide |
| Dwarf | Shaman | 19.2 | 17.6 | 15.7 | 15.7 | plan |
| Dwarf | Warrior | 15.7 | 14.5 | 14.5 | 13.9 | plan |
| Gnome | Mage | 20.0 | 18.8 | 17.1 | 16.7 | plan |
| Gnome | Priest | 19.1 | 17.6 | 17.6 | 17.3 | guide |
| Gnome | Rogue | 16.1 | 14.8 | 15.1 | 14.8 | guide |
| Gnome | Warlock | 16.7 | 16.1 | 15.7 | 15.8 | guide |
| Gnome | Warrior | 15.7 | 14.5 | 14.5 | 13.9 | plan |
| Human | Hunter | 18.7 | 16.8 | 16.8 | 16.6 | guide |
| Human | Mage | 19.2 | 18.4 | 16.9 | 17.1 | plan |
| Human | Paladin | 18.2 | 15.6 | 15.2 | 15.1 | guide |
| Human | Priest | 18.0 | 17.4 | 17.5 | 17.3 | guide |
| Human | Rogue | 16.0 | 15.0 | 14.4 | 14.1 | plan |
| Human | Warlock | 16.8 | 15.4 | 16.3 | 15.3 | guide |
| Human | Warrior | 15.2 | 14.2 | 15.2 | 15.3 | guide |
| NightElf | Druid | 24.4 | 20.1 | 18.1 | 17.6 | plan |
| NightElf | Hunter | 19.6 | 17.8 | 17.4 | 17.5 | guide |
| NightElf | Priest | 21.9 | 18.9 | 17.8 | 17.5 | plan |
| NightElf | Rogue | 18.2 | 16.3 | 14.0 | 14.4 | plan |
| NightElf | Warrior | 17.8 | 16.0 | 14.3 | 14.4 | plan |
| Orc | Hunter | 19.9 | 19.5 | 16.4 | 16.3 | plan |
| Orc | Mage | 18.4 | 16.2 | 16.1 | 16.3 | guide |
| Orc | Rogue | 15.1 | 13.3 | 13.7 | 13.8 | guide |
| Orc | Shaman | 18.4 | 17.9 | 15.5 | 15.3 | plan |
| Orc | Warlock | 16.1 | 14.9 | 15.0 | 15.3 | guide |
| Orc | Warrior | 14.5 | 12.6 | 13.5 | 13.9 | guide |
| Skyborne (A) | Druid | 20.7 | 19.2 | 18.7 | 18.2 | plan |
| Skyborne (A) | Hunter | 19.7 | 18.0 | 17.8 | 17.4 | plan |
| Skyborne (A) | Mage | 20.8 | 19.8 | 18.5 | 18.1 | plan |
| Skyborne (A) | Rogue | 17.3 | 16.3 | 14.9 | 14.5 | plan |
| Skyborne (A) | Warrior | 17.0 | 15.6 | 15.4 | 15.0 | plan |
| Skyborne (H) | Druid | 18.5 | 17.0 | 16.7 | 16.5 | guide |
| Skyborne (H) | Hunter | 16.9 | 16.1 | 15.9 | 16.1 | guide |
| Skyborne (H) | Rogue | 15.6 | 13.3 | 15.3 | 15.1 | guide |
| Skyborne (H) | Shaman | 17.5 | 14.5 | 15.9 | 15.7 | guide |
| Skyborne (H) | Warrior | 15.4 | 13.3 | 13.7 | 14.1 | guide |
| Tauren | Druid | 19.9 | 18.4 | 16.6 | 16.5 | plan |
| Tauren | Hunter | 17.7 | 16.5 | 16.2 | 16.2 | guide |
| Tauren | Shaman | 17.8 | 15.2 | 15.0 | 15.2 | guide |
| Tauren | Warrior | 16.2 | 14.5 | 13.2 | 13.1 | plan |
| Troll | Hunter | 19.9 | 19.5 | 16.4 | 16.3 | plan |
| Troll | Mage | 18.4 | 16.3 | 17.2 | 16.3 | guide |
| Troll | Priest | 17.9 | 15.3 | 16.1 | 16.1 | guide |
| Troll | Rogue | 14.8 | 12.7 | 13.7 | 13.8 | guide |
| Troll | Shaman | 18.4 | 17.9 | 15.5 | 15.3 | plan |
| Troll | Warlock | 16.1 | 14.9 | 15.0 | 15.3 | guide |
| Troll | Warrior | 14.5 | 12.6 | 13.5 | 13.9 | guide |
| Undead | Mage | 18.3 | 16.0 | 17.5 | 17.9 | guide |
| Undead | Paladin | 15.1 | 13.6 | 13.9 | 13.9 | guide |
| Undead | Priest | 17.9 | 15.0 | 18.6 | 18.0 | guide |
| Undead | Rogue | 14.9 | 12.8 | 14.5 | 14.3 | guide |
| Undead | Warlock | 16.4 | 15.2 | 15.1 | 14.8 | guide |
| Undead | Warrior | 14.3 | 12.5 | 14.5 | 14.2 | guide |

Average over the 56 combinations: RestedXP 17.66 h, the kept route 15.33 h (13.2 % faster). A planned route stays at home until the first class quests are done and crosses between Kalimdor and the Eastern Kingdoms at most twice up to level 20.

## Group routes

`<race>-<class>-group.json`: the same, with dungeon runs (`foreverroutes improve -dungeons`). The route takes a
dungeon's quests on the way (from three levels below it, within ten minutes), runs it with a group and hands the quests
in, across the sea if need be (Ragefire's quest from Undercity). The addon shows it after `/fr gruppe`. Run times are
estimates (`data/sources/dungeons.json`); better gear from the runs is not counted yet, so the group routes are rather
faster in game than here.

Average over the 56 combinations: 13.25 h to level 20 (solo 15.33 h).

| Race | Class | Hours | Basis |
|---|---|---|---|
| Dwarf | Hunter | 13.8 | plan |
| Dwarf | Paladin | 13.7 | plan |
| Dwarf | Priest | 14.3 | plan |
| Dwarf | Rogue | 12.7 | plan |
| Dwarf | Shaman | 13.9 | plan |
| Dwarf | Warrior | 12.4 | plan |
| Gnome | Mage | 15.2 | plan |
| Gnome | Priest | 15.5 | plan |
| Gnome | Rogue | 12.7 | plan |
| Gnome | Warlock | 14.0 | plan |
| Gnome | Warrior | 12.4 | plan |
| Human | Hunter | 14.8 | plan |
| Human | Mage | 15.3 | plan |
| Human | Paladin | 13.7 | plan |
| Human | Priest | 15.0 | plan |
| Human | Rogue | 13.2 | plan |
| Human | Warlock | 13.4 | plan |
| Human | Warrior | 12.8 | plan |
| NightElf | Druid | 15.2 | plan |
| NightElf | Hunter | 14.6 | plan |
| NightElf | Priest | 15.8 | plan |
| NightElf | Rogue | 12.8 | plan |
| NightElf | Warrior | 13.2 | plan |
| Orc | Hunter | 13.9 | plan |
| Orc | Mage | 14.4 | plan |
| Orc | Rogue | 11.4 | plan |
| Orc | Shaman | 13.3 | plan |
| Orc | Warlock | 13.1 | plan |
| Orc | Warrior | 11.6 | plan |
| Skyborne (A) | Druid | 14.9 | plan |
| Skyborne (A) | Hunter | 13.9 | plan |
| Skyborne (A) | Mage | 15.9 | plan |
| Skyborne (A) | Rogue | 13.6 | plan |
| Skyborne (A) | Warrior | 13.4 | plan |
| Skyborne (H) | Druid | 13.3 | plan |
| Skyborne (H) | Hunter | 13.8 | plan |
| Skyborne (H) | Rogue | 12.6 | plan |
| Skyborne (H) | Shaman | 11.3 | plan |
| Skyborne (H) | Warrior | 11.5 | plan |
| Tauren | Druid | 12.8 | plan |
| Tauren | Hunter | 11.8 | plan |
| Tauren | Shaman | 12.1 | plan |
| Tauren | Warrior | 10.1 | plan |
| Troll | Hunter | 13.9 | plan |
| Troll | Mage | 13.1 | plan |
| Troll | Priest | 13.2 | plan |
| Troll | Rogue | 11.4 | plan |
| Troll | Shaman | 13.3 | plan |
| Troll | Warlock | 13.1 | plan |
| Troll | Warrior | 11.6 | plan |
| Undead | Mage | 13.6 | plan |
| Undead | Paladin | 12.8 | plan |
| Undead | Priest | 12.4 | plan |
| Undead | Rogue | 10.6 | plan |
| Undead | Warlock | 12.1 | plan |
| Undead | Warrior | 10.7 | plan |

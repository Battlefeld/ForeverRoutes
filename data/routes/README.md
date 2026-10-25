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
| Dwarf | Hunter | 18.7 | 16.8 | 16.7 | 16.5 | guide |
| Dwarf | Paladin | 17.7 | 15.6 | 15.1 | 15.1 | plan |
| Dwarf | Priest | 19.1 | 18.0 | 16.8 | 16.6 | plan |
| Dwarf | Rogue | 16.1 | 14.7 | 13.7 | 13.6 | plan |
| Dwarf | Shaman | 19.9 | 16.1 | 15.6 | 15.3 | plan |
| Dwarf | Warrior | 15.7 | 14.3 | 14.6 | 14.6 | guide |
| Gnome | Mage | 20.0 | 18.7 | 17.1 | 17.1 | plan |
| Gnome | Priest | 19.1 | 18.0 | 17.1 | 16.7 | plan |
| Gnome | Rogue | 16.1 | 14.7 | 13.7 | 13.6 | plan |
| Gnome | Warlock | 16.7 | 16.1 | 15.8 | 15.6 | plan |
| Gnome | Warrior | 15.7 | 14.3 | 14.6 | 14.6 | guide |
| Human | Hunter | 18.7 | 16.8 | 16.8 | 16.6 | guide |
| Human | Mage | 19.2 | 18.4 | 17.0 | 16.9 | plan |
| Human | Paladin | 18.2 | 15.5 | 17.2 | 16.8 | guide |
| Human | Priest | 18.0 | 17.2 | 18.1 | 17.9 | guide |
| Human | Rogue | 16.0 | 14.9 | 13.5 | 13.6 | plan |
| Human | Warlock | 16.8 | 15.7 | 14.7 | 14.7 | plan |
| Human | Warrior | 15.2 | 14.2 | 15.2 | 14.9 | guide |
| NightElf | Druid | 24.4 | 19.6 | 17.7 | 17.6 | plan |
| NightElf | Hunter | 19.6 | 17.1 | 16.8 | 16.8 | guide |
| NightElf | Priest | 21.9 | 18.7 | 17.8 | 17.7 | plan |
| NightElf | Rogue | 18.2 | 16.2 | 15.0 | 14.7 | plan |
| NightElf | Warrior | 17.8 | 15.8 | 15.0 | 15.2 | plan |
| Orc | Hunter | 19.9 | 19.5 | 15.4 | 15.5 | plan |
| Orc | Mage | 18.4 | 16.2 | 16.6 | 16.8 | guide |
| Orc | Rogue | 15.1 | 12.7 | 13.7 | 13.8 | guide |
| Orc | Shaman | 18.4 | 17.9 | 15.1 | 15.2 | plan |
| Orc | Warlock | 16.1 | 14.9 | 14.3 | 14.5 | guide |
| Orc | Warrior | 14.5 | 12.7 | 13.2 | 13.2 | guide |
| Skyborne (A) | Druid | 20.7 | 19.2 | 18.7 | 18.2 | plan |
| Skyborne (A) | Hunter | 19.7 | 18.0 | 17.8 | 17.4 | plan |
| Skyborne (A) | Mage | 20.8 | 19.8 | 17.9 | 18.1 | plan |
| Skyborne (A) | Rogue | 17.3 | 16.3 | 14.9 | 14.5 | plan |
| Skyborne (A) | Warrior | 17.0 | 15.6 | 15.4 | 15.0 | plan |
| Skyborne (H) | Druid | 18.5 | 16.2 | 19.1 | 19.0 | guide |
| Skyborne (H) | Hunter | 16.9 | 16.1 | 19.4 | 19.6 | guide |
| Skyborne (H) | Rogue | 15.6 | 13.3 | 15.3 | 15.1 | guide |
| Skyborne (H) | Shaman | 17.5 | 14.5 | 15.9 | 15.7 | guide |
| Skyborne (H) | Warrior | 15.4 | 13.3 | 13.7 | 14.1 | guide |
| Tauren | Druid | 19.9 | 16.9 | 16.2 | 15.8 | plan |
| Tauren | Hunter | 17.7 | 16.5 | 15.9 | 15.2 | plan |
| Tauren | Shaman | 18.5 | 15.3 | 14.7 | 14.8 | plan |
| Tauren | Warrior | 16.2 | 14.5 | 13.4 | 13.5 | plan |
| Troll | Hunter | 19.9 | 19.5 | 15.4 | 15.5 | plan |
| Troll | Mage | 18.4 | 16.2 | 16.2 | 16.1 | guide |
| Troll | Priest | 17.9 | 15.1 | 15.7 | 16.2 | guide |
| Troll | Rogue | 14.8 | 12.7 | 13.7 | 13.8 | guide |
| Troll | Shaman | 18.4 | 17.9 | 15.1 | 15.2 | plan |
| Troll | Warlock | 16.1 | 14.9 | 14.3 | 14.5 | guide |
| Troll | Warrior | 14.5 | 12.7 | 13.2 | 13.2 | guide |
| Undead | Mage | 18.3 | 16.0 | 17.4 | 17.3 | guide |
| Undead | Paladin | 15.1 | 13.5 | 17.5 | 17.4 | guide |
| Undead | Priest | 17.9 | 15.0 | 18.8 | 18.9 | guide |
| Undead | Rogue | 14.9 | 12.8 | 13.6 | 13.3 | guide |
| Undead | Warlock | 16.4 | 15.2 | 15.0 | 15.3 | guide |
| Undead | Warrior | 14.3 | 12.5 | 13.8 | 13.8 | guide |

Average over the 56 combinations: RestedXP 17.67 h, the kept route 15.16 h (14.2 % faster). A planned route crosses between Kalimdor and the Eastern Kingdoms at most twice up to level 20.

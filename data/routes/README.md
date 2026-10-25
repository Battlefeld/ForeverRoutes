# Routes 1–20

The best route ForeverRoutes knows for each race and class of WoW Forever, from level 1 to 20, for a character with
its starter weapon and no other gear or talents (the class's kill profile in `data/profiles`). ForeverCore follows
the character's route from wherever it stands (`foreverroutes next -route`); without ForeverCore.exe it is the route
to follow by hand.

Each file was made by `foreverroutes improve`: the RestedXP guide for the race and class and our planner's own route,
each improved by local edits that reach level 20 sooner under the model (leaving out a quest that costs more time than
it brings, putting in one the route passes by). The faster of the two is kept (`winner`), with the hours of all four
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
| Dwarf | Hunter | 17.4 | 15.6 | 16.3 | 16.0 | guide |
| Dwarf | Paladin | 17.2 | 15.1 | 15.7 | 15.4 | guide |
| Dwarf | Priest | 17.2 | 16.2 | 17.9 | 17.9 | guide |
| Dwarf | Rogue | 15.9 | 14.6 | 13.4 | 13.9 | plan |
| Dwarf | Shaman | 18.3 | 15.2 | 15.4 | 14.9 | plan |
| Dwarf | Warrior | 15.4 | 13.9 | 15.3 | 14.9 | guide |
| Gnome | Mage | 17.4 | 16.4 | 16.2 | 16.4 | guide |
| Gnome | Priest | 17.2 | 16.2 | 17.7 | 17.4 | guide |
| Gnome | Rogue | 15.9 | 14.6 | 13.4 | 13.9 | plan |
| Gnome | Warlock | 15.5 | 14.7 | 16.3 | 15.0 | guide |
| Gnome | Warrior | 15.4 | 13.9 | 15.3 | 14.9 | guide |
| Human | Hunter | 17.4 | 15.3 | 16.4 | 15.5 | guide |
| Human | Mage | 17.0 | 15.9 | 17.2 | 16.3 | guide |
| Human | Paladin | 17.7 | 15.7 | 15.7 | 15.5 | plan |
| Human | Priest | 16.7 | 16.2 | 16.9 | 16.9 | guide |
| Human | Rogue | 15.7 | 14.8 | 14.6 | 14.4 | plan |
| Human | Warlock | 15.5 | 14.5 | 14.9 | 14.6 | guide |
| Human | Warrior | 15.0 | 14.0 | 14.6 | 14.4 | guide |
| NightElf | Druid | 23.2 | 18.8 | 18.3 | 18.3 | plan |
| NightElf | Hunter | 18.3 | 16.0 | 16.6 | 16.4 | guide |
| NightElf | Priest | 20.5 | 17.6 | 17.6 | 17.2 | plan |
| NightElf | Rogue | 17.9 | 16.0 | 16.0 | 15.3 | plan |
| NightElf | Warrior | 17.7 | 15.7 | 15.0 | 15.1 | plan |
| Orc | Hunter | 18.5 | 18.2 | 15.4 | 15.4 | plan |
| Orc | Mage | 15.5 | 13.1 | 16.0 | 15.4 | guide |
| Orc | Rogue | 15.0 | 12.6 | 14.4 | 14.2 | guide |
| Orc | Shaman | 16.9 | 16.7 | 15.0 | 14.3 | plan |
| Orc | Warlock | 14.4 | 13.0 | 14.4 | 13.9 | guide |
| Orc | Warrior | 14.3 | 12.5 | 14.9 | 14.8 | guide |
| Skyborne (A) | Druid | 19.6 | 18.5 | 16.5 | 16.7 | plan |
| Skyborne (A) | Hunter | 18.8 | 18.1 | 16.8 | 16.4 | plan |
| Skyborne (A) | Mage | 19.1 | 18.1 | 17.2 | 17.2 | plan |
| Skyborne (A) | Rogue | 17.1 | 16.1 | 15.6 | 15.7 | plan |
| Skyborne (A) | Warrior | 16.7 | 15.5 | 15.7 | 15.7 | guide |
| Skyborne (H) | Druid | 17.5 | 15.6 | 15.6 | 15.7 | guide |
| Skyborne (H) | Hunter | 16.2 | 14.9 | 15.9 | 15.8 | guide |
| Skyborne (H) | Rogue | 15.4 | 13.4 | 15.0 | 14.5 | guide |
| Skyborne (H) | Shaman | 16.6 | 14.6 | 14.8 | 14.6 | plan |
| Skyborne (H) | Warrior | 15.2 | 13.1 | 14.2 | 14.1 | guide |
| Tauren | Druid | 18.7 | 15.6 | 16.1 | 15.6 | plan |
| Tauren | Hunter | 16.7 | 15.2 | 15.5 | 15.2 | plan |
| Tauren | Shaman | 17.0 | 14.1 | 14.4 | 14.0 | plan |
| Tauren | Warrior | 16.2 | 14.5 | 14.2 | 14.5 | plan |
| Troll | Hunter | 18.5 | 18.2 | 15.4 | 15.4 | plan |
| Troll | Mage | 15.5 | 13.1 | 16.0 | 15.5 | guide |
| Troll | Priest | 16.1 | 13.6 | 15.1 | 15.0 | guide |
| Troll | Rogue | 14.7 | 12.6 | 14.4 | 14.2 | guide |
| Troll | Shaman | 16.9 | 16.7 | 15.0 | 14.3 | plan |
| Troll | Warlock | 14.4 | 13.0 | 14.4 | 13.9 | guide |
| Troll | Warrior | 14.3 | 12.5 | 14.9 | 14.8 | guide |
| Undead | Mage | 15.7 | 13.4 | 15.6 | 15.0 | guide |
| Undead | Paladin | 14.8 | 13.1 | 15.3 | 14.6 | guide |
| Undead | Priest | 16.2 | 14.0 | 15.8 | 15.8 | guide |
| Undead | Rogue | 14.9 | 12.4 | 14.6 | 14.5 | guide |
| Undead | Warlock | 15.0 | 13.4 | 14.1 | 14.5 | guide |
| Undead | Warrior | 14.2 | 12.3 | 14.0 | 13.4 | guide |


Average over the 56 combinations: RestedXP 16.65 h, the kept route 14.63 h (12.2 % faster).

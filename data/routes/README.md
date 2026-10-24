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

**Limits.** The hours are the model's (fight times from MythicSim for a naked character, walking as straight lines
times 1.2); in game they are checked by the recordings after the release. Quests the dataset cannot model yet (items
no source places) are left out of every route alike.

**Hours to level 20** (model, dataset of this commit):

| Race | Class | RestedXP | RestedXP improved | Plan | Plan improved | Kept |
|---|---|---|---|---|---|---|
| Dwarf | Hunter | 17.3 | 15.9 | 16.0 | 15.2 | plan |
| Dwarf | Paladin | 17.8 | 15.4 | 15.5 | 15.0 | plan |
| Dwarf | Priest | 17.9 | 16.4 | 16.1 | 15.8 | plan |
| Dwarf | Rogue | 16.6 | 14.0 | 14.3 | 13.7 | plan |
| Dwarf | Shaman | 18.6 | 15.5 | 14.9 | 14.6 | plan |
| Dwarf | Warrior | 16.2 | 14.7 | 14.8 | 14.0 | plan |
| Gnome | Mage | 18.1 | 15.9 | 16.9 | 16.0 | guide |
| Gnome | Priest | 17.9 | 16.4 | 16.1 | 15.6 | plan |
| Gnome | Rogue | 16.6 | 14.0 | 14.3 | 13.7 | plan |
| Gnome | Warlock | 16.4 | 14.3 | 15.2 | 13.5 | plan |
| Gnome | Warrior | 16.2 | 14.7 | 14.8 | 14.0 | plan |
| Human | Hunter | 17.3 | 15.3 | 16.6 | 15.4 | guide |
| Human | Mage | 17.2 | 15.9 | 16.2 | 15.4 | plan |
| Human | Paladin | 17.9 | 14.9 | 15.3 | 15.0 | guide |
| Human | Priest | 17.3 | 15.9 | 16.9 | 15.4 | plan |
| Human | Rogue | 16.1 | 14.8 | 14.6 | 14.4 | plan |
| Human | Warlock | 15.9 | 14.7 | 15.6 | 15.1 | guide |
| Human | Warrior | 15.3 | 14.2 | 14.2 | 14.1 | plan |
| NightElf | Druid | 21.2 | 17.6 | 15.9 | 15.6 | plan |
| NightElf | Hunter | 17.7 | 15.7 | 16.7 | 16.2 | guide |
| NightElf | Priest | 20.1 | 17.9 | 17.6 | 16.5 | plan |
| NightElf | Rogue | 18.1 | 15.7 | 14.8 | 14.4 | plan |
| NightElf | Warrior | 17.3 | 14.7 | 14.9 | 14.1 | plan |
| Orc | Hunter | 18.4 | 18.0 | 15.6 | 15.3 | plan |
| Orc | Mage | 14.8 | 13.4 | 15.1 | 14.9 | guide |
| Orc | Rogue | 14.3 | 12.6 | 14.3 | 14.6 | guide |
| Orc | Shaman | 17.7 | 16.6 | 13.6 | 13.4 | plan |
| Orc | Warlock | 13.9 | 13.1 | 14.2 | 13.8 | guide |
| Orc | Warrior | 13.7 | 12.2 | 13.5 | 13.4 | guide |
| SkyborneA | Druid | 20.1 | 17.1 | 16.9 | 15.9 | plan |
| SkyborneA | Hunter | 17.6 | 15.9 | 16.5 | 16.2 | guide |
| SkyborneA | Mage | 18.5 | 16.8 | 17.3 | 16.0 | plan |
| SkyborneA | Rogue | 17.7 | 14.5 | 14.2 | 14.2 | plan |
| SkyborneA | Warrior | 17.2 | 14.6 | 14.5 | 13.9 | plan |
| SkyborneH | Druid | 16.6 | 14.7 | 16.1 | 15.8 | guide |
| SkyborneH | Hunter | 15.9 | 15.4 | 15.4 | 15.2 | plan |
| SkyborneH | Rogue | 15.3 | 12.9 | 14.0 | 14.0 | guide |
| SkyborneH | Shaman | 15.6 | 12.8 | 14.3 | 14.1 | guide |
| SkyborneH | Warrior | 14.8 | 13.4 | 14.3 | 14.0 | guide |
| Tauren | Druid | 26.3 | 15.2 | 16.0 | 15.3 | guide |
| Tauren | Hunter | 24.2 | 14.5 | 15.0 | 15.0 | guide |
| Tauren | Shaman | 24.5 | 13.6 | 14.6 | 14.6 | guide |
| Tauren | Warrior | 24.0 | 13.8 | 13.5 | 13.5 | plan |
| Troll | Hunter | 18.4 | 18.0 | 15.6 | 15.3 | plan |
| Troll | Mage | 14.8 | 13.4 | 14.6 | 14.4 | guide |
| Troll | Priest | 15.3 | 13.8 | 15.5 | 15.3 | guide |
| Troll | Rogue | 14.1 | 12.6 | 14.3 | 14.6 | guide |
| Troll | Shaman | 17.7 | 16.6 | 13.6 | 13.4 | plan |
| Troll | Warlock | 13.9 | 13.1 | 14.2 | 13.8 | guide |
| Troll | Warrior | 13.7 | 12.2 | 13.5 | 13.4 | guide |
| Undead | Mage | 14.7 | 13.2 | 16.4 | 15.8 | guide |
| Undead | Paladin | 14.4 | 13.1 | 14.3 | 13.7 | guide |
| Undead | Priest | 14.6 | 13.5 | 15.8 | 15.7 | guide |
| Undead | Rogue | 13.7 | 13.0 | 14.1 | 14.2 | guide |
| Undead | Warlock | 14.0 | 13.5 | 13.2 | 12.9 | plan |
| Undead | Warrior | 13.2 | 12.0 | 13.5 | 13.5 | guide |

Average: RestedXP 17.04 h, kept route 14.19 h (16.7 % faster), 56 combinations.

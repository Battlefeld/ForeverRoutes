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

Options: besides the class chains, Forever goals worth a detour are options too (`extraOptions`): the Cozy Sleeping Bag
chain from level 14. An option's `events` are the route with it, woven in place by place (a turn-in and the next quest
taken there are one visit), each visit where it costs the route least within a few levels after it can be taken,
measured with the rest of the chain straight after it (Barrens steps while the route is in the Barrens, Eastern
Kingdoms steps when it is there), and finished before level 20. The addon shows them with `/fr optionen` and switches one on with
`/fr option <number>`: its steps go into the route after the step they follow, the rest stays as it is (a grind step
is done at its level, so the XP of an option shortens the grinding by itself).

Wands: mages, priests and warlocks finish their mobs with a wand from when they have one (`route/wand.go`; no mana,
fewer rests). Enchanting trainers and suppliers stand in the capitals: from level 4 (the money for the essence) the
route goes there where it adds least to its way, either when a capital lies on its way (leaving the start area) or
back and forth with the hearthstone home, whichever reaches level 20 sooner (`wandDetour` in the route file: the most
the visit may add). One visit crafts both: the Novice's Practice Wand (new in
Forever: Enchanting 1, Simple Wood and a Mote of Magic from the supplier), worn until level 5, and the Lesser Magic
Wand (Enchanting 10 with practice wands, a Lesser Magic Essence from the auction house), worn from level 5. A better wand from a quest on the way takes their place.
The addon shows the crafting as a step of its own.

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
| Dwarf | Hunter | 19.7 | 16.6 | 17.0 | 17.0 | guide |
| Dwarf | Paladin | 18.3 | 15.7 | 14.9 | 14.9 | plan |
| Dwarf | Priest | 20.5 | 16.2 | 18.2 | 16.9 | guide |
| Dwarf | Rogue | 16.1 | 14.0 | 13.3 | 13.2 | plan |
| Dwarf | Shaman | 19.3 | 17.5 | 15.5 | 17.2 | guide |
| Dwarf | Warrior | 16.0 | 14.0 | 14.2 | 14.5 | guide |
| Gnome | Mage | 20.2 | 16.2 | 17.6 | 16.4 | guide |
| Gnome | Priest | 20.5 | 16.4 | 17.3 | 17.0 | guide |
| Gnome | Rogue | 16.1 | 14.0 | 13.3 | 13.2 | plan |
| Gnome | Warlock | 19.9 | 15.4 | 16.6 | 15.9 | guide |
| Gnome | Warrior | 16.0 | 14.0 | 14.2 | 14.7 | guide |
| Human | Hunter | 22.2 | 17.0 | 16.0 | 15.9 | plan |
| Human | Mage | 18.8 | 16.0 | 16.9 | 16.1 | guide |
| Human | Paladin | 19.5 | 15.2 | 15.6 | 14.9 | guide |
| Human | Priest | 18.9 | 16.1 | 16.8 | 16.2 | guide |
| Human | Rogue | 15.7 | 13.9 | 13.4 | 13.4 | plan |
| Human | Warlock | 20.9 | 14.9 | 15.2 | 14.6 | guide |
| Human | Warrior | 15.2 | 13.5 | 13.9 | 13.6 | guide |
| NightElf | Druid | 20.9 | 18.5 | 17.7 | 18.4 | guide |
| NightElf | Hunter | 19.0 | 16.9 | 17.5 | 17.7 | guide |
| NightElf | Priest | 17.8 | 16.5 | 17.6 | 16.8 | guide |
| NightElf | Rogue | 15.8 | 14.4 | 13.8 | 13.8 | plan |
| NightElf | Warrior | 15.9 | 14.2 | 14.5 | 14.3 | guide |
| Orc | Hunter | 19.1 | 18.6 | 15.8 | 16.4 | plan |
| Orc | Mage | 15.3 | 14.1 | 15.2 | 14.8 | guide |
| Orc | Rogue | 13.5 | 12.8 | 12.2 | 12.8 | guide |
| Orc | Shaman | 17.4 | 17.2 | 14.8 | 16.1 | plan |
| Orc | Warlock | 14.4 | 13.4 | 14.7 | 14.4 | guide |
| Orc | Warrior | 12.9 | 11.9 | 12.6 | 13.7 | guide |
| SkyborneA | Druid | 22.7 | 19.7 | 18.3 | 17.7 | guide |
| SkyborneA | Hunter | 21.2 | 17.9 | 17.2 | 17.7 | guide |
| SkyborneA | Mage | 22.1 | 17.4 | 17.8 | 16.0 | plan |
| SkyborneA | Rogue | 17.5 | 16.2 | 13.1 | 13.3 | plan |
| SkyborneA | Warrior | 16.9 | 15.3 | 15.1 | 14.7 | plan |
| SkyborneH | Druid | 20.9 | 15.5 | 16.3 | 17.4 | guide |
| SkyborneH | Hunter | 19.8 | 16.5 | 16.0 | 16.9 | guide |
| SkyborneH | Rogue | 14.8 | 13.5 | 13.0 | 14.6 | guide |
| SkyborneH | Shaman | 18.2 | 16.2 | 17.2 | 18.1 | guide |
| SkyborneH | Warrior | 14.7 | 12.8 | 13.1 | 13.5 | guide |
| Tauren | Druid | 19.7 | 18.0 | 15.6 | 18.0 | guide |
| Tauren | Hunter | 18.8 | 15.4 | 15.3 | 16.0 | guide |
| Tauren | Shaman | 18.5 | 16.6 | 15.3 | 15.8 | plan |
| Tauren | Warrior | 16.1 | 14.1 | 12.6 | 13.0 | plan |
| Troll | Hunter | 19.1 | 18.6 | 15.8 | 16.4 | plan |
| Troll | Mage | 15.3 | 14.1 | 15.5 | 14.9 | guide |
| Troll | Priest | 15.8 | 14.3 | 16.0 | 15.6 | guide |
| Troll | Rogue | 13.3 | 12.9 | 12.2 | 12.8 | guide |
| Troll | Shaman | 17.4 | 17.2 | 14.8 | 16.1 | plan |
| Troll | Warlock | 14.4 | 13.4 | 14.7 | 14.5 | guide |
| Troll | Warrior | 12.9 | 11.9 | 12.6 | 13.7 | guide |
| Undead | Mage | 14.9 | 13.8 | 17.0 | 16.2 | guide |
| Undead | Paladin | 14.4 | 13.4 | 14.9 | 15.0 | guide |
| Undead | Priest | 14.9 | 13.9 | 17.6 | 16.2 | guide |
| Undead | Rogue | 13.2 | 12.8 | 12.3 | 13.5 | guide |
| Undead | Warlock | 14.5 | 14.9 | 15.3 | 17.4 | guide |
| Undead | Warrior | 12.6 | 11.9 | 12.0 | 12.3 | guide |

Average over the 56 combinations: RestedXP 17.33 h, the kept route 14.91 h (13.9 % faster). A planned route stays at home until the first class quests are done and crosses between Kalimdor and the Eastern Kingdoms at most twice up to level 20.

## Solo with dungeons

`<race>-<class>-dungeons.json` ("Solo mit Dungeon"): the same, with dungeon runs (a group only for the run itself) (`foreverroutes improve -dungeons`). The route takes a
dungeon's quests on the way (from three levels below it, within ten minutes), runs it with a group and hands the quests
in, across the sea if need be (Ragefire's quest from Undercity). The addon shows it after `/fr dungeon` (`/fr solo` goes back to the route without dungeons, which takes no dungeon quests). A druid or warrior gets its bear form or defensive stance before the first run (it finds a group as a tank). Run times are
estimates (`data/sources/dungeons.json`); better gear from the runs is not counted yet, so the routes with dungeons are rather
faster in game than here.

Average over the 56 combinations: 13.31 h to level 20 (without dungeons 14.91 h). Where the dungeons do not pay (Skyborne Horde druid: its bear form before the first run costs more than the runs bring), the route with dungeons is the one without.

| Race | Class | Hours | Kept |
|---|---|---|---|
| Dwarf | Hunter | 13.9 | guide |
| Dwarf | Paladin | 13.5 | plan |
| Dwarf | Priest | 13.3 | guide |
| Dwarf | Rogue | 11.7 | plan |
| Dwarf | Shaman | 15.5 | guide |
| Dwarf | Warrior | 12.8 | plan |
| Gnome | Mage | 14.3 | plan |
| Gnome | Priest | 13.8 | guide |
| Gnome | Rogue | 11.7 | plan |
| Gnome | Warlock | 13.3 | guide |
| Gnome | Warrior | 12.8 | plan |
| Human | Hunter | 14.3 | plan |
| Human | Mage | 14.4 | guide |
| Human | Paladin | 12.6 | plan |
| Human | Priest | 12.6 | plan |
| Human | Rogue | 12.1 | guide |
| Human | Warlock | 12.8 | plan |
| Human | Warrior | 11.9 | plan |
| NightElf | Druid | 15.9 | plan |
| NightElf | Hunter | 14.3 | plan |
| NightElf | Priest | 13.9 | plan |
| NightElf | Rogue | 12.9 | guide |
| NightElf | Warrior | 12.5 | guide |
| Orc | Hunter | 13.8 | plan |
| Orc | Mage | 14.0 | guide |
| Orc | Rogue | 12.8 | guide |
| Orc | Shaman | 13.3 | plan |
| Orc | Warlock | 12.2 | plan |
| Orc | Warrior | 11.3 | guide |
| SkyborneA | Druid | 13.9 | plan |
| SkyborneA | Hunter | 15.6 | guide |
| SkyborneA | Mage | 15.6 | plan |
| SkyborneA | Rogue | 12.1 | plan |
| SkyborneA | Warrior | 13.3 | plan |
| SkyborneH | Druid | 15.5 | guide |
| SkyborneH | Hunter | 14.7 | guide |
| SkyborneH | Rogue | 12.9 | plan |
| SkyborneH | Shaman | 14.5 | guide |
| SkyborneH | Warrior | 11.7 | plan |
| Tauren | Druid | 15.7 | plan |
| Tauren | Hunter | 15.4 | guide |
| Tauren | Shaman | 13.0 | plan |
| Tauren | Warrior | 11.3 | plan |
| Troll | Hunter | 13.8 | plan |
| Troll | Mage | 14.0 | guide |
| Troll | Priest | 12.4 | plan |
| Troll | Rogue | 12.9 | guide |
| Troll | Shaman | 13.3 | plan |
| Troll | Warlock | 12.2 | plan |
| Troll | Warrior | 11.3 | guide |
| Undead | Mage | 13.8 | guide |
| Undead | Paladin | 12.4 | plan |
| Undead | Priest | 13.2 | plan |
| Undead | Rogue | 11.7 | plan |
| Undead | Warlock | 13.8 | plan |
| Undead | Warrior | 11.2 | plan |

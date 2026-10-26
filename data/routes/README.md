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

Camping (`route/camp.go`): every character learns Skinning, Cooking and Leatherworking on the way (casters at the
capital they fetch their wands in: Enchanting for the wands, then unlearned for Leatherworking). Skinning the beasts
it kills brings Leatherworking to 20 for the Camp Tent: from level 10, once the hour's cooldown is over and the rested experience is
used up (with some left the tent raises nothing), a camp (fire, tent, a minute's rest) raises rested experience to 5 % of a level (when it is lower; the Leatherworking camp buff is "Well Rested
Shelter"), which doubles kill XP until used up. Up to level 20 this costs
about 10 minutes on average (skinning, trainers, camps); it pays from level 20 on, where a level takes hours: over
20–60 an estimated 5 to 12 hours (Classic XP table, 80–110 hours of guided leveling). The addon follows the camp
buff: 10 minutes before it ends it asks for leather for the next tent, 5 minutes before it says what the camp lacks,
then it says to make camp, all only once the rested experience is (nearly) used up (`/fr lager` marks a camp).

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
| Dwarf | Hunter | 19.2 | 16.1 | 16.3 | 15.7 | guide |
| Dwarf | Paladin | 20.3 | 15.0 | 14.9 | 14.7 | guide |
| Dwarf | Priest | 20.1 | 16.1 | 16.2 | 15.6 | guide |
| Dwarf | Rogue | 15.6 | 13.7 | 13.2 | 12.9 | plan |
| Dwarf | Shaman | 18.9 | 17.9 | 15.1 | 17.2 | plan |
| Dwarf | Warrior | 15.5 | 14.4 | 13.9 | 13.6 | plan |
| Gnome | Mage | 19.8 | 16.2 | 17.4 | 15.9 | guide |
| Gnome | Priest | 20.1 | 16.6 | 17.5 | 16.4 | guide |
| Gnome | Rogue | 15.6 | 13.7 | 13.2 | 12.9 | plan |
| Gnome | Warlock | 19.5 | 15.2 | 14.9 | 14.5 | plan |
| Gnome | Warrior | 15.5 | 14.4 | 13.9 | 13.6 | plan |
| Human | Hunter | 21.7 | 16.4 | 17.4 | 16.8 | guide |
| Human | Mage | 21.1 | 15.6 | 17.3 | 15.8 | guide |
| Human | Paladin | 22.2 | 15.1 | 14.8 | 14.6 | guide |
| Human | Priest | 21.3 | 15.5 | 17.0 | 15.7 | guide |
| Human | Rogue | 20.9 | 14.2 | 12.2 | 12.4 | plan |
| Human | Warlock | 20.2 | 14.9 | 15.4 | 14.7 | guide |
| Human | Warrior | 19.8 | 13.3 | 12.9 | 13.1 | guide |
| NightElf | Druid | 20.3 | 18.5 | 17.0 | 18.6 | guide |
| NightElf | Hunter | 18.5 | 16.1 | 16.8 | 16.4 | guide |
| NightElf | Priest | 17.2 | 15.8 | 16.7 | 16.1 | guide |
| NightElf | Rogue | 15.5 | 14.1 | 14.0 | 13.6 | plan |
| NightElf | Warrior | 15.7 | 13.7 | 13.8 | 14.0 | guide |
| Orc | Hunter | 18.6 | 18.1 | 15.8 | 15.5 | plan |
| Orc | Mage | 15.5 | 13.7 | 15.7 | 15.1 | guide |
| Orc | Rogue | 13.4 | 12.8 | 12.4 | 13.2 | guide |
| Orc | Shaman | 18.0 | 16.7 | 14.8 | 14.5 | plan |
| Orc | Warlock | 14.5 | 13.3 | 14.1 | 14.1 | guide |
| Orc | Warrior | 13.0 | 11.9 | 13.1 | 13.3 | guide |
| SkyborneA | Druid | 23.1 | 19.6 | 16.0 | 16.5 | plan |
| SkyborneA | Hunter | 22.8 | 18.3 | 16.3 | 16.8 | plan |
| SkyborneA | Mage | 22.1 | 18.7 | 17.8 | 17.0 | plan |
| SkyborneA | Rogue | 17.2 | 15.7 | 14.7 | 14.1 | plan |
| SkyborneA | Warrior | 16.7 | 15.5 | 14.9 | 14.7 | plan |
| SkyborneH | Druid | 21.1 | 16.1 | 16.2 | 16.8 | guide |
| SkyborneH | Hunter | 19.3 | 15.9 | 16.5 | 17.2 | guide |
| SkyborneH | Rogue | 15.4 | 14.1 | 12.9 | 13.9 | guide |
| SkyborneH | Shaman | 18.8 | 16.5 | 15.2 | 15.2 | plan |
| SkyborneH | Warrior | 15.3 | 13.6 | 13.5 | 13.6 | guide |
| Tauren | Druid | 20.1 | 18.5 | 16.1 | 17.6 | plan |
| Tauren | Hunter | 18.4 | 15.4 | 16.0 | 15.6 | guide |
| Tauren | Shaman | 18.3 | 16.8 | 14.9 | 15.2 | plan |
| Tauren | Warrior | 16.8 | 14.1 | 12.4 | 12.8 | plan |
| Troll | Hunter | 18.6 | 18.1 | 15.8 | 15.5 | plan |
| Troll | Mage | 15.5 | 13.7 | 15.7 | 15.1 | guide |
| Troll | Priest | 16.1 | 14.1 | 15.5 | 15.1 | guide |
| Troll | Rogue | 13.2 | 12.9 | 12.3 | 13.0 | guide |
| Troll | Shaman | 18.0 | 16.7 | 14.8 | 14.5 | plan |
| Troll | Warlock | 14.5 | 13.3 | 14.1 | 14.1 | guide |
| Troll | Warrior | 13.0 | 11.9 | 13.1 | 13.3 | guide |
| Undead | Mage | 14.9 | 13.7 | 16.3 | 15.9 | guide |
| Undead | Paladin | 14.6 | 13.0 | 14.1 | 13.9 | guide |
| Undead | Priest | 14.9 | 13.9 | 15.8 | 15.6 | guide |
| Undead | Rogue | 13.3 | 12.2 | 11.5 | 12.4 | guide |
| Undead | Warlock | 13.9 | 14.1 | 14.7 | 16.4 | guide |
| Undead | Warrior | 12.8 | 12.0 | 12.6 | 12.6 | guide |

Average over the 56 combinations: RestedXP 17.61 h, the kept route 14.59 h (17.2 % faster). A planned route stays at home until the first class quests are done and crosses between Kalimdor and the Eastern Kingdoms at most twice up to level 20.

## Solo with dungeons

`<race>-<class>-dungeons.json` ("Solo mit Dungeon"): the same, with dungeon runs (a group only for the run itself) (`foreverroutes improve -dungeons`). Each dungeon is tried at every step of the route
that has its level (and the solo route is a start too, `-seed`); the route takes a
dungeon's quests on the way (from three levels below it, within ten minutes), runs it with a group and hands the quests
in, across the sea if need be (Ragefire's quest from Undercity). The addon shows it after `/fr dungeon` (`/fr solo` goes back to the route without dungeons, which takes no dungeon quests). A druid or warrior gets its bear form or defensive stance before the first run (it finds a group as a tank). Run times are
estimates (`data/sources/dungeons.json`); better gear from the runs is not counted yet, so the routes with dungeons are rather
faster in game than here.

Average over the 56 combinations: 13.17 h to level 20 (without dungeons 14.59 h). Where the dungeons do not pay (the runs on the way cost more than they bring), the route with dungeons is the one without.

| Race | Class | Hours | Kept |
|---|---|---|---|
| Dwarf | Hunter | 14.3 | guide |
| Dwarf | Paladin | 13.0 | plan |
| Dwarf | Priest | 14.4 | guide |
| Dwarf | Rogue | 12.9 | plan |
| Dwarf | Shaman | 15.9 | plan |
| Dwarf | Warrior | 12.7 | plan |
| Gnome | Mage | 14.5 | plan |
| Gnome | Priest | 14.5 | plan |
| Gnome | Rogue | 12.9 | plan |
| Gnome | Warlock | 13.9 | plan |
| Gnome | Warrior | 12.7 | plan |
| Human | Hunter | 13.6 | plan |
| Human | Mage | 14.7 | guide |
| Human | Paladin | 12.8 | plan |
| Human | Priest | 13.8 | guide |
| Human | Rogue | 12.4 | plan |
| Human | Warlock | 13.6 | plan |
| Human | Warrior | 12.9 | plan |
| NightElf | Druid | 15.2 | plan |
| NightElf | Hunter | 14.9 | guide |
| NightElf | Priest | 14.3 | guide |
| NightElf | Rogue | 13.6 | plan |
| NightElf | Warrior | 12.9 | guide |
| Orc | Hunter | 12.3 | plan |
| Orc | Mage | 12.6 | guide |
| Orc | Rogue | 12.1 | guide |
| Orc | Shaman | 12.6 | plan |
| Orc | Warlock | 11.8 | plan |
| Orc | Warrior | 11.1 | plan |
| SkyborneA | Druid | 15.2 | plan |
| SkyborneA | Hunter | 15.9 | plan |
| SkyborneA | Mage | 15.9 | plan |
| SkyborneA | Rogue | 13.9 | plan |
| SkyborneA | Warrior | 13.6 | plan |
| SkyborneH | Druid | 13.8 | plan |
| SkyborneH | Hunter | 14.0 | plan |
| SkyborneH | Rogue | 12.2 | plan |
| SkyborneH | Shaman | 13.1 | plan |
| SkyborneH | Warrior | 11.7 | plan |
| Tauren | Druid | 14.6 | plan |
| Tauren | Hunter | 13.0 | plan |
| Tauren | Shaman | 12.9 | plan |
| Tauren | Warrior | 11.4 | plan |
| Troll | Hunter | 12.3 | plan |
| Troll | Mage | 12.6 | guide |
| Troll | Priest | 12.2 | plan |
| Troll | Rogue | 12.0 | plan |
| Troll | Shaman | 12.6 | plan |
| Troll | Warlock | 11.8 | plan |
| Troll | Warrior | 11.1 | plan |
| Undead | Mage | 13.0 | guide |
| Undead | Paladin | 11.8 | guide |
| Undead | Priest | 12.4 | guide |
| Undead | Rogue | 11.6 | plan |
| Undead | Warlock | 13.2 | plan |
| Undead | Warrior | 10.9 | guide |

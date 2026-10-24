local _,ns=...

-- 1-30 dungeon associations reviewed against current WoW Forever dungeon quest lists.
-- ATTMetadata already carries the classic/Forever associations for RFC, WC, DM, SFK,
-- BFD, Stockades, Gnomeregan, RFK, Hall of Thanes and Ruins of Lordaeron.
-- This file fills current beta gaps and the level-30 boundary content.
local function tag(id,key,role)
 ns.MergeQuest({id=id,dungeon=key,dungeonRole=role or 'QUEST',dungeonReviewed='2026-10-04'})
end

-- Excavation Site: Wetlands (26-31). Current beta dungeon quests.
for _,id in ipairs({95697,98823,95664,95795,95646,95810,95682,98824}) do tag(id,'excavation') end

-- Razorfen Downs becomes available in the level-30 beta, but only the quests obtainable
-- at/before 30 are relevant to this build. An Unholy Alliance starts from RFK.
tag(6522,'rfk','CHAIN')
tag(6521,'rfd','CHAIN')
tag(6626,'rfd') -- A Host of Evil, min 28

-- Scarlet Monastery: only quests whose prerequisite/start can matter by level 30.
tag(1051,'sm')  -- Vorrel's Revenge (starts inside Graveyard)
tag(1113,'sm','CHAIN') -- Hearts of Zeal, min 30; requires RFK guano chain
tag(1049,'sm')  -- Compendium of the Fallen, min 28
tag(1050,'sm')  -- Mythology of the Titans, min 28
tag(6628,'sm','CHAIN') -- Test of Lore chain, min 25; later dungeon step follows

-- Deadmines new Forever destruction chain: the actual dungeon step is 92753;
-- 92819 is its immediate outside follow-up and stays attached to the package.
tag(92753,'dm','QUEST')
tag(92819,'dm','FOLLOWUP')

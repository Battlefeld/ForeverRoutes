package route

import (
	"math"
	"sync"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Zones: the dataset has no zone of a place, so a place is named by the nearest of a few known anchors (inns, flight
// masters and the first camps of the start zones) on its continent. Near a border the name can be the neighbour's;
// it names the legs of a plan and keeps a run in its area, it is not a map.

// zoneAnchors: NPC (innkeeper, flight master or a known camp NPC) -> zone; capitals by their city.
var zoneAnchors = map[int]string{
	// Eastern Kingdoms: inns
	295: "Elwynn Forest", 1247: "Dun Morogh", 1464: "Wetlands", 2352: "Hillsbrad Foothills", 2388: "Hillsbrad Foothills",
	5111: "Ironforge", 5688: "Tirisfal Glades", 5814: "Stranglethorn Vale", 6727: "Redridge Mountains", 6734: "Loch Modan",
	6739: "Silverpine Forest", 6740: "Stormwind", 6741: "Undercity", 6790: "Duskwood", 6807: "Stranglethorn Vale",
	6930: "Swamp of Sorrows", 7744: "The Hinterlands", 8931: "Westfall", 9356: "Badlands", 9501: "Arathi Highlands",
	14731: "The Hinterlands", 16256: "Eastern Plaguelands",
	// Eastern Kingdoms: flight masters
	352: "Stormwind", 523: "Westfall", 931: "Redridge Mountains", 1387: "Stranglethorn Vale", 1571: "Wetlands",
	1572: "Loch Modan", 1573: "Ironforge", 2226: "Silverpine Forest", 2299: "Burning Steppes", 2389: "Hillsbrad Foothills",
	2409: "Duskwood", 2432: "Hillsbrad Foothills", 2835: "Arathi Highlands", 2851: "Arathi Highlands", 2858: "Stranglethorn Vale",
	2859: "Stranglethorn Vale", 2861: "Badlands", 2941: "Searing Gorge", 3305: "Searing Gorge", 4314: "The Hinterlands",
	4551: "Undercity", 6026: "Swamp of Sorrows", 8018: "The Hinterlands", 8609: "Blasted Lands", 12596: "Western Plaguelands",
	12617: "Eastern Plaguelands", 12636: "Eastern Plaguelands", 13177: "Burning Steppes",
	1740: "Tirisfal Glades", 1938: "Silverpine Forest", // Deathknell; the Sepulcher
	// Kalimdor: inns
	3934: "The Barrens", 6272: "Dustwallow Marsh", 6735: "Darnassus", 6736: "Teldrassil", 6737: "Darkshore", 6738: "Ashenvale",
	6746: "Thunder Bluff", 6747: "Mulgore", 6791: "The Barrens", 6928: "Durotar", 6929: "Orgrimmar", 7714: "The Barrens",
	7731: "Stonetalon Mountains", 7733: "Tanaris", 7736: "Feralas", 7737: "Feralas", 11103: "Desolace", 11106: "Desolace",
	11116: "Thousand Needles", 11118: "Winterspring", 12196: "Ashenvale", 15174: "Silithus", 16458: "Stonetalon Mountains",
	// Kalimdor: camps far from an inn or flight master (the Mirage Raceway on the Shimmering Flats is Thousand Needles,
	// nearer to Gadgetzan than to Freewind Post)
	4452: "Thousand Needles", 4454: "Thousand Needles",
	// Kalimdor: flight masters
	2995: "Thunder Bluff", 3310: "Orgrimmar", 3615: "The Barrens", 3838: "Teldrassil", 3841: "Darkshore", 4267: "Ashenvale",
	4312: "Stonetalon Mountains", 4317: "Thousand Needles", 4319: "Feralas", 4321: "Dustwallow Marsh", 4407: "Stonetalon Mountains",
	6706: "Desolace", 6726: "Desolace", 7823: "Tanaris", 7824: "Tanaris", 8019: "Feralas", 8020: "Feralas", 8610: "Azshara",
	10378: "The Barrens", 10897: "Moonglade", 11138: "Winterspring", 11139: "Winterspring", 11899: "Dustwallow Marsh",
	11900: "Felwood", 11901: "Ashenvale", 12577: "Azshara", 12578: "Felwood", 12616: "Ashenvale", 12740: "Moonglade",
	15177: "Silithus", 15178: "Silithus", 16227: "The Barrens", 10583: "Un'Goro Crater",
	10176: "Durotar", 3304: "Durotar", 2980: "Mulgore", 3337: "The Barrens", 5878: "The Barrens", // Valley of Trials, Sen'jin, Camp Narache, Far Watch, the Crossroads' east
}

type zoneAnchor struct {
	at   dataset.Point
	zone string
}

// capitalRadius: a capital's name holds within this distance of its anchors; beyond it the zone around it.
const capitalRadius = 450.0

var capitals = map[string]string{"Orgrimmar": "Durotar", "Thunder Bluff": "Mulgore", "Undercity": "Tirisfal Glades",
	"Stormwind": "Elwynn Forest", "Ironforge": "Dun Morogh", "Darnassus": "Teldrassil"}

func (w *World) anchors() []zoneAnchor {
	w.anchorOnce.Do(func() {
		seen := map[int]bool{}
		add := func(id int, p dataset.Point) {
			if z, ok := zoneAnchors[id]; ok && !seen[id] {
				seen[id] = true
				w.zoneAnchors = append(w.zoneAnchors, zoneAnchor{p, z})
			}
		}
		t := w.Data.Travel
		for _, list := range [][]dataset.Node{t.Inns, t.FlightMasters} {
			for _, n := range list {
				add(n.NPC, n.At)
			}
		}
		for i := range w.Data.NPCs {
			if n := &w.Data.NPCs[i]; len(n.Spawns) > 0 {
				add(n.ID, n.Spawns[0])
			}
		}
	})
	return w.zoneAnchors
}

// zoneOf names the zone of a place.
func (w *World) zoneOf(p dataset.Point) string {
	w.zoneOnce.Do(func() { w.zoneCache = &sync.Map{} })
	key := [3]int{p.Map, int(math.Floor(p.X / 25)), int(math.Floor(p.Y / 25))}
	if z, ok := w.zoneCache.Load(key); ok {
		return z.(string)
	}
	zone, bd, around := "", math.Inf(1), ""
	for _, a := range w.anchors() {
		if a.at.Map != p.Map {
			continue
		}
		if d := math.Hypot(a.at.X-p.X, a.at.Y-p.Y); d < bd {
			zone, bd = a.zone, d
		}
	}
	if z, ok := capitals[zone]; ok && bd > capitalRadius {
		around = z
	}
	if around != "" {
		zone = around
	}
	w.zoneCache.Store(key, zone)
	return zone
}

// ZoneName is the zone of a place (zoneOf) for reports.
func (w *World) ZoneName(p dataset.Point) string { return w.zoneOf(p) }

package route

import (
 "fmt"
 "math"
 "sort"
)

// MageWaterRank is a verified, level-gated conjured-water item.
// RestoreMana is the total mana restored over DurationSeconds.
// It is a universal planning reference, regardless of class or inventory.
type MageWaterRank struct {
 ItemID int
 RequiredLevel int
 RestoreMana float64
 DurationSeconds float64
}

// BestMageWater chooses the highest-level usable rank; item IDs resolve ties.
// Missing data is an error rather than an invented water value.
func BestMageWater(level int, ranks []MageWaterRank) (MageWaterRank, error) {
 var eligible []MageWaterRank
 for _, r := range ranks {
  if r.ItemID <= 0 || r.RequiredLevel < 1 || r.DurationSeconds <= 0 || r.RestoreMana <= 0 ||
   math.IsNaN(r.RestoreMana) || math.IsInf(r.RestoreMana, 0) ||
   math.IsNaN(r.DurationSeconds) || math.IsInf(r.DurationSeconds, 0) {
   return MageWaterRank{}, fmt.Errorf("invalid mage water rank")
  }
  if r.RequiredLevel <= level { eligible=append(eligible,r) }
 }
 if len(eligible)==0 { return MageWaterRank{},fmt.Errorf("no verified mage water for level %d",level) }
 sort.Slice(eligible,func(i,j int) bool {
  if eligible[i].RequiredLevel!=eligible[j].RequiredLevel {return eligible[i].RequiredLevel>eligible[j].RequiredLevel}
  if eligible[i].RestoreMana!=eligible[j].RestoreMana {return eligible[i].RestoreMana>eligible[j].RestoreMana}
  return eligible[i].ItemID<eligible[j].ItemID
 })
 return eligible[0],nil
}

// WithMageWater sets the same drink reference for all mana-using classes.
// Does not assume the character owns the water.
func WithMageWater(p ManaChain, level int, ranks []MageWaterRank) (ManaChain,error) {
 r,err:=BestMageWater(level,ranks)
 if err!=nil{return p,err}
 p.DrinkManaPerSecond=r.RestoreMana/r.DurationSeconds
 p.DrinkTargetFraction=1
 return p,nil
}

// ClassUsesMana reports the primary resource of the standard playable classes.
// Druids use mana in caster form; shapeshifted energy/rage is handled separately.
// Hunters use mana in the Classic/Forever ruleset.
func ClassUsesMana(class string) bool {
 switch class {
 case "MAGE", "PRIEST", "WARLOCK", "SHAMAN", "PALADIN", "DRUID", "HUNTER":
  return true
 default:
  return false
 }
}

// WithMageWaterForClass leaves non-mana classes unchanged.
// Actual power type must additionally be checked by the live bridge.
func WithMageWaterForClass(p ManaChain, class string, level int, ranks []MageWaterRank) (ManaChain,error) {
 if !ClassUsesMana(class) { return p,nil }
 return WithMageWater(p,level,ranks)
}

package route

import (
 "fmt"
 "math"
)

// ManaChain models consecutive kills with explicit mana consumption and recovery.
// All rates are mana per second. The caller must supply measured or simulated
// spell costs; no class-specific values are assumed here.
type ManaChain struct {
 Current, Maximum float64
 CostPerKill float64
 FightSeconds float64
 RegenInCombat, RegenOutOfCombat float64
 TravelBetweenKills float64
 DrinkManaPerSecond float64
 DrinkSetupSeconds float64
 DrinkTargetFraction float64
}

type ManaChainResult struct {
 Seconds, Fighting, Traveling, Drinking, Waiting float64
 Drinks int
 EndMana float64
}

func SimulateManaChain(p ManaChain, kills int) (ManaChainResult, error) {
 var out ManaChainResult
 valid := func(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) && x >= 0 }
 for _, v := range []float64{p.Current,p.Maximum,p.CostPerKill,p.FightSeconds,p.RegenInCombat,p.RegenOutOfCombat,p.TravelBetweenKills,p.DrinkManaPerSecond,p.DrinkSetupSeconds,p.DrinkTargetFraction} {
  if !valid(v) { return out, fmt.Errorf("invalid mana chain parameter") }
 }
 if kills < 0 || p.Maximum <= 0 || p.Current > p.Maximum || p.DrinkTargetFraction > 1 {
  return out, fmt.Errorf("invalid mana chain bounds")
 }
 if p.CostPerKill > p.Maximum + p.FightSeconds*p.RegenInCombat {
  return out, fmt.Errorf("one kill requires more mana than can be supplied")
 }
 mana := p.Current
 for i:=0;i<kills;i++ {
  if i>0 {
   out.Traveling += p.TravelBetweenKills
   mana=math.Min(p.Maximum,mana+p.TravelBetweenKills*p.RegenOutOfCombat)
  }
  required:=math.Max(0,p.CostPerKill-p.FightSeconds*p.RegenInCombat)
  if mana < required {
   target:=math.Max(required,p.Maximum*p.DrinkTargetFraction)
   target=math.Min(p.Maximum,target)
   if p.DrinkManaPerSecond>0 {
    out.Drinks++
    out.Drinking+=p.DrinkSetupSeconds
    recovery:=(target-mana)/p.DrinkManaPerSecond
    out.Drinking+=recovery
    mana=target
   } else if p.RegenOutOfCombat>0 {
    recovery:=(required-mana)/p.RegenOutOfCombat
    out.Waiting+=recovery
    mana=required
   } else {
    return ManaChainResult{},fmt.Errorf("insufficient mana and no recovery available")
   }
  }
  mana=math.Min(p.Maximum,math.Max(0,mana-required))
  out.Fighting+=p.FightSeconds
 }
 out.Seconds=out.Fighting+out.Traveling+out.Drinking+out.Waiting
 out.EndMana=mana
 return out,nil
}

package route

import (
 "math"
 "testing"
)

func TestManaChainNoRest(t *testing.T) {
 p:=ManaChain{Current:100,Maximum:100,CostPerKill:10,FightSeconds:2}
 r,e:=SimulateManaChain(p,5)
 if e!=nil || r.Seconds!=10 || r.Drinks!=0 || r.EndMana!=50 {t.Fatalf("got %+v %v",r,e)}
}
func TestManaChainDrink(t *testing.T) {
 p:=ManaChain{Current:100,Maximum:100,CostPerKill:20,FightSeconds:2,DrinkManaPerSecond:10,DrinkTargetFraction:1,DrinkSetupSeconds:2}
 r,e:=SimulateManaChain(p,10)
 if e!=nil || r.Drinks!=1 || math.Abs(r.Seconds-32)>1e-9 {t.Fatalf("got %+v %v",r,e)}
}
func TestManaChainTravelRegen(t *testing.T) {
 p:=ManaChain{Current:10,Maximum:100,CostPerKill:10,FightSeconds:2,TravelBetweenKills:5,RegenOutOfCombat:2}
 r,e:=SimulateManaChain(p,2)
 if e!=nil || r.Seconds!=9 || r.Drinks!=0 {t.Fatalf("got %+v %v",r,e)}
}
func TestManaChainNoRecovery(t *testing.T) {
 _,e:=SimulateManaChain(ManaChain{Current:0,Maximum:100,CostPerKill:10,FightSeconds:2},1)
 if e==nil {t.Fatal("expected insufficient mana error")}
}

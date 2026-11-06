package route
import "testing"
func TestManaClassGate(t *testing.T) {
 for _,c:=range []string{"MAGE","PRIEST","WARLOCK","SHAMAN","PALADIN","DRUID","HUNTER"} {
  if !ClassUsesMana(c) {t.Fatalf("%s should use mana",c)}
 }
 for _,c:=range []string{"WARRIOR","ROGUE","DEATHKNIGHT",""} {
  if ClassUsesMana(c) {t.Fatalf("%s should not use mana",c)}
  p:=ManaChain{Maximum:0}
  q,e:=WithMageWaterForClass(p,c,10,nil)
  if e!=nil || q!=p {t.Fatalf("nonmana changed: %s %+v %v",c,q,e)}
 }
}

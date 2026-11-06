package route
import "testing"
func TestBestMageWaterUniversal(t *testing.T) {
 ranks:=[]MageWaterRank{{ItemID:1,RequiredLevel:1,RestoreMana:100,DurationSeconds:20},{ItemID:2,RequiredLevel:10,RestoreMana:300,DurationSeconds:20}}
 for _,level:=range []int{10,15,30} {
  r,e:=BestMageWater(level,ranks)
  if e!=nil||r.ItemID!=2 {t.Fatalf("level %d: %+v %v",level,r,e)}
 }
 p,e:=WithMageWater(ManaChain{Maximum:200,Current:100},15,ranks)
 if e!=nil||p.DrinkManaPerSecond!=15||p.DrinkTargetFraction!=1 {t.Fatalf("%+v %v",p,e)}
}
func TestBestMageWaterMissing(t *testing.T) {
 _,e:=BestMageWater(5,[]MageWaterRank{{ItemID:2,RequiredLevel:10,RestoreMana:300,DurationSeconds:20}})
 if e==nil {t.Fatal("expected no rank")}
}

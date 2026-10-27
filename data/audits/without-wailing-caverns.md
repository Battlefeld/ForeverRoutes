# RF und RoL mit oder ohne HDW, Untoter Hexer 1–20

Gepaarter Vergleich auf demselben korrigierten Datensatz: gleicher Startseed je Ausstattungsvariante, jeweils 180 Sekunden Optimierung mit Suchseed 20261027 und GOMAXPROCS=1. RF und RoL bleiben über ihre bereits vorhandenen Questziele in beiden Varianten verpflichtend. In der Kontrolle bleibt zusätzlich HDW über Smart Drinks verpflichtend; ohne HDW sind der Dungeon und seine Dungeonquests aus der verfügbaren Welt entfernt. Fehlende XP werden bis zum gleichen Ziellevel ersetzt.

Alle acht Ergebnisse erreichen Level 20, enthalten die gewünschten Dungeons, Pflicht-Klassenquests und Optionen und wurden anhand ihrer tatsächlichen Aktionen erfolgreich nachgespielt.

| Ausstattung | Mit HDW | Ohne HDW | Ohne HDW länger |
|---|---:|---:|---:|
| Ohne Zusatzoption | 11:41:15 | 12:50:17 | +69.0 min |
| Zauberstab | 11:28:04 | 12:28:03 | +60.0 min |
| Schlafsack | 13:27:42 | 14:12:36 | +44.9 min |
| Zauberstab + Schlafsack | 12:36:07 | 13:46:10 | +70.0 min |

Die Kontrollzeiten mit HDW können vom vorigen Bericht abweichen, weil beide Seiten zusätzlich neu optimiert wurden. Das Ergebnis ist ein begrenzter Suchlauf im Modell, kein Beweis eines globalen Optimums und keine gemessene Spielzeit. Bekannte Unsicherheiten bei Ereignisdauern, Kampfprofil und Dungeonzeit bleiben bestehen. Der Schlafsack enthält seine Beschaffung, noch nicht seinen späteren XP-Bonus.

[Maschinenlesbares Ergebnis mit Events, Wand-Detour und Datensatz-SHA-256](without-wailing-caverns.json). Die Startseeds liegen in `no-wc-seeds/`.

Reproduktion je Variante (`dungeon`, `dungeon_wand`, `dungeon_bag`, `dungeon_wand_bag`):

```sh
# Kontrolle mit HDW
FR_NO_WC_SCENARIO=dungeon_wand FR_EXCLUDE_WC=0 FR_BENCH_OUTPUT=/tmp/no-wc GOMAXPROCS=1 \
  go test ./route -run '^TestUndeadNoWCComparison$' -count=1 -v -timeout=8m
# Ohne HDW
FR_NO_WC_SCENARIO=dungeon_wand FR_EXCLUDE_WC=1 FR_BENCH_OUTPUT=/tmp/no-wc GOMAXPROCS=1 \
  go test ./route -run '^TestUndeadNoWCComparison$' -count=1 -v -timeout=8m
```

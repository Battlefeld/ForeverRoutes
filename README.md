# ForeverRoutes

> **Datengrundlage:** ForeverRoutes nutzt Daten von **[ForeverDB.net](https://foreverdb.net)** (Data from ForeverDB.net),
> QuestieDB Forever, EverythingQuests Forever, [WarcraftDB Forever](https://forever.warcraftdb.com), der Forever-Datenbank
> von [Polestar](https://github.com/fitzforge/polestar) (forever-db), den Spieler-Traces von QuestieDB,
> [NorgannonDB](https://github.com/zmarotrix/NorgannonDB) und der Weltdatenbank von
> [cmangos classic-db](https://github.com/cmangos/classic-db). Herkunft und Revision jeder Quelle stehen
> im Datensatz selbst (`sources`). ForeverDB ist nicht mit Blizzard Entertainment verbunden.

Routenplaner für WoW Forever, Level 1–30: welche Quests annehmen, wo was töten, wann abgeben und wann Grinden schneller
ist. Go-Modul ohne Fremdabhängigkeiten. Der Planer läuft offline als Kommandozeilenwerkzeug und ist als Bibliothek für
ForeverCore.exe gebaut (Paket `route`). Die Kampfzeiten kommen aus MythicSim (ForeverSim) über ForeverCores Auftrag
`kill-profile` (siehe „Kampfprofil“).

## Ingame-Addon

Das eigenständige HUD liegt in `ForeverRoutes/`. Im aktuellen GitHub-Release enthält
`ForeverRoutes-addon.zip` das vollständige Addon inklusive Grafiken und frisch exportierter Routen.
Den enthaltenen Ordner `ForeverRoutes` nach `Interface/AddOns` entpacken und mit `/fr` öffnen.
ForeverSim ist keine Abhängigkeit. [HUD-Funktionen und Befehle](ForeverRoutes/README-HUD.md).

Die Routendaten werden beim Paketbau erzeugt; ein reiner Git-Checkout ist noch kein installierbares Addon.

## Stand

- Datensatz mit 2922 Quests, 3751 NPCs, 630 Objekten, 59 Flugmeistern, 45 Gasthäusern, Zeppelinen und Schiffen, dazu das
  Skyborne-Startgebiet Zephras Isle mit der Abreise beider Fraktionen. Skyborne brauchen `-faction A` oder `-faction H`.
- Alle 56 unterstützten Volk-/Klassenkontexte werden bis 20 geprüft; je Kontext werden Solo und optionale Dungeons sowie passende Zauberstab-/Schlafsackvarianten berechnet. [Gesamtprüfung](data/audits/roster-v22.md).
- Orc Hexer und Mensch Magier bleiben zusätzliche Ausgangstests bis Level 30.
- **Die Stundenangaben sind nicht kalibriert.** Ohne `-profile` gilt das eingebaute Kampfmodell (`DefaultProfile`), eine Schätzung. Es
  bewertet Grinden im Vergleich zu Quests etwa doppelt so gut wie die Wirklichkeit: Ein Guide-Ablauf (RestedXP) und
  reines Grinden landen bis Level 21 im Modell fast gleichauf. Darum grinden die Routen heute zu viel. Klassenbezogene Simulationswerte kommen
  über die eingefrorenen MythicSim-Profile (`KillProfile`); auch sie sind keine Ingame-Messung. Mit `guide` lassen sich Guides unter denselben Regeln nachgehen und so
  das Modell prüfen.
- **Questprüfung bis 20:** [vollständiger Bericht](data/audits/quests-through-20.json), [Ergebnis und Grenzen](data/audits/README.md).
  Alle 1908 Quests mit Mindestlevel ≤20 werden über die 56 ausgelieferten Volk-/Klassenkombinationen geprüft.
  1450 sind für mindestens einen Kontext bis 20 modellierbar (19 wiederholbare davon einmalig); 179 haben offene Datenlücken.
  Unbekannte Mengen, Orte und Quellen erzeugen keine kostenlosen Questbelohnungen. Unbekannte XP bleiben gekennzeichnet bei 0.
  Statische Routendateien müssen nach Datenänderungen neu berechnet werden; ihre alten Stunden sind kein Vergleich mit diesem Datensatz.

## Neue Zeitstrahl-Optimierung

`timeline` vergleicht unabhängige Startdateien, Dungeon-Teilmengen einschließlich keiner Instanz sowie Vorbereitungsvarianten.
Größere Questabschnitte werden neu zusammengesetzt; ein kompatibler Checkpoint bewahrt die beste nachgespielte Lösung.
Die Suche minimiert die Zeit bis zum Ziellevel. Originalguides werden nicht überschrieben.

[Verfahren, Befehle, Tests und Ergebnisse](data/audits/timeline-v2.md). [ForeverSim-Schnittstelle und Freigabesperre](docs/foreversim-feedback.md).
Die Schnittstelle ist vorbereitet, bleibt aber bis nach dem Release und ausdrücklicher Nutzerbestätigung deaktiviert.

## Normaler RXP-Fallback bis 20

`data/fallbacks/rxp.json.gz` enthält 30 unveränderte normale RXP-Kapitel sowie 56 getrennte Kapitelketten. AoE- und Dungeon-Guides werden nicht als normaler Fallback eingesetzt. Die mitgelieferte Lizenz und die festgeschriebene Quellrevision bleiben erhalten.

`next -route auto` wählt die passende aktuelle Routendatei; fehlt sie, ist sie veraltet oder gehört sie zu einem anderen Charakter, wird RXP verwendet. `-route ""` plant frei. Der CLI-Nachgang eines RXP-Fallbacks ist ausdrücklich nur eine Quest-Arbeitsprojektion: dynamische RXP-Bedingungen, Geld, Inventar und alle Reise-/Lehreranweisungen sind darin nicht vollständig simuliert. Für das spätere Ingame-UI exportiert `export-addon` die vollständigen Originalanweisungen und `ForeverRoutesData.SelectRoute(...)`. [Fallback-Vertrag](docs/rxp-fallback.md).

## Befehle

```sh
go test ./...                      # alle Tests, inkl. Ausgangstest Orc Hexer / Mensch Magier (~3 min)
go test -short ./...               # ohne die langen Planungsläufe
go run ./cmd/foreverroutes check-quests -target 20 -out quest-audit.json
go run ./cmd/foreverroutes timeline -seeds data/routes/undead-warlock.json,data/routes/undead-warlock-dungeons.json -profile data/profiles/warlock.json -budget 3m -out data/optimized/undead-warlock.json

go run ./cmd/foreverroutes plan -race Orc -class Warlock -out orc.json
go run ./cmd/foreverroutes plan -race Human -class Mage -target 30 -restarts 4 -out human.json
go run ./cmd/foreverroutes plan -race Orc -class Warlock -profile data/profiles/warlock.json   # Kampfzeiten aus MythicSim
go run ./cmd/foreverroutes replay -race Orc -class Warlock -route orc.json
go run ./cmd/foreverroutes next -live state.json -profile data/profiles/warlock.json -horizon 3600   # ab Live-Stand
go run ./cmd/foreverroutes guide -race Orc -class Warlock -events guide.json   # Guide unter denselben Regeln
go run ./cmd/foreverroutes audit                                              # was der Datensatz kennt und was fehlt
```

`guide` liest eine Liste von Schritten `[{"op":"accept","quest":4641},{"op":"complete","quest":4641},
{"op":"turnin","quest":4641},{"op":"level","level":6}]`. Der normale RXP-Fallback liegt mit Originalanweisungen und CC-BY-NC-SA-4.0-Lizenz in `data/fallbacks`; andere Guide-Quellen werden dadurch nicht ersetzt. Einzelziele können mit `objective` (1-basiert), XP-Zwischenziele mit `xp` angegeben werden.

### Solo mit Dungeon (`-dungeons`)

Klassenregeln: Hexer und Jäger kämpfen einfache Elites bis zu ihrem Level allein (das Pet tankt). Krieger, Schurke,
Paladin und Schamane kämpfen höchstens ein Level nach oben: Die Route kommt erst in ein Gebiet, wenn dessen Gegner
höchstens +1 sind (höhere Gegner kosten mehr Zeit je Kill, darum bevorzugt die Planung gleiche oder niedrigere). Schickt
ein Guide sie zu früh hin, levelt der Nachspieler vorher bis zu zwei Level, statt die Quest auszulassen; ob sich das
lohnt, entscheidet das Verbessern (es kann die Quest samt Grinden weglassen).

Klassenquests (`route/classquests.go`): Jede Route macht die Pflicht-Klassenquests samt Vorquests, auch wenn es länger
dauert: Klassenbrief (Level 1–2), Wichtel und Leerwandler (Hexenmeister), Begleiter zähmen (Jäger), Bärengestalt
(Druide), Erd- und Feuertotem (Schamane), Verteidigungshaltung (Krieger), Schlossknacken (Schurke). Von mehreren
Versionen zählt die aus dem eigenen Startgebiet oder der eigenen Hauptstadt, sonst die nächstgelegene. Die Einfügeheuristik sucht zunächst innerhalb von zwei Leveln nach dem Mindestlevel einen passenden Besuch. Die Optimierung darf den Zeitpunkt verschieben; die tatsächliche Abschlussstufe steht in `classQuests.level`. Pflichtreihen müssen vor dem Ziellevel abgeschlossen sein und dürfen nicht entfallen. Empfohlene (Waffen, Roben, Volkszauber der Priester) und sonstige Klassenquests sind Optionen. Die
Routendatei nennt für jede Reihe unter `classQuests`, was sie die Route kostet (`minutes`; negativ: ihre XP machen die
Route schneller). Klassenquest-Gegner zählen auch als Elite als solo machbar.

Voraussetzungen: Ein Breadcrumb (cmangos `BreadcrumbForQuestId`) ist keine Voraussetzung, auch wenn die Client-Daten
ihn so führen (Cutting Teeth, Redridge Rendezvous, Surena Caledon). Nimmt ein RestedXP-Forever-Guide eine Quest an,
während die angebliche Voraussetzung noch offen ist, kann diese entfallen (nicht bei Klassenquests).
Dass kein Guide eine Vorquest erwähnt, reicht nicht zum Entfernen. Alternative Voraussetzungen bleiben als Alternativen erhalten;
nur wenn der Guide alle gleichzeitig offen hat, kann die OR-Bedingung entfallen. Eine Voraussetzung einer anderen Rasse zählt für den Charakter nicht.

Es gibt je Volk und Klasse zwei Routen: **Solo ohne Dungeon** (`<volk>-<klasse>.json`, keine Dungeonquests) und **Solo mit erlaubten Dungeons** (`<volk>-<klasse>-dungeons.json`, auch kein Lauf ist erlaubt). Mit `-dungeons` (bei `plan`, `improve`, `extend`, `next`) gehören Dungeon-Läufe zur Route. Ein Dungeon-Baustein nimmt die
Dungeonquests an, die draußen vergeben werden und die der Lauf erledigt (nächstgelegene zuerst; unterwegs schon ab drei
Stufen vor dem Dungeon, wenn der Geber nahe ist), läuft den Dungeon (dafür eine Gruppe) und gibt danach ab, wenn nötig
auf dem anderen Kontinent. Im Dungeon vergebene und abgegebene Quests samt Folgequests erledigt der Lauf
selbst. Der Planer wägt den Baustein wie jeden anderen gegen Grinden ab. Jeder Dungeon wird höchstens einmal gelaufen,
nur innerhalb seiner Stufen.

- Dungeons: `data/sources/dungeons.json` (Stufen, übliche Laufzeit als Schätzung, Seite). Der Eingang kommt von Polestar,
  der Trash aus den cmangos-Spawns der Instanz (5er-Gruppe mit Classic-Gruppenbonus 1,4; ein Questlauf tötet 40 %, was im Weg steht). Vor jedem Lauf 10 Minuten Gruppensuche; Klassen, die tanken oder heilen können (Krieger, Paladin, Schamane, Priester, Druide), finden fast sofort eine Gruppe (1 Minute); Druide und Krieger erst mit Bärengestalt bzw. Verteidigungshaltung, die Routen mit Dungeon holen sie vor dem ersten Lauf. Sammelquests im Dungeon: jedes Gruppenmitglied braucht eigene Drops, die vier weiteren Anteile kosten zusätzliche Kills (15 s je Kill); einzelne Bossgegenstände nicht.
- `extend -route data/routes/orc-warrior.json -target 40 [-dungeons]` setzt eine Route fort: Er spielt sie bis zu
  ihrem Ende nach, plant ab dem erreichten Stand und verbessert das Ganze wie `improve`.
- Beim Optimieren werden Dungeonquests gemeinsam mit ihrer vollständigen Vorquestkette als ein Kandidat bewertet.
  Ein Lauf, der sich nicht lohnt, fällt beim Verbessern heraus wie eine Quest. Laufzeiten werden nach dem Release im
  Spiel kalibriert.

### Planen beim Spielen (`next`)

`next` plant ab dem Stand, den das Addon im Spiel liest, die nächste Stunde (`-horizon` in Sekunden). Mitgegeben
werden Level, Erfahrung, Position, erledigte Quests, das Questlog mit dem Fortschritt jedes Ziels, bekannte
Flugmeister, Ruhestein-Ort und -Abklingzeit (`route.Live`, Eingabe `{race, class, live}`). ForeverCore ruft das bei
jeder Quest-Änderung neu auf. So folgt die Route dem Charakter, statt ihn auf einer festen Route zu halten.

- Eine Entscheidung (Quest, Hub oder Grinden) dauert 0,3–0,8 s und deckt etwa 10 Minuten Spiel ab. Ein Aufruf für eine
  Stunde braucht mit Laden des Datensatzes etwa 1–3 s.
- Quests im Log, die der Planer für diesen Charakter nicht kennt, belegen trotzdem einen Platz im Log und stehen in
  `notes`. Der Fortschritt eines Ziels wird nach benötigter Anzahl und Art zugeordnet, weil das Spiel die Ziele in
  eigener Reihenfolge liefert. Was sich nicht zuordnen lässt, zählt als nicht begonnen. Der Planer nimmt also nie
  Fortschritt an, den er nicht sieht.

### Kampfprofil

`-profile` (bei `plan`, `replay` und `guide`) nimmt die Kampfzeiten aus einem Profil, das ForeverCore mit MythicSim und
dem Level-Overlay rechnet (`kill-profile`, Schema `foreverroutes.killprofile.v1`). In `data/profiles` liegen die
Profile aller neun Klassen (Level 10–40): synthetischer Charakter mit der weißen Startwaffe, ohne weitere Ausrüstung
und Talente. Für alle Völker derselben Klasse wird dasselbe Vergleichsprofil verwendet; individuelle Volksfähigkeiten sind damit nicht separat kalibriert.

- **Kampf:** Für jedes Level stehen Schaden, Manaverbrauch und selbst verbrauchte Gesundheit nach 6, 10, 15, 22 und
  30 Sekunden gegen ein gleich hohes Ziel und gegen eines zwei Level darüber. Der Kampf dauert, bis der Schaden die
  Lebenspunkte des Gegners erreicht. Zwischen den beiden Zielen wird nach dem Level des Gegners interpoliert.
- **Pause:** Essen und Trinken holen zurück, was der Kampf gekostet hat, zu den Raten des Händleressens und -wassers des
  Levels (z. B. ab 15: 243 Leben und 437 Mana in 21 s). Der erlittene Schaden ist der Durchschnittstreffer des Gegners
  aus cmangos, ein Schlag alle 2 s. Gezählt wird er für die Zeit, in der der Gegner den Charakter erreicht: beim Hexer
  mit Leerwandler ein Viertel des Kampfs, sonst der Kampf ohne die ersten 3 s Anlauf. Dazu kommen 4 s für Plündern und
  den nächsten Pull.
- **Grenzen:** Die Profile gelten für einen Charakter ohne Ausrüstung und Talente; Ausrüstungs- und Talentunterschiede sind damit nur eingeschränkt abgebildet.
  Unter Level 10 und außerhalb der Tabellen gilt `DefaultProfile`, an die Randwerte des Klassenprofils angepasst. Rüstung und Ausweichen des Charakters
  sind nicht berücksichtigt. Die Annahmen zu erlittenem Schaden prüft erst die Aufzeichnung im Spiel.

### Daten aktuell halten

```sh
FOREVERDB_API_KEY=… go run ./cmd/foreverroutes update   # alle Quellen prüfen, bei Neuem neu bauen und prüfen
go run ./cmd/foreverroutes build-data                     # Datensatz exakt aus data/sources/lock.json neu bauen
```

`update` fragt die neuesten Stände von QuestieDB, EverythingQuests und cmangos ab (Git), liest die Questliste von
ForeverDB über die API (höchstens einmal pro Stunde, 2,1 s Abstand) und holt für Quests, denen noch Angaben fehlen,
die Clientdaten von WarcraftDB (1,5 s Abstand, Zwischenspeicher). Ändert sich am Datensatz etwas, wird er neu gebaut und
gegen den bisherigen geprüft (keine großen Verluste an Quests, NPCs, Positionen); nur dann werden die Dateien
geschrieben. Neue Commits upstream, die am Inhalt nichts ändern, lassen alles, wie es ist.

Auf GitHub läuft das als Workflow `update` (beim Start von WoW über ForeverCore angestoßen, sonst täglich): Bei
Änderungen laufen alle Tests und der Ausgangstest, dann wird auf `main` gepusht und der Datensatz mit SHA-256 als Release
veröffentlicht. Schlägt etwas fehl, bleibt alles beim alten Stand und ein Issue „Data update failed“ beschreibt den
Grund. Der Workflow `data` prüft bei jeder Änderung, dass der Datensatz byte-genau aus den festgeschriebenen Quellen
entsteht.

## Aufbau

| Pfad | Inhalt |
|---|---|
| `route/` | Planer: Welt je Charakter, Zustand und Regeln (`Apply`/`Replay`), Reisen (Laufen, Flugmeister, Zeppelin, Schiff, Ruhestein), Suche, Guide-Nachgang, Kampfmodell |
| `fallback/` | Normale RXP-Kapitel und getrennte Charakter-/Fraktionsketten laden und prüfen |
| `data/fallbacks/` | Festgeschriebener normaler RXP-Fallback einschließlich Lizenz |
| `dataset/` | Format des Datensatzes (`foreverroutes.dataset.v1`), Laden und Schreiben |
| `xp/` | Classic-XP-Regeln: Quest-XP nach Levelabstand, Kill-XP (cmangos), Levelkurve |
| `internal/update/` | Quellen prüfen, neu bauen, Ergebnis prüfen (`update`, `build-data`) |
| `internal/sources/` | Importer: QuestieDB (Lua-Datentabellen), ForeverDB (API), EverythingQuests, WCLBox (Gegenstandsquellen), ForeverChanges (Questart, Zielsatz) |
| `internal/merge/` | Zusammenführen der Questquellen feldweise mit Quelle und Vorrangregeln |
| `internal/build/` | Bau des Datensatzes aus Quest-Datenbank, cmangos und WarcraftDB; Umrechnung Kartenprozent → Weltyards |
| `internal/lua/` | Leser für Lua-Tabellenliterale (führt keinen Code aus) |
| `internal/sqldump/` | Leser für den cmangos-SQL-Dump |
| `internal/warcraftdb/` | Abruf und Format der WarcraftDB-Ergänzung |
| `cmd/foreverroutes/` | Kommandozeile |
| `data/routes-data.json.gz` | der fertige Datensatz |
| `data/sources/` | festgeschriebene Stände der Quellen (`lock.json`), ForeverDB-Stand, WarcraftDB-Ergänzung, einzeln belegte Werte |
| `data/archive/` | aufbewahrte Daten für später (Dungeon-Journal), vom Planer noch nicht gelesen |

## Modell in Kürze

- Positionen in Weltyards je Kontinent. Forever-Positionen (Kartenprozent) werden je Karte über NPCs umgerechnet, die in
  beiden Systemen bekannt sind (mittlerer Fehler ~1 Yard). Fehlt Forever eine Position, gilt cmangos.
- Ziele werden an den dichtesten Spawn-Gruppen erledigt (bis zu 4 je Kreatur); die Route nimmt die schnellste.
  Gegenstände kommen aus Beute (cmangos, sonst Classic-Dropraten aus QuestieDB), vom Händler (hinlaufen und kaufen)
  oder als Belohnung einer anderen Quest (die wird dann Voraussetzung). Quests, die ein Gegenstand startet, beginnen
  dort, wo er droppt.
  Kills zählen für alle Quests im Log, Respawns werden abgewartet, geleerte Gruppen erholen sich über die Respawnzeit.
- Reisen: Laufen (7 Yards/s) auf gemessenen Wegen (siehe „Laufwege“), Flugmeister ab Entdeckung (im Vorbeigehen innerhalb 60 Yards), Zeppeline und
  Schiffe mit Zeiten aus der cmangos-Transporttabelle, Ruhestein (10 s, 1 h Abklingzeit, gebunden am letzten Gasthaus).
- Zauberstab: Magier, Priester und Hexenmeister bauen sich bei einem Besuch in der nächsten Hauptstadt
  (Verzauberkunst-Lehrer, -Händler, Auktionshaus) den Novice's Practice Wand (getragen bis Stufe 5) und den Geringen
  Zauberstab (ab Stufe 5), sofern diese Vorbereitung gewählt wurde; die Kampfwahl vergleicht Zauber und Zauberstab ohne Manakosten. Quest- und Dungeon-Zauberstäbe ersetzen ihn, wenn sie besser sind.
- Lager: Die Optimierung vergleicht Kürschnern, Kochkunst und Lederverarbeitung mit dem Verzicht darauf. Gewählte Vorbereitungen bezahlen Trainerweg und die geschätzten Beschaffungskosten. Ab Stufe 10 können Lager mit Lagerzelt erholte XP geben; Ruhezeit und Abklingzeit werden bezahlt. Materialbeschaffung und Berufsplätze sind weiterhin ein vereinfachtes Kostenmodell.
- Laufwege: Zwischen allen Orten, die Routen anlaufen (Questgeber, Ziele, Grindplätze, Flugmeister, Friedhöfe,
  Eingänge, Lehrer; 5630 Orte), liegen 1,6 Millionen Wege bis 3000 Yards in `data/walktable.json.gz`, gemessen mit
  der Wegsuche von Shortest Path Forever auf ihrem 8-Yard-Navigationsnetz mit Stockwerken, auf der Höhe der Orte
  (Unterstadt unter den Ruinen, Minen unter dem Hügel): der Weg, den ein Spieler läuft, samt Steigungen, Schwimmen mit
  eigenem Tempo. Sie decken 96,6 % der Laufwege der Routen ab. Alle anderen Wege (Live-Positionen, Todespunkte, Wege
  durch einen Aufzug) gehen über das 200-Yard-Gitter, kalibriert an denselben Messungen; Aufzüge kosten ihre Fahrt
  samt Warten. [Messung und Neuaufbau](tools/walktable/README.md).
- Laufgitter: Ein Punkt auf einer abgeschnittenen Gitter-Insel (Dammkrone, Felsvorsprung) wird über die nächste
  verbundene Zelle erreicht statt mit dem Notwert (sechsfache Luftlinie).
- Optionen: Klassenreihen, die nicht Pflicht sind, und Forever-Ziele wie der Gemütliche Schlafsack (ab Level 14) werden
  Ort für Ort eingewoben (was an einem Ort zu tun ist, als ein Besuch), jeweils an der Stelle der Route, die am
  wenigsten kostet, und sind bis Level 20 fertig; im Addon zuschaltbar.
- Suche: Bausteine (eine Quest, ein Questgebiet in drei Größen, Grinden) bewertet nach dem Gewinn gegenüber Grinden;
  die besten werden einige Entscheidungen weit vorausgespielt; mehrere Läufe, das beste Ergebnis wird nachgeprüft.
  Ergebnis ist die beste gefundene Route, kein bewiesenes Optimum.

## Lizenz

Code: MIT (siehe `LICENSE`). Daten: siehe [`data/README.md`](data/README.md); der Datensatz enthält aus cmangos
classic-db, Polestars forever-db und NorgannonDB abgeleitete Werte (GPL-3.0).

Die normalen RXP-Kapitel und daraus abgeleitete Routensequenzen unterliegen CC BY-NC-SA 4.0; siehe [RXP-Lizenz](data/fallbacks/RXP-LICENSE.txt).

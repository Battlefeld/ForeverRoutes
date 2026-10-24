# Daten

| Datei | Inhalt |
|---|---|
| `routes-data.json.gz` | Datensatz des Planers (`foreverroutes.dataset.v1`), gebaut von `foreverroutes update` / `build-data` |
| `sources/lock.json` | festgeschriebene Stände: QuestieDB-, EverythingQuests- und cmangos-Commit, ForeverDB-Build und Prüfsumme |
| `sources/foreverdb-snapshot.json.gz` | ForeverDB-Questliste Level 1–30 mit Spielmetadaten (Data from ForeverDB.net) |
| `sources/warcraftdb-forever.json` | Questangaben von WarcraftDB Forever, die den anderen Quellen fehlen |
| `sources/wclbox-items.json` | woher ein Gegenstand kommt (Gegner und Gebiet), gelesen von den Gegenstandsseiten der WCLBox; die Seite nennt keine Dropchance |
| `sources/foreverchanges-quests.json` | Zusammenfassung je offener Quest von ForeverChanges: Questart (normal, Elite, Dungeon) und der Gegner aus dem Zielsatz |
| `sources/manual-facts.json` | einzeln belegte Werte (XP, Level, wiederholbar, Questziele mit Mengen) von WarcraftDB, OctoWow, ClassicDB, Wowhead Forever und Beobachtungen aus ForeverDB, jeweils mit Link; übernommen aus dem früheren Python-Werkzeug und dem früheren Addon ForeverQuests |
| `archive/dungeon-journal/` | Dungeon-Daten des früheren Addons ForeverQuests (Bosse, Beute, Dungeonquests) mit Quellenliste; vom Planer noch nicht gelesen |

QuestieDB, EverythingQuests und cmangos werden beim Bau in den festgeschriebenen Commits geladen und nicht eingecheckt.

## Quellen und Hinweise

- **ForeverDB** — Data from [ForeverDB.net](https://foreverdb.net). Wo ForeverDB-Daten angezeigt werden, muss dieser
  Hinweis mit Link sichtbar sein. ForeverDB ist nicht mit Blizzard Entertainment verbunden.
- **QuestieDB** (https://github.com/Questie/QuestieDB) — Forever-Quests, NPCs, Objekte, Gegenstände; Classic-XP und
  Beutechancen als Classic-Referenz gekennzeichnet.
- **EverythingQuests** (https://github.com/wheelbarrel00/EverythingQuests, MIT) — Forever-Questgeber, Abgaben, Ziele,
  Voraussetzungen (teils aus AllTheThings).
- **WarcraftDB Forever** (https://forever.warcraftdb.com) — Spielclient-Questdaten (Ziele mit Mengen, XP, Level,
  Positionen). Abruf laut robots.txt erlaubt, mit 1,5 s Abstand und Zwischenspeicher.
- **WCLBox** (https://wowforever.wclbox.com, „新手盒子·魔兽世界无限数据库“) — Gegenstandsseiten mit „Source: Gegner, Gebiet“.
  robots.txt erlaubt die Seiten, die Seite lädt automatische Leser ausdrücklich ein (llms.txt) und bittet um Nennung mit
  Seitenlink; jeder Eintrag behält seinen Link. Abruf: eine Anfrage pro Sekunde, nur für Gegenstände ohne andere Quelle.
  Dazu die Questseiten (Daten aus Wowheads Forever-Datenbank) der Quests, die sonst ohne Quelle blieben: welcher
  Gegenstand vom Questgeber übergeben wird („Provided“), woher ein Gegenstand kommt (Gegner, Objekt, Händler) und die
  Punkte auf der Gebietskarte, wo die Ziele liegen. Genannte Gegner und Objekte nahe den Punkten werden Quellen
  (Gegner mit unbekannter Chance), sonst werden die Punkte die Orte des Ziels.
- **ForeverChanges** (https://foreverchanges.pro) — Questseiten (Daten aus dem Spielclient und aus Wowheads Forever-Datenbank):
  Questart und Zielsatz. robots.txt erlaubt die Questseiten; eine Anfrage pro Sekunde, nur für offene Quests.
- **Polestar forever-db** (https://github.com/fitzforge/polestar, Ordner `forever-db`, GPL-3.0) — aus dem Forever-Spielclient
  gebaute Questdaten: Forever-XP (Dungeonquests mit Ziel im Dungeon wurden auf etwa das 2- bis 2,7-Fache angehoben),
  Level, Volks- und Klassenmasken mit den Skyborne-Bits, Questgeber, Abgaben, Ziele mit Mengen, Voraussetzungen, was einen
  Gegenstand droppt, Spawnpunkte und Kartenrahmen. Wo Polestar eine Quest kennt, gelten seine Clientwerte vor allen
  Referenzen außer im Spiel beobachteten Werten; Quests, die nur Polestar kennt, kommen hinzu. Ereignisziele (Gebiet
  erkunden, Eskorte) haben keinen Ort in den Daten und werden ohne sie geplant. Dungeonquests bleiben aus Solo-Routen
  draußen, außer alle Ziele liegen vor dem Instanzportal: Gegner oder Objekte in der offenen Welt, höchstens 450 Yards von
  einem Dungeoneingang (Polestars `data/codex/entrances.json`), keine Elite. So sind Red Silk Bandanas (Defias im Stollen
  vor den Todesminen) und The Essence of Aku'Mai (Saphire vor der Tiefschwarzen Grotte) solo planbar. Die Abreise der Skyborne von Zephras Isle
  (Luftschiff nach Mulgore, Skycutter nach Alterac und Portal nach Sturmwind) stammt aus Polestars Skyborne-Guides, die
  Fahrzeiten sind geschätzt. Stand und Commit stehen in `data/sources/lock.json` (`polestar`).
- **Guide-Fakten aus RestedXP** (https://github.com/RestedXP/RXPGuides, `Guides/forever`, CC BY-NC-SA 4.0) **und Polestars
  Forever-Guides** (`guides/forever`, GPL-3.0) — was die Guides dem Spieler zu einem Questziel sagen: welche Gegner man
  dafür tötet, wo man hingeht, ob die Gegenstände am Boden liegen, welches Objekt man benutzt, wo der Gegenstand droppt,
  der eine Quest startet. Nur Fakten (IDs, Namen, Orte), kein Guidetext. Sie füllen nur Lücken, die keine Datenbank
  schließt: genannte Gegner werden Drops mit unbekannter Chance, Objekte Drops zu 100 %, sonst werden die Orte des Guides
  die Orte des Ziels; Gegner und Objekte ohne Spawnpunkt bekommen die Orte des Guideschritts. Polestar-Schritte ohne
  Koordinaten zählen über ihr Gebiet: die Gegner, die der Hinweis des Schritts nennt (nur in diesem Gebiet, keine
  Questgeber, Händler oder Lehrer), oder die Orte der Gegner, die der Schritt ohnehin tötet. Lieferquests (ein einziger
  Gegenstand, den nichts droppt und kein Guide suchen lässt, Geber und Abnehmer verschieden) gelten als vom Geber
  übergeben; die Quest trägt dann den Vermerk „item assumed handed over by the giver“. RestedXPs Stand steht in
  `data/sources/lock.json` (`rxp`), Polestars im selben Commit wie forever-db.
- **cmangos classic-db** (https://github.com/cmangos/classic-db, GPL-3.0) — Classic-Weltdaten: Zielmengen,
  Beutechancen, Spawnpositionen, Respawnzeiten, Lebenspunkte und Schaden der Kreaturen, Flugmeister, Gasthäuser,
  Transporte. Classic, nicht Forever: Forever-Angaben haben Vorrang. Werte aus cmangos stehen unter GPL-3.0; damit
  steht auch `routes-data.json.gz` unter GPL-3.0 (ebenso die Werte aus Polestars forever-db).

**Dropchancen:** Wo eine Quelle nur den Gegner nennt, nicht die Chance, steht im Datensatz `unknown` (Chance 0, kein Wert).
Der Planer nimmt dafür einen ausdrücklich gekennzeichneten Planungswert an (30 %) und meldet im Ergebnis, wie viele Schritte
davon abhängen (`assumedDropSteps`). Echte Werte kommen aus dem Mitschreiben im Spiel.

Nichts wird erfunden. Was keine Quelle kennt, bleibt im Datensatz als unbekannt markiert (`unsupported` mit Grund).

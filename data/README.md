# Daten

| Datei | Inhalt |
|---|---|
| `routes-data.json.gz` | Datensatz des Planers (`foreverroutes.dataset.v1`), gebaut von `foreverroutes update` / `build-data` |
| `sources/lock.json` | festgeschriebene Stände: Commits von QuestieDB, EverythingQuests, cmangos, Polestar, RestedXP, AEGIS, EG Link, Shortest Path Forever und NorgannonDB, ForeverDB-Build und Prüfsumme |
| `sources/foreverdb-snapshot.json.gz` | ForeverDB-Questliste Level 1–30 mit Spielmetadaten (Data from ForeverDB.net) |
| `sources/warcraftdb-forever.json` | Questangaben von WarcraftDB Forever, die den anderen Quellen fehlen |
| `sources/wclbox-items.json` | woher ein Gegenstand kommt (Gegner und Gebiet), gelesen von den Gegenstandsseiten der WCLBox; die Seite nennt keine Dropchance |
| `sources/foreverchanges-quests.json` | Zusammenfassung je offener Quest von ForeverChanges: Questart (normal, Elite, Dungeon) und der Gegner aus dem Zielsatz |
| `sources/manual-facts.json` | einzeln belegte Werte (XP, Level, wiederholbar, Questziele mit Mengen) von WarcraftDB, OctoWow, ClassicDB, Wowhead Forever und Beobachtungen aus ForeverDB, jeweils mit Link; übernommen aus dem früheren Python-Werkzeug und dem früheren Addon ForeverQuests |
| `sources/quest-fixes.json` | einzeln belegte Korrekturen (Voraussetzungen, Startobjekte, Orte, reine Gesprächsquests), jeweils mit Link |
| `sources/dungeons.json` | Dungeons der Gruppenrouten: Stufen (Forever-Beta laut Dungeon-Journal, sonst Classic), übliche Laufzeit (Schätzung), Seite, Gebiets-IDs der Dungeonquests |
| `archive/dungeon-journal/` | Dungeon-Daten des früheren Addons ForeverQuests (Bosse, Beute, Dungeonquests) mit Quellenliste; die Stufen sind in `sources/dungeons.json` übernommen |

QuestieDB, EverythingQuests und cmangos werden beim Bau in den festgeschriebenen Commits geladen und nicht eingecheckt.

## Quellen und Hinweise

- **ForeverDB** — Data from [ForeverDB.net](https://foreverdb.net). Wo ForeverDB-Daten angezeigt werden, muss dieser
  Hinweis mit Link sichtbar sein. ForeverDB ist nicht mit Blizzard Entertainment verbunden.
- **QuestieDB** (https://github.com/Questie/QuestieDB) — Forever-Quests, NPCs, Objekte, Gegenstände; Classic-XP und
  Beutechancen als Classic-Referenz gekennzeichnet. Dazu die Forever-Traces (`src/corrections/Forever/traces`, aus den
  Aufzeichnungen der Spieler mit QuestieTrace): von welchen Gegnern und Objekten ein Gegenstand droppt, Questgeber und
  Abgaben, Spawns von Gegnern und Objekten, die den Datendateien fehlen (Gegner mit unbekannter Chance).
- **NorgannonDB** (https://github.com/zmarotrix/NorgannonDB, GPL-3.0, Datei `data/items.lua`) — offene, gesammelte
  Forever-Datenbank (aus QuestieDB, pfQuest, Forevers Spieltabellen und Spielerdaten): wo ein Gegenstand droppt (Gegner,
  Objekte, mit Chance wo bekannt) und welche Quest ihn als Belohnung gibt (die wird dann Voraussetzung). Nur für
  Gegenstände ohne andere Quelle.
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
- **Shortest Path Forever** (https://github.com/cjber/shortest-path-forever, GPL-3.0-or-later) von Cillian Berragan —
  Reisedaten: Flugzeiten zwischen benachbarten Flugpunkten (zu 93 % im Spiel gemessen, aus InFlight, MIT), Fahrpläne
  der Schiffe und Zeppeline (Fahrzeit plus im Mittel eine halbe Runde Warten), Portale; die Deeprun-Tram ist
  geschätzt (106 s); die Aufzüge von Thunder Bluff und Undercity (Fahrt plus mittleres Warten) als Zuschlag für jeden
  Weg auf die Hochebene oder in die Stadt hinein oder heraus (`travel.lifts`). Dazu `walks.json.gz`: Laufkosten zwischen benachbarten 200-Yard-Zellen, gemessen mit dem eigenen
  Wegsucher von Shortest Path Forever auf seinen selbst aus dem Client gebauten Navigationskarten
  (`tools/walkgrid/README.md`). Stand in `data/sources/lock.json` (`spf`).
- **EG Link output von Efficient Games** (https://github.com/JIVESCORP/eg-link-output-wowf, CC BY 4.0) — „Data from
  Efficient Games (https://efficient.games), licensed under CC BY 4.0.“ Was das Addon EG Link bei Spielern aufzeichnet,
  ein Ordner je Beta-Build. Gelesen werden `loot` (was ein Gegner oder Objekt droppte, mit der Chance je Kill, gezählt
  während der Quest, die den Gegenstand braucht) und `objectives` (wo der Zielzähler hochging, in Weltkoordinaten).
  ForeverRoutes fasst die Builds zusammen (Chancen nach Beutefenstern gewichtet, erst ab 5 Fenstern); das ist eine
  Veränderung der Daten im Sinne der Lizenz. Gemessene Drops füllen Gegenstände ohne Quelle und ersetzen die angenommene
  Chance; die Zielorte füllen Ziele ohne andere Quelle. Stand in `data/sources/lock.json` (`eglink`).
- **AEGIS: Pathfinder (Forever)** (https://github.com/Torchlite-bit/Aegis_Pathfinder_Forever, `data/merged.json`, MIT)
  — Questdaten, die Spieler mit dem Addon im Spiel sammeln und einschicken: wo eine Quest angenommen und abgegeben wird,
  die Kartenpunkte des Clients für die Ziele, wo der Zielzähler hochging, die XP der Abgabe. Füllt nur Lücken (Orte von
  Zielen ohne andere Quelle, Geber- und Abgabeorte, fehlende XP); wo es sich mit den Clientdaten überschneidet, wird die
  Übereinstimmung der XP gezählt (`aegis_xp_agrees`, `aegis_xp_differs`). Stand in `data/sources/lock.json` (`aegis`).
- **cmangos classic-db** (https://github.com/cmangos/classic-db, GPL-3.0) — Classic-Weltdaten: Zielmengen,
  Beutechancen, Spawnpositionen, Respawnzeiten, Lebenspunkte und Schaden der Kreaturen, Flugmeister, Gasthäuser,
  Transporte, Friedhöfe (`world_safe_locs`, nur mit Geistheiler) und welcher Friedhof für einen Tod in welcher Zone
  gilt (`game_graveyard_zone`; die Zone eines Orts nach den QuestieDB-Spawns der Kreaturen dort, als Raster aus
  200-Yard-Zellen in `travel.deaths`). Forevers eigene Friedhöfe sind Geistheiler aus forever-db fern jedem
  klassischen. Dazu die Death Skips der RestedXP-Forever-Guides (`travel.deathskips`): Todesort (der Wegpunkt vor
  `.deathskip`), der Friedhof nahe dem nächsten Wegpunkt (sonst der nach der Regel am Todesort) und der Volks- und
  Klassenfilter des Schritts; abgeschaltete Schritte (`skip`) bleiben draußen.
  Classic, nicht Forever: Forever-Angaben haben Vorrang. Werte aus cmangos stehen unter GPL-3.0; damit
  steht auch `routes-data.json.gz` unter GPL-3.0 (ebenso die Werte aus Polestars forever-db).

**Was als machbar gilt (Prüfungen beim Bau):** Startet ein Gegenstand eine Quest, ist sie nur mit ihm zu haben (die
NPCs daneben sind die Abgabe, z. B. die Bücher für Owen Thadd und Garion Wendell). Ein NPC ohne Fraktionsangabe, der nur
in einer Hauptstadt steht, gehört deren Seite (Hauptstadtgrenzen aus Shortest Path Forevers `Zones.lua`). Ein
Ereignisziel (erkunden, Eskorte, etwas benutzen) braucht einen Ort: aus den Guides, aus EG Link oder AEGIS, sonst den
Questgeber oder die Abgabe, wenn ein Guide es dort erledigt; ohne Ort bleibt die Quest draußen. Eine Quest ohne Ziel,
die beim selben NPC angenommen und abgegeben wird, gilt nur am Ende einer Kette (mit Vorquest); sonst ist ihr Ziel
unbekannt und sie bleibt draußen.

**Dropchancen:** Wo eine Quelle nur den Gegner nennt, nicht die Chance, steht im Datensatz `unknown` (Chance 0, kein Wert).
Der Planer nimmt dafür einen ausdrücklich gekennzeichneten Planungswert an (30 %) und meldet im Ergebnis, wie viele Schritte
davon abhängen (`assumedDropSteps`). Echte Werte kommen aus dem Mitschreiben im Spiel.

Nichts wird erfunden. Was keine Quelle kennt, bleibt im Datensatz als unbekannt markiert (`unsupported` mit Grund).

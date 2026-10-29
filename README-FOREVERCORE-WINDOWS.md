# ForeverRoutes für ForeverCore 1.0 – Routengenerator, Timeline und schlankes WoW-HUD

> **STATUS: KONZEPT / UMBAU-SPEZIFIKATION, NICHT IMPLEMENTIERT.**
>
> Diese neue, ergänzende README ist die Routen-Seite der gemeinsamen ForeverCore-Windows-Planung. Der bestehende Code, die Routen, Tests, Workflows und die README.md werden hierdurch nicht geändert.
> Gegenstück: [ForeverSim / ForeverCore – Windows-App, Installer und Bridge](https://github.com/Battlefeld/ForeverSim/blob/main/README-FOREVERCORE-WINDOWS.md).
> **Produktregel:** Der Nutzer lädt/startet nur **ForeverCore.exe**; kein separates `foreverroutes.exe`-Fenster, keine Konsole, keine ZIP-/CMD-/PowerShell-Installation. Die Routes-Engine bleibt intern ein eigenständig getestetes Modul. Das ForeverRoutes-Ingame-Addon wird automatisch durch ForeverCore verwaltet.

## 1. Zielbild und Zuständigkeiten

- **ForeverCore (Windows):** zentrale grafische Anwendung mit Dashboard, Auto Route Builder, Drag-and-drop Timeline, Simulationsvergleichen, Profil-/Datenverwaltung, Installer, Update-Center und Integration zu MythicSim/ForeverSim.
- **ForeverRoutes (dieses Repo):** reine Engine/Bibliothek für Welt-/Questmodell, Routenplanung, Dungeon- und Berufsbausteine, Optimierung, Replay, Export, Tests sowie schlankes `ForeverRoutes`-HUD.
- **ForeverSim / MythicSim:** liefern versionierte Kampfprofile und Auswirkungen geplanter Ausrüstung/Zauberstäbe/Talente auf Kampf- und Regenerationszeiten. **Nicht** aus dem Routes-Repositorium verändern.
- **ForeverCoreBridge:** transportiert nur zulässige Live-Zustandsinformationen, Antwort-/Route-Snapshots, Steuer- und Versionsmeldungen. Nicht den vollständigen offline optimierten Routenverlauf ständig über die Pixelstreifen senden.
- **WoW:** zeigt Pfeil, Questlog-Anreicherung, aktuelle Aufgabe, wenige nächste Schritte, wichtige Side-Tasks und Ist-vs.-Soll-Fortschritt; keine schwere Routenberechnung.

Ingame-HUD gemäß `ForeverRoutes/README-HUD.md` als eigenständiges, funktionsfähiges Modul erhalten; kein 1:1-Klon eines anderen Guides, vorhandenes Artwork/Kristall/Pfeil respektieren. Wenn ForeverCore fehlt, soll ein klar gekennzeichneter kompatibler Offline-Fallback benutzbar bleiben.

## 2. Bestand: Was wiederverwendet und umgebaut werden muss

Bereits im Repository laut bestehender README/Dokumenten:
- Go-Modul `foreverroutes` mit `route/`, `dataset/`, `xp/`, `fallback/`, `internal/update/`, `internal/merge/` und `cmd/foreverroutes/`.
- CLI-Kommandos `plan`, `replay`, `timeline`, `improve`, `next`, `guide`, `check-quests`, `audit`, `update`, `export-addon`; alle Businessfunktionen von CLI-Eingabe/-Ausgabe entkoppeln.
- `data/audits/timeline-v2.md`: Optimierung und Replay mit Quests, Reisekosten, Dungeons, vorbereitenden Aktionen, Checkpoint sowie Dungeon-Pflichtlisten, z. B. `-require-dungeons ragefire-chasm,ruins-of-lordaeron`.
- `data/sources/dungeons.json`: RF (Ragefire Chasm) und RoL (Ruins of Lordaeron) mit **geschätzten** Level-/Laufzeitdaten; nicht als bestätigte echte Runs darstellen.
- `data/routes-data.json.gz` und `data/sources/lock.json`: Daten und belegbare Revisions-/Quellenbindung.
- `ForeverRoutes/` und `ForeverRoutes/README-HUD.md`: HUD, lokale Grafiken, Pfeil, Questhover, Side-Tasks, Skip/Undo, Crystal; Tests beruhen derzeit auf Mocks, echte Client-Abnahme erforderlich.
- `docs/foreversim-feedback.md`: Rückübertragung von Combat-Feedback ist derzeit **gesperrt**. Nicht beim Windows-Umbau still aktivieren; nur nach den dokumentierten Freigabevoraussetzungen.

**Wichtige bekannte Lücken:** Berufsmaterialien, Geld/Auktionshaus, Berufsslotwechsel, teilweise Buff- und Ausrüstungsmodelle sind noch vereinfacht; XP-/Mob-/Travel-Daten teils Schätzungen. Die vorhandene Suche liefert das beste **gefundene** Ergebnis, keinen Beweis globaler Optimalität. Laufzeiten nie künstlich auf Zielwerte trimmen.

## 3. Eine engine-seitige API statt nutzerseitiger CLI

Die CLI bleibt für Entwicklung/Tests möglich, ist aber nicht länger die Endnutzer-Schnittstelle. Implementiere/exportiere eine stabile Go-API, damit ForeverCore in einem Windows-Prozess mit Context/Cancellation, strukturierten Fortschrittsereignissen und Daten-Snapshots arbeiten kann:

| API (Konzeptname) | Auftrag |
|---|---|
| `LoadDataset(revision)` | Validierten, unveränderlichen Datenstand öffnen |
| `InspectBlocks(context)` | Verfügbare Quests, Dungeon-/Berufs-/Reise-/Vorbereitungsbausteine für Figur und Level anzeigen |
| `GenerateRoute(PlanSpec)` | Aus Pflicht- und optionalen Bausteinen eine vollständige Route berechnen |
| `ReplayRoute(RouteDraft)` | Jeden Schritt vorwärts prüfen, XP/Level/Position/Geld/Inventar/Zeiten verifizieren |
| `OptimizeRoute(RouteDraft, Locks, Budget)` | Zulässige Varianten vergleichen, gesperrte Segmente respektieren, bessere gültige Kandidaten als neue Variante sichern |
| `PreviewMove(Draft, Edit)` | Drag/Drop auf Timeline inkrementell prüfen, Blockaden und Zeitfolgen anzeigen |
| `CompareRoutes(ScenarioSet)` | RXP, Journey, ForeverWisp, Polestar, eigene Varianten unter denselben Annahmen vergleichen, sofern Lizenzen/Importe vorliegen |
| `NextLive(HUDSnapshot)` | Aus Live-Status nur die nächsten sinnvollen Aktionen bestimmen |
| `ExportRoute(ValidatedRoute)` | Zum aktuellen ForeverRoutes-HUD passende, versionierte Addondaten erzeugen |
| `AuditDataset()` | Lücken, Konflikte, Quellen, Modellverfügbarkeit reporten |

**Technische Forderungen:** Kein `os.Exit`, `flag`-Parsing, `stdout` als API oder implizite Dateimutationen im Bibliothekskern. Der CLI-Wrapper in `cmd/foreverroutes/` darf vorhandene Methoden weiterhin aufrufen. Versioniertes Contract-Paket für ForeverCore: `forevercore.contract.v1`, charakter- und datensatzgebundene IDs, Hash/Engine-Revisionsprüfung, typisierte Fehlerrückgaben. Keine zyklische Modulabhängigkeit ForeverRoutes -> ForeverSim-Host; nur geteilte DTOs/Interfaces oder festgelegte Datenaustauschformate.

## 4. Automatischer Route Builder: Auswahl von Bausteinen

Der Nutzer wählt **Volk, Klasse, Startlevel, Ziellevel, Ziele, Pflichten und Optimierungskriterium**. Er muss keine einzelnen Quests von Hand anklicken.

### Baustein-Typen

| Baustein | Inhalt/Regeln |
|---|---|
| `QuestChain` / `QuestHub` | Quest-IDs, Annahme/Abgabe, Ketten, Vorbedingungen, Mob-/Objektziele, Fortschritt, XP, Travel |
| `DungeonRun` | Zutrittslevel/Gruppe, Quest-Sammelroute inkl. Vorquests, Instanzreise, Gruppensuche, Boss-/Trash-/Quest-XP, Abschluss/Abgabe |
| `ProfessionTarget` | Beruf, Start-/Zielskill, Stufen-/Trainerschranken, Skillchancen, Rezeptketten, Materialien, Einkauf/Sammeln, Gold, Craft-/Disenchantzeit |
| `PrepBlock` | Zauberstab, Talente, Klassenquests, Ausrüstung, Schlafsack, Bufffood, Trainer, definierte Mindest-/Maximallevel |
| `TravelBlock` | Gehen, Flug, Schiff/Zeppelin, Ruhestein, Cooldowns, Positionen |
| `GrindBlock` | Mobs, XP/Kill, Kills, Kampf- und Pausenzeiten, Drop-/Respawn-Annahmen |
| `RestBuffBlock` | Schlafsack/Ruhe/XP-Buff mit Erwerb, Aktivierung, Dauer, Ablauf, Cooldown |

Jeder Baustein besitzt **eindeutige ID**, Schematyp, Herkunft, Datenrevision, erforderlichen Zustand, erzeugten Zustand, Kosten und Unsicherheitsgrad. **Modi:** `required` (muss vorkommen), `optional` (darf entfallen), `forbidden`, `locked` (Position/Gruppe bleibt fix), `auto` (Engine entscheidet). Optimierer darf Pflichtbausteine **nicht** still weglassen. Nicht erfüllbare Ziele -> erklärbare Blockade, keine Fake-Erfolgsroute.

Automatische Quest-/Gebietsauswahl ergänzt alle fehlenden XP, Wege und erforderlichen Schritte. Dungeonquests vor und nach dem Lauf korrekt einsammeln/abgeben; nur berechtigte Quests, keine doppelten Belohnungen und kein kostenloses Material. Jeder Kandidat wird anschließend vollständig replayt. Importierte Guides bleiben **eigenständige Eingaben** und dürfen nicht überschrieben werden.

## 5. Referenz-Szenario: Troll Magier, Level 1–20

**Nur Test-/Produktanforderung; keine echt simulierten Resultate, Zeiten, XP oder Goldkosten vorgeben.**

```json
{
  "schema": "forevercore.planspec.v1",
  "character": {"race": "Troll", "class": "Mage", "faction": "H", "startLevel": 1},
  "goal": {"characterLevel": 20, "completion": "all-required-blocks"},
  "requiredBlocks": [
    {"type": "DungeonRun", "id": "ragefire-chasm", "required": true},
    {"type": "DungeonRun", "id": "ruins-of-lordaeron", "required": true},
    {"type": "ProfessionTarget", "profession": "Enchanting", "skill": 225, "required": true},
    {"type": "ProfessionTarget", "profession": "Tailoring", "skill": 225, "required": true},
    {"type": "PrepBlock", "id": "wand-preparation", "required": true}
  ],
  "optimize": {"primary": "minimum-wall-clock-time", "goldBudget": null},
  "preserveImportedGuides": true
}
```

**Definition „fertig“:** beide Dungeons samt zwingenden Questvorbedingungen abgeschlossen, Zauberstab-Vorbereitung erfüllt, Magier erreicht Level 20 **und** beide Berufe haben nach Regelprüfung Skill 225. Falls ein höherer Trainer-/Spielerlevel nötig ist oder die Daten das Ziel unmöglich machen, darf die Engine nicht behaupten, die Route sei fertig. Berufe ggf. vorbereiten, Endskill erst nach tatsächlicher Freischaltung. Keine willkürliche XP-Vorgabe „1.000.000 XP“ statt der gültigen Levelkurve.

Optimierung muss wenigstens diese Varianten vergleichen:
1. Früh/spät mögliche RF- und RoL-Einfügung mit Quest- und Reise-Nebenkosten.
2. Questhubs/Grind-Alternativen bis Erreichen aller Levelvoraussetzungen.
3. Stoff unterwegs sammeln vs. gezielter Farm-/Einkauf-/Trainingsweg; Schneiderprodukte zum Entzaubern benutzen vs. alternative VZ-Materialien.
4. Zauberstab früher/später vorbereiten, dessen Gesamtkosten gegen verbesserte Kill-/Trinkzeiten abwägen.
5. Goldbedarf als Nebenbedingung, wenn Budget vorgegeben; sonst „schnellste Route“ samt realistischem, separat ausgewiesenem Goldverbrauch.

Ergebnisbericht: **Gesamtdauer, Questanzahl, Dungeonanzahl, XP-Zerlegung (Quests/Kills/Dungeons), Berufsskills, Mindest-/Istgold, Reisekosten, Gear-/Buff-Revisionsstand, Datenlücken und Konfidenz**. Bis alles berechnet und validiert ist: „nicht berechnet“ statt erfundener Zahlen.

## 6. Manuelles Timeline Studio und Hybrid-Modus

Dasselbe `RouteDraft`-Format muss **Auto Builder** und **Timeline Editor** bedienen. In Windows unter ForeverCore grafisch, mit frei verschiebbaren Blöcken auf Spuren:
- Quests und Questhubs
- Dungeons und Gruppensuche
- Berufe/Materialien/Trainer
- Reise-/Wartewege
- Kampf, Ausrüstung, Buffs, Zauberstab, Schlafsack

Spuren dienen der Übersicht; die simulierte Weltzeit ist konsistent, Aktivitäten dürfen nur überlappen, wenn sie wirklich gleichzeitig möglich sind. Drag/Drop: `PreviewMove` prüft Level, Reiseposition, Vorquests, Cooldowns, Material-/Geldverfügbarkeit, Questlog, Dungeonlevel, Buffablauf, Berufs-/Trainerzugang. Grün = gültig, Gelb = unsicher, Rot = unmöglich – jeweils mit erklärbarer Ursache und ggf. automatisch vorgeschlagener Verschiebung.

**Hybrid:** automatisch generierte Route öffnen -> Segmente/Questketten manuell verschieben -> einzelne Fixpunkte sperren -> nur übrige Teile optimieren -> vorherige Version jederzeit wiederherstellen. Alle Varianten separat mit Datensatz-/Engine-Hash speichern; alte Routen nicht heimlich überschreiben. Verschiedene Importquellen/Guidevarianten bleiben unterscheidbar.

## 7. Rechenmodell: realistische Zeiten und Ressourcen

Eine Route darf nicht aus frei erfundenen „XP/min“-Angaben oder geschönten Stundenzahlen bestehen.

- **XP:** gültige Levelkurve, Quest-IDs/Quest-XP/Turn-ins, Voraussetzungsketten, Wiederholbarkeit, Levelabstände, Kill-XP/RestedXP/Buffs; kein doppelt gezählter Kill, keine Belohnung bei unbekanntem Ziel. Unbekannte XP als `unknown` markieren, keine implizite Gratis-Quest.
- **Zeit:** Reise, Laufen/Transport/Wartezeit, Questkampf/Kills, Mana- und HP-Regeneration, Loot, Gruppensuche, Instanzzeit, Handwerk/Trainer/Bank/AH, Materialbeschaffung, Buff-Setup, Tod als optionales Risikoszenario. Konservative Schätzungen von bestätigten Messungen unterscheiden.
- **Ressourcen:** echtes Inventar, Questloglimit, Gold- und Item-Bilanz, Berufsslots, Lehrergebühren, Materialien, zufällige Skillpunkte (erwartete Menge + Unsicherheit), Craft-/Disenchant-Verwertung; Kaufpreise regions-/marktbezogen als Annahmen mit Datum.
- **Kampfprofile:** aus MythicSim/ForeverSim versionierte Level-/Gear-/Talent-/Rotation-Snapshots. Änderungen an Zauberstab/Ausrüstung invalidieren nachfolgende Kills; niemals ein Level-60- oder falsches Gegnerprofil still für Level 10 verwenden.
- **Optimierung:** primär minimale **Gesamtabschlusszeit bis alle Pflichtziele erfüllt sind**, nicht nur „frühest Level 20“; Gold bei fehlendem Budget als Bericht, optional Preis-/Gold-Zeit-Pareto-Kurven. Deterministischer Seed, Suchbudget, Fortschritt und geprüfter Checkpoint; gefundene Beste statt behauptetes globales Optimum.
- **Quellen:** QuestieDB, ForeverDB, WarcraftDB, cmangos, weitere belegte Quellen wie heute; Feldprovenienz, Stichtag/Spielbuild, Datenkollisionen und veraltete Modellannahmen in jedem Ergebnis. Patch/Blue-Post ist Auslöser zur Prüfung, kein automatischer Wahrheitsbeleg.

Die neue `ProfessionTarget`-Engine ist ein eigener Meilenstein; der aktuell dokumentierte Berufskostenansatz reicht für Skill 225 nicht. Testfälle müssen die reale Zugänglichkeit der Expertentrainer/Skillränge prüfen, statt einen fixen Level-20-Wert hart in Code zu schreiben.

## 8. Übergabe vom Windows-Editor an das Ingame-HUD

**Installationsweg:** ForeverCore.exe aus dem anderen Repo liefert `ForeverRoutes` als gepacktes, versioniertes Addon mit; findet WoW-Installationspfade, sichert alte eigene Dateien, spielt nach Hash-/Versionsprüfung neue Dateien ein. Kein manuelles Addon-ZIP, keine separaten Shell-Skripte oder `foreverroutes.exe`-Downloads.

**Datenweg:** vollständige validierte Route wird als kompaktes, versioniertes lokales Addon-Datenpaket generiert und bei sicherem Spielzustand installiert; keine gigantischen Routen über Pixelstreifen. Live-Änderungen des Questlogs/Levels/Position werden über `ForeverCoreBridge` als kleine Snapshots zum Host gesendet. Der Host antwortet mit aktuellem Abschnitt, nächste Schritte, Wegpunkt, Warnung und Modellrevision. Addon liest bereits geladene Pakete; für neue Addon-Lua-Dateien ist ggf. ein WoW-Neustart oder `/reload` nötig, Status ehrlich anzeigen.

**HUD MUSS behalten:** unabhängiger Pfeil, nächstes Ziel, Questlog/Mouseover, sichtbare Teilziele, wichtige Side-Tasks, Bag-Warnung, Skip/Undo, Kristall als echte Verbindungsmeldung, Lock/Skalierung und gespeicherte Positionen. Kein großer Routen- oder Berufe-Editor im WoW-Client. Ein kompakter Zeitvergleich „geplant / bisher tatsächlich“ nur bei belastbarem Live-Snapshot, keine Schätzung als Messwert präsentieren.

**Offline-Fallback:** letzter kompatibler bereits geladener, validierter Routenexport oder bestehender datengebundener Guide-Fallback; sichtbare Kennzeichnung veraltet/nicht verbunden. Import-Lizenzen und Attribution behalten. Der Core darf nicht per eigenem Client-Injection-/Memory-Hack oder im Kampf geschützte Aktionen umgehen.

## 9. Code-/Modulgrenzen und Release-Vertrag

ForeverRoutes liefert an ForeverCore mindestens:
- **Go-Library:** stabiler API-Einstieg mit Context/Cancel, Fortschrittsereignissen, immutable World-/Character-State und reproduzierbaren Ergebnissen.
- **Versionierte Daten:** `foreverroutes.dataset.v1` (oder bewusst migrierte Version), Quellenlock, Cache-Hash, Engine-API-Version, Version des WoW-Exports und kompatible Profilrevision.
- **Addon-Payload:** `ForeverRoutes` inkl. TOC, Texturen, Daten, korrekter Attribution, ohne Fremd-/Sim-Addon zu überschreiben.
- **Tests:** Unit/Integrations-/Replay-/Timeline-Tests, deterministische Fixtures, Export-Tests und WoW-Mock-HUD-Tests, plus realer WoW-Client-Abnahmekatalog.

**Besitzverhältnisse:** ForeverRoutes-Repositorium veröffentlicht Bibliothek, Dataset und Export; ForeverSim-Repositorium baut die einzelne Nutzer-`ForeverCore.exe` und installiert beide Addons über einen konsolidierten Release-Bundle. Cross-Repo-Revisionspaar **pinning** statt „neueste zwei Releases“ ohne Verträglichkeitsnachweis. Ein GitHub Actions Job darf einen Bundle-Test/kompatibles Manifest anstoßen; bis dahin alte gültige Pakete beibehalten.

**Manifest (Beispielfelder, kein bereits vorhandenes API):** `coreVersion`, `routesEngineRevision`, `datasetHash`, `simEngineRevision`, `gameBuildRange`, `addonApiVersion`, `routeSchemaVersion`, `assetSha256`, `licenseManifest`. Hash und Inhaltsvalidierung beim Erstellen, Installieren und Laden. Fehler können GUI-seitig behandelt werden, ohne stille inkompatible Dateikopien.

## 10. Umbauphasen

1. **A – Struktur/Audit:** vorhandene CLI-Funktionen, Datenmodelle, Exporte, Profile und allenfalls lizenzierbare Guides inventarisieren. Contract-Tests für identische CLI- und API-Ergebnisse.
2. **B – Library API:** `cmd/foreverroutes` auf kleine Wrapper reduzieren, cancellation/progress/error, reentrant Immutable-Snapshots, versionsgebundene Cache-Invalidierung.
3. **C – Baukasten:** Blockschema, Pflicht-/Verbotsregeln, Quest-/Dungeon-/Professions-Abhängigkeiten, Kosten- und Erreichbarkeitsmodell.
4. **D – Zeitstrahl:** Replay eines manuell veränderten Plans, PreviewMove, Locks, Varianten-Historie, begrenzte Optimierung ohne Mutation von Originalguides.
5. **E – Professionen:** vollständiges 1–225-Modell inkl. Levelgates, Craft-/Disenchant-Ketten, Sammeln/AH, Gold und Skillschwankungen; offene Daten explizit melden.
6. **F – Kampfkopplung:** versionierte MythicSim-Killprofile und Segment-Recalculation, ohne vorzeitig die gesperrte Combat-Feedback-Funktion einzuschalten.
7. **G – Export/HUD:** kompatible Addon-Pakete, aktuelle Schritte und Progress-vs.-Plan in HUD; Offline-Fallback und echte Clienttests.
8. **H – Windows-Bundle:** gemeinsame CI von ForeverSim/ForeverRoutes, gepinntes Dataset, ein Endnutzer-`ForeverCore.exe`-Download, Erstinstall/Reparatur/Update/Offline ohne CMD.

## 11. Abnahmetests / Definition of Done

- [ ] Auto Builder erstellt für Troll Magier 1–20 mit **beiden** Pflichtruns, VZ 225, Schneider 225 und Zauberstab-Vorbereitung einen vollständig replaybaren Plan **oder** eine konkrete, nachvollziehbare Unmöglichkeitsmeldung.
- [ ] Zielzeit umfasst **alle** Pflichtaktivitäten einschließlich Level-20-Berufsabschluss; keine XP-/Gold-/Material-Fiktion.
- [ ] Dungeon-Minimums, Questreihen/Questlog und Reisen korrekt validiert; keine doppelten XP, keine unzulässigen Dungeonläufe.
- [ ] Berufsplanung berücksichtigt Rezept-/Trainerlimits, Erfolge bei Skillups, Materialverbrauch, Kauf/Grind, Kosten und Entzaubern.
- [ ] Windows Auto Builder und manuelle Timeline liefern für dieselbe Ereignisliste dieselbe Replay-Zeit; Locks werden nie verletzt.
- [ ] Änderungen an Gear/Zauberstab/Talenten invalidieren abhängige Kampfergebnisse; identischer Snapshot reproduzierbar.
- [ ] Originalguides (RXP/Journey/ForeverWisp/Polestar soweit legal importiert) bleiben unangetastet und vergleichbar.
- [ ] `ForeverRoutes`-HUD bekommt alle notwendigen kompakt aufbereiteten Live-Schritte ohne große Routenberechnung im Spiel; Crystal zeigt nur echte Bridge-Verbindung.
- [ ] Install/AddOn-Export und Reinstall funktionieren ausschließlich über ForeverCore.exe; kein sichtbares CMD/zusätzliches manuelles Programm.
- [ ] Offline-/Rollback-/unkompatible-Version-Tests bestanden; echte Forever-Clientprüfung für Pfeil, Mouseover, Waypoints, Pixel-/Slotkanal und `/reload`.
- [ ] Cross-Repo-CI validiert gepinnte Engine-, Datensatz-, Addon- und Protokollversionen, Lizenzen und Paket-Hashes.

## 12. Was diese Datei bewusst nicht behauptet

Sie implementiert **keine** Windows-GUI, keine Go-API und keinen installierbaren Build. Sie garantiert weder „perfekte“ Laufzeit noch echte 1–20-Ergebnisse, eine zuverlässige Hot-Swap-Bridge oder vollständige Spieldaten. Solche Aussagen sind erst nach Tests zulässig. Spieler braucht für die installierten WoW-Addons weiterhin Dateien im AddOns-Verzeichnis; nur der **Download und Startpunkt** soll eine ForeverCore.exe sein.

**Umsetzung erst nach separatem Auftrag. Bis dahin bleiben Engine, Branches, Workflows und Routen unverändert.**

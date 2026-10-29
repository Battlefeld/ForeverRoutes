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

**Umsetzung erst nach separatem Auftrag. Bis dahin bleiben Engine, Branches, Workflows und Routen unverändert.** Die ergänzenden Abschnitte 13–21 definieren die technischen Schnittstellen, Datenmodelle, Grenzfälle und Testvorgaben für die spätere Implementierung.

---

## 13. Tatsächlicher Ist-Stand und Datei-/Vertrags-Audit

**Keine Annahme, dass die neuen GUI-APIs schon existieren.** Im `main`-Branch sind nachgewiesen:

| Vorhandener Bestandteil | Umbau für ForeverCore |
|---|---|
| `cmd/foreverroutes/main.go` enthält `build-data`, `audit`, `update`, `plan`, `replay`, `timeline`, `improve`, `next`, `guide`, `bundle`, `export-addon`, `extend`, `check-routes`, `check-quests` | Bestehende Geschäftslogik als Go-Library mit typisierten Parametern/Cancellation verfügbar machen; CLI als Test-/Developer-Wrapper behalten |
| `route/` besitzt bereits `timeline`-/Optimierungs-/Checkpoint-/Dungeon-/Wand-/Live-/Reise-Logik | Bestehende Algorithmen/Tests weiterverwenden; **nicht** alle Routen zu einer gemeinsamen Guide-Datei zusammenführen |
| `data/audits/timeline-v2.md` weist vollständig nachgespielte historische Szenarien aus, nicht automatisch aktuelle, ingame bestätigte Bestzeiten | Ergebnis-Snapshot mit konkreter Modellrevision, Charakter/Profil, Datensatz, Seed, Suchbudget und Replay-Status speichern |
| `data/sources/dungeons.json` beschreibt `ragefire-chasm` und `ruins-of-lordaeron`, Laufzeiten sind laut Quelldatei **Schätzungen** | Dungeon-Komponenten gesondert mit Quest-/Kill-/Travel-/Wartezeit und Unsicherheit ausweisen; echte Zeiten später kalibrieren |
| `ForeverRoutes/` mit `README-HUD.md` und `README-COLLECTOR.md` | Eigenständiges HUD und Quest-Collector **behalten**; nicht durch ForeverSim-HUD überschreiben oder dessen Aussehen erfinden |
| `docs/foreversim-feedback.md` deaktiviert produktive Kampfrückmeldungen bis expliziter Release-/Nutzerfreigabe | Auch die neue Windows-GUI darf diese Sperre **nicht** automatisch umgehen |
| `.github/workflows/release.yml` erzeugt bisher eigenständige Bundles, `.github/workflows/update.yml` verarbeitet Quelldaten | Für gemeinsames Produkt kompatible gepinnte Artefakte/Contracts liefern; kein willkürliches `latest` beider Repos |
| Im ForeverSim-Repo liegt eine ältere `ForeverRoutes/`-Kopie; `ForeverCore/package.py` schließt sie vom Paket aus | **Nur dieses Repository besitzt die aktuelle ForeverRoutes-HUD-Quelle**. CI muss Altdateien im Endnutzerpaket ausschließen |

Der Installer liegt beim **anderen** Repo. Dort dokumentiert `ForeverCore/setup/installer.go.txt` heute noch Konsoleneingabe und PowerShell-Ausführung; erst dessen Ablösung erfüllt den Wunsch „eine ForeverCore.exe, keine CMD/PowerShell“. Dieses Routes-Repo liefert die aktuelle Engines/Addon-Artefakte, aber baut **keine zweite vom Nutzer zu startende Anwendung**.

## 14. Gemeinsamer technischer Datenvertrag (einmal definieren, überall verwenden)

**Architekturregel:** ForeverCore-Windows-Host (ForeverSim-Repo) ist Eigentümer von Installationsmanagement, GUI, Bridge, MythicSim-Host und Auth. ForeverRoutes ist Eigentümer von Datenmodell, World-/Routen-/XP-Engine, Plan-/Optimierungslogik, Routenimporten und Routes-HUD. Eine **einzige** versionierte Schemaquelle (gepinnte JSON-Schemas oder generiertes DTO-Paket) ist notwendig; nicht in beiden Repos auseinanderlaufende Kopien manuell pflegen.

**Vorgeschlagene DTOs (Designnamen, nicht als implementiert ausgeben):**

- `PlanSpec`: `planId`, `character{race,class,faction,startLevel,startXp,knownFlightPaths,...}`, `target{level,requiredCompletedBlocks}`, `required`, `optional`, `forbidden`, `locks`, `optimization{goal,goldBudget,seed,timeBudget}`.
- `ActivityBlock`: `blockId`, `kind`, `sourceRef`, `requiredState`, `producedState`, `prerequisites`, `location`, `costModel`, `durationModel`, `uncertainty`, `provenance`, `dataRevision`.
- `RouteDraft`: eigenständige Route-ID und revisionierte zeitlich sortierte Events mit stabilem `eventId`; `originGuideId` und `originRevision` optional; `edited`-/`locked`-Markierung. **Originalguide niemals direkt mutieren.**
- `ReplayResult`: `valid`, `blockingErrors`, `warnings`, `startState`, `endState`, `events`, `elapsedSeconds`, `xpLedger`, `goldLedger`, `inventoryLedger`, `professionLedger`, `confidence`.
- `TimelineEdit`: Operation `insert/move/split/merge/delete/lock/unlock`, erwartete Route-Revision (optimistisches Concurrency-Control), Eingabe- und Ausgabezustand.
- `CombatProfile`: Character/Gear/Talente/Rotation/Levelbereich/Gegner, `engineRevision`, `datasetHash`, `source=modeled|measured`, Gültigkeitszeitpunkt.
- `HudRouteSnapshot`: routeId, sequence, currentStepId, nächste Schritte, Quest-/NPC-IDs, Zielkoordinaten, Side-Tasks, Zeit-/XP-Vergleich, Daten-/Bridge-Version, Verfalls-/Gültigkeitsstatus.

**Protokollhärtung:** `schemaVersion`, `gameBuild`, `coreVersion`, `routesEngineRevision`, `datasetHash`, `characterRevision` obligatorisch, soweit die jeweilige Nachricht sie benötigt. Ausdrückliche Versionen für neue APIs, eigener Migrationspfad für historische Exporte. Alte `foreverroutes.dataset.v1` nicht kommentarlos überschreiben; zuerst API-Kompatibilität/Reader-/Writer-Migration prüfen.

**Einheiten:** Zeiten intern Integer-Sekunden (keine beliebig gerundeten Minuten), XP und Gold als Integer der kleinsten relevanten Einheit, Wegpunkte mit Koordinatenraum/Map-ID, Level und Berufe als klar getrennte Werte; fehlender Wert = `unknown` statt `0` (ausdrücklich belegte 0-XP-Quests bleiben 0). Alle Zufallsannahmen mit Seed/Verteilung und alle externen Quellen mit Revision/Datum.

## 15. Baukasten-Pflichten, Quests und Dungeon-Abhängigkeiten

Der automatisch erstellte Plan nimmt **Bausteine** an, nicht bloß einen Zeitwunsch:

1. `required` = muss tatsächlich abgeschlossen werden; `optional` = nur verwenden, wenn schneller; `forbidden` = niemals verwenden; `locked` = manuelle Position/Gruppe fest; `auto` = Engine bewertet die Wahl. Bei Konflikt `required` + `forbidden` für dasselbe Objekt **Validierungsfehler**, keine automatische Streichung.
2. Pflicht-Dungeons umfassen grundsätzlich alle zwingenden Vorbereitungen der **ausgewählten** Dungeonquests, Eingang, Gruppenphase, Lauf, Questabschluss und Abgabe. Optionale Dungeonquests sind eigene mit der Dungeon-Aktivität verknüpfte Blöcke. Keine Belohnung für unerledigte/unerreichbare Quests.
3. Quest-IDs und Voraussetzungen werden aus der validierten Quelldatenbank ermittelt; Reihung darf nicht blind nur nach XP/Hr erfolgen. Annahme und Abgabe sind unterschiedliche Ereignisse; Questloggrenze/gleichzeitige Questziele, Levelgates und Positionen zählen.
4. XP-Herkunft ereignisbasiert: `questTurnInXp`, `mobKillXp`, `discoveryXp` **nur wenn modelliert**, `dungeonMobXp`, `restedOrBonusXp` mit klarer Baseline; **kein** separater pauschaler Dungeon-XP-Posten, der dieselben Mobkills erneut addiert.
5. Ein Pflichtbaustein darf nicht verschwinden, nur weil Level 20 früher erreicht ist. Abschlusszeit ist `max(alle Pflichtziele tatsächlich erfüllt)`; Berufe auf 225 werden daher **nach** einem möglicherweise notwendigen Level-/Trainer-Gate beendet.
6. Vorhandene eigene Routen, RestedXP, Journey, ForeverWisp, Polestar usw. sind **unabhängige Kandidaten**, keine vermischte falsche „Original-Route“. Nur lizenzierte/importierbare Inhalte verwenden; Quellen/Attribution annehmen und beibehalten.

**Infeasible-Report statt Fake-Route:** `UNREACHABLE_TRAINER`, `MISSING_QUEST_PRECONDITION`, `REQUIRES_HIGHER_LEVEL`, `UNKNOWN_XP`, `INSUFFICIENT_GOLD`, `MISSING_MATERIAL_SOURCE`, `NO_LEGAL_ROUTE`; mit Block-ID, Quelle, erforderlichem Zustand und konkreter Behebungsoption. Auch bei `goldBudget = null` muss ein Golddefizit auf der Zeitachse finanziert werden; „kein Limit“ heißt nicht „unendlich Gold zu jeder Zeit“.

## 16. Professionen als vollständige Event-Simulation (notwendig für 225)

**Heutige Lücke:** `data/audits/timeline-v2.md` bezeichnet Geld, Auktionshausversorgung, Berufsslotwechsel und Materialmodell als vereinfacht. Das muss durch eine testbare eigene Skill-/Craft-/Ressourcen-Engine ersetzt bzw. ergänzt werden.

**Ereignisse:** `learnProfession`, `trainRank`, `learnRecipe`, `buyMaterials`, `collectMaterials`, `craftItem`, `disenchantItem`, `sellItem`, `visitTrainer`, `visitBank`. Je Event: benötigtes Charakterlevel und Professionsrang, Ort, Reisezeit, NPC/Recipe-ID, Materialmengen und Itemverbrauch, Geldabbuchung, Herstellung/Kanalzeit, mögliche Fehlschläge und Skillpoint-Verteilung.

- **Schneiderei** erzeugt herstellbare Items; deren Ergebnis ist bei passender Regel als Material für **Verzauberkunst/Entzaubern** verwendbar. Material weder doppelt verkaufen noch mehrfach entzaubern.
- Rezept-/Trainerränge, Mindestlevel und Profession-Skill-Gates nach aktuellem Forever-Datensatz modellieren. **Level 20 als Wunsch-Endpunkt, nicht als unbelegte Garantie** für 225 fest verdrahten. Wenn Serverregeln 225 dort verhindern: sauber melden und Alternative vorschlagen, nie bescheinigen.
- Zwei Hauptberufsslots, eventuelle Wechsel/Spezialisierungen und vorhandene Spielerberufe explizit modellieren. Zeit fürs Sammeln und Reisen mit Questaktivitäten nur dann überlappen lassen, wenn dieselben tatsächlich getöteten/geplünderten Mobs/Orte diese Materialien liefern.
- Einkaufspreise nur aus markierter Preisquelle/Szenarioannahme, nie als aktuelle AH-Realpreise ohne Nachweis. Geschätzte Materialkosten mit Bandbreite, Zeitpunkt und Angebotseinschränkungen ausweisen.
- Simulierte Craft-Skillups probabilistisch behandeln: Erwartungswert, Worst-/Best-Szenario oder Monte Carlo mit Seed; bei ungesicherten Raten **Warnung**, keine exakte Dauer ausgeben.

**Abnahme:** Replay einer Route mit Schneider 225 + VZ 225 zeigt für **jedes** Item, wer es beschafft hat, wann es verbraucht wurde, verfügbaren Goldstand und Skillstand nach jeder Aktion. Ein Spieler darf nie mit negativem Inventar oder nicht freigeschaltetem Rezept weiterrechnen.

## 17. Drag-and-drop-Timeline: Bedienvertrag und deterministische Neuberechnung

**UI liegt in ForeverCore (Windows), nicht im WoW-Addon.** Routes liefert den Event-/Fehler-/Diff-Kern.

- Mehrspurige Übersicht `Quests`, `Dungeons`, `Professionen/Material`, `Reisen`, `Combat/Items/Buffs`, `Zeit-/Gold-/XP-Bilanz`. Spuren sind **Darstellung**, nicht erlaubte Gleichzeitigkeit.
- Links Katalog mit Questsuche (ID/Name/Hub), Berufen, Dungeons, Buff-/Vorbereitungsbausteinen; Filter nach Level/Fraktion/Prereqs/Verfügbarkeit. Rechts Details/Abhängigkeiten/Quelle, Kosten und Preview.
- `PreviewMove` muss alle **nachfolgenden** abhängigen Events invalidieren und ab letztem unveränderlichen Checkpoint neu replayen; Level, XP, Questlog, Wege, Inventar, Geld, Trainer, Buffdauer, Dungeon-Gruppenphase und Kampfprofil zurückrechnen.
- Verbindliches Feedback: **grün** gültig, **gelb** angenommen/unsicher, **rot** unmöglich. Vorschläge („Vorquest vorziehen“, „Material einkaufen“, „Trainer nach Levelgate“) ersetzen **niemals** die manuelle Entscheidung ohne Bestätigung.
- Auto-Route -> Timeline -> manuelle Sperren -> selektive Optimierung -> komplette Revalidierung -> als **neue eigene Version** speichern; Undo/Redo, Vergleich „vorher/nachher“, Variantenklonen und Export. Importierte Ausgangsguides nicht überschreiben.
- `CompareRoutes` nutzt dieselbe Engine, dieselbe Datenrevision, dasselbe CombatProfile und dieselben Annahmen; unterschiedliche Starter/Gruppen-/AH-Annahmen klar kennzeichnen. Keine geschönten Zahlen oder zeitlich unvollständige 1–30-Guides als echte 1–30-Vergleiche.
- Performanceziel zunächst **als Messziel** festlegen, nicht behaupten: GUI bleibt beim Verschieben reaktionsfähig; schnelle lokale Preview ohne Full-Optimization, aufwendige Suche im Worker mit Abbrechen/Fortschritt und Checkpoint.

## 18. Bridge, Quest-Collector und HUD – keine vorgetäuschte Live-Synchronisierung

Das Dokument `ForeverRoutes/README-COLLECTOR.md` nennt aktuell den Client-Collector (`C_QuestLog.RequestLoadQuestByID`, nur Build 1.60.x), kurz übertragene Questtitel während freier Verbindung und gespeicherte Beobachtungen in `WTF/Account/.../SavedVariables/ForeverRoutes.lua`. Diese Quelle erhält Titel und verfügbare Ziele, **nicht automatisch** vollständige XP, Questvoraussetzungen, NPC-Drop-Raten oder sämtliche Clientdaten.

**Drei getrennte Wege:**
1. **Große Routen-/Questpakete:** ForeverCore baut komplette `Data/Routes.lua`/Addon-Manifestdaten offline, prüft Hash/TOC/Schema und installiert sie sicher, wenn WoW nicht aktiv ist; neue Dateien wirken erst nach wirklich erforderlichem WoW-Reload/Neustart. Status `STAGED`, `INSTALLED`, `RESTART_REQUIRED`, `ACTIVE` nicht verwechseln.
2. **Kleine Live-Aktualisierungen:** erlaubter `ForeverCoreBridge`-Pixel-/Antwortslotkanal für Questfortschritt, nächstes Ziel, Kristall, Warnung; 2.048 LoadOnDemand-Slots pro Sitzung und mögliche Capture-Probleme als echte Grenze behandeln. Kein Versprechen einer grenzenlosen bidirektionalen Datei-Live-API.
3. **Beobachtungen:** Client-Collector-Daten bei zulässigen Ereignissen oder gesicherten SavedVariables-Schreibzeitpunkten importieren; nach Build/Region/Sprache trennen, unbekannte/ungesicherte Einträge im Rohbestand behalten, keine unvalidierten XP in Routen übernehmen.

**Live-Status-Vertrag:** `routeId`, `stepId`, `sequence`, `sessionId`, `schemaVersion`, `gameBuild`, `datasetHash`, `lastConfirmedQuestEvent`, `progressActual`, `progressPlanned`, `offlineFallback`. Stale/inkompatible/aus fremder Session kommende Antworten verwerfen. Der blaue Kristall steht nur für die **wirklich bestätigte Core-Verbindung dieser Sitzung**. Ist-vs.-Soll-Zeit ohne tatsächliche Start-/Messpunkte nicht als gemessene Pace ausgeben.

**Erhalt des bestehenden HUD:** Pfeil, kompakter Quest-/Zieltext, native Tracker-Mouseover, NPC-/Target-Hinweise **nur mit belegter NPC-ID**, Händler-/Nebenaufgaben, Dungeon-Hinweise, `Skip`/`Undo`, `Ziel erreicht`, einfache Zustandsanzeige, Platzwarnung, Lock/Position/Skalierung. Alle komplexen Editor-/Kosten-/Routenentscheidungen gehören nach Windows. Keine vollständige Bildschirmkopie von Journey/RXP und keine unausführbaren Ingame-Eingriffe.

**Rückfallebenen:** (a) Live-Route gültig -> aktuelle Schritte; (b) Host weg -> letzte kompatible installierte Route mit Warnung; (c) Paket fehlt/veraltet -> belegter eigenständiger Guide-Fallback, sofern passender Import/Attribution vorhanden; (d) unvereinbare Zustände -> Plan pausieren und Fehler zeigen, statt falschen Schritt vorzugeben.

## 19. Release und automatische Installation durch die eine ForeverCore.exe

**Kein eigenständiger Nutzer-Installer für dieses Repo.** ForeverRoutes liefert `route`-Bibliothek, Quelldatensatz, Datenpaket und **aktuelles** HUD. Finale `ForeverCore.exe` wird im ForeverSim-Repo gebaut und installiert die beiden HUDs + CoreBridge automatisch.

Pipeline-Übertragungsvertrag:
- `routesCommit`, `routesApiVersion`, `datasetHash`, `hudVersion`, `addonTocInterface`, `routeExportSchema`, `gameBuildRange`, `simEngineRevision`, `licenseManifest`, je Datei SHA-256 plus validierte Signatur-/Manifest-Herkunft.
- `ForeverRoutes`-Build **als Teil der gemeinsamen Release-Matrix**; kein zufälliges Herunterladen des aktuellsten inkompatiblen Routenbundles. Alter ForeverSim-Ordner `ForeverRoutes/` ist *nicht* autoritativ.
- Lizenzprüfung explizit: GPL-3.0-abhängige Datensätze, CC-BY-NC-SA-RXP-Guides, MIT-Code und sonstige Third-Party-Assets/Guide-Importe können unterschiedliche Weitergabepflichten auslösen. Inhalt nur mit geklärten Vertriebsrechten, Autorenhinweisen, erforderlichen Quellen/Notices paketieren. Fremde proprietäre Guides nicht einfach mitverteilen.
- Vollständige Release-Verifikation: Prüf-Import, Rebuild gegen `data/sources/lock.json`, Replay und Klassen-/Rassenmatrix, Route-Diff, Export nach Addon, Mock-Lua-HUD-Tests, echter WoW-Client-Test, Windows-Erstinstallation, GitHub-Releasemanifest und Rollback.
- Auto-Update-Weg respektiert private GitHub-Rechte; ForeverCore bietet Browser-/Token-Authentifizierung grafisch **nur bei berechtigtem Konto**, speichert Credentials sicher und zeigt 401/403/Rate-Limit/Offline getrennt. Kein Developer-Token in Nutzerversion.
- WoW beim Update aktiv -> sicher stagen, kein live behauptetes Dateihotswap; `RESTART_REQUIRED` korrekt anzeigen. Installer löscht keine fremden Addons und keine SavedVariables.

Die bestehende `.github/workflows/release.yml` bleibt bis zum geplanten Umbau unverändert. Das neue One-EXE-Release erst nach vollständiger Cross-Repo-Integration freigeben.

## 20. Erweiterte Testmatrix und Qualitätsgrenzen

| Testgruppe | Konkrete Testfälle und Akzeptanz |
|---|---|
| **Pflichtbausteine** | RF und RoL müssen im Troll-Magier-Szenario enthalten bleiben; fehlende Quest-/Dungeonvoraussetzungen -> Blockade mit Begründung, niemals stilles Entfernen |
| **Level-/Berufsgates** | VZ/Schneider 225 als echter Endzustand nach Trainer-/Rezept-/Levelcheck; unmögliche Kombination => `INFEASIBLE` |
| **Ökonomie** | Gold/Inventar nie negativ; kein Item doppelt verbraucht, verkauft oder entzaubert; Skillup-Verteilung und Quellen für Preise sichtbar |
| **XP/Kampf** | Kill-/Quest-/Dungeon-XP genau einmal, Levelkurve prüfbar; Gear/Wand/Talente ändern nur abhängige Segmentprofile |
| **Zeit** | Reise-/Gruppensuche/Kampfpause/Trainer/Craft inklusive; keine nicht erlaubte Überlappung; deterministischer Replay im selben Snapshot |
| **Auto vs. Timeline** | Gleicher `RouteDraft`-/`ReplayResult`-Vertrag; Drag-/Lock-/Undo führt nicht zu heimlichem Quest-/Pflichtverlust |
| **Guides** | RXP, Journey, ForeverWisp, Polestar nur als echte getrennte lizenzierte Imports; keine Fake-1–30-Ergebnisse durch unvollständige Kapitel |
| **Daten** | fehlende XP/Koordinaten/Questvoraussetzungen, veraltete Source-Locks, Patch-Konflikte/403 -> Warnung oder Blockade; keine stillen 0-Werte |
| **Live/HUD** | WoW-Mock und echter Client: Questfortschritt, Skip/Undo, Kristall, Tracker, Mouseover, Pfeil, Dungeonruhe, Offline-Fallback, Session-/Slot-/Reload-Limit |
| **Windows** | GUI erzeugt/optimiert/exportiert Route und installiert Addon via die **eine** `ForeverCore.exe`; keine CMD/PowerShell/zweite Nutzer-EXE; Rollback nach unterbrochenem Update |
| **CI/Release** | Gepinnte Revisionen, Daten-/Engine-/HUD-Hash, Lizenz- und Signaturcheck, End-to-End-Tests; unabhängige GitHub-Fails nie als erfolgreich darstellen |

**Teststufen:** `unit` (reine Rechenregeln), `contract` (CLI vs. Library, Sim-Profil-Schema), `replay` (jede Planaktion und Ressourcenbilanz), `integration` (Cross-Repo), `desktop-e2e` (UI/Installer), `live-client` (WoW). Ein Mock-Test ist **kein** Ersatz für letzten Schritt.

## 21. Nicht implementiert / offene Entscheidungen für das erste Development-Issue

Folgende Punkte sind derzeit **ausdrücklich offen** und dürfen ohne Messung nicht als fertig in README, Dashboard oder Release Notes stehen:
- Allgemeingültige optimal schnelle 1–30-Routen für jede Klassen-/Rassen-/Guide-Kombination; unvollständige Guides bleiben unvollständig.
- Vollständig echte Forever-Quest-/Mob-/Dungeon-/Berufs-/AH-/Hotfixdaten und belastbare Gold-/Zeitwerte.
- Vollständiger Live-Austausch von großen Daten während laufendem Spiel; Bridge-/Addon-/Reload-Limits gelten.
- Ausreichende Routen-/Berufs-Solver-Qualität bei großen Suchräumen, echte Windows-GUI-Performance und reproduzierbare Cross-Repo-Releases.
- Endgültiges professionelles Theme/Branding (Original-Forever-Assets, blaues „ever“, vorhandenes HUD/Kristall; keine Fantasie-Assets); Windows-UX wird nach den Engine-Verträgen finalisiert.
- Ausdrückliche Freigabe der derzeit gesperrten produktiven Combat-Feedback-Pipeline.

**Abschlusskriterium:** ForeverCore erzeugt aus den gewählten Pflichtbausteinen eine **nachgespielte, ressourcen- und regelkonforme** Route oder einen nachvollziehbaren Unmöglichkeitsbericht; derselbe Stand ist im Windows-Timeline-Editor modifizierbar und nach geprüftem Export im eigenständigen ForeverRoutes-HUD sichtbar. Keine Konsoleninteraktion auf Nutzerseite. Die offizielle Windows-`ForeverCore.exe` bleibt einziger Download-/Startpunkt.

**Dieses Dokument ist eine vollständige Ziel- und Prüfspezifikation, keine Behauptung einer bereits fertigen Implementierung.**

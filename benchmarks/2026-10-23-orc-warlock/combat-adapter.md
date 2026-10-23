# Isolierte Hexenmeister-Kampfanbindung

Stand: 24.10.2026. Implementierter Diagnoseadapter; keine validierte 1–30-Bestzeit.

## Schutz von ForeverSim

Alle Änderungen liegen in ForeverRoutes. ForeverSim wird ausschließlich gelesen. Die fünf Quelldateien sind auf `Battlefeld/ForeverSim@2083b58935fc17dfae86fb4b188a4659bd0c845c` festgehalten. Das Quellenpaket unter `vendor/foreversim-2083b589` enthält ihre exakten Bytes einschließlich Lizenzhinweis. Vor jedem Lauf werden SHA-256 und die ursprünglichen Git-Blob-Hashes geprüft. Ein geänderter oder unvollständiger Stand wird abgewiesen.

Ein separater Lua-Prozess lädt temporäre, schreibgeschützte Kopien. Der Routenadapter stellt daneben einen eigenen Einstieg mit Ressourcenweitergabe bereit. Die Originaldateien und der Originaleinstieg `P.run` werden nicht geändert. Rangwahl, Schaden, Treffer und Procs stammen aus dem vorhandenen Code. Die Zusatzfunktion wird nur im isolierten Prozess mit überprüften Ankern aus dem gepinnten Code erzeugt.

**Wichtige Abgrenzung:** Verwendet wird die vorhandene Lua-Referenzroutine. Die produktive ForeverCore-/MythicSim-EXE wird nicht verändert oder angesteuert. Eine Ergebnisgleichheit mit dieser EXE ist noch nicht nachgewiesen. Die bewiesene Gleichheit bezieht sich auf den ursprünglichen Lua-Einstieg bei identischen Eingaben und frischem Start.

## Was funktioniert

- Echte, gelernte Zauberränge und explizite Rotationen bestimmen den Kampf statt des Polestar-Klassenfaktors.
- Der verbleibende Mana-/Lebenszustand, Abklingzeiten, globale Abklingzeit, Besitzer-Buffs und Proc-Sperren werden an den nächsten Gegner weitergegeben. Ressourcenmaxima werden dabei nicht auf den Restbestand verkleinert.
- Begleitermana und nächste Begleiteraktionen bleiben erhalten. Begleiterleben und Bedrohung sind weiterhin nicht abgebildet.
- Reise-/Wartezeit lässt Ressourcen entsprechend den vorhandenen Regeln regenerieren. Zusätzliche Erholung braucht explizite Raten und Dauer; die ganze Dauer wird bezahlt.
- Ein nicht getöteter Gegner oder Spielertod bleibt ein bezahlter Fehlschlag. Es gibt kein kostenloses Auffüllen.
- Rotationsversuche können dieselben Seeds verwenden. Alle Fehlschläge bleiben erhalten; ausgegeben werden Median und 10.–90. Perzentil. Ein diagnostisch schnellster Rotationskandidat ist kein Routensieger.
- Der optionale Routenmodus verwendet die Kampfergebnisse direkt für Questarbeit und Kill-XP. Klassenfaktoren und pauschale Grindraten dienen dabei nicht als Fallback. Für unbekannte Gegner, fehlende Profile oder unbekannte Grindorte hält der Lauf mit einem konkreten Blocker an.

## Ausführen

Node 24, Lua 5.1 und für Routentests der festgehaltene Polestar-Checkout werden benötigt. `LUA` wählt den Interpreter. Aus diesem Benchmarkordner:

```sh
node scripts/run_combat_probe.cjs --verify-source
LUA=/pfad/zu/lua POLESTAR_ROOT=/pfad/zu/polestar node --test tests/*.cjs
LUA=/pfad/zu/lua node scripts/run_combat_probe.cjs --input kampffolge.json --output bericht.json
LUA=/pfad/zu/lua node scripts/run_combat_probe.cjs --compare-choices --input rotationsversuch.json --output vergleich.json
node scripts/run_combat_probe.cjs --audit-route-input --input kampfdaten.json --output abdeckung.json
LUA=/pfad/zu/lua POLESTAR_ROOT=/pfad/zu/polestar node scripts/run_route_comparison.cjs --saved-inputs --class Warlock --combat-input kampfdaten.json --require-complete --output-dir .comparison/stateful-warlock
```

`kampffolge.json` enthält eine Ereignisliste `events`: `fight` mit `player`, `enemy`, `choice`, `seed`, optional `duration`; `idle` mit `seconds` und optionalen Erholungsraten. Ein Rotationsversuch enthält `job` mit einer solchen Folge und `choices` mit eindeutiger `id` und `rotation` je Kandidat. Es werden bewusst keine erfundenen Spielerdaten als ausführbarer Standard mitgeliefert. Die Softwaretests nutzen ausdrücklich synthetische Eingaben.

Für die Route ist die Eingabestruktur:

| Feld | Inhalt |
|---|---|
| `schema` | `foreverroutes.combat_input.v1` |
| `profile` | Klasse und Volk, identisch mit dem Routenprofil |
| `levels[L].source` | Herkunft der Daten für diese Stufe |
| `levels[L].player` | Klasse, Stufe, gelernte Ränge, Talente, Ausrüstung und explizite Charakterwerte |
| `levels[L].player.stats` | health, mana, baseMana, intellect, spirit, spellPower, hastePct, hitPct, critPct und manaRegen |
| `levels[L].choice.policies` | Erlernte Fähigkeiten in der zu prüfenden Priorität |
| `levels[L].recoveryBeforeFight` | Optional: source, seconds, rates mit manaPerSecond/healthPerSecond |
| `enemies[NPC-ID]` | source, level, elite, health und encounter mit armor, targetResist, levelResistance=false, incomingDamage, hitEvery |

Aktive Begleiter benötigen außerdem explizite Stufe, Mana, Manaregeneration und gelernte Zauberränge. Quellenangaben werden dokumentiert, noch nicht automatisch als echte Spielmessung verifiziert. Lehrerfreischaltungen, Gegenstandsprogression und Verbrauchsvorräte brauchen weitere Routenprüfung.

## Eingaben gezielt beschaffen

```sh
POLESTAR_ROOT=/pfad/zu/polestar node scripts/run_route_comparison.cjs --saved-inputs --class Warlock --extend-to-30 --collect-combat-requirements --output-dir .comparison/combat-requests
```

Dieser reine Beobachter lässt die bisherigen Modellzeiten, XP und Fehler unverändert. Die Ergebniszeilen enthalten `combatRequirements` mit Gegner-IDs, konservativen Levelbereichen und Grindanforderungen. Es handelt sich um eine Anforderungsliste aus einem Diagnosemodell, nicht um aufgezeichnetes Spiel. Original-/Ergänzungsanteile bleiben getrennt.

## Grenzen für eine Freigabe

Die gepinnte Schnittstelle gibt Level 1–4 nicht frei; der Adapter umgeht das nicht. Level 5–9 bleiben wegen angenäherter Klassendaten markiert. Für 10–30 bestehen Softwareprüfungen, aber noch keine Validierung gegen echte Levelruns. Haustiergesundheit/Bedrohung, Besitzer-Waffentimer zwischen Gegnern, buffabhängige Regeneration und Verbrauchsvorräte sind offen. Die Klassendaten entsprechen dem festgehaltenen Stand, nicht automatisch einem späteren Spielpatch.

Erwartete Bruchteile von Kills aus dem alten Drop-/Zusatzkillmodell werden im optionalen Modus auf ganze bezahlte Kämpfe aufgerundet und als Annahme protokolliert. Das ersetzt keine geprüfte Drop-/Spawn-Verteilung. Der ursprüngliche Umgang der Referenzengine mit vorab bezahlten und beim Zielsterben abgebrochenen Zaubern bleibt ebenfalls ein Validierungspunkt.

Deshalb bleiben `validated` und die Routensiegerfreigabe aus. Fehlende Daten werden weder durch Wunschzeiten noch durch einen alten Klassenfaktor ersetzt. Der bisherige Diagnosemodus bleibt zum Vergleich verfügbar und wird weiterhin als unkalibriert ausgewiesen.

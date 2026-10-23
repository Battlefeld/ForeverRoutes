# Klassenvergleich 1–30

Zwischenstand: Alle neun Klassen können in einem Durchlauf gegen alle vier Quellen ausgeführt werden. Mit ausdrücklich generierten Fortsetzungen erreichen alle 36 Kombinationen Level 30. **Ein belastbarer schnellster Weg mit validierten Zeiten ist noch nicht implementiert.**

Die Referenzprofile sind Krieger/Orc, Paladin/Mensch, Jäger/Orc, Schurke/Orc, Priester/Troll, Schamane/Orc, Magier/Troll, Hexenmeister/Orc und Druide/Taure. Verglichen werden vier Routen innerhalb desselben Profils. Der Runner unterstützt zusätzlich die in `scripts/class_profiles.cjs` geprüften Volk/Klassenkombinationen per `--race` und `--class`; die Matrix prüft jeweils ein Referenzvolk pro Klasse.

## Ausführen

Node 24 und ein Polestar-Checkout auf `eb7b47840f032bf711a3c8d7b1df9f3f6551b990` werden benötigt. Aus diesem Ordner:

```sh
# Alle Originalquellen: fehlende Kapitel bleiben sichtbar.
POLESTAR_ROOT=/pfad/zu/polestar node scripts/run_class_matrix.cjs --output-dir .comparison/class-matrix/originals

# Technischer Probelauf aller 36 Kombinationen bis 30.
POLESTAR_ROOT=/pfad/zu/polestar node scripts/run_class_matrix.cjs --diagnostic-continuations --require-target --output-dir .comparison/class-matrix/diagnostic

# Belastbare Sieger verlangen. Aktuell erwarteter Exitcode 2.
POLESTAR_ROOT=/pfad/zu/polestar node scripts/run_class_matrix.cjs --require-validated --output-dir .comparison/class-matrix/validated

# Einzelprofil mit eigener Kapitel- und Klassenwahl.
POLESTAR_ROOT=/pfad/zu/polestar node scripts/run_route_comparison.cjs --saved-inputs --class Paladin --race Human --extend-to-30 --output-dir .comparison/paladin

POLESTAR_ROOT=/pfad/zu/polestar node --test tests/*.cjs
```

`--require-target` prüft nur, ob alle Kombinationen das Ziel im Modell erreichen. Das bestätigt keine vollständige Route und keine plausible Spielzeit. `--require-validated` lässt fehlende Kapitel, ausgelassene Aktionen, ungelöste Bedingungen oder unvalidierte Klassenzeiten nicht als Sieger durch. Prozessfehler ergeben Exitcode 1; ein nicht bestandenes fachliches Gate ergibt Exitcode 2.

In `matrix.json` stehen alle 36 Zellen einschließlich Fehlern, Quellabdeckung, erreichter Stufe, Zeitannahmen und Kostenbuch. Die Unterordner enthalten die vollständigen Aktionsfehler und Bedingungen. Laufzeiten sind ausschließlich Diagnosewerte; `winner` bleibt aktuell `null`.

## Daten und Nachvollziehbarkeit

Beide Fraktionen benutzen gespeicherte Weltmodelle aus denselben Questie-, Quest- und Geometriedaten. Der Runner wählt Klassen- und Volkzweige aus dem vollständigen Guidekatalog; er skaliert keine kopierte Hexenmeisterroute. Klassenbedingungen gelten vor dem Ausführen von Bewegung und Quests.

`scripts/build_matrix_worlds.cjs` baut die gespeicherten Modelle aus den geprüften Checkouts neu. Dafür zusätzlich `QUESTIEDB_ROOT` auf `1f538335cfbb240c14a97c340f48c555fdc8a52b` setzen und den gebündelten Lua-Interpreter über `LUA` angeben. Der normale Matrixlauf benötigt diese Aufbereitung nicht.

Das Kostenbuch erfasst tatsächliche Differenzen der Enginezeit je Aktionsart und getrennt nach Quellroute/generierter Fortsetzung. Es prüft Gesamt-, Reise- und Grindsekunden auf vollständige Zuordnung. Es erfindet keine Dauern für nicht ausgeführte Aktionen. Die Zeit beim Erreichen einer Stufe (`timeTo`) kann vor dem pauschalen Levelaufstiegsaufwand liegen; `endSeconds` und Kostenbuch enthalten auch diesen Aufwand.

## Offene Voraussetzungen für einen Sieger

- Originale RestedXP-/Wisp-Kapitel fehlen in den gelieferten Quellen. Generierte Fortsetzungen bleiben gesonderte Routenvarianten.
- Vollständige Ausführung von Lehrer-, Händler-, Inventar-, Transport-, Todes- und Skriptaktionen sowie dynamischen Journey-Bedingungen fehlt teilweise. Bekannte Lücken stehen in jeder Ergebniszelle.
- Kampf und Erholung basieren weiterhin auf groben Klassenfaktoren. Es fehlen integrierte Messungen beziehungsweise validierte Mechaniken für Fähigkeiten, Ränge, Ausrüstung, Talente, Begleiter und Ressourcen von 1–30. Ein beliebiger Parameter-Fit oder ein bestandener Softwaretest ersetzt diesen Nachweis nicht.
- Die freigegebenen Null-XP-Annahmen bleiben Annahmen. Unbekannte Zielmengen und Drop-Raten bleiben gesondert zu prüfen.

Die Matrix ist damit eine reproduzierbare Prüfgrundlage für das gewünschte Ergebnis, keine abgeschlossene Geschwindigkeitsmessung. Der aktuelle belegte Stand steht in `../../simulation-results/2026-10-23-class-matrix.md`.

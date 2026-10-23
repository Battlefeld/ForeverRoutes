Aktueller Stand für alle Klassen: [class-matrix.md](class-matrix.md). Der folgende Bericht dokumentiert den früheren Orc-Hexenmeister-Lauf.

# Levelrouten: erster gemeinsamer Simulationslauf, 23.10.2026

**Status: vier diagnostische Routen-Replays abgeschlossen. Eine belastbare Rangliste ist noch blockiert.**

Profil: Horde, Orc, Hexenmeister, solo, ohne AoE und Rested-XP. Zielpunkte: Level 6, 12, 20 und 30. Alle Quellen laufen mit derselben Reise-, Kampf- und XP-Berechnung aus Polestar. Die 160 freigegebenen Null-XP-Annahmen bleiben als Annahmen gekennzeichnet. Explizite Nullwerte erhalten im Simulator keine ersatzweise geschätzte Quest-XP.

| Route | Import reicht bis Level | Im Diagnose-Replay abgegeben | Im Solo-Weltmodell nicht auflösbare Quest-IDs | Quests mit offenen Zielmengen | Offene Bedingungen |
|---|---:|---:|---:|---:|---:|
| RestedXP Forever (ohne AoE) | 22 | 164 | 10 | 11 | 0 |
| Journey 4.00.02 | 30 | 181 | 8 | 3 | 2 |
| ForeverWisp 0.32.20-beta.1 | 20 | 155 | 11 | 9 | 0 |
| Polestar default | 60 | 228 | 2 | 10 | 0 |

Die Questzahlen sind Ausführungsdiagnostik, keine Speed-Wertung. Nicht auflösbare IDs umfassen auch Instanz-, Gruppen-, Berufs- und unplatzierbare Schritte. Sie bedeuten nicht automatisch, dass die Quest in der Quelldatenbank fehlt.

## Was der Lauf belegt

- Die vier importierten Quellen wurden unter einem gemeinsamen Modell tatsächlich ausgeführt.
- Null-XP-Werte bleiben auch im zugrundeliegenden Simulator null. Das wird mit den echten spielbaren Questdaten geprüft.
- RestedXP endet hier bei 22 und ForeverWisp bei 20; es gibt keine künstlich verlängerte Zeit bis Level 30.
- Die Rohresultate und präzisen Fehlerlisten stehen in `.comparison/results/`.
- Fünf gezielte Prüfungen bestehen: Klassen-/Volkbedingungen, dynamische Questzustände, unbekanntes Inventar, Null-XP-Behandlung und Teilrouten.

## Warum noch kein Sieger ausgewiesen wird

Die numerischen Zeiten in den JSON-Dateien sind ausdrücklich **Diagnosewerte**: fehlende Ziele, ausgeschlossene Schritte und ungelöste Bedingungen können Routen verkürzen oder verändern. Deshalb steht `rankingPublished` auf `false`. Kein dortiger Zeitwert ist ein validierter /played-Vergleich.

Zusätzlich fehlen die genaue Ausführung von Todesabkürzungen, einzelnen Händler-/Lehrerwegen, Geld-/Inventarbedingungen und teils parallelen Zielen. Journey benötigt noch zwei Inventarbedingungen. Bei RestedXP werden die Klassenverzweigung über Tirisfal/Silverpine und explizite XP-Checkpoints berücksichtigt. Die CSV von Journey enthält keine vollständigen Grind-/Reisebefehle.

Die Wegberechnung verwendet veröffentlichte Kartenrahmen und Transportdaten des festgehaltenen Polestar-Standes. Unvermessene Fußwege und unbekannte Drops benutzen dessen offengelegte Modellparameter; die Kampfzeiten sind Klassenannahmen und keine ForeverSim-Integration. Der Liveabruf der Wago-Kartentabellen war mit HTTP 403 gesperrt, deshalb wurde die bereits veröffentlichte Kartengeometrie aus den generierten Addondaten verwendet.

## Festgehaltene Quellen

- ForeverRoutes-Datenstand: `52e4996bee4dc84189134c4dd545eebe5e9e1fd3` (`artifacts` vor diesem Lauf), mit den gespeicherten XP- und Beobachtungs-Ergänzungen.
- Gemeinsame Engine und Polestar-Route: `fitzforge/polestar@eb7b47840f032bf711a3c8d7b1df9f3f6551b990` (GPL-3.0).
- QuestieDB Forever: `1f538335cfbb240c14a97c340f48c555fdc8a52b`.
- RestedXP: `9536b5a9ee7e55745618a57dad43df2c4312ea16`, Standard-Forever-Guides ohne AoE.
- Journey: importierte CSV aus `Journey-4.00.02.zip`.
- ForeverWisp: importierte CSV aus `0.32.20-beta.1`.
- Classic-Zielmengen: festgehaltener CMaNGOS-Stand `22b51464f1625f6ef6275771de1f5466c6f5d19e`; Forever-Ziele behalten offene Mengen, sofern nicht belegt.

## Gespeicherten Lauf erneut ausführen

Node 24 oder neuer und ein lokaler Polestar-Checkout genau am oben genannten Commit werden benötigt. Im Ordner dieses Berichts:

```sh
POLESTAR_ROOT=/absoluter/pfad/zu/polestar node scripts/run_route_comparison.cjs --saved-inputs
POLESTAR_ROOT=/absoluter/pfad/zu/polestar node --test tests/test_route_comparison.cjs
```

Der gespeicherte Weltzustand und die normalisierten Routenereignisse reichen dafür aus; die ursprünglichen Guides und großen Rohdaten werden nicht erneut geladen. `prepare_sim_inputs.cjs` und `prepare_guide_events.py` dokumentieren die ursprüngliche lokale Aufbereitung und benötigen für einen Neubau die genannten Quelldateien.


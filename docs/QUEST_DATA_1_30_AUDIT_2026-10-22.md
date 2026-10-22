# Questdaten-Audit Level 1–30 (22.10.2026)

**Dies ist ein Quellen- und Integritätsaudit, keine vollständige Validierung aller Forever-Questbelohnungen.** Keine Route wird hier zeitlich bewertet.

## Maschinell vollständig geprüfte Rohdaten

- Quelle: [QuestieDB Forever](https://github.com/Questie/QuestieDB/blob/f844007b329b72f18cf2bda3d87540012294732a/data/Forever/foreverQuestDB.lua), gegen [QuestieDB Classic](https://github.com/Questie/QuestieDB/blob/f844007b329b72f18cf2bda3d87540012294732a/data/Classic/classicQuestDB.lua) und [XP-Datei (explizit xpDB-classic.lua)](https://github.com/Questie/QuestieDB/blob/f844007b329b72f18cf2bda3d87540012294732a/support/Forever/QuestXP/xpDB-classic.lua).
- Festgehaltene Revision: `f844007b329b72f18cf2bda3d87540012294732a`.
- Beide Quest-Dateien: jeweils 4.244 IDs, identische Quest-ID-Mengen. Forever enthält **keine zusätzlichen Quest-IDs** gegenüber der Classic-Rohdatei.
- 1.413 Datensätze mit Queststufe 1 bis 30 (das ist nicht gleich der Zahl aller annehmbaren Quests bis Level 30). Für 1.312 dieser Datensätze enthält die XP-Referenztabelle einen Zahlenwert; **101 haben dort keinen Eintrag**, was auch XP-lose oder Spezialquests umfassen kann.
- 1.394 mit hinterlegtem Starter; 1.390 mit hinterlegtem Abgeber. Leere Felder können absichtlich (z.B. automatische Quests) sein.
- 1.414 Starterreferenzen und 1.392 Abgeberreferenzen sind alle im jeweiligen Forever-Referenz-NPC-, Objekt- oder Item-File vorhanden (rein formale ID-Prüfung). 803 Questvoraussetzungs-Verweise haben ebenfalls keine fehlende Ziel-ID.
- Unterschiede Forever gegen Classic: 2.491 geänderte `requiredRaces`-Masken und **nur 3** geänderte `triggerEnd`-Felder. Keine anderen Quest-Datenfelder unterscheiden sich in dieser Revision. Die Race-Masken können auf Forever-Rassen erweitert worden sein; das ist kein Nachweis, dass sonstiges Gameplay unverändert ist.
- Das neue Beta-Quest `97279` fehlt in QuestieDB. Eine realistische Forever-1–30-Routenberechnung darf daher diese Datenbank nicht als alleinige Quelle nutzen.

## Beobachtete echte Forever-Daten bei ForeverDB (gezielte Gegenprobe; **nicht** für alle 1.413 Quests validiert)

| ID | Quest | XP-Beobachtung (Level) | Beleg |
|---|---|---|---|
| 788 | Cutting Teeth | 170 (Level 1, 2) | https://foreverdb.net/quest/788-cutting-teeth |
| 97279 | Wayward Weapons | 170 (Level 2, 3) | https://foreverdb.net/quest/97279-wayward-weapons |
| 248 | Looking Further | 1.300 (Level 25) | https://foreverdb.net/quest/248-looking-further |
| 387 | Quell the Uprising | 5.550 (Level 24) | https://foreverdb.net/quest/387-quell-the-uprising |
| 325 | Armed and Ready | Nicht als Forever-XP aufgezeichnet | https://foreverdb.net/quest/325-armed-and-ready |

ForeverDB markiert Felder pro Quelle: `seen in Forever`, `game quest map` oder `Classic reference`. Quelle: https://foreverdb.net/about und https://foreverdb.net/quests.

## Quellenhierarchie für ForeverRoutes

1. **Serverbeobachtetes Forever-Questfeld:** pro Feld speichern: ID, Wert, Beobachtungslevel, Beta-Build, Anzahl unabhängiger Spieler, Quelle. Erfahrungspunkte ggf. stufenabhängig auswerten; Beobachtung auf Level X ist nicht automatisch ein Wert für jedes Spielerlevel.
2. **Direkte Forever-Clientdaten:** Quest-Existenz, clientseitige Flags, angezeigte Karte; nicht als Server-Droprate oder NPC-HP ausgeben.
3. **Classic/Era-Baseline:** verwendbar als ausdrücklich unbestätigter Referenzwert, nicht als bestätigter Forever-Wert.
4. **Guides:** Anweisungen als Routen-Kandidaten, **niemals** ultimative Quelle für Werte.

Für Quest XP, Mob HP/XP, Respawn, Lootchance, Warlock-spezifische Kampfdauer und reale Wege ist die Vollabdeckung 1–30 **noch nicht nachgewiesen**. Deshalb darf kein 1–30-Routensieger behauptet werden.

## Datendateien

- `reports/questie_classic_reference_quest_levels_1_30.csv`: ein Eintrag pro QuestieDB-Quest, Fakten und Datenlücken ohne geschützte Guide-Texte.
- `reports/questie_classic_reference_coverage_by_level.csv`: Abdeckung für jede Queststufe von 1 bis 30.

### Auffälligkeiten / richtige Interpretation

- `missing_xp_reference`: 101 – nicht automatisch ein defekter Questdatensatz.
- `missing_starter_field`: 19 – Spezialquests möglich.
- `missing_finisher_field`: 23 – Spezialquests möglich.
- `unresolved_starter_reference`: 0.
- `unresolved_finisher_reference`: 0.
- `unresolved_prequest_reference`: 0.

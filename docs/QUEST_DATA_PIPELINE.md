# QuestieDB / ForeverDB / Wowhead: Classic-Fallback für 1–30

Stand 22.10.2026. Diese Policy ist vom Nutzer autorisiert: **Lieber echte historische Classic-Referenzwerte als blockierte Simulationen**, solange aktuelle Forever-Beobachtungen fehlen.

## Priorität je Datenfeld

1. ForeverDB **in Forever beobachtet** (z.B. Quest-XP auf beobachtetem Spielerlevel, Mob-HP bei bestimmtem Moblevel).
2. Wowhead **Forever** / ForeverDB belegte Clientdaten (obwohl die Existenz einer Quest ihre XP/HP nicht belegt).
3. QuestieDBs Version für **Forever** für Quest-IDs, Annahme- und Abgabe-NPCs, Stufen, Questketten und Klassenbedingungen.
4. **Classic/Era-Fallback** aus denselben gepinnten QuestieDB-Quellen: XP aus `support/Forever/QuestXP/xpDB-classic.lua`, NPC-HP aus `data/Classic/classicNpcDB.lua`.

Quellenwerte erhalten `source`, `url`, `quality` und ggf. `method`. Konflikte werden protokolliert. Ein von Classic stammender Wert ist eine **vorläufige Arbeitsreferenz**, kein Forever-Servermesswert.

**ACHTUNG:** Die dokumentierte QuestieDB-Laufzeit-API `NpcDB.minLevelHealth/maxLevelHealth` liefert Kompatibilitäts-Platzhalter 0 und 1. Nur rohe Classic-NPC-Datei mit expliziten numerischen HP-Spannen verwenden. Bei verschiedenen NPC-Stufen werden für Zwischenstufen HP linear interpoliert und als Modell gekennzeichnet. Unbrauchbare 0/1-Zeilen werden ausgeschlossen.

## Bisher überprüfte Quellabdeckung

- QuestieDB v. `f844007b329b72f18cf2bda3d87540012294732a`: **1.413 Queststufen 1–30**, **1.312** davon mit einem Classic-XP-Referenzwert.
- Rohe Classic-NPC-Datei: **4.879 NPCs mit numerischen HP-Angaben**, deren Mindeststufe 1–35 beträgt. Weitere **40** passende NPC-Datensätze haben keine verwendbaren HP-Werte.
- Beispiel: NPC 3098 Mottled Boar Level 1–2, **42–55 Classic-HP**. Bei Forever-Datenänderungen werden die entsprechenden Einträge überlagert.

## Lokale Generierung

```bash
python -m foreverroutes.reference_facts \
  --quest-overlay data/foreverdb_observations.example.json \
  --output .generated/forever_quest_npc_1_30.json
```

Alle QuestieDB-Rohdateien werden **nur lokal** von dem festgepinnten Commit heruntergeladen. Bei fehlendem Netz können Dateien vorher im Cache `.cache/questie` abgelegt werden. Dieses MIT-Repo enthält **keine kopierten Questrohdaten oder vollständigen Drittanbieter-Guides**; deren Rechte sind separat zu beachten.

Für ForeverDB ist die API-Freischaltung noch ausstehend. Beobachtungen können bis dahin als lokale JSON-Overlays eingespielt werden; Wowhead-Details nur aufnehmen, wenn der konkrete Wert auf einer Forever-Seite erkennbar ist. Nicht aus Questbeschreibung oder Guidezeiten abgeleitete HP-/XP-Zahlen erfinden.

**Noch offen für eine echte vergleichbare Modellzeit:** vollständige Guide-Aktionslisten mit Bedingungen, Grind-Mob-XP, Begehbarkeit, Travel-Messungen, Respawn-/Dropchance und Warlock-Kampfwerte. Classic-Referenzen sollen diesen Fortschritt ermöglichen, nicht eine bereits verifizierte Bestzeit vortäuschen.

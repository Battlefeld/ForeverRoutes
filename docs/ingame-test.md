# Ingame-Test zum Release (ab 2.11.2026)

Was am PC im echten Spiel zu prüfen ist, bevor die Routen als kalibriert gelten. Alles hier ist Modell, bis es im
Spiel bestätigt ist: Laufwege sind gemessen (Shortest Path Forever), Kampfzeiten simuliert (MythicSim), Dungeonläufe
und Gruppensuche geschätzt.

## 1. Installation

1. WoW vollständig schließen.
2. `ForeverRoutes-addon.zip` entpacken; den Ordner `ForeverRoutes` nach `World of Warcraft/_classic_/Interface/AddOns`
   kopieren (einen älteren Ordner `ForeverRoutes` vorher löschen). Eine alte Kopie unter ForeverSim nicht zusätzlich
   aktivieren.
3. WoW starten, in der Addon-Liste „ForeverRoutes“ aktiv lassen, einloggen, `/fr` öffnet das HUD.

## 2. Route und Anzeige

- Wird die Route für Volk und Klasse geladen (Solo oder mit Dungeons)? Steht kein „veraltet“/RXP-Fallback, obwohl
  eine passende Route existiert?
- Zeigt der Pfeil zum nächsten Schritt, springt der Schritt bei Annahme, Ziel und Abgabe weiter?
- Optionen (`/fr optionen`, `/fr option N`): Schlafsack-Reihe zuschalten und prüfen, dass ihre Schritte erscheinen.
- `/reload` und Neu-Einloggen: bleibt der Fortschritt erhalten?

## 3. Laufzeiten stichprobenartig messen

Ziel: Weichen echte Laufzeiten von den Routen ab? Drei bis fünf Wege mit Stoppuhr (oder `/played`-Differenz), ohne
Reittier und ohne Kampf unterwegs:

| Weg | Warum |
|---|---|
| Ein Weg im Startgebiet zwischen zwei Questgebern (100–300 Yards) | kurze Wege |
| Ein Weg über eine Brücke oder einen Fluss (z. B. Seenhain in Rotkamm) | Umwege |
| Ein Weg in einer Stadt mit Ebenen (Unterstadt, Eisenschmiede) | Stockwerke |
| Ein langer Weg über Land (1500–3000 Yards) | lange Wege |
| Eine Fahrt mit dem Aufzug (Donnerfels oder Unterstadt) | Aufzugszeit |

Notieren: Start-NPC, Ziel-NPC, gemessene Sekunden. Das Modell rechnet 7 Yards pro Sekunde auf dem gemessenen Weg.

## 4. Dungeons

- Gruppensuche: Wie lange dauerte es bis zur Gruppe (als DD, als Heiler/Tank)? Das Modell nimmt an, dass parallel
  weiter gequestet wird; gewartet wird nur, was das Questen nicht überbrückt.
- Laufdauer je Dungeon von Betreten bis Verlassen (Flammenschlund, Ruinen von Lordaeron, Höhlen des Wehklagens,
  Todesminen, Hallen der Thane). Das Modell schätzt 30–50 Minuten.
- Stufenbereiche aus dem Dungeon-Browser notieren (Hallen der Thane, Höhlen des Wehklagens: zwei Quellen sind sich
  uneinig).

## 5. Was zurückkommt

Die gemessenen Werte (Tabelle oben, Dungeonzeiten) reichen als Text. Damit werden Aufzugszeiten, Gruppensuche und
Dungeondauer kalibriert und die Routen neu berechnet.

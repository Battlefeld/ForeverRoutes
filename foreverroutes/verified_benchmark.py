"""Fail-closed reference-route benchmarking for WoW Forever.

The module reports *why* a guide cannot yet be timed. It never promotes a
Classic/Era seed, another guide's estimated XP or a synthetic fixture into a
Forever measurement. These audits are not elapsed-time simulations.
"""
from __future__ import annotations

from dataclasses import dataclass
import json
from pathlib import Path
from typing import Iterable

from .baselines import parse_rows

VALID_GAME_ORIGINS = frozenset({"forever_server_observed", "forever_client_direct"})
SERVER_FACTS = frozenset({"quest_xp", "mob_hp", "mob_xp", "drop_chance", "kill_seconds"})


class EvidenceError(ValueError):
    pass


@dataclass(frozen=True)
class Fact:
    field: str
    identity: str
    value: float
    origin: str
    source: str
    revision: str
    observed_at: str = ""
    sample_count: int = 0

    def is_forever_verified(self) -> bool:
        if not self.source or not self.revision or self.origin not in VALID_GAME_ORIGINS:
            return False
        if self.field in SERVER_FACTS:
            return self.origin == "forever_server_observed" and bool(self.observed_at) and self.sample_count > 0
        return True


class FactBook:
    def __init__(self, facts: Iterable[Fact]):
        self._facts = {}
        for f in facts:
            k = (f.field, f.identity)
            if k in self._facts:
                raise EvidenceError(f"duplicate fact {k}: conflicting sources must be reconciled explicitly")
            if f.value < 0:
                raise EvidenceError(f"negative fact {k}")
            self._facts[k] = f

    @classmethod
    def load(cls, path: str | Path) -> "FactBook":
        doc = json.loads(Path(path).read_text(encoding="utf-8"))
        if doc.get("schema") != 1 or not isinstance(doc.get("facts"), list):
            raise EvidenceError("expected schema=1, facts=[...]")
        facts = []
        for r in doc["facts"]:
            if not isinstance(r.get("value"), (int,float)) or isinstance(r.get("value"),bool):
                raise EvidenceError(f"fact must have a numeric value: {r}")
            facts.append(Fact(field=str(r["field"]), identity=str(r["id"]), value=float(r["value"]),
                              origin=str(r["origin"]), source=str(r.get("source", "")),
                              revision=str(r.get("revision", "")), observed_at=str(r.get("observed_at", "")),
                              sample_count=int(r.get("sample_count", 0))))
        return cls(facts)

    def get(self, field: str, identity: str | int) -> Fact | None:
        return self._facts.get((field, str(identity)))


def inspect_candidate(*, source: str, csv_path: str | Path, facts: FactBook,
                      guide: str | None = None, target_level: int = 30,
                      observed_target_level: int | None = None) -> dict:
    """Audit historical guide IDs without executing unknown guide conditions.

    observed_target_level must come from an independent replay: a file title
    that claims 1-30 does not prove that its route actually reaches level 30.
    """
    rows = list(parse_rows(csv_path, source, guide))
    ids = sorted({r["quest_id"] for r in rows if r["action"] == "turnin"})
    missing, unverified, verified = [], [], []
    for quest in ids:
        f = facts.get("quest_xp", quest)
        if f is None:
            missing.append(quest)
        elif not f.is_forever_verified():
            unverified.append({"quest_id":quest,"value":f.value,"origin":f.origin,"source":f.source})
        else:
            verified.append(quest)
    cond = sum(bool(r["condition"]) for r in rows)
    optional = sum(bool(r["optional"]) for r in rows)
    blockers = []
    if not rows:
        blockers.append("no_reference_route_actions")
    if cond:
        blockers.append("conditions_not_yet_evaluated_for_character")
    if optional:
        blockers.append("optional_branches_not_yet_resolved")
    if missing or unverified:
        blockers.append("quest_xp_not_verified_in_forever")
    # Full replay requires verified objectives, combat and terrain, not
    # merely a complete list of quest XP values.
    blockers.extend(("quest_prerequisites_and_objectives_not_replayed",
                     "npc_hp_kill_xp_drop_rates_and_class_combat_not_verified",
                     "walkable_travel_network_and_measured_times_missing"))
    if observed_target_level is None or observed_target_level < target_level:
        blockers.append("target_level_not_confirmed_by_replay")
    return {
        "source":source,"guide":guide,"target_level":target_level,
        "status":"NICHT_VERGLEICHBAR", "elapsed_seconds":None,
        "actions":len(rows),"quest_ids_with_turnins":len(ids),
        "verified_forever_quest_xp":len(verified),
        "missing_quest_xp":len(missing),"missing_quest_xp_ids":missing[:25],
        "non_forever_verified_xp":len(unverified),"non_forever_verified_examples":unverified[:12],
        "unresolved_conditions":cond,"optional_actions":optional,
        "blockers":blockers,
        "interpretation":"Datenvollstaendigkeit, keine Routenzeit. Weder Guide-Schaetzwerte noch Era-Werte werden zur Wertung benutzt."
    }


def compare_candidates(candidates: list[dict]) -> dict:
    """Never name a winner when any baseline has not been verified."""
    if not candidates or any(c.get("status") != "VERIFIED_REPLAY" for c in candidates):
        return {"status":"KEIN_SIEGER", "winner":None,
                "reason":"Mindestens eine Vergleichsroute ist nicht mit denselben Forever-Fakten bis zum Ziellevel reproduziert."}
    goals = {(c.get("race"),c.get("class"),c.get("faction"),c.get("target_level"),c.get("world_revision")) for c in candidates}
    if len(goals)!=1:
        raise EvidenceError("different characters, world revisions or target levels")
    winner=min(candidates,key=lambda r:r["elapsed_seconds"])
    return {"status":"VERIFIED_REPLAY","winner":winner["source"],"elapsed_seconds":winner["elapsed_seconds"]}

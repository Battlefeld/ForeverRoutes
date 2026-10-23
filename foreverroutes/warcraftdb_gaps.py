"""WarcraftDB Forever/Classic first, OctoWow evidence fallback, then Wowhead.

The input ID list is derived from the published unified quality audit, NOT by
guessing a numeric quest-ID range. Facts are stored with evidence and source.
No errors/404s become zero XP, and Classic is tagged as provisional.
"""
from __future__ import annotations
import argparse
import copy
import datetime as dt
import hashlib
import json
import re
import time
from pathlib import Path
from urllib.parse import urlparse
from urllib.request import Request, urlopen
from urllib.error import HTTPError
from collections import Counter
from urllib.robotparser import RobotFileParser

SCHEMA = "foreverroutes.warcraftdb_gaps.v1"
AGENT = "ForeverRoutes-QuestAudit/1.0 (source-referenced; respectful rate limit)"
SOURCES = (
    ("warcraftdb_forever", "https://forever.warcraftdb.com/data/quest/{id}", "forever_reference"),
    ("warcraftdb_classic", "https://classic.warcraftdb.com/data/quest/{id}", "classic_reference"),
    ("octowow_classic", "https://octowow.st/db/?quest={id}", "classic_reference"),
    ("wowhead_forever", "https://www.wowhead.com/forever/quest={id}", "forever_reference"),
    ("wowhead_classic", "https://www.wowhead.com/classic/quest={id}", "classic_reference"),
)
SCALAR = {"name":"name","level":"quest_level","min_level":"required_level","xp":"xp","repeatable":"repeatable"}

def load(path, fallback=None):
    if not path or not Path(path).is_file():
        return {} if fallback is None else fallback
    return json.loads(Path(path).read_text(encoding="utf-8"))

def write(path, obj):
    p=Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    temp=p.with_suffix(p.suffix+".tmp")
    temp.write_text(json.dumps(obj, ensure_ascii=False, separators=(",",":"))+"\n",encoding="utf-8")
    temp.replace(p)

def ids_from_audit(audit):
    if audit.get("schema") != "foreverroutes.unified_audit.v1":
        raise ValueError("Expected existing unified quality-audit snapshot")
    ids=audit.get("missing",{}).get("quests_without_xp")
    if not isinstance(ids,list) or not ids:
        raise ValueError("Audit missing expected XP gap ID list")
    result=[]
    for value in ids:
        if type(value) is not int or value<1 or value in result:
            raise ValueError("Invalid or duplicate gap ID")
        result.append(value)
    return sorted(result)

def text_lines(body):
    return [re.sub(r"\s+"," ",s).strip() for s in body.splitlines() if s.strip()]

def field_line(lines, key):
    # WarcraftDB details are in a table; the browser turns td text into tabs or
    # individual lines. Support both without confusing the navigation sidebar.
    heading={"name":"Name","level":"Level","min_level":r"(?:Minimum Level|Requires Level|Required Level)",
             "zone":"Zone","side":"Side","type":"Type"}[key]
    rex=re.compile(r"^\s*"+heading+r"\s*(?:\||:|\t)\s*(.+)\s*$",re.I)
    rex_bare=re.compile(r"^(?:"+heading+r")$",re.I)
    for i,s in enumerate(lines):
        m=rex.match(s)
        if m:return m.group(1).strip()
        if rex_bare.match(s) and i+1<len(lines):
            candidate=lines[i+1]
            if candidate and len(candidate)<120:return candidate
    return None

def section(lines,heading):
    stops={"objectives","rewards","quest item drops","provided item","provided items",
           "storyline","name","zone","level","requires level","minimum level",
           "tags","difficulty","started by","completed by","log description",
           "quest description","objective map","progress","completion log","related"}
    indexes=[i for i,v in enumerate(lines) if v.lower()==heading.lower()]
    for start in indexes:
        out=[]
        for line in lines[start+1:]:
            if line.lower() in stops or line.startswith("### ") or line.startswith("## "):
                break
            if line in ("Image","--"):continue
            out.append(line.removeprefix("• ").removeprefix("* ").strip())
            if len(out)>=80:break
        if out:return out
    return []

def parse_warcraftdb(title, text):
    """Only copy values explicitly present in the quest page."""
    lines=text_lines(text)
    joined="\n".join(lines)
    info={}
    if title:
        title=re.sub(r"(?i)\s*[-–]\s*(?:Forever|WoW Classic|Warcraft DB|Quest).*$","",title).strip()
    name=field_line(lines,"name") or title
    if name and len(name)<160 and not any(t in name.lower() for t in ("{{","warcraft db","not found")):
        info["name"]=name
    for src in ("level","min_level"):
        raw=field_line(lines,src)
        if raw:
            m=re.match(r"^\s*(\d{1,3})\b",raw)
            if m:info[src]=int(m.group(1))
    for src in ("zone","side","type"):
        raw=field_line(lines,src)
        if raw and len(raw)<160:info[src]=raw
    # An explicit metadata flag is evidence; ordinary prose is not.
    if re.search(r"(?im)^(?:Type|Flags?)\s*(?:\||:)\s*[^\n]*\brepeatable\b",joined):
        info["repeatable"]=True
    xp = re.search(r"(?im)(?<![\w])([\d,]+)\s+Experience Points\b",joined)
    if xp is None:
        xp = re.search(r"(?im)^([\d,]+)\s+XP(?:\s|$)",joined)
    if xp is not None:info["xp"]=int(xp.group(1).replace(",",""))
    for key,head in (("objectives","Objectives"),
                     ("quest_item_drops","Quest Item Drops"),
                     ("rewards","Rewards"),
                     ("provided_items","Provided Item")):
        values=section(lines,head)
        if values:info[key]=values
    reputation=re.findall(r"(?im)\+([\d,]+)\s+reputation with ([^\n]+)",joined)
    if reputation:info["reputation"]=[{"faction":n.strip(),"amount":int(a.replace(",",""))} for a,n in reputation]
    if not info.get("name") or not any(k in info for k in ("xp","level","min_level","objectives")):
        raise ValueError("Page did not expose a verifiable quest record")
    return info

def parse_wowhead(title,text):
    # A Wowhead page may show a scaled player-level reward. Do not silently
    # interpret it as unscaled base XP without an explicit amount.
    d=parse_warcraftdb(title,text)
    return d

def parse_octowow(title,text):
    """Parse an explicit Classic quest template, never infer 0 from absent XP.

    The separate RewXP field is a base-quest reward. OctoWow also displays
    player-scaled 'Gains' experience, which must not replace base XP.
    """
    lines=text_lines(text)
    joined="\n".join(lines)
    if not re.search(r"(?im)^Quick Facts$",joined):
        raise ValueError("OctoWow did not expose a quest Quick Facts record")
    if not title or not re.search(r"(?i)Quests?\s*-\s*OctoWow",title):
        raise ValueError("Unexpected quest page or page title")
    head=joined.split("Quick Facts",1)[1].split("RewXP",1)[0]
    info={}
    name=title.split(" - Quests",1)[0].strip()
    if name and len(name)<160:info["name"]=name
    for key,pattern in (("level",r"(?im)^Level:\s*(\d{1,3})\s*$"),
                        ("min_level",r"(?im)^Requires level:\s*(\d{1,3})\s*$")):
        m=re.search(pattern,head)
        if m:info[key]=int(m.group(1))
    if re.search(r"(?im)^Repeatable$",head):
        info["repeatable"]=True
    m=re.search(r"(?im)^RewXP\s*:\s*([\d,]+)\s*$",joined)
    if m:
        info["xp"]=int(m.group(1).replace(",",""))
    # The page can be readable but not expose an explicit base XP.
    # Never translate a missing field into zero.
    if not info.get("name") or not any(k in info for k in ("xp","level","repeatable")):
        raise ValueError("No verified OctoWow quest fact")
    return info


def robots_allowed(url,cache):
    host=urlparse(url).netloc
    if host not in cache:
        parser=RobotFileParser()
        try:
            with urlopen(Request("https://"+host+"/robots.txt",
                    headers={"User-Agent":AGENT}),timeout=12) as resp:
                data=resp.read(500000).decode("utf-8","replace")
            parser.parse(data.splitlines())
            cache[host]=parser
        except HTTPError as exc:
            if exc.code in (404, 410):
                # RFC 9309: an absent robots file has no crawl restrictions.
                parser.parse([])
                cache[host]=parser
            else:
                cache[host]=str(exc)
        except Exception as exc:
            cache[host]=str(exc)
    value=cache[host]
    if isinstance(value,str):
        raise PermissionError("robots unavailable for "+host+": "+value[:100])
    return value.can_fetch(AGENT,url)

def get_page(page,url,robots,delay):
    if not robots_allowed(url,robots):
        return "disallowed",None,{}
    response=page.goto(url,wait_until="domcontentloaded",timeout=20000)
    status=response.status if response else 0
    if status==404:return "not_found",None,{"http_status":status}
    if status in (403,429):return "blocked",None,{"http_status":status}
    if status>=400 or status==0:return "fetch_error",None,{"http_status":status}
    try:
        if urlparse(url).netloc=="octowow.st":
            page.wait_for_function("() => document.body.innerText.includes('RewXP:')",timeout=4500)
        else:
            page.wait_for_function("""() => {
              const t = document.body.innerText;
              return t.includes('Rewards') && (t.includes('Experience Points') || t.includes('Level'));
            }""",timeout=7500)
    except Exception:pass
    title=page.title()
    # The body is inspected because the page's semantic <main> sometimes
    # excludes the quest metadata table and sometimes includes the sidebar.
    text=page.locator("body").inner_text(timeout=5000)
    if re.search(r"(?i)\b(quest not found|no such quest|this quest doesn't exist)\b",text):
        return "not_found",None,{"http_status":status}
    time.sleep(delay)
    return "ok",(title,text),{"http_status":status,
             "body_sha256":hashlib.sha256(text.encode()).hexdigest()}

def attempts_for_quest(page,qid,robots,delay,previous=None):
    """Try Forever first, then explicit OctoWow base XP, preserving earlier evidence.

    For previously inspected quests that lack XP, newly added OctoWow lookup is
    performed without pointlessly repeating the other four remote requests.
    """
    old=previous if isinstance(previous,dict) else {}
    previous_attempts=list(old.get("attempts",[]))
    old_sources={a.get("source") for a in previous_attempts}
    old_fields=dict(old.get("fields",{}))
    old_evidence=dict(old.get("field_sources",{}))
    catchup=bool(previous_attempts and "xp" not in old_fields
                 and "octowow_classic" not in old_sources)
    observations=list(previous_attempts) if catchup else []
    combined=dict(old_fields) if catchup else {}
    provenance=dict(old_evidence) if catchup else {}
    sources_to_try=[s for s in SOURCES if s[0]=="octowow_classic"] if catchup else SOURCES
    for source,template,quality in sources_to_try:
        if "xp" in combined:break
        url=template.format(id=qid)
        obs={"source":source,"url":url,"quality":quality,
             "checked":dt.datetime.now(dt.timezone.utc).isoformat()}
        try:
            status,doc,meta=get_page(page,url,robots,delay)
            obs["status"]=status
            obs.update(meta)
            if status=="ok":
                parser=(parse_octowow if source=="octowow_classic" else
                        parse_wowhead if source.startswith("wowhead") else parse_warcraftdb)
                info=parser(*doc)
                obs["fields"]=info
                for key,value in info.items():
                    if key not in combined:
                        combined[key]=value
                        provenance[key]={"source":source,"url":url,"quality":quality}
        except Exception as exc:
            obs["status"]="error"
            obs["error"]=str(exc)[:180]
        observations.append(obs)
    return {"id":qid,"fields":combined,"field_sources":provenance,"attempts":observations,
            "checked":dt.datetime.now(dt.timezone.utc).isoformat()}

def unscanned_ids(target_ids, rows):
    """IDs still missing either their initial lookup or the explicit XP fallback."""
    return [qid for qid in target_ids
            if not isinstance(rows.get(str(qid)),dict)
            or ("xp" not in rows[str(qid)].get("fields",{})
                and not any(a.get("source")=="octowow_classic"
                            for a in rows[str(qid)].get("attempts",[])))]


def pending_ids(target_ids, rows, retry_hours, now=None):
    """Unseen IDs are immediately due, verified XP (including 0) is never retried."""
    now = now or dt.datetime.now(dt.timezone.utc)
    pending = []
    for qid in target_ids:
        record = rows.get(str(qid))
        if not isinstance(record, dict):
            pending.append(qid)
            continue
        if "xp" in record.get("fields", {}):
            continue
        # Revisit earlier scanned quests once for the newly added explicit
        # RewXP and Repeatable metadata, even within normal retry cooldown.
        if not any(a.get("source")=="octowow_classic"
                   for a in record.get("attempts",[])):
            pending.append(qid)
            continue
        checked = record.get("checked")
        if checked:
            try:
                when = dt.datetime.fromisoformat(checked.replace("Z", "+00:00"))
                if now - when < dt.timedelta(hours=retry_hours):
                    continue
            except (ValueError, TypeError):
                pass
        pending.append(qid)
    return pending

def collect(args):
    from playwright.sync_api import sync_playwright
    audit=load(args.audit)
    target_ids=ids_from_audit(audit)
    previous=load(args.previous,{"schema":SCHEMA,"quests":{}})
    if previous.get("schema")!=SCHEMA:raise ValueError("Wrong prior gaps schema")
    rows=dict(previous.get("quests",{}))
    pending=pending_ids(target_ids, rows, args.retry_hours)
    batch=pending[:args.limit] if args.limit else pending
    robots={}
    if batch:
        with sync_playwright() as browser_api:
            browser=browser_api.chromium.launch(headless=True)
            page=browser.new_page(locale="en-US",user_agent=AGENT)
            for j,qid in enumerate(batch,1):
                rows[str(qid)]=attempts_for_quest(page,qid,robots,args.delay,
                                                  previous=rows.get(str(qid)))
                if j%10==0:print("WarcraftDB checked",j,"/",len(batch),flush=True)
            browser.close()
    # Current quality audits shrink as XP gaps are recovered. Keep all earlier
    # source evidence; dropping recovered rows would undo fixes on the next rebuild.
    original_size=max(int(previous.get("target_count",0)),len(target_ids),len(rows))
    values={"schema":SCHEMA,"generated":dt.datetime.now(dt.timezone.utc).isoformat(),
            "source_audit":str(args.audit),"target_count":original_size,
            "quests":{str(i):rows[str(i)] for i in sorted(map(int,rows))}}
    write(args.output,values)
    covered=sum("xp" in e.get("fields",{}) for e in values["quests"].values())
    checked=[rows[str(qid)] for qid in batch]
    statuses=Counter(a.get("status","unknown") for rec in checked for a in rec.get("attempts",[]))
    newly_found=sum("xp" in rec.get("fields",{}) for rec in checked)
    outcome={"target":len(target_ids),"scanned_this_run":len(batch),
             "xp_recovered":covered,"xp_found_this_run":newly_found,
             "remaining":len(target_ids)-covered,
             "unscanned_remaining":len(unscanned_ids(target_ids,rows)),
             "source_statuses":dict(statuses)}
    if args.status_output:write(args.status_output,outcome)
    print(json.dumps(outcome),flush=True)

def apply(base,document):
    if document.get("schema")!=SCHEMA:raise ValueError("Wrong gap overlay schema")
    if base.get("schema")!="foreverroutes.unified_quests.v1":raise ValueError("Wrong unified quest schema")
    data=copy.deepcopy(base)
    changes={"xp_inserted":0,"other_fields_inserted":0,"overrides_skipped":0}
    for sid,record in document.get("quests",{}).items():
        if sid not in data.get("quests",{}):continue
        dest=data["quests"][sid].get("fields",{})
        values=record.get("fields",{})
        evidence=record.get("field_sources",{})
        for key,unified in SCALAR.items():
            if key not in values or key not in evidence:continue
            value=values[key]
            if key in ("xp","level","min_level"):
                if type(value) is not int or value<0 or value>1000000:continue
            elif key=="repeatable":
                if value is not True:continue
            else:
                if not isinstance(value,str) or not value.strip():continue
            incoming=evidence[key]
            source=incoming.get("source")
            url=incoming.get("url","")
            source_map=dict((name,(template,quality)) for name,template,quality in SOURCES)
            # Directly verified zero-XP values from independent Classic DBs
            # are explicit evidence, never inferred from missing rewards.
            source_map.update({
                "classicdb_classic": ("https://classicdb.com/?quest={id}", "classic_reference"),
                "octowow_classic": ("https://octowow.st/db/?quest={id}", "classic_reference"),
            })
            expected_url=source_map.get(source, ("", ""))[0].format(id=sid)
            if source not in source_map or url != expected_url:
                continue
            if dest.get(unified) is not None:
                changes["overrides_skipped"]+=1
                continue
            dest[unified]={"value":value,"source":source,
                "quality":source_map[source][1],"url":url,
                "evidence":"source-page-2026-10-23; provisional until in-game check"}
            changes["xp_inserted" if unified=="xp" else "other_fields_inserted"]+=1
        # Website fields that lack a safe normalized simulator mapping are
        # retained exactly as evidence; they cannot overwrite objective IDs.
        data["quests"][sid]["warcraftdb_page_evidence"]={
           "fields":{k:v for k,v in values.items() if k not in SCALAR},
           "sources":{k:v for k,v in evidence.items() if k not in SCALAR}}
    data["stats"]["quests_missing_xp"]=sum(
        v.get("status")!="unresolved_guide_candidate"
        and not any(k in v.get("fields",{}) for k in ("xp","xp_observed_by_player_level"))
        for v in data["quests"].values())
    data["stats"]["warcraftdb_xp_inserted"]=sum(
        q.get("fields",{}).get("xp",{}).get("source","").startswith("warcraftdb")
        for q in data["quests"].values())
    data["stats"]["wowhead_gap_xp_inserted"]=sum(
        q.get("fields",{}).get("xp",{}).get("source","").startswith("wowhead")
        for q in data["quests"].values())
    data["sources"]["warcraftdb_gaps"]={"schema":SCHEMA,
        "updated":document.get("generated"),"target":document.get("target_count"),
        "source_priority":["warcraftdb_forever","warcraftdb_classic","octowow_classic","wowhead_forever","wowhead_classic"]}
    return data,changes

def apply_cli(args):
    from foreverroutes.unified import audit as create_audit
    data,changes=apply(load(args.base),load(args.overlay))
    write(args.output,data)
    if args.audit_output:write(args.audit_output,create_audit(data))
    print(json.dumps(changes))

def main():
    p=argparse.ArgumentParser(description=__doc__)
    sub=p.add_subparsers(dest="command",required=True)
    c=sub.add_parser("collect")
    c.add_argument("--audit",required=True)
    c.add_argument("--previous")
    c.add_argument("--output",required=True)
    c.add_argument("--limit",type=int,default=50)
    c.add_argument("--delay",type=float,default=1.5)
    c.add_argument("--retry-hours",type=int,default=24)
    c.add_argument("--status-output",help="Save structured progress diagnostics for workflow checkpoints")
    a=sub.add_parser("apply")
    a.add_argument("--base",required=True)
    a.add_argument("--overlay",required=True)
    a.add_argument("--output",required=True)
    a.add_argument("--audit-output")
    args=p.parse_args()
    if args.command=="collect":collect(args)
    else:apply_cli(args)

if __name__=="__main__":main()

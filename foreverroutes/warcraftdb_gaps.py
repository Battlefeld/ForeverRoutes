"""WarcraftDB Forever/Classic first, Wowhead fallback for the *831* audited XP gaps.

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
from urllib.robotparser import RobotFileParser

SCHEMA = "foreverroutes.warcraftdb_gaps.v1"
AGENT = "ForeverRoutes-QuestAudit/1.0 (source-referenced; respectful rate limit)"
SOURCES = (
    ("warcraftdb_forever", "https://forever.warcraftdb.com/data/quest/{id}", "forever_reference"),
    ("warcraftdb_classic", "https://classic.warcraftdb.com/data/quest/{id}", "classic_reference"),
    ("wowhead_forever", "https://www.wowhead.com/forever/quest={id}", "forever_reference"),
    ("wowhead_classic", "https://www.wowhead.com/classic/quest={id}", "classic_reference"),
)
SCALAR = {"name":"name","level":"quest_level","min_level":"required_level","xp":"xp"}

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
        except Exception as exc:
            cache[host]=str(exc)
    value=cache[host]
    if isinstance(value,str):
        raise PermissionError("robots unavailable for "+host+": "+value[:100])
    return value.can_fetch(AGENT,url)

def get_page(page,url,robots,delay):
    if not robots_allowed(url,robots):
        return "disallowed",None,{}
    response=page.goto(url,wait_until="domcontentloaded",timeout=35000)
    status=response.status if response else 0
    if status==404:return "not_found",None,{"http_status":status}
    if status in (403,429):return "blocked",None,{"http_status":status}
    if status>=400 or status==0:return "fetch_error",None,{"http_status":status}
    try:
        page.wait_for_function("""() => {
          const t = document.body.innerText;
          return t.includes('Rewards') && (t.includes('Experience Points') || t.includes('Level'));
        }""",timeout=12000)
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

def attempts_for_quest(page,qid,robots,delay):
    observations=[]
    combined={}
    provenance={}
    for ix,(source,template,quality) in enumerate(SOURCES):
        # Only ask Wowhead if WarcraftDB did not supply every previously
        # missing scalar value. Never needlessly scrape Wowhead.
        if ix>=2 and "xp" in combined and "name" in combined and "level" in combined and "min_level" in combined:
            break
        url=template.format(id=qid)
        obs={"source":source,"url":url,"quality":quality,
             "checked":dt.datetime.now(dt.timezone.utc).isoformat()}
        try:
            status,doc,meta=get_page(page,url,robots,delay)
            obs["status"]=status
            obs.update(meta)
            if status=="ok":
                info=(parse_wowhead if source.startswith("wowhead") else parse_warcraftdb)(*doc)
                obs["fields"]=info
                for key,value in info.items():
                    if key not in combined:
                        combined[key]=value
                        provenance[key]={"source":source,"url":url,"quality":quality}
        except Exception as exc:
            obs["status"]="error"
            obs["error"]=str(exc)[:180]
        observations.append(obs)
        if ix==0 and "xp" in combined and "level" in combined and "min_level" in combined:
            break
    return {"id":qid,"fields":combined,"field_sources":provenance,"attempts":observations,
            "checked":dt.datetime.now(dt.timezone.utc).isoformat()}

def collect(args):
    from playwright.sync_api import sync_playwright
    audit=load(args.audit)
    target_ids=ids_from_audit(audit)
    previous=load(args.previous,{"schema":SCHEMA,"quests":{}})
    if previous.get("schema")!=SCHEMA:raise ValueError("Wrong prior gaps schema")
    rows=dict(previous.get("quests",{}))
    remaining=[qid for qid in target_ids
               if not isinstance(rows.get(str(qid)),dict)
               or "xp" not in rows[str(qid)].get("fields",{})]
    # Skip permanent 404s only for the selected number of days, not forever.
    def retry_due(qid):
        rec=rows.get(str(qid),{})
        if not rec or "xp" in rec.get("fields",{}):return False
        checked=rec.get("checked")
        if not checked:return True
        try:
            return (dt.datetime.now(dt.timezone.utc)-
                    dt.datetime.fromisoformat(checked.replace("Z","+00:00"))).total_seconds()>=args.retry_hours*3600
        except (ValueError,TypeError):return True
    pending=[qid for qid in remaining if retry_due(qid)]
    batch=pending[:args.limit] if args.limit else pending
    robots={}
    if batch:
        with sync_playwright() as browser_api:
            browser=browser_api.chromium.launch(headless=True)
            page=browser.new_page(locale="en-US",user_agent=AGENT)
            for j,qid in enumerate(batch,1):
                rows[str(qid)]=attempts_for_quest(page,qid,robots,args.delay)
                if j%10==0:print("WarcraftDB checked",j,"/",len(batch),flush=True)
            browser.close()
    values={"schema":SCHEMA,"generated":dt.datetime.now(dt.timezone.utc).isoformat(),
            "source_audit":str(args.audit),"target_count":len(target_ids),
            "quests":{str(i):rows[str(i)] for i in sorted(map(int,rows)) if i in target_ids}}
    write(args.output,values)
    covered=sum("xp" in e.get("fields",{}) for e in values["quests"].values())
    print(json.dumps({"target":len(target_ids),"scanned_this_run":len(batch),
                      "xp_recovered":covered,"remaining":len(target_ids)-covered}))

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
            else:
                if not isinstance(value,str) or not value.strip():continue
            incoming=evidence[key]
            source=incoming.get("source")
            url=incoming.get("url","")
            source_map=dict((name,(template,quality)) for name,template,quality in SOURCES)
            if source not in source_map or not url.startswith(source_map[source][0].split("{id}")[0]):
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
        "source_priority":["warcraftdb_forever","warcraftdb_classic","wowhead_forever","wowhead_classic"]}
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
    a=sub.add_parser("apply")
    a.add_argument("--base",required=True)
    a.add_argument("--overlay",required=True)
    a.add_argument("--output",required=True)
    a.add_argument("--audit-output")
    args=p.parse_args()
    if args.command=="collect":collect(args)
    else:apply_cli(args)

if __name__=="__main__":main()

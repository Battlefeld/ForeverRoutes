#!/usr/bin/env python3
"""Check repository Markdown paths and every shipped JSON/gzip JSON document offline."""
import gzip,json,pathlib,re,urllib.parse

ROOT=pathlib.Path(__file__).resolve().parents[1]
errors=[];links=0;documents=0
for p in sorted(ROOT.rglob('*.md')):
    if any(part in {'.git','.cache','.generated'} for part in p.relative_to(ROOT).parts):continue
    for m in re.finditer(r'!?\[[^\]]*\]\(([^\s)]+)(?:\s+"[^"]*")?\)',p.read_text()):
        target=m[1].strip('<>')
        if target.startswith(('#','https://','http://','mailto:')):continue
        target=urllib.parse.unquote(target.split('#',1)[0].split('?',1)[0])
        if target and not (p.parent/target).exists():errors.append(str(p.relative_to(ROOT))+': missing '+target)
        links+=1
for p in sorted(ROOT.rglob('*')):
    if not p.is_file() or any(part in {'.git','.cache','.generated'} for part in p.relative_to(ROOT).parts):continue
    if not (p.name.endswith('.json') or p.name.endswith('.json.gz')):continue
    try:
        data=p.read_bytes()
        if p.name.endswith('.gz'):data=gzip.decompress(data)
        json.loads(data,parse_constant=lambda s:(_ for _ in ()).throw(ValueError('non-finite '+s)))
        documents+=1
    except Exception as e:errors.append(str(p.relative_to(ROOT))+': '+str(e))
if errors:raise SystemExit('\n'.join(errors))
print('Validated',links,'local Markdown links and',documents,'JSON documents')

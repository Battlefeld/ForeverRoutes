import re,zlib,base64,sys,os
repo,out=sys.argv[1],sys.argv[2]
os.makedirs(out,exist_ok=True)
for m in ('0','1','2991'):
    lines=[]
    for f in sorted(os.listdir(repo+'/Nav')):
        if re.match(r'^Nav%s_\d+\.lua$'%m,f):
            s=open(repo+'/Nav/'+f).read()
            for field,key,body in re.findall(r'data\.packed\.(\w+)\[(\d+)\]\s*=\s*\{(.*?)\}',s,re.S):
                blocks=re.findall(r'"([^"]*)"',body)
                raw=b''.join(zlib.decompress(base64.b64decode(b)) for b in blocks)
                txt=raw.decode('latin-1')
                assert '"' not in txt and '\\' not in txt and '\n' not in txt, (f,field,key)
                lines.append('D.%s[%s]="%s"'%(field,key,txt))
    open(out+'/nav%s_data.lua'%m,'w').write('local D=ShortestPathForeverPathData[%s]\n'%m+'\n'.join(lines)+'\nD.packed=nil\n')
    print(m,len(lines))

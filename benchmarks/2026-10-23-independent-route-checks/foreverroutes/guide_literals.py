"""Read literal Lua guide tables without executing Lua or addon code."""
from __future__ import annotations
import re

NUMBER=re.compile(r'-?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?')
IDENT=re.compile(r'[A-Za-z_][A-Za-z_0-9]*')
LONG=re.compile(r'\[(=*)\[')

class GuideSyntaxError(ValueError):
    pass

class LuaDataReader:
    def __init__(self,text,start=0):self.text=text;self.pos=start
    def error(self,message):
        raise GuideSyntaxError(f'{message} at line {self.text.count(chr(10),0,self.pos)+1}')
    def skip(self):
        s=self.text
        while self.pos<len(s):
            if s[self.pos].isspace():self.pos+=1;continue
            if not s.startswith('--',self.pos):return
            self.pos+=2;m=LONG.match(s,self.pos)
            if m:
                end=s.find(']'+m[1]+']',m.end())
                if end<0:self.error('Unclosed comment')
                self.pos=end+len(m[1])+2
            else:
                end=s.find('\n',self.pos);self.pos=len(s) if end<0 else end+1
    def take(self,t):
        self.skip()
        if self.text.startswith(t,self.pos):self.pos+=len(t);return True
        return False
    def value(self):
        self.skip();s=self.text
        if self.pos>=len(s):self.error('Unexpected end')
        if s[self.pos]=='{':return self.table()
        if s[self.pos] in '\"\'':
            quote=s[self.pos];self.pos+=1;out=[]
            while self.pos<len(s):
                c=s[self.pos];self.pos+=1
                if c==quote:return ''.join(out)
                if c=='\\':
                    if self.pos>=len(s):self.error('Truncated escape')
                    c=s[self.pos];self.pos+=1
                    if c.isdigit():
                        m=re.match(r'\d{0,2}',s[self.pos:]);n=c+m[0];self.pos+=len(m[0]);out.append(chr(int(n)));continue
                    c={'n':'\n','r':'\r','t':'\t','a':'\a','b':'\b','f':'\f','v':'\v'}.get(c,c)
                out.append(c)
            self.error('Unclosed string')
        m=LONG.match(s,self.pos)
        if m:
            end=s.find(']'+m[1]+']',m.end())
            if end<0:self.error('Unclosed long string')
            self.pos=end+len(m[1])+2;return s[m.end():end]
        m=NUMBER.match(s,self.pos)
        if m:self.pos=m.end();return float(m[0]) if any(c in m[0] for c in '.eE') else int(m[0])
        m=IDENT.match(s,self.pos)
        if m:
            self.pos=m.end()
            return {'true':True,'false':False,'nil':None}.get(m[0],{'symbol':m[0]})
        self.error('Nonliteral expression')
    def table(self):
        self.take('{');out={};next_index=1
        while not self.take('}'):
            self.skip();m=IDENT.match(self.text,self.pos)
            if self.take('['):
                key=self.value()
                if not self.take(']') or not self.take('='):self.error('Invalid table key')
                value=self.value()
            elif m and re.match(r'\s*=',self.text[m.end():m.end()+20]):
                key=m[0];self.pos=m.end()
                if not self.take('='):self.error('Expected assignment')
                value=self.value()
            else:key=next_index;next_index+=1;value=self.value()
            # Lua permits repeated keys; a later literal assignment wins.
            out[key]=value
            if self.take('}'):return out
            if not self.take(',') and not self.take(';'):self.error('Expected comma')
        return out

def read_literal(text,start=0,require_end=True):
    reader=LuaDataReader(text,start);value=reader.value();reader.skip()
    if require_end and reader.pos!=len(text):reader.error('Trailing executable expression')
    return value,reader.pos

def children(table):
    if not isinstance(table,dict):return []
    return [v for k,v in table.items() if isinstance(k,int)]

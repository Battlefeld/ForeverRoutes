#!/usr/bin/env python3
"""The crest in the route window's top edge: the panel's gem ornament with the ForeverCore crystal in place of its
stone, 1.3 times the ornament's size so the crystal reads well and reaches a little into the blue.
Usage: build_crest.py <ForeverRoutes/Media/UI> <out dir>   (writes crest_on.tga and crest_off.tga, 2x resolution)"""
import sys
from PIL import Image, ImageDraw, ImageFilter, ImageChops
ui,out=sys.argv[1].rstrip('/')+'/',sys.argv[2].rstrip('/')+'/'
panel=Image.open(ui+'panel_square_medium.tga').convert('RGBA').crop((0,0,263,256))
K=0.85*2            # the window draws the panel at 85 %; the crest is stored at twice that
SCALE=1.3           # the ornament, bigger than the panel's gem
C0,C1=98,165        # the gem ornament in the panel's top edge
CRYSTAL=112         # crystal size in the stored image (56 points in the window)
def crest(crystal_file):
    g=panel.crop((C0,0,C1,68)).resize((round((C1-C0)*K*SCALE),round(68*K*SCALE)),Image.LANCZOS)
    # the dark lower part of the piece fades out (the frame's own background shows), its sides fade into the bar
    m=Image.new('L',g.size,0);md=ImageDraw.Draw(m)
    for y in range(g.height):
        t=y/g.height;md.line([(0,y),(g.width,y)],fill=255 if t<.62 else max(0,int(255*(1-(t-.62)/.12))))
    side=Image.new('L',g.size,255);sd=ImageDraw.Draw(side)
    for x in range(g.width):
        e=min(x,g.width-1-x);sd.line([(x,0),(x,g.height)],fill=min(255,int(255*e/(g.width*.18))))
    g.putalpha(ImageChops.multiply(ImageChops.multiply(g.getchannel('A'),m),side))
    # the crystal sits where the stone was, on a dark seat
    cx,cy=g.width//2,round(29*K*SCALE)
    seat=Image.new('RGBA',g.size,(0,0,0,0))
    ImageDraw.Draw(seat).ellipse((cx-CRYSTAL*.42,cy-CRYSTAL*.46,cx+CRYSTAL*.42,cy+CRYSTAL*.46),fill=(4,10,28,255))
    g.alpha_composite(seat.filter(ImageFilter.GaussianBlur(2)))
    c=Image.open(ui+crystal_file).convert('RGBA').resize((CRYSTAL,CRYSTAL),Image.LANCZOS)
    g.alpha_composite(c,(cx-CRYSTAL//2,cy-CRYSTAL//2))
    return g
for name,src in(('crest_on','crystal_on.tga'),('crest_off','crystal_off.tga')):
    im=crest(src);tex=Image.new('RGBA',(256,256),(0,0,0,0));tex.alpha_composite(im,(0,0));tex.save(out+name+'.tga')
    print(name,im.size)

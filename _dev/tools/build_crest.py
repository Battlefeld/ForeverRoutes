import sys
from PIL import Image, ImageDraw, ImageFilter
root=sys.argv[1];out=sys.argv[2]
panel=Image.open(root+'/panel_square_medium.tga').convert('RGBA').crop((0,0,263,256))
# work at 4x the panel's pixels for a smooth result
S=4
def big(im):return im.resize((im.width*S,im.height*S),Image.LANCZOS)
W,H=170,124           # crest in panel pixels: wider than the gem piece; the crystal hangs below the top bar
TOP=0                 # the crest's top is the frame's top edge
def crest(crystal_file):
    c=Image.new('RGBA',(W*S,H*S),(0,0,0,0))
    # the gem's gold wings only (the frame's own bar runs on beneath the crest): the gem piece, its dark lower part
    # faded out, its sides faded into the bar
    gem=big(panel.crop((90,0,173,68)))
    mask=Image.new('L',gem.size,0);md=ImageDraw.Draw(mask)
    for y in range(gem.height):
        a=255 if y<44*S else max(0,int(255*(1-(y-44*S)/(10*S))))
        md.line([(0,y),(gem.width,y)],fill=a)
    side=Image.new('L',gem.size,255);sd=ImageDraw.Draw(side)
    for x in range(gem.width):
        e=min(x,gem.width-1-x);sd.line([(x,0),(x,gem.height)],fill=min(255,int(255*e/(10*S))))
    from PIL import ImageChops
    alpha=ImageChops.multiply(ImageChops.multiply(gem.getchannel('A'),mask),side)
    gem.putalpha(alpha)
    c.alpha_composite(gem,((W*S-gem.width)//2,TOP*S))
    # a soft dark shadow and a gold glow behind the crystal so it sits in the frame
    cx,cy=W*S//2,(TOP+70)*S;R=38*S
    sh=Image.new('RGBA',c.size,(0,0,0,0));d=ImageDraw.Draw(sh)
    d.ellipse((cx-R,cy-R+4*S,cx+R,cy+R+4*S),fill=(0,0,0,170));sh=sh.filter(ImageFilter.GaussianBlur(5*S))
    c.alpha_composite(sh)
    gl=Image.new('RGBA',c.size,(0,0,0,0));d=ImageDraw.Draw(gl)
    d.ellipse((cx-R-4*S,cy-R-4*S,cx+R+4*S,cy+R+4*S),fill=(255,196,90,90));gl=gl.filter(ImageFilter.GaussianBlur(8*S))
    c.alpha_composite(gl)
    # gold: a neck from the bar down to the crystal, two curved struts from the bar to its sides, a ring around it
    GL,GM,GD,EDGE=(246,206,120,255),(206,150,62,255),(122,78,26,255),(40,24,8,255)
    def gold_shape(draw_fn):
        m=Image.new('L',c.size,0);draw_fn(ImageDraw.Draw(m))
        # vertical gold gradient through the shape, a dark rim around it
        grad=Image.new('RGBA',c.size,(0,0,0,0));gd=ImageDraw.Draw(grad)
        for y in range(c.size[1]):
            t=y/c.size[1];col=tuple(int(GL[i]*(1-t)+GD[i]*t) if t<.5 else int(GM[i]*(1-(t-.5)*2)+GD[i]*(t-.5)*2) for i in range(3))+(255,)
            gd.line([(0,y),(c.size[0],y)],fill=col)
        rim=m.filter(ImageFilter.MaxFilter(5))
        out=Image.new('RGBA',c.size,(0,0,0,0))
        edge=Image.new('RGBA',c.size,EDGE);out.paste(edge,(0,0),rim)
        out.paste(grad,(0,0),m)
        hi=m.filter(ImageFilter.MinFilter(9));hl=Image.new('RGBA',c.size,(255,236,180,90));out.paste(hl,(0,0),ImageChops.subtract(m,hi))
        c.alpha_composite(out)
    from PIL import ImageChops
    bar=33*S   # the bar's middle in the panel
    def bez(p0,p1,p2,n=40):
        return [((1-t)**2*p0[0]+2*(1-t)*t*p1[0]+t*t*p2[0],(1-t)**2*p0[1]+2*(1-t)*t*p1[1]+t*t*p2[1]) for t in [i/n for i in range(n+1)]]
    for sgn in (-1,1):
        pts=bez((cx+sgn*62*S,bar),(cx+sgn*58*S,cy-6*S),(cx+sgn*(R+2*S),cy+4*S))
        gold_shape(lambda d,pts=pts:d.line(pts,fill=255,width=7*S,joint='curve'))
    gold_shape(lambda d:d.polygon([(cx-12*S,bar-6*S),(cx+12*S,bar-6*S),(cx+6*S,cy-R),(cx-6*S,cy-R)],fill=255))
    gold_shape(lambda d:(d.ellipse((cx-R-7*S,cy-R-7*S,cx+R+7*S,cy+R+7*S),fill=255),d.ellipse((cx-R+1*S,cy-R+1*S,cx+R-1*S,cy+R-1*S),fill=0)))
    # the crystal's own dark seat inside the ring
    seat=Image.new('RGBA',c.size,(0,0,0,0));ImageDraw.Draw(seat).ellipse((cx-R,cy-R,cx+R,cy+R),fill=(6,14,34,235));c.alpha_composite(seat)
    cr=Image.open(root+'/'+crystal_file).convert('RGBA').resize((2*R,2*R),Image.LANCZOS)
    c.alpha_composite(cr,(cx-R,cy-R))
    return c.resize((W*2,H*2),Image.LANCZOS)   # stored at 2x the panel's pixels: crisp when drawn large
for name,src in (('crest_on','crystal_on.tga'),('crest_off','crystal_off.tga')):
    im=crest(src)
    tex=Image.new('RGBA',(512,256),(0,0,0,0));tex.alpha_composite(im,(0,0))
    tex.save(out+'/'+name+'.tga')
    im.save(out+'/'+name+'_preview.png')
print(W*2,H*2)

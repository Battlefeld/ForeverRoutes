import sys
from PIL import Image, ImageDraw, ImageFilter
root=sys.argv[1];out=sys.argv[2]
panel=Image.open(root+'/panel_square_medium.tga').convert('RGBA').crop((0,0,263,256))
# work at 4x the panel's pixels for a smooth result
S=4
def big(im):return im.resize((im.width*S,im.height*S),Image.LANCZOS)
W,H=150,92            # crest in panel pixels: wider than the gem piece, taller (it rises above the frame)
TOP=18                # how far the crest rises above the frame's top edge
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
    cx,cy=W*S//2,(TOP+24)*S;R=32*S
    sh=Image.new('RGBA',c.size,(0,0,0,0));d=ImageDraw.Draw(sh)
    d.ellipse((cx-R,cy-R+4*S,cx+R,cy+R+4*S),fill=(0,0,0,170));sh=sh.filter(ImageFilter.GaussianBlur(5*S))
    c.alpha_composite(sh)
    gl=Image.new('RGBA',c.size,(0,0,0,0));d=ImageDraw.Draw(gl)
    d.ellipse((cx-R-4*S,cy-R-4*S,cx+R+4*S,cy+R+4*S),fill=(255,196,90,90));gl=gl.filter(ImageFilter.GaussianBlur(8*S))
    c.alpha_composite(gl)
    cr=Image.open(root+'/'+crystal_file).convert('RGBA').resize((2*R,2*R),Image.LANCZOS)
    c.alpha_composite(cr,(cx-R,cy-R))
    return c.resize((W*2,H*2),Image.LANCZOS)   # stored at 2x the panel's pixels: crisp when drawn large
for name,src in (('crest_on','crystal_on.tga'),('crest_off','crystal_off.tga')):
    im=crest(src)
    tex=Image.new('RGBA',(512,256),(0,0,0,0));tex.alpha_composite(im,(0,0))
    tex.save(out+'/'+name+'.tga')
    im.save(out+'/'+name+'_preview.png')
print(W*2,H*2)

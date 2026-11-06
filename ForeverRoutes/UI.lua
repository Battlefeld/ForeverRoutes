-- ForeverRoutes owns every frame and setting here. Existing art is reused locally.
local FR=ForeverRoutes
local UI=FR.UI
local C=UI.COLOR
local call=FR.Call
local frame,menu,view
local refresh -- FR.Refresh, set below
local function label(parent,size,color)
 return UI.Text(parent,size or 12,color or C.text,'body','LEFT')
end
local function tip(button,title,body)
 button:SetScript('OnEnter',function(self)
  GameTooltip:SetOwner(self,'ANCHOR_RIGHT');GameTooltip:AddLine(title,C.gold[1],C.gold[2],C.gold[3])
  if body then GameTooltip:AddLine(body,1,1,1,true)end
  if self==frame.crystal and FR.dataStatus then GameTooltip:AddLine(FR.dataStatus.message or'Datenpruefung ausstehend',1,.82,0,true)end
  GameTooltip:Show()
 end)
 button:SetScript('OnLeave',function()GameTooltip:Hide()end)
end
local function button(parent,text,width,fn)
 local b=CreateFrame('Button',nil,parent);b:SetSize(width,28)
 UI.Style(b,'button');b.label=label(b,12);b.label:SetPoint('CENTER');b.label:SetText(text)
 b:SetScript('OnClick',fn);return b
end
-- A small icon button for the window head (no frame art: it sits inside the border, not on it).
local function iconButton(parent,normal,pushed,fn)
 local b=CreateFrame('Button',nil,parent);b:SetSize(20,20)
 b:SetNormalTexture(normal);b:SetPushedTexture(pushed or normal)
 b:SetHighlightTexture('Interface\\Buttons\\UI-Panel-MinimizeButton-Highlight','ADD')
 b:SetScript('OnClick',fn);return b
end
-- A check box with its text: the whole line toggles. get() tells whether it is on, set(on) switches.
local function checkbox(parent,text,get,set,width)
 local b=CreateFrame('Button',nil,parent);b:SetSize(width or 300,22)
 b.box=b:CreateTexture(nil,'ARTWORK');b.box:SetSize(20,20);b.box:SetPoint('LEFT',0,0);b.box:SetTexture('Interface\\Buttons\\UI-CheckBox-Up')
 b.check=b:CreateTexture(nil,'OVERLAY');b.check:SetSize(20,20);b.check:SetPoint('LEFT',0,0);b.check:SetTexture('Interface\\Buttons\\UI-CheckBox-Check')
 b.label=label(b,12,C.text);b.label:SetPoint('LEFT',b.box,'RIGHT',6,0);b.label:SetWidth((width or 300)-26);b.label:SetText(text)
 b.get=get;b.refresh=function(self)self.check:SetShown(self.get()and true or false)end
 b:SetScript('OnClick',function(self)set(not self.get());self:refresh();refresh()end)
 b:refresh();return b
end
local function position(f,key,defaultPoint,dx,dy)
 f:ClearAllPoints();local p=FR.Config()[key]
 if type(p)=='table'and tonumber(p.x)and tonumber(p.y)then f:SetPoint('CENTER',UIParent,'CENTER',p.x,p.y)
 else f:SetPoint(defaultPoint,UIParent,defaultPoint,dx,dy)end
end
local function movable(f,key)
 f:SetMovable(true);f:EnableMouse(true);f:RegisterForDrag('LeftButton')
 f:SetScript('OnDragStart',function(self)if not FR.Config().locked then self:StartMoving()end end)
 f:SetScript('OnDragStop',function(self)
  self:StopMovingOrSizing();local x,y=self:GetCenter();local ux,uy=UIParent:GetCenter()
  if x and ux then local s=self:GetScale();FR.Config()[key]={x=x-ux/s,y=y-uy/s};if key=='window'then FR.Config().docked=false end end
 end)
end
-- children (rows, buttons) take the mouse; a drag on them moves the window all the same
local function passDrag(child,win)
 child:RegisterForDrag('LeftButton')
 child:SetScript('OnDragStart',function()local f=win:GetScript('OnDragStart');if f then f(win)end end)
 child:SetScript('OnDragStop',function()local f=win:GetScript('OnDragStop');if f then f(win)end end)
end
-- One quest display: the window takes the place of the game's quest tracker and lists every quest itself.
-- Config().blizzTracker=true brings the game's tracker back; the window then sits left of it.
local trackerFrame
local hiddenTrackers,hookedTrackers={},{}
local function replacing()return FR.Config().blizzTracker~=true end
local function trackers()return{ObjectiveTrackerFrame,QuestWatchFrame,WatchFrame}end
local function tracker()
 if trackerFrame then return trackerFrame end
 for _,f in ipairs(trackers())do if f and f.IsShown and f:IsShown()then trackerFrame=f;return f end end
end
-- hide the game's tracker while our window shows the quests; out of combat only, Blizzard shows it again on its own
local function applyTracker()
 if call(InCombatLockdown)==true then return end
 local replace=replacing()and frame and frame:IsShown()
 for _,f in ipairs(trackers())do
  if f and f.Hide then
   if not hookedTrackers[f]then
    hookedTrackers[f]=true
    f:HookScript('OnShow',function(self)
     if replacing()and frame and frame:IsShown()and call(InCombatLockdown)~=true then hiddenTrackers[self]=true;self:Hide()end
    end)
   end
   if replace and f:IsShown()then trackerFrame=trackerFrame or f;hiddenTrackers[f]=true;f:Hide()
   elseif not replace and hiddenTrackers[f]then hiddenTrackers[f]=nil;f:Show()end
  end
 end
end
local function dock(f)
 local t=tracker();if not t then return false end
 f:ClearAllPoints()
 if replacing()then f:SetPoint('TOPRIGHT',t,'TOPRIGHT',0,0)else f:SetPoint('TOPRIGHT',t,'TOPLEFT',-14,0)end
 return true
end
local items,placeItems,questList,dungeonSection
local function hide()frame:Hide();if menu then menu:Hide()end;FR.Config().hidden=true;applyTracker();if items and call(InCombatLockdown)~=true then items:Hide()end end
refresh=function()FR.Refresh()end
-- Icons: each step and hint shows what it is at a glance (the game's own icons).
local ICON={
 a='Interface\\GossipFrame\\AvailableQuestIcon',t='Interface\\GossipFrame\\ActiveQuestIcon',
 kill='Interface\\Icons\\Ability_DualWield',item='Interface\\Icons\\INV_Misc_Bag_10',object='Interface\\Icons\\INV_Misc_Gear_01',
 talk='Interface\\GossipFrame\\GossipGossipIcon',g='Interface\\Icons\\Ability_Warrior_BattleShout',d='Interface\\Icons\\INV_Misc_Key_03',
 sleep='Interface\\Icons\\Spell_Nature_Sleep',p='Interface\\Icons\\INV_Misc_Pelt_Wolf_01',z='Interface\\Icons\\INV_Wand_07',
 buy='Interface\\Icons\\INV_Misc_Coin_01',collect='Interface\\Icons\\INV_Misc_Bag_10',travel='Interface\\Icons\\Ability_Mount_Wyvern_01',
 note='Interface\\Icons\\INV_Misc_Note_01',hearth='Interface\\Icons\\INV_Misc_Rune_01',deathskip='Interface\\Icons\\Spell_Holy_Resurrection',
 teleport='Interface\\Icons\\Spell_Arcane_TeleportMoonglade',camp='Interface\\Icons\\Spell_Fire_Fire',bag='Interface\\Icons\\INV_Misc_Bag_08',
 option='Interface\\Icons\\INV_Misc_Note_02',blocked='Interface\\DialogFrame\\UI-Dialog-Icon-AlertNew',
 done='Interface\\RaidFrame\\ReadyCheck-Ready',open='Interface\\RaidFrame\\ReadyCheck-Waiting',
}
function FR.StepIcon(step)
 if not step then return nil end
 if step.op=='w'then return ICON[step.k]or ICON.kill end
 if step.op=='a'and step.sk then return ICON.kill end
 return ICON[step.op]or ICON.note
end
-- callout colors: a band behind the hint, the text in the same tone
local CALLOUT={gold={.95,.75,.3,.16},warn={.9,.3,.25,.2},info={.4,.6,.95,.14}}
local function row(parent)
 local r=CreateFrame('Button',nil,parent);r:SetWidth(340);r:SetHeight(25);r:RegisterForClicks('LeftButtonUp','RightButtonUp')
 r.bg=r:CreateTexture(nil,'BACKGROUND');r.bg:SetPoint('TOPLEFT',-4,1);r.bg:SetPoint('BOTTOMRIGHT',4,-1);r.bg:Hide()
 r.icon=r:CreateTexture(nil,'ARTWORK');r.icon:SetSize(18,18);r.icon:SetPoint('TOPLEFT',0,-1);r.icon:Hide()
 r.text=label(r,13);r.text:SetPoint('TOPLEFT',0,-2);r.text:SetWidth(340);r.text:SetWordWrap(true)
 r:SetScript('OnClick',function(self,which)
  if which=='RightButton'and self.task then FR.HideTask(self.task);refresh()
  elseif self.q then FR.OpenQuest(self.q)end
 end)
 r:SetScript('OnEnter',function(self)
  if self.item then GameTooltip:SetOwner(self,'ANCHOR_RIGHT');GameTooltip:SetItemByID(self.item);GameTooltip:Show()
  elseif self.q then FR.ShowQuestTooltip(self,self.q,self.objective)
  elseif self.task then GameTooltip:SetOwner(self,'ANCHOR_RIGHT');GameTooltip:AddLine(FR.Text(self.task),1,1,1,true);GameTooltip:AddLine('Rechtsklick: Nebenaufgabe ausblenden');GameTooltip:Show()end
 end)
 r:SetScript('OnLeave',function()GameTooltip:Hide();FR.HideQuestTooltip()end)
 r:Hide();return r
end
-- opts: icon (texture path), indent (pixels), callout (a CALLOUT tone: the line as a highlighted band)
local function setRow(r,text,y,color,opts)
 opts=opts or{}
 local x=(opts.indent or 0)+(opts.icon and 24 or 0)
 if opts.icon then r.icon:SetTexture(opts.icon);r.icon:ClearAllPoints();r.icon:SetPoint('TOPLEFT',opts.indent or 0,opts.callout and -3 or -1);r.icon:Show()else r.icon:Hide()end
 r.text:ClearAllPoints();r.text:SetPoint('TOPLEFT',x,opts.callout and -4 or -2);r.text:SetWidth(340-x)
 local tone=opts.callout and CALLOUT[opts.callout]
 if tone then r.bg:SetColorTexture(tone[1],tone[2],tone[3],tone[4]);r.bg:Show()else r.bg:Hide()end
 r.text:SetText(text);r.text:SetTextColor(unpack(color or C.text));r:ClearAllPoints();r:SetPoint('TOPLEFT',frame.content,'TOPLEFT',0,-y)
 local h=math.max(opts.callout and 26 or 20,r.text:GetStringHeight()+(opts.callout and 10 or 6));r:SetHeight(h);r:Show()
 return y+h+(opts.callout and 4 or 0)
end
local function makeFrame()
 if frame then return frame end
 frame=CreateFrame('Frame','ForeverRoutesFrame',UIParent);FR.frame=frame
 frame:SetSize(400,300);frame:SetClampedToScreen(true);frame:SetFrameStrata('MEDIUM');UI.Style(frame,'panel')
 movable(frame,'window')
 -- layout 3: the window takes the quest tracker's place; a place saved before that is dropped once
 if FR.Config().layout~=3 then FR.Config().layout=3;FR.Config().window=nil;FR.Config().docked=true end
 position(frame,'window','LEFT',40,120)
 if FR.Config().docked~=false and not FR.Config().window then frame.docked=dock(frame)end
 frame:SetScale(math.max(.65,math.min(1.5,tonumber(FR.Config().scale)or 1)))
 frame.crystal=CreateFrame('Button',nil,frame);frame.crystal:SetSize(26,26);frame.crystal:SetPoint('TOPLEFT',28,-38)
 frame.crystal.tex=frame.crystal:CreateTexture(nil,'ARTWORK');frame.crystal.tex:SetAllPoints()
 tip(frame.crystal,'ForeverCore','Blau: ForeverCore hat in dieser Sitzung geantwortet. Ohne Verbindung folgt ForeverRoutes der gespeicherten Route.')
 local title=UI.Text(frame,16,C.gold,'title');title:SetPoint('LEFT',frame.crystal,'RIGHT',8,0);title:SetText('ForeverRoutes')
 local close=iconButton(frame,'Interface\\Buttons\\UI-Panel-MinimizeButton-Up','Interface\\Buttons\\UI-Panel-MinimizeButton-Down',hide);close:SetSize(24,24);close:SetPoint('TOPRIGHT',-30,-39)
 tip(close,'Schließen','Mit /fr wieder öffnen.')
 local collapse=iconButton(frame,'Interface\\Buttons\\UI-MinusButton-Up','Interface\\Buttons\\UI-MinusButton-Down',function()FR.Config().compact=not FR.Config().compact;refresh()end);collapse:SetSize(16,16);collapse:SetPoint('RIGHT',close,'LEFT',-4,0)
 tip(collapse,'Kompaktansicht','Nur den aktuellen Schritt mit Fortschritt zeigen.')
 frame.status=label(frame,11,C.muted);frame.status:SetPoint('TOPLEFT',30,-72);frame.status:SetWidth(340);frame.status:SetWordWrap(false)
 frame.current=CreateFrame('Button',nil,frame);frame.current:SetPoint('TOPLEFT',30,-90);frame.current:SetWidth(340)
 frame.current.icon=frame.current:CreateTexture(nil,'ARTWORK');frame.current.icon:SetSize(26,26);frame.current.icon:SetPoint('TOPLEFT',0,2)
 frame.current.text=label(frame.current,15,C.gold);frame.current.text:SetPoint('TOPLEFT',32,0);frame.current.text:SetWidth(308);frame.current.text:SetWordWrap(true)
 frame.current:SetScript('OnClick',function()if view and view.step and view.step.q then FR.OpenQuest(view.step.q)end end)
 frame.current:SetScript('OnEnter',function(self)if view and view.step and view.step.q then FR.ShowQuestTooltip(self,view.step.q)end end)
 frame.current:SetScript('OnLeave',FR.HideQuestTooltip)
 -- below the current step the window has a fixed height: its lines scroll with the mouse wheel instead of
 -- stretching the frame art
 frame.scroll=CreateFrame('ScrollFrame',nil,frame);frame.scroll:EnableMouseWheel(true)
 frame.content=CreateFrame('Frame',nil,frame.scroll);frame.content:SetSize(340,10);frame.scroll:SetScrollChild(frame.content)
 frame.scroll:SetScript('OnMouseWheel',function(self,delta)
  local max=math.max(0,(frame.content:GetHeight()or 0)-(self:GetHeight()or 0))
  self:SetVerticalScroll(math.max(0,math.min(max,(self:GetVerticalScroll()or 0)-delta*40)))
  if placeItems then placeItems()end
 end)
 frame.more=frame:CreateTexture(nil,'OVERLAY');frame.more:SetSize(16,16);frame.more:SetTexture('Interface\\Buttons\\Arrow-Down-Up')
 frame.more:SetPoint('BOTTOMRIGHT',frame.scroll,'BOTTOMRIGHT',4,-14);frame.more:Hide()
 local content=frame.content
 frame.rows={};for i=1,24 do frame.rows[i]=row(content)end
 frame.nextLabel=label(content,11,C.muted)
 frame.nextRows={};for i=1,3 do frame.nextRows[i]=row(content)end
 frame.dungeonLabel=label(content,11,C.muted)
 frame.dungeonRows={};for i=1,2 do frame.dungeonRows[i]=row(content)end
 frame.questLabel=label(content,11,C.muted)
 frame.questRows={};for i=1,60 do frame.questRows[i]=row(content)end
 frame.skip=button(frame,'Überspringen',124,function()FR.Skip();refresh()end);frame.skip:SetPoint('BOTTOM',0,40)
 frame.back=button(frame,'Zurück',90,function()FR.Back();refresh()end);frame.back:SetPoint('RIGHT',frame.skip,'LEFT',-6,0)
  frame.confirm=button(frame,'Erledigt',90,function()FR.Confirm();refresh()end);frame.confirm:SetPoint('LEFT',frame.skip,'RIGHT',6,0)
 tip(frame.confirm,'Schritt erledigt','Markiert den Schritt als fertig. Bei einer Quest, die es im Spiel nicht gibt oder die du nicht annehmen kannst, nimmt es die ganze Quest aus der Planung. Zurück macht es rückgängig.')
 frame.options=iconButton(frame,'Interface\\Icons\\INV_Misc_Gear_01',nil,function()FR.ToggleMenu()end);frame.options:SetSize(16,16);frame.options:SetPoint('RIGHT',collapse,'LEFT',-6,0)
 tip(frame.options,'Routenoptionen','Mit oder ohne Dungeons, Zusatzziele, Position.')
 for _,child in ipairs({frame.current,frame.back,frame.skip,frame.confirm,frame.options})do passDrag(child,frame)end
 for _,r in ipairs(frame.rows)do passDrag(r,frame)end;for _,r in ipairs(frame.nextRows)do passDrag(r,frame)end
 for _,r in ipairs(frame.questRows)do passDrag(r,frame)end;for _,r in ipairs(frame.dungeonRows)do passDrag(r,frame)end
 tip(frame.skip,'Schritt überspringen','Mit Zurück den letzten übersprungenen Schritt wiederherstellen. Die Quest wird nicht als erledigt markiert.')
 return frame
end
-- Route options: switches as check boxes, the route's extra goals as check boxes with their cost, actions as buttons.
local function optionMenu()
 if menu then return menu end
 menu=CreateFrame('Frame','ForeverRoutesOptions',UIParent);menu:SetSize(380,500);menu:SetFrameStrata('DIALOG');menu:SetClampedToScreen(true);UI.Style(menu,'panel')
 menu:SetMovable(true);menu:EnableMouse(true);menu:RegisterForDrag('LeftButton')
 menu:SetScript('OnDragStart',menu.StartMoving);menu:SetScript('OnDragStop',menu.StopMovingOrSizing)
 local title=UI.Text(menu,16,C.gold,'title');title:SetPoint('TOPLEFT',30,-40);title:SetText('Routenoptionen')
 local close=iconButton(menu,'Interface\\Buttons\\UI-Panel-MinimizeButton-Up','Interface\\Buttons\\UI-Panel-MinimizeButton-Down',function()menu:Hide()end);close:SetSize(24,24);close:SetPoint('TOPRIGHT',-30,-37)
 local y=70
 local function head(text)local h=label(menu,11,C.muted);h:SetPoint('TOPLEFT',30,-y);h:SetText(text);y=y+18 end
 menu.checks={}
 local function check(text,get,set,hint)
  local c=checkbox(menu,text,get,set,320);c:SetPoint('TOPLEFT',30,-y);y=y+24
  if hint then tip(c,text,hint)end
  menu.checks[#menu.checks+1]=c
 end
 head('ROUTE')
 check('Mit Dungeons',function()return FR.CharDB().dungeons==true end,function(on)FR.SetDungeons(on)end,'Route mit Gruppen-Dungeons (Questreihen der Dungeons) statt nur solo.')
 check('Blizzard-Questtracker anzeigen',function()return FR.Config().blizzTracker==true end,function()SlashCmdList.FOREVERROUTES('tracker')end,'Aus: dieses Fenster zeigt alle Quests selbst.')
 check('Kompaktansicht',function()return FR.Config().compact==true end,function(on)FR.Config().compact=on or nil end,'Nur den aktuellen Schritt mit Fortschritt zeigen.')
 check('Pfeil anzeigen',function()return not FR.Config().arrowHidden end,function(on)FR.Config().arrowHidden=not on or nil end)
 check('Fenster festsetzen',function()return FR.Config().locked==true end,function(on)FR.Config().locked=on or nil end,'Das Fenster lässt sich nicht mehr verschieben.')
 y=y+6;head('ZUSATZZIELE')
 menu.optY=y
 menu.scroll=CreateFrame('ScrollFrame',nil,menu,'UIPanelScrollFrameTemplate');menu.scroll:SetPoint('TOPLEFT',30,-y);menu.scroll:SetSize(300,140)
 menu.child=CreateFrame('Frame',nil,menu.scroll);menu.child:SetSize(300,140);menu.scroll:SetScrollChild(menu.child);menu.rows={}
 y=y+150;head('AKTIONEN')
 local actions={
  {'Live-Plan neu berechnen',function()FR.ReplanNow()end},
  {'An den Questtracker setzen',function()SlashCmdList.FOREVERROUTES('dock')end},
  {'Übersprungene zurückholen',function()FR.RestoreSkipped()end},
  {'Nebenaufgaben wieder zeigen',function()FR.RestoreTasks()end},
 }
 -- one button per line, as wide as the menu: the texts fit
 for i,a in ipairs(actions)do
  local b=button(menu,a[1],320,function()a[2]();refresh()end)
  b:SetPoint('TOPLEFT',30,-(y+(i-1)*32))
 end
 menu:SetHeight(y+#actions*32+44)
 menu:Hide();return menu
end
function FR.ToggleMenu()
 local f=optionMenu();if f:IsShown()then f:Hide();return end
 f:ClearAllPoints();f:SetPoint('CENTER',UIParent,'CENTER',0,0)
 for _,c in ipairs(f.checks)do c:refresh()end
 for _,r in ipairs(f.rows)do r:Hide()end
 local opts=FR.Options()
 for i,o in ipairs(opts)do
  local b=f.rows[i]
  if not b then
   b=checkbox(f.child,'',function()return false end,function()end,290)
   b.cost=label(b,10,C.muted);b.cost:SetPoint('RIGHT',b,'RIGHT',0,0);b.cost:SetJustifyH('RIGHT')
   f.rows[i]=b
  end
  b.get=function()return(FR.CharDB().opts or{})[o.id]==true end
  b:SetScript('OnClick',function(self)FR.SetOption(i);self:refresh();refresh()end)
  b:ClearAllPoints();b:SetPoint('TOPLEFT',0,-(i-1)*24);b.label:SetText(o.n);b.label:SetWidth(200)
  b.cost:SetText(string.format('%+d Min',o.min or 0));b:refresh()
  tip(b,o.n,(o.nt or'')..string.format('\n%+d Min · Abschluss Stufe %d',o.min or 0,o.l or 0));b:Show()
 end
 f.none=f.none or label(f.child,11,C.muted);f.none:SetPoint('TOPLEFT',0,-2);f.none:SetText('Keine Zusatzziele für diese Route.');f.none:SetShown(#opts==0)
 f.child:SetHeight(math.max(140,#opts*24));f:Show()
end
-- An independent, movable arrow with the same card/crystal art as the route HUD.
local arrow
local function makeArrow()
 if arrow then return end
 -- just the arrow, distance and target under it; no panel behind
 arrow=CreateFrame('Frame','ForeverRoutesArrow',UIParent);FR.arrow=arrow;arrow:SetSize(184,108);arrow:SetClampedToScreen(true)
 movable(arrow,'arrow');position(arrow,'arrow','TOP',0,-110)
 arrow.tex=arrow:CreateTexture(nil,'ARTWORK');arrow.tex:SetSize(64,64);arrow.tex:SetPoint('TOP',0,0)
 if not UI.SetArt(arrow.tex,'route_arrow')then arrow.tex:SetTexture('Interface\\Minimap\\MinimapArrow')end
 arrow.distance=UI.Text(arrow,16,C.gold,'title','CENTER');arrow.distance:SetJustifyH('CENTER');arrow.distance:SetPoint('TOP',0,-66);arrow.distance:SetWidth(158)
 arrow.target=label(arrow,11,C.text);arrow.target:SetPoint('TOP',0,-86);arrow.target:SetWidth(156);arrow.target:SetHeight(24);arrow.target:SetJustifyH('CENTER')
 arrow:Hide()
end
local function updateArrow(step)
 local nav=FR.Navigation(step);FR.UpdateWaypoint(step,nav);makeArrow()
 if not nav or FR.Config().arrowHidden then arrow:Hide();return end
 arrow.target:SetText(nav.dying and'Death Skip'or(step.tg or step.n or FR.Text(step)))
 if nav.distance then
  arrow.distance:SetText(nav.arrived and'Ziel erreicht'or(nav.distance>=1000 and string.format('%.1f km',nav.distance/1000)or string.format('%d m',nav.distance)))
 else arrow.distance:SetText('Anderes Gebiet')end
 if nav.rotation and not nav.arrived then arrow.tex:SetRotation(nav.rotation);arrow.tex:Show()else arrow.tex:Hide()end
 arrow:Show()
end
-- The quest log as the game's tracker would show it, each quest with its place in the route.
local function questLog()
 local out={}
 local n=C_QuestLog and C_QuestLog.GetNumQuestLogEntries and call(C_QuestLog.GetNumQuestLogEntries)or 0
 for i=1,tonumber(n)or 0 do
  local info=call(C_QuestLog.GetInfo,i)
  if type(info)=='table'and type(info.questID)=='number'and info.questID>0 and not info.isHeader and not info.isHidden then
   out[#out+1]={q=info.questID,title=info.title,index=i}
  end
 end
 return out
end
local function role(q)
 if view.step and view.step.q==q then return'jetzt',C.gold end
 local steps=FR.Steps();local ahead=0
 for i=view.index+1,#steps do
  local s=steps[i]
  if not FR.Done(s)and(view.live or not FR.CharDB().skipped[i])then
   ahead=ahead+1
   if s.q==q then
    local what=s.op=='t'and'abgeben'or(s.op=='a'and'annehmen'or'Ziele')
    return ahead==1 and(what..' als Nächstes')or string.format('%s in %d Schritten',what,ahead),C.text
   end
  end
 end
 return'nicht in der Route',C.muted
end
-- Quest items (the tracker's use button): secure buttons on UIParent, never anchored to our window, so the window
-- stays free to move and resize in combat. They are only set and placed out of combat.
local wanted={}
local function itemButton(i)
 items=items or CreateFrame('Frame','ForeverRoutesQuestItems',UIParent)
 items.buttons=items.buttons or{}
 local b=items.buttons[i];if b then return b end
 b=CreateFrame('Button','ForeverRoutesQuestItem'..i,items,'SecureActionButtonTemplate');b:SetSize(22,22);b:SetFrameStrata('HIGH')
 b:RegisterForClicks('AnyUp','AnyDown');b:SetAttribute('type','item')
 b.icon=b:CreateTexture(nil,'ARTWORK');b.icon:SetAllPoints()
 b:SetScript('OnEnter',function(self)if self.link then GameTooltip:SetOwner(self,'ANCHOR_RIGHT');GameTooltip:SetHyperlink(self.link);GameTooltip:Show()end end)
 b:SetScript('OnLeave',function()GameTooltip:Hide()end)
 items.buttons[i]=b;return b
end
placeItems=function()
 if call(InCombatLockdown)==true then return end
 local show=frame and frame:IsShown()and #wanted>0
 if items then items:SetShown(show and true or false)end
 if not show then return end
 items=items or CreateFrame('Frame','ForeverRoutesQuestItems',UIParent);items:Show()
 local scale=(frame:GetEffectiveScale()or 1)/(UIParent:GetEffectiveScale()or 1)
 for i,w in ipairs(wanted)do
  local b=itemButton(i);local left,top=w.row:GetLeft(),w.row:GetTop()
  -- only for a line that is in view (the window's lines scroll)
  local st,sb=frame.scroll:GetTop(),frame.scroll:GetBottom()
  if top and st and sb and(top>st+1 or top-22<sb-1)then top=nil end
  if left and top then
   if b.link~=w.link then b.link=w.link;b:SetAttribute('item',w.item);b.icon:SetTexture(w.icon)end
   b:ClearAllPoints();b:SetPoint('TOPLEFT',UIParent,'BOTTOMLEFT',left*scale,top*scale);b:Show()
  else b:Hide()end
 end
 for i=#wanted+1,#(items.buttons or{})do items.buttons[i]:Hide()end
end
questList=function(y)
 wanted={}
 for _,r in ipairs(frame.questRows)do r:Hide()end
 frame.questLabel:Hide()
 if FR.Config().compact or not replacing()then return y end
 local list=questLog();if #list==0 then return y end
 frame.questLabel:ClearAllPoints();frame.questLabel:SetPoint('TOPLEFT',frame.content,'TOPLEFT',0,-y-6);frame.questLabel:SetText(string.format('QUESTS (%d)',#list));frame.questLabel:Show();y=y+25
 local ri=0
 local function add(text,color,q,objective,indent)
  ri=ri+1;local r=frame.questRows[ri];if not r then return end
  r.q,r.objective,r.item,r.task=q,objective,nil,nil
  y=setRow(r,text,y,color,{indent=indent});return r
 end
 for _,quest in ipairs(list)do
  local tag,color=role(quest.q)
  local link,icon,_,showDone=call(GetQuestLogSpecialItemInfo,quest.index)
  local done=FR.QuestDone(quest.q)
  local hasItem=type(link)=='string'and(not done or showDone)
  local title=quest.title or(C_QuestLog.GetTitleForQuestID and call(C_QuestLog.GetTitleForQuestID,quest.q))or('Quest '..quest.q)
  local r=add(string.format('%s  |cff9aa9c0· %s|r',title,tag),color,quest.q,nil,hasItem and 28 or 0)
  if r and hasItem then wanted[#wanted+1]={row=r,link=link,icon=icon,item=link:match('item:%d+')or link} end
  if done then add('Bereit zum Abgeben',C.good,quest.q,nil,14)
  elseif not(view.step and view.step.q==quest.q)then
   for i,o in ipairs(call(C_QuestLog.GetQuestObjectives,quest.q)or{})do
    if type(o)=='table'and not o.finished then add('- '..(o.text or string.format('%d/%d',o.numFulfilled or 0,o.numRequired or 0)),C.muted,quest.q,i,14)end
   end
  end
 end
 return y
end
-- The window's height limit: a few lines more than the head; the rest scrolls.
FR.WINDOW_MAX=560
-- With dungeons: the next dungeon run of the plan, when (level, steps away) and the quests to take along.
dungeonSection=function(y,group)
 frame.dungeonLabel:Hide();for _,r in ipairs(frame.dungeonRows)do r:Hide()end
 if not group or FR.Config().compact then return y end
 local steps=FR.Steps();local found,ahead
 for i=view.index,#steps do
  local s=steps[i]
  if s.op=='d'and not FR.Done(s)then found,ahead=s,i-view.index;break end
 end
 -- not in the live plan's horizon: the stored route with dungeons knows the next one
 if not found then
  local data=ForeverRoutesData and ForeverRoutesData.routes
  local r=data and data[(FR.Key()or'')..'-dungeons']
  local level=call(UnitLevel,'player')or 0
  for _,s in ipairs(r and r.s or{})do
   if s.op=='d'and not FR.Done(s)and(s.l or 0)>=level then found=s;break end
  end
 end
 frame.dungeonLabel:ClearAllPoints();frame.dungeonLabel:SetPoint('TOPLEFT',frame.content,'TOPLEFT',0,-y-6);frame.dungeonLabel:SetText('DUNGEON');frame.dungeonLabel:Show();y=y+25
 local r1,r2=frame.dungeonRows[1],frame.dungeonRows[2]
 r1.q,r1.item,r1.task,r1.objective=nil,nil,nil,nil;r2.q,r2.item,r2.task,r2.objective=nil,nil,nil,nil
 if not found then return setRow(r1,'Kein Dungeon mehr geplant',y,C.muted,{icon=ICON.d})end
 local when=ahead and(ahead==0 and'jetzt'or string.format('in %d Schritten',ahead))or string.format('ab Stufe %d',found.l or 0)
 y=setRow(r1,string.format('%s  |cff9aa9c0· %s|r',found.n or'Dungeon',when),y,C.gold,{icon=ICON.d})
 if type(found.qi)=='table'and #found.qi>0 then y=setRow(r2,'Mitnehmen: '..table.concat(found.qi,', '),y,C.muted,{indent=24})end
 return y
end
function FR.Refresh()
 FR.CheckBlock();view=FR.TaskView();updateArrow(view.step)
 -- hidden window: give the quests back to the game (also when it was hidden in combat)
 if not frame or not frame:IsShown()then if frame then applyTracker();placeItems()end;return view.step end
 -- the quest tracker comes up after login: dock as soon as it is there
 if not frame.docked and FR.Config().docked~=false and not FR.Config().window then frame.docked=dock(frame)end
 local connected=FR.Connected();UI.SetArt(frame.crystal.tex,connected and'crystal_on'or'crystal_off')
 local group=FR.CharDB().dungeons==true or FR.CharDB().group==true
 -- one short line: where the steps come from and how far (the data check is in the crystal's tooltip)
 local status
 if view.paused then status='Route pausiert: im Dungeon'
 elseif view.live and FR.liveBlock then status=string.format('Live-Plan · %d Quests im Block',FR.BlockQuests())
 elseif view.live then status='Live-Plan'
 else status=string.format('%s · Schritt %d von %d',view.kind=='rxp-fallback'and'Guide'or'Route',math.min(view.index,view.total),view.total)end
 frame.status:SetText(status..(group and' · mit Dungeons'or' · solo'))
 frame.current.text:SetText(view.text)
 local icon=FR.StepIcon(view.step);frame.current.icon:SetShown(icon~=nil);if icon then frame.current.icon:SetTexture(icon)end
 local ch=math.max(28,frame.current.text:GetStringHeight()+8);frame.current:SetHeight(ch)
 local top=90+ch+2
 frame.scroll:ClearAllPoints();frame.scroll:SetPoint('TOPLEFT',frame,'TOPLEFT',30,-top);frame.scroll:SetPoint('BOTTOMRIGHT',frame,'BOTTOMRIGHT',-30,76)
 local y=0
 local ri=0
 local function add(text,color,q,objective,item,task,opts)
  ri=ri+1;local r=frame.rows[ri];if not r then return end
  r.q,r.objective,r.item,r.task=q,objective,item,task;y=setRow(r,text,y,color,opts)
 end
 -- What to do first, as highlighted lines with their icon: travel (hearthstone, deathskip, teleport), the camp,
 -- binding the hearthstone, full bags, a blocked step.
 for _,c in ipairs(FR.Callouts(view))do add(c.text,c.tone=='warn'and{1,.55,.5}or(c.tone=='gold'and C.gold or C.text),nil,nil,nil,nil,{icon=ICON[c.icon],callout=c.tone})end
 -- Quest progress comes from the game, not from the route's estimated counts.
 if view.step and view.step.q then
  for i,o in ipairs(FR.Objectives(view.step))do
   add(o.text or string.format('%d/%d',o.numFulfilled or 0,o.numRequired or 0),o.finished and C.good or C.text,view.step.q,i,nil,nil,{icon=o.finished and ICON.done or ICON.open})
  end
 end
 for _,line in ipairs(FR.GrindProgress(view.step)or{})do add(line,C.text,nil,nil,nil,nil,{icon=ICON.kill})end
 for _,it in ipairs(view.items)do add(it.text,it.done and C.good or C.text,nil,nil,it.id,nil,{icon=it.icon or ICON.item})end
 if not FR.Config().compact then
  local _,camp=FR.CampStatus();if camp then add(camp,C.muted,nil,nil,nil,nil,{icon=ICON.p})end
  for _,task in ipairs(view.parallel)do add('Nebenbei: '..FR.Text(task),C.muted,nil,nil,nil,task,{icon=FR.StepIcon(task)})end
 end
 for i=ri+1,#frame.rows do frame.rows[i]:Hide()end
 frame.nextLabel:Hide();for _,r in ipairs(frame.nextRows)do r:Hide()end
 if not FR.Config().compact and view.step then
  local steps=FR.Steps();local count=0;local last=FR.Text(view.step)
  for i=view.index+1,#steps do
   local s=steps[i]
   -- two steps in a row with the same text (a grind split at a level) read as one line
   if not FR.Done(s)and(view.live or not FR.CharDB().skipped[i])and FR.Text(s)~=last then
    last=FR.Text(s)
    count=count+1
    if count==1 then frame.nextLabel:ClearAllPoints();frame.nextLabel:SetPoint('TOPLEFT',frame.content,'TOPLEFT',0,-y-6);frame.nextLabel:SetText('DANACH');frame.nextLabel:Show();y=y+25 end
    local r=frame.nextRows[count];r.q,r.task,r.item,r.objective=s.q,nil,nil,nil
    y=setRow(r,FR.Text(s),y,C.muted,{icon=FR.StepIcon(s)});if count==3 then break end
   end
  end
 end
 y=dungeonSection(y,group)
 y=questList(y)
 frame.confirm:SetEnabled(FR.CanConfirm(view.step)and true or false)
 frame.skip:SetEnabled(view.step~=nil and not view.paused);frame.back:SetEnabled(not view.paused and(FR.undo~=nil or not view.live))
 -- the window: as high as its lines, at most WINDOW_MAX; more lines scroll
 frame.content:SetHeight(math.max(10,y))
 frame:SetHeight(math.max(230,math.min(FR.WINDOW_MAX,top+y+80)))
 local visible=frame:GetHeight()-top-76
 local over=y-visible
 if over<=0 then frame.scroll:SetVerticalScroll(0)elseif(frame.scroll:GetVerticalScroll()or 0)>over then frame.scroll:SetVerticalScroll(over)end
 frame.more:SetShown(over>4 and(frame.scroll:GetVerticalScroll()or 0)<over-4)
 applyTracker();placeItems()
 return view.step
end
function FR.Show()makeFrame():Show();FR.Config().hidden=nil;FR.Refresh()end
local driver=CreateFrame('Frame');local elapsed=0;local pending=false;local full=0
local events={'PLAYER_LOGIN','PLAYER_ENTERING_WORLD','QUEST_ACCEPTED','QUEST_TURNED_IN','QUEST_REMOVED','QUEST_LOG_UPDATE','PLAYER_LEVEL_UP','PLAYER_DEAD','PLAYER_UNGHOST','PLAYER_ALIVE','UNIT_AURA','BAG_UPDATE_DELAYED','GET_ITEM_INFO_RECEIVED','PLAYER_REGEN_ENABLED','ADDON_LOADED','HEARTHSTONE_BOUND'}
for _,e in ipairs(events)do pcall(driver.RegisterEvent,driver,e)end
driver:SetScript('OnEvent',function(_,event,unit)
 if event=='UNIT_AURA'and unit~='player'then return end
 if event=='PLAYER_LOGIN'then if not FR.Config().hidden then FR.Show()end;FR.ScanQuestTracker();return end
 if event=='PLAYER_DEAD'then FR.Died()end
 if event=='HEARTHSTONE_BOUND'then FR.HearthBound()end
 pending=true
end)
driver:SetScript('OnUpdate',function(_,dt)
 elapsed=elapsed+dt;full=full+dt;if elapsed<.15 then return end;elapsed=0
 if pending or full>=1 then pending=false;full=0;FR.Refresh();FR.ScanQuestTracker()
 else updateArrow(view and view.step);placeItems()end
end)
SLASH_FOREVERROUTES1='/fr'
SlashCmdList.FOREVERROUTES=function(msg)
 msg=strtrim(tostring(msg or'')):lower()
 if msg==''then if frame and frame:IsShown()then hide()else FR.Show()end;return end
 if msg=='hide'then if frame then hide()else FR.Config().hidden=true end;return end
 if msg=='skip'then FR.Skip()elseif msg=='back'or msg=='zurueck'then FR.Back()
 elseif msg=='done'then FR.Confirm()elseif msg=='reset'then FR.Reset()
 elseif msg=='dungeon'or msg=='dungeons'or msg=='mit'or msg=='gruppe'then FR.SetDungeons(true)
 elseif msg=='solo'or msg=='ohne'then FR.SetDungeons(false)
 elseif msg=='lager'then FR.Camped()
 elseif msg=='camp'or msg=='lagercheck'then FR.CampReport();return
 elseif msg=='optionen'or msg=='options'or msg=='option'then FR.ToggleMenu();return
 elseif msg:match('^option%s+%d+$')then FR.SetOption(msg:match('(%d+)$'))
 elseif msg=='restore'then FR.RestoreSkipped();FR.RestoreTasks()
 elseif msg=='compact'then FR.Config().compact=not FR.Config().compact
 elseif msg=='lock'then FR.Config().locked=not FR.Config().locked
 elseif msg=='arrow'then FR.Config().arrowHidden=not FR.Config().arrowHidden
 elseif msg:match('^scale%s+[%d%.]+$')then
  FR.Config().scale=math.max(.65,math.min(1.5,tonumber(msg:match('([%d%.]+)$'))or 1));if frame then frame:SetScale(FR.Config().scale)end
 elseif msg=='neu'or msg=='replan'then FR.ReplanNow()
 elseif msg=='tracker'or msg=='questtracker'then FR.Config().blizzTracker=not FR.Config().blizzTracker or nil;FR.Config().window=nil;FR.Config().docked=true;if frame then frame.docked=dock(frame)end;applyTracker()
 elseif msg=='dock'or msg=='andocken'then FR.Config().window=nil;FR.Config().docked=true;if frame and not dock(frame)then position(frame,'window','LEFT',40,120)end
 elseif msg=='position'then FR.Config().window=nil;FR.Config().arrow=nil;if frame then position(frame,'window','LEFT',40,120)end;if arrow then position(arrow,'arrow','TOP',0,-110)end
 elseif msg~='show'then print('ForeverRoutes: /fr · show · hide · skip · back · done · options · compact · tracker · lock · arrow · scale 1 · position · restore');return end
 FR.Show()
end


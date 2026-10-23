'use strict';
// A red quest's displayed level does not prevent taking it. Keep the real
// minimum-level/prerequisite/log checks; do not relax item-start combat work.
function stepEvent(ev,run,kind,q,task,pt,acceptKind){
 if(kind!==acceptKind||run.qAccept[q]>=0||ev.level<run.qMinLevel[q])return ev.step(kind,q,task,pt);
 const previous=ev.p;
 ev.p={...previous,dangerGap:Math.max(previous.dangerGap,run.qLevel[q]-ev.level)};
 try{return ev.step(kind,q,task,pt);}finally{ev.p=previous;}
}
module.exports={stepEvent};

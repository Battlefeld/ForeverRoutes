'use strict';
const fs=require('node:fs'),path=require('node:path'),{run,verifySnapshot,auditInput}=require('./combat/bridge.cjs');
const {compare}=require('./combat/experiment.cjs');
const args=process.argv.slice(2),value=k=>{const i=args.indexOf(k);return i<0?undefined:args[i+1];};
try{
 if(args.includes('--verify-source'))console.log(JSON.stringify(verifySnapshot(),null,2));
 else{
  const input=value('--input'),output=value('--output');if(!input||!output)throw Error('Use --input fight-sequence.json --output report.json; LUA selects the interpreter');
  const data=JSON.parse(fs.readFileSync(input));
  const report=args.includes('--audit-route-input')?auditInput(data):args.includes('--compare-choices')?compare(data):run(data);
  fs.mkdirSync(path.dirname(path.resolve(output)),{recursive:true});fs.writeFileSync(output,JSON.stringify(report,null,2)+'\n');
  console.log(JSON.stringify({output,combatSeconds:report.combatSeconds,idleSeconds:report.idleSeconds,stopped:report.stopped||null,validated:false,rankingEligible:false}));
  if(report.stopped)process.exitCode=2;
 }
}catch(e){console.error(e.message);process.exitCode=1;}

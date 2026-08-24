import { appendFile, readFile } from 'node:fs/promises';
import process from 'node:process';
import { pathToFileURL } from 'node:url';

export const JIRA_TARGETS = {
  todo: { label: 'À faire', statusId: '10000', transitionId: '11' },
  inProgress: { label: 'En cours', statusId: '10001', transitionId: '21' },
  review: { label: 'Revue en cours', statusId: '10002', transitionId: '31' },
  done: { label: 'Terminé(e)', statusId: '10003', transitionId: '41' }
};
const MANAGED='github-';
const EXPECTED_COUNT=17;
const RETRYABLE=new Set([429,500,502,503,504]);
const sleep=(ms)=>new Promise(r=>setTimeout(r,ms));
const uniq=(a)=>[...new Set(a)];
const safe=(v)=>String(v).replace(/[\r\n|`]+/g,' ').replace(/\s+/g,' ').trim().slice(0,300);
const slug=(v)=>v.toLowerCase().replace(/[^a-z0-9_-]+/g,'-').replace(/^-+|-+$/g,'');

export function validateMapping(mapping){
  if(mapping.version!==1) throw new Error(`Version de mapping non prise en charge : ${mapping.version}`);
  if(mapping.jiraProject!=='SCRUM') throw new Error(`Projet Jira inattendu : ${mapping.jiraProject}`);
  if(!Array.isArray(mapping.mappings)||mapping.mappings.length!==EXPECTED_COUNT) throw new Error(`Le mapping doit contenir exactement les ${EXPECTED_COUNT} tickets Jira SCRUM.`);
  const keys=new Set();
  for(const e of mapping.mappings){
    if(!/^SCRUM-\d+$/.test(e.jira)) throw new Error(`Clé Jira invalide : ${e.jira}`);
    if(keys.has(e.jira)) throw new Error(`Clé Jira dupliquée : ${e.jira}`);
    keys.add(e.jira);
    if(!Array.isArray(e.githubIssues)||!e.githubIssues.length) throw new Error(`${e.jira} ne possède aucun ticket GitHub.`);
    if(!e.githubIssues.includes(e.primary)) throw new Error(`Le ticket principal de ${e.jira} doit figurer dans githubIssues.`);
  }
  const expected=Array.from({length:EXPECTED_COUNT},(_,i)=>`SCRUM-${i+1}`).sort().join('\n');
  if([...keys].sort().join('\n')!==expected) throw new Error(`Le mapping doit couvrir exactement SCRUM-1 à SCRUM-${EXPECTED_COUNT}.`);
  return true;
}

export function jiraAuthHeader(baseUrl,email,token){
  const u=new URL(baseUrl);
  if(u.hostname==='api.atlassian.com') return `Bearer ${token}`;
  if(!email) throw new Error('JIRA_EMAIL requis pour une URL Jira classique.');
  return `Basic ${Buffer.from(`${email}:${token}`).toString('base64')}`;
}

export function validateJiraBaseUrl(baseUrl){
  const u=new URL(baseUrl);
  const scoped=u.protocol==='https:'&&u.hostname==='api.atlassian.com'&&/^\/ex\/jira\/[0-9a-f-]+\/?$/i.test(u.pathname);
  const classic=u.protocol==='https:'&&u.hostname.endsWith('.atlassian.net')&&(u.pathname==='/'||u.pathname==='');
  if(u.username||u.password||u.search||u.hash||(!scoped&&!classic)) throw new Error('URL Jira invalide.');
  if(!u.pathname.endsWith('/'))u.pathname+='/';
  return u;
}

export function buildJiraCandidates(baseUrl,siteUrl,email,token){
  const raw=[];
  if(siteUrl) raw.push({label:'site Jira direct · Basic',base:validateJiraBaseUrl(siteUrl)});
  if(baseUrl) raw.push({label:new URL(baseUrl).hostname==='api.atlassian.com'?'gateway Atlassian · Bearer':'URL Jira configurée · Basic',base:validateJiraBaseUrl(baseUrl)});
  const seen=new Set();
  return raw.filter(c=>{const k=c.base.href;if(seen.has(k))return false;seen.add(k);return true}).map(c=>({...c,auth:jiraAuthHeader(c.base,email,token)}));
}

export function deriveJiraTarget(entry,issues,pulls){
  const primary=issues.get(entry.primary); if(!primary) throw new Error(`Ticket GitHub principal #${entry.primary} introuvable pour ${entry.jira}.`);
  const mapped=entry.githubIssues.map(n=>{const i=issues.get(n);if(!i)throw new Error(`Ticket GitHub #${n} introuvable pour ${entry.jira}.`);return i});
  const linked=uniq(entry.githubIssues.flatMap(n=>pulls.get(n)??[]));
  if(primary.state==='CLOSED') return {key:'done',...JIRA_TARGETS.done,pullRequests:linked};
  if(linked.some(p=>!p.isDraft)) return {key:'review',...JIRA_TARGETS.review,pullRequests:linked};
  if(mapped.some(i=>i.assignees.nodes.length)) return {key:'in-progress',...JIRA_TARGETS.inProgress,pullRequests:linked};
  return {key:'todo',...JIRA_TARGETS.todo,pullRequests:linked};
}

export function managedLabels(existing,entry,issues,target){
  const kept=existing.filter(l=>!l.startsWith(MANAGED));
  const assignees=uniq(entry.githubIssues.flatMap(n=>issues.get(n).assignees.nodes.map(a=>slug(a.login))));
  return uniq([...kept,`github-issue-${entry.primary}`,...entry.githubIssues.filter(n=>n!==entry.primary).map(n=>`github-child-${n}`),...assignees.map(a=>`github-assignee-${a}`),`github-state-${target.key}`,...target.pullRequests.map(p=>`github-pr-${p.number}`)]).sort();
}

async function request(url,options={},attempt=0){
  let r;
  try{r=await fetch(url,{...options,signal:AbortSignal.timeout(20000)});}catch(e){if(attempt<3){await sleep(500*2**attempt);return request(url,options,attempt+1)}throw e}
  if(r.ok)return r.status===204?null:r.json();
  if(RETRYABLE.has(r.status)&&attempt<3){await sleep(500*2**attempt);return request(url,options,attempt+1)}
  let d='';try{d=safe(await r.text())}catch{}
  throw new Error(`HTTP ${r.status} ${new URL(url).pathname}${d?` : ${d}`:''}`);
}

function jiraClient(candidate){
  return (path,options={})=>request(new URL(path,candidate.base),{...options,headers:{accept:'application/json','content-type':'application/json',authorization:candidate.auth,...options.headers}});
}

async function preflightJira(candidates){
  if(!candidates.length) throw new Error('Aucune URL Jira candidate configurée.');
  const failures=[];
  for(const candidate of candidates){
    try{
      const jira=jiraClient(candidate);
      await jira('rest/api/3/issue/SCRUM-1?fields=id,key');
      return {candidate,jira};
    }catch(e){failures.push(`${candidate.label}: ${safe(e.message)}`)}
  }
  throw new Error(`Pré-test SCRUM-1 refusé. ${failures.join(' || ')}`);
}

async function githubState(repository,token){
  const [owner,name]=repository.split('/');
  const query=`query($owner:String!,$name:String!){repository(owner:$owner,name:$name){issues(first:100,states:[OPEN,CLOSED]){nodes{number state assignees(first:20){nodes{login}}}} pullRequests(first:100,states:OPEN){nodes{number isDraft url closingIssuesReferences(first:20){nodes{number}}}}}}`;
  const r=await request('https://api.github.com/graphql',{method:'POST',headers:{authorization:`Bearer ${token}`,'content-type':'application/json'},body:JSON.stringify({query,variables:{owner,name}})});
  if(r.errors?.length)throw new Error(r.errors.map(e=>e.message).join('; '));
  return r.data.repository;
}

function lookups(state){
  const issues=new Map(state.issues.nodes.map(i=>[i.number,i])); const pulls=new Map();
  for(const p of state.pullRequests.nodes) for(const i of p.closingIssuesReferences.nodes){const a=pulls.get(i.number)??[];a.push(p);pulls.set(i.number,a)}
  return {issues,pulls};
}

async function synchronizeEntry(entry,issues,pulls,jira,dryRun){
  const target=deriveJiraTarget(entry,issues,pulls);
  const current=await jira(`rest/api/3/issue/${entry.jira}?fields=status,labels`);
  const desired=managedLabels(current.fields.labels??[],entry,issues,target);
  const labelsChanged=JSON.stringify([...(current.fields.labels??[])].sort())!==JSON.stringify(desired);
  const statusChanged=current.fields.status.id!==target.statusId;
  if(!dryRun&&labelsChanged) await jira(`rest/api/3/issue/${entry.jira}`,{method:'PUT',body:JSON.stringify({fields:{labels:desired}})});
  if(!dryRun&&statusChanged) await jira(`rest/api/3/issue/${entry.jira}/transitions`,{method:'POST',body:JSON.stringify({transition:{id:target.transitionId}})});
  return {jira:entry.jira,github:entry.githubIssues.map(n=>`#${n}`).join(', '),target:target.label,changed:labelsChanged||statusChanged};
}

async function summary(lines){if(process.env.GITHUB_STEP_SUMMARY)await appendFile(process.env.GITHUB_STEP_SUMMARY,lines.join('\n')+'\n')}
function env(name,optional=false){const v=process.env[name]?.trim();if(!v&&!optional)throw new Error(`Configuration manquante : ${name}`);return v??''}

async function run(){
  const repository=env('GITHUB_REPOSITORY'),ghToken=env('GITHUB_TOKEN'),baseUrl=env('JIRA_BASE_URL',true),siteUrl=env('JIRA_SITE_URL',true),jiraToken=env('JIRA_API_TOKEN'),email=env('JIRA_EMAIL',true),dryRun=process.env.DRY_RUN==='true';
  const mapping=JSON.parse(await readFile(new URL('../jira-map.json',import.meta.url),'utf8')); validateMapping(mapping);
  const candidates=buildJiraCandidates(baseUrl,siteUrl,email,jiraToken);
  let selected;
  try{selected=await preflightJira(candidates)}catch(e){
    const lines=['## Synchronisation GitHub → Jira ❌','',`- Pré-test : **SCRUM-1**`,`- Résultat : ${safe(e.message)}`,'- Tickets modifiés : **0**','',`Le workflow s’arrête avant de parcourir les ${mapping.mappings.length} tickets.`];
    await summary(lines);console.log(lines.join('\n'));throw e;
  }
  const state=await githubState(repository,ghToken),{issues,pulls}=lookups(state),jira=selected.jira;
  const results=[],failures=[];
  for(const entry of mapping.mappings){try{results.push(await synchronizeEntry(entry,issues,pulls,jira,dryRun))}catch(e){failures.push({jira:entry.jira,message:safe(e.message)})}}
  const lines=[`## Synchronisation GitHub → Jira ${failures.length?'❌':'✅'}`,'',`- Mode Jira : **${selected.candidate.label}**`,`- Mode : **${dryRun?'simulation':'écriture'}**`,`- Pré-test SCRUM-1 : **OK**`,`- Tickets attendus : **${mapping.mappings.length}**`,`- Tickets traités : **${results.length}**`,`- Échecs : **${failures.length}**`,'','| Jira | GitHub | État cible |','|---|---|---|',...results.map(r=>`| ${r.jira} | ${r.github} | ${r.target} |`)];
  if(failures.length)lines.push('','### Échecs',...failures.map(f=>`- ${f.jira}: ${f.message}`));
  await summary(lines);console.log(lines.join('\n'));if(failures.length)throw new Error(`${failures.length} ticket(s) Jira n’ont pas pu être synchronisés.`);
}
if(process.argv[1]&&import.meta.url===pathToFileURL(process.argv[1]).href)run().catch(e=>{console.error(e.message);process.exitCode=1});

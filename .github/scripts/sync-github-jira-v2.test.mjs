import assert from 'node:assert/strict';
import test from 'node:test';
import {readFile} from 'node:fs/promises';
import {validateMapping,jiraAuthHeader,jiraPreflight,deriveJiraTarget,managedLabels} from './sync-github-jira-v2.mjs';
const issue=(number,state='OPEN',assignees=[])=>({number,state,assignees:{nodes:assignees.map(login=>({login}))}});

test('mapping couvre 17 tickets Jira',async()=>{const m=JSON.parse(await readFile(new URL('../jira-map.json',import.meta.url),'utf8'));assert.equal(validateMapping(m),true);assert.equal(m.mappings.length,17)});
test('jeton avec scopes utilise Basic sur le gateway Atlassian',()=>{assert.equal(jiraAuthHeader('https://api.atlassian.com/ex/jira/696029ff-b2e6-40cb-83e6-4484d60935a8/','x@y.test','abc'),`Basic ${Buffer.from('x@y.test:abc').toString('base64')}`)});
test('jeton classique utilise aussi Basic',()=>{assert.equal(jiraAuthHeader('https://demo.atlassian.net/','x@y.test','abc'),`Basic ${Buffer.from('x@y.test:abc').toString('base64')}`)});
test('email obligatoire avec un jeton API',()=>{assert.throws(()=>jiraAuthHeader('https://api.atlassian.com/ex/jira/696029ff-b2e6-40cb-83e6-4484d60935a8/','','abc'),/JIRA_EMAIL requis/)});
test('pré-vérification contrôle identité puis projet',async()=>{const calls=[];await jiraPreflight(async path=>{calls.push(path);return{}},'SCRUM');assert.deepEqual(calls,['rest/api/3/myself','rest/api/3/project/SCRUM'])});
test('pré-vérification explique une authentification refusée',async()=>{await assert.rejects(jiraPreflight(async()=>{throw new Error('HTTP 401')},'SCRUM'),/Authentification Jira refusée/) });
test('ticket fermé devient terminé',()=>{const e={jira:'SCRUM-17',primary:31,githubIssues:[31]};assert.equal(deriveJiraTarget(e,new Map([[31,issue(31,'CLOSED')]]),new Map()).key,'done')});
test('ticket assigné devient en cours',()=>{const e={jira:'SCRUM-17',primary:31,githubIssues:[31]};assert.equal(deriveJiraTarget(e,new Map([[31,issue(31,'OPEN',['dev'])]]),new Map()).key,'in-progress')});
test('labels métier sont préservés',()=>{const e={jira:'SCRUM-17',primary:31,githubIssues:[31]},issues=new Map([[31,issue(31,'OPEN',['Dev789'])]]),target={key:'in-progress',pullRequests:[]};assert.deepEqual(managedLabels(['tests','github-old'],e,issues,target),['github-assignee-dev789','github-issue-31','github-state-in-progress','tests'])});

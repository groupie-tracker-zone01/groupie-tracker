import assert from 'node:assert/strict';
import test from 'node:test';
import {readFile} from 'node:fs/promises';
import {validateMapping,jiraAuthHeader,buildJiraCandidates,deriveJiraTarget,managedLabels} from './sync-github-jira-v2.mjs';
const issue=(number,state='OPEN',assignees=[])=>({number,state,assignees:{nodes:assignees.map(login=>({login}))}});

test('mapping couvre 17 tickets Jira',async()=>{const m=JSON.parse(await readFile(new URL('../jira-map.json',import.meta.url),'utf8'));assert.equal(validateMapping(m),true);assert.equal(m.mappings.length,17)});
test('gateway scoped utilise Bearer',()=>{assert.equal(jiraAuthHeader('https://api.atlassian.com/ex/jira/696029ff-b2e6-40cb-83e6-4484d60935a8/','x@y.test','abc'),'Bearer abc')});
test('site classique utilise Basic',()=>{assert.match(jiraAuthHeader('https://demo.atlassian.net/','x@y.test','abc'),/^Basic /)});
test('candidats Jira essaient le site direct avant le gateway',()=>{const c=buildJiraCandidates('https://api.atlassian.com/ex/jira/696029ff-b2e6-40cb-83e6-4484d60935a8/','https://demo.atlassian.net/','x@y.test','abc');assert.equal(c.length,2);assert.equal(c[0].label,'site Jira direct · Basic');assert.equal(c[0].base.href,'https://demo.atlassian.net/');assert.match(c[0].auth,/^Basic /);assert.equal(c[1].label,'gateway Atlassian · Bearer');assert.equal(c[1].auth,'Bearer abc')});
test('candidats Jira dédupliquent la même URL',()=>{const c=buildJiraCandidates('https://demo.atlassian.net/','https://demo.atlassian.net/','x@y.test','abc');assert.equal(c.length,1)});
test('ticket fermé devient terminé',()=>{const e={jira:'SCRUM-17',primary:31,githubIssues:[31]};assert.equal(deriveJiraTarget(e,new Map([[31,issue(31,'CLOSED')]]),new Map()).key,'done')});
test('ticket assigné devient en cours',()=>{const e={jira:'SCRUM-17',primary:31,githubIssues:[31]};assert.equal(deriveJiraTarget(e,new Map([[31,issue(31,'OPEN',['dev'])]]),new Map()).key,'in-progress')});
test('labels métier sont préservés',()=>{const e={jira:'SCRUM-17',primary:31,githubIssues:[31]},issues=new Map([[31,issue(31,'OPEN',['Dev789'])]]),target={key:'in-progress',pullRequests:[]};assert.deepEqual(managedLabels(['tests','github-old'],e,issues,target),['github-assignee-dev789','github-issue-31','github-state-in-progress','tests'])});

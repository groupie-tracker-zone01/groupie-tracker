import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  buildLabelUpdates,
  buildManagedLabels,
  deriveJiraTarget,
  fetchWithRetry,
  jiraEndpoint,
  RequestError,
  synchronizeEntry,
  validateJiraBaseUrl,
  validateMapping
} from "./sync-github-jira.mjs";

function issue(number, state = "OPEN", assignees = []) {
  return { number, state, assignees: { nodes: assignees.map((login) => ({ login })) } };
}

function pullRequest(number, isDraft = false) {
  return { number, isDraft, url: `https://github.test/pull/${number}` };
}

const entry = { jira: "SCRUM-9", primary: 6, githubIssues: [6, 20, 21] };

test("le mapping versionné couvre exactement les 16 tickets Jira", async () => {
  const mapping = JSON.parse(await readFile(new URL("../jira-map.json", import.meta.url), "utf8"));
  assert.equal(validateMapping(mapping), true);
  assert.equal(mapping.mappings.length, 16);
});

test("un ticket principal fermé passe Jira à Terminé", () => {
  const issues = new Map([
    [6, issue(6, "CLOSED")],
    [20, issue(20, "OPEN", ["Rafapsou24"])],
    [21, issue(21)]
  ]);
  assert.equal(deriveJiraTarget(entry, issues, new Map()).key, "done");
});

test("une PR non brouillon liée passe Jira en revue", () => {
  const issues = new Map([
    [6, issue(6, "OPEN", ["Rafapsou24"])],
    [20, issue(20)],
    [21, issue(21)]
  ]);
  const pulls = new Map([[20, [pullRequest(42)]]]);
  assert.equal(deriveJiraTarget(entry, issues, pulls).key, "review");
});

test("une assignation passe Jira en cours", () => {
  const issues = new Map([
    [6, issue(6)],
    [20, issue(20, "OPEN", ["Rafapsou24"])],
    [21, issue(21)]
  ]);
  assert.equal(deriveJiraTarget(entry, issues, new Map()).key, "in-progress");
});

test("un ticket libre sans PR reste à faire", () => {
  const issues = new Map([
    [6, issue(6)],
    [20, issue(20)],
    [21, issue(21)]
  ]);
  assert.equal(deriveJiraTarget(entry, issues, new Map()).key, "todo");
});

test("les labels GitHub sont remplacés sans effacer les labels métier", () => {
  const issues = new Map([
    [6, issue(6, "OPEN", ["Rafapsou24"])],
    [20, issue(20, "OPEN", ["Rafapsou24"])],
    [21, issue(21)]
  ]);
  const target = {
    key: "review",
    pullRequests: [pullRequest(42)]
  };
  const labels = buildManagedLabels(
    ["frontend", "github-1", "github-state-todo", "github-assignee-old-user"],
    entry,
    issues,
    target
  );

  assert.deepEqual(labels, [
    "frontend",
    "github-assignee-rafapsou24",
    "github-child-20",
    "github-child-21",
    "github-issue-6",
    "github-pr-42",
    "github-state-review"
  ]);
});

test("la mise à jour Jira ne touche qu’aux labels github-*", () => {
  assert.deepEqual(
    buildLabelUpdates(
      ["frontend", "github-assignee-old", "github-issue-6"],
      ["frontend", "github-assignee-new", "github-issue-6"]
    ),
    [{ remove: "github-assignee-old" }, { add: "github-assignee-new" }]
  );
});

test("une PR brouillon ne passe pas Jira en revue", () => {
  const issues = new Map([
    [6, issue(6, "OPEN", ["Rafapsou24"])],
    [20, issue(20)],
    [21, issue(21)]
  ]);
  const pulls = new Map([[6, [pullRequest(42, true)]]]);
  assert.equal(deriveJiraTarget(entry, issues, pulls).key, "in-progress");
});

test("la fermeture d’un enfant n’achève pas le ticket principal", () => {
  const issues = new Map([
    [6, issue(6)],
    [20, issue(20, "CLOSED")],
    [21, issue(21)]
  ]);
  assert.equal(deriveJiraTarget(entry, issues, new Map()).key, "todo");
});

test("un état déjà synchronisé n’écrit rien dans Jira", async () => {
  const issues = new Map([
    [6, issue(6, "OPEN", ["Rafapsou24"])],
    [20, issue(20)],
    [21, issue(21)]
  ]);
  const calls = [];
  const jira = async (path, options = {}) => {
    calls.push({ path, options });
    return {
      fields: {
        status: { id: "10001" },
        labels: [
          "frontend",
          "github-assignee-rafapsou24",
          "github-child-20",
          "github-child-21",
          "github-issue-6",
          "github-state-in-progress"
        ]
      }
    };
  };

  const result = await synchronizeEntry({
    entry,
    issuesByNumber: issues,
    pullRequestsByIssue: new Map(),
    jira,
    dryRun: false
  });

  assert.equal(result.changed, false);
  assert.equal(calls.length, 1);
  assert.equal(calls[0].options.method, undefined);
});

test("le mode simulation ne modifie ni labels ni statut", async () => {
  const issues = new Map([
    [6, issue(6, "OPEN", ["Rafapsou24"])],
    [20, issue(20)],
    [21, issue(21)]
  ]);
  const calls = [];
  const jira = async (path, options = {}) => {
    calls.push({ path, options });
    return { fields: { status: { id: "10000" }, labels: ["frontend"] } };
  };

  const result = await synchronizeEntry({
    entry,
    issuesByNumber: issues,
    pullRequestsByIssue: new Map(),
    jira,
    dryRun: true
  });

  assert.equal(result.action, "simulation");
  assert.equal(calls.length, 1);
});

test("une transition déjà appliquée après une erreur ambiguë n’est pas rejouée", async () => {
  const issues = new Map([
    [6, issue(6, "OPEN", ["Rafapsou24"])],
    [20, issue(20)],
    [21, issue(21)]
  ]);
  const calls = [];
  const labels = [
    "github-assignee-rafapsou24",
    "github-child-20",
    "github-child-21",
    "github-issue-6",
    "github-state-in-progress"
  ];
  const jira = async (path, options = {}) => {
    calls.push({ path, options });
    if (options.method === "POST") {
      throw new RequestError("réponse ambiguë", { status: 503, retryable: true });
    }
    if (path.endsWith("?fields=status")) {
      return { fields: { status: { id: "10001" } } };
    }
    return { fields: { status: { id: "10000" }, labels } };
  };

  await synchronizeEntry({
    entry,
    issuesByNumber: issues,
    pullRequestsByIssue: new Map(),
    jira,
    dryRun: false
  });

  assert.equal(calls.filter((call) => call.options.method === "POST").length, 1);
});

test("les erreurs HTTP 500 sont retentées", async () => {
  let attempts = 0;
  const response = await fetchWithRetry(
    "https://example.test/resource",
    {},
    async () => {
      attempts += 1;
      if (attempts === 1) {
        return new Response("", { status: 500, headers: { "retry-after": "0" } });
      }
      return new Response("{}", { status: 200 });
    }
  );

  assert.equal(response.status, 200);
  assert.equal(attempts, 2);
});

test("le mapping refuse toute autre série que SCRUM-1 à SCRUM-16", async () => {
  const mapping = JSON.parse(await readFile(new URL("../jira-map.json", import.meta.url), "utf8"));
  mapping.mappings[15].jira = "SCRUM-17";
  assert.throws(() => validateMapping(mapping), /exactement SCRUM-1 à SCRUM-16/);
});

test("l’URL Jira doit rester sur un hôte Atlassian HTTPS sûr", () => {
  assert.equal(validateJiraBaseUrl("https://example.atlassian.net").hostname, "example.atlassian.net");
  const scoped = "https://api.atlassian.com/ex/jira/696029ff-b2e6-40cb-83e6-4484d60935a8";
  assert.equal(
    jiraEndpoint(scoped, "/rest/api/3/issue/SCRUM-1").href,
    `${scoped}/rest/api/3/issue/SCRUM-1`
  );
  assert.throws(() => validateJiraBaseUrl("http://example.atlassian.net"), /URL HTTPS du site Atlassian/);
  assert.throws(() => validateJiraBaseUrl("https://evil.test"), /URL HTTPS du site Atlassian/);
  assert.throws(() => validateJiraBaseUrl("https://user:secret@example.atlassian.net"), /URL HTTPS du site Atlassian/);
});

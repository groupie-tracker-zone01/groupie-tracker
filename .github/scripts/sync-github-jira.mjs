import { appendFile, readFile } from "node:fs/promises";
import process from "node:process";
import { pathToFileURL } from "node:url";

export const JIRA_TARGETS = {
  todo: {
    label: "À faire",
    statusId: "10000",
    transitionId: "11"
  },
  inProgress: {
    label: "En cours",
    statusId: "10001",
    transitionId: "21"
  },
  review: {
    label: "Revue en cours",
    statusId: "10002",
    transitionId: "31"
  },
  done: {
    label: "Terminé(e)",
    statusId: "10003",
    transitionId: "41"
  }
};

const MANAGED_LABEL_PREFIX = "github-";

const RETRYABLE_STATUS_CODES = new Set([429, 500, 502, 503, 504]);
const MAXIMUM_RETRY_DELAY_MS = 120_000;

export class RequestError extends Error {
  constructor(message, { status, retryable = false, retryAfterMs } = {}) {
    super(message);
    this.name = "RequestError";
    this.status = status;
    this.retryable = retryable;
    this.retryAfterMs = retryAfterMs;
  }
}

function unique(values) {
  return [...new Set(values)];
}

function comparable(values) {
  return unique(values).sort().join("\n");
}

function slug(value) {
  return value.toLowerCase().replace(/[^a-z0-9_-]+/g, "-").replace(/^-+|-+$/g, "");
}

function issueLookup(issues) {
  return new Map(issues.map((issue) => [issue.number, issue]));
}

function pullRequestLookup(pullRequests) {
  const byIssue = new Map();
  for (const pullRequest of pullRequests) {
    for (const issue of pullRequest.closingIssuesReferences.nodes) {
      const current = byIssue.get(issue.number) ?? [];
      current.push(pullRequest);
      byIssue.set(issue.number, current);
    }
  }
  return byIssue;
}

export function validateMapping(mapping) {
  if (mapping.version !== 1) {
    throw new Error(`Version de mapping non prise en charge : ${mapping.version}`);
  }

  if (!Array.isArray(mapping.mappings) || mapping.mappings.length !== 16) {
    throw new Error("Le mapping doit contenir exactement les 16 tickets Jira SCRUM.");
  }

  if (mapping.jiraProject !== "SCRUM") {
    throw new Error(`Projet Jira inattendu : ${mapping.jiraProject}`);
  }

  const jiraKeys = new Set();
  for (const entry of mapping.mappings) {
    if (!/^SCRUM-\d+$/.test(entry.jira)) {
      throw new Error(`Clé Jira invalide : ${entry.jira}`);
    }
    if (jiraKeys.has(entry.jira)) {
      throw new Error(`Clé Jira dupliquée : ${entry.jira}`);
    }
    jiraKeys.add(entry.jira);

    if (!Array.isArray(entry.githubIssues) || entry.githubIssues.length === 0) {
      throw new Error(`${entry.jira} ne possède aucun ticket GitHub.`);
    }
    if (!entry.githubIssues.includes(entry.primary)) {
      throw new Error(`Le ticket principal de ${entry.jira} doit figurer dans githubIssues.`);
    }
    if (entry.githubIssues.some((number) => !Number.isInteger(number) || number < 1)) {
      throw new Error(`Numéro de ticket GitHub invalide pour ${entry.jira}.`);
    }
    if (new Set(entry.githubIssues).size !== entry.githubIssues.length) {
      throw new Error(`Numéro de ticket GitHub dupliqué pour ${entry.jira}.`);
    }
  }

  const expectedJiraKeys = Array.from({ length: 16 }, (_, index) => `SCRUM-${index + 1}`);
  if (comparable(jiraKeys) !== comparable(expectedJiraKeys)) {
    throw new Error("Le mapping doit couvrir exactement SCRUM-1 à SCRUM-16.");
  }

  return true;
}

export function deriveJiraTarget(entry, issuesByNumber, pullRequestsByIssue) {
  const primaryIssue = issuesByNumber.get(entry.primary);
  if (!primaryIssue) {
    throw new Error(`Ticket GitHub principal #${entry.primary} introuvable pour ${entry.jira}.`);
  }

  const mappedIssues = entry.githubIssues.map((number) => {
    const issue = issuesByNumber.get(number);
    if (!issue) throw new Error(`Ticket GitHub #${number} introuvable pour ${entry.jira}.`);
    return issue;
  });

  const linkedPullRequests = unique(
    entry.githubIssues.flatMap((number) => pullRequestsByIssue.get(number) ?? [])
  );

  if (primaryIssue.state === "CLOSED") {
    return { key: "done", ...JIRA_TARGETS.done, pullRequests: linkedPullRequests };
  }

  if (linkedPullRequests.some((pullRequest) => !pullRequest.isDraft)) {
    return { key: "review", ...JIRA_TARGETS.review, pullRequests: linkedPullRequests };
  }

  if (mappedIssues.some((issue) => issue.assignees.nodes.length > 0)) {
    return { key: "in-progress", ...JIRA_TARGETS.inProgress, pullRequests: linkedPullRequests };
  }

  return { key: "todo", ...JIRA_TARGETS.todo, pullRequests: linkedPullRequests };
}

export function buildManagedLabels(existingLabels, entry, issuesByNumber, target) {
  const preserved = existingLabels.filter((label) => !label.startsWith(MANAGED_LABEL_PREFIX));

  const mappedIssues = entry.githubIssues.map((number) => issuesByNumber.get(number));
  const assignees = unique(
    mappedIssues.flatMap((issue) => issue.assignees.nodes.map((assignee) => slug(assignee.login)))
  );

  const managed = [
    `github-issue-${entry.primary}`,
    ...entry.githubIssues
      .filter((number) => number !== entry.primary)
      .map((number) => `github-child-${number}`),
    ...assignees.map((login) => `github-assignee-${login}`),
    `github-state-${target.key}`,
    ...target.pullRequests.map((pullRequest) => `github-pr-${pullRequest.number}`)
  ];

  return unique([...preserved, ...managed]).sort();
}

export function buildLabelUpdates(existingLabels, desiredLabels) {
  const currentManaged = existingLabels.filter((label) => label.startsWith(MANAGED_LABEL_PREFIX));
  const desiredManaged = desiredLabels.filter((label) => label.startsWith(MANAGED_LABEL_PREFIX));
  const desiredSet = new Set(desiredManaged);
  const currentSet = new Set(currentManaged);

  return [
    ...currentManaged.filter((label) => !desiredSet.has(label)).map((label) => ({ remove: label })),
    ...desiredManaged.filter((label) => !currentSet.has(label)).map((label) => ({ add: label }))
  ];
}

function retryDelay(response, attempt) {
  const retryAfter = response.headers.get("retry-after");
  if (retryAfter) {
    const seconds = Number(retryAfter);
    if (Number.isFinite(seconds)) return Math.max(seconds * 1000, 0);

    const date = Date.parse(retryAfter);
    if (Number.isFinite(date)) return Math.max(date - Date.now(), 0);
  }

  return Math.min(1000 * 2 ** attempt, 10_000) + Math.floor(Math.random() * 250);
}

function sleep(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

function endpointPath(url) {
  try {
    return new URL(url).pathname;
  } catch {
    return "l’API distante";
  }
}

function safeMessage(value) {
  return String(value).replace(/[\r\n|`]+/g, " ").replace(/\s+/g, " ").trim().slice(0, 240);
}

async function responseDiagnostic(response) {
  try {
    const raw = await response.text();
    const body = JSON.parse(raw);
    const messages = [
      ...(Array.isArray(body.errorMessages) ? body.errorMessages : []),
      ...(body.errors && typeof body.errors === "object" ? Object.values(body.errors) : [])
    ].filter(Boolean);
    return safeMessage(messages.join("; "));
  } catch {
    return "";
  }
}

export async function fetchWithRetry(
  url,
  options = {},
  fetchImplementation = fetch,
  { maximumAttempts = 4, timeoutMs = 20_000 } = {}
) {
  let lastNetworkError;

  for (let attempt = 0; attempt < maximumAttempts; attempt += 1) {
    let response;
    try {
      response = await fetchImplementation(url, {
        ...options,
        signal: options.signal ?? AbortSignal.timeout(timeoutMs)
      });
    } catch (error) {
      lastNetworkError = error;
      if (attempt === maximumAttempts - 1) {
        throw new RequestError(
          `Échec réseau vers ${endpointPath(url)} après ${maximumAttempts} tentative(s).`,
          { retryable: true }
        );
      }
      await sleep(Math.min(1000 * 2 ** attempt, 10_000) + Math.floor(Math.random() * 250));
      continue;
    }

    if (response.ok) return response;

    const retryable = RETRYABLE_STATUS_CODES.has(response.status);
    const delay = retryable ? retryDelay(response, attempt) : undefined;
    if (!RETRYABLE_STATUS_CODES.has(response.status) || attempt === maximumAttempts - 1) {
      const diagnostic = await responseDiagnostic(response);
      throw new RequestError(
        `Requête HTTP ${response.status} vers ${endpointPath(url)}${diagnostic ? ` : ${diagnostic}` : ""}`,
        { status: response.status, retryable, retryAfterMs: delay }
      );
    }

    if (delay > MAXIMUM_RETRY_DELAY_MS) {
      throw new RequestError(
        `Requête HTTP ${response.status} vers ${endpointPath(url)} : délai Retry-After de ${Math.ceil(delay / 1000)} s, prochain passage planifié sans nouvelle tentative.`,
        { status: response.status, retryable: true, retryAfterMs: delay }
      );
    }
    await sleep(delay);
  }

  throw new RequestError(`Échec réseau vers ${endpointPath(url)} : ${safeMessage(lastNetworkError?.name ?? "erreur inconnue")}.`);
}

async function jsonRequest(url, options, fetchImplementation = fetch, retryOptions = {}) {
  const response = await fetchWithRetry(url, options, fetchImplementation, retryOptions);
  if (response.status === 204) return null;
  return response.json();
}

async function loadGitHubState(repository, token) {
  const [owner, name] = repository.split("/");
  if (!owner || !name) throw new Error(`GITHUB_REPOSITORY invalide : ${repository}`);

  const query = `
    query RepositoryWork($owner: String!, $name: String!) {
      repository(owner: $owner, name: $name) {
        issues(first: 100, states: [OPEN, CLOSED], orderBy: {field: CREATED_AT, direction: ASC}) {
          pageInfo { hasNextPage }
          nodes {
            number
            state
            assignees(first: 20) { nodes { login } }
          }
        }
        pullRequests(first: 100, states: OPEN, orderBy: {field: UPDATED_AT, direction: DESC}) {
          pageInfo { hasNextPage }
          nodes {
            number
            isDraft
            url
            closingIssuesReferences(first: 20) {
              pageInfo { hasNextPage }
              nodes { number }
            }
          }
        }
      }
    }
  `;

  const response = await jsonRequest("https://api.github.com/graphql", {
    method: "POST",
    headers: {
      accept: "application/vnd.github+json",
      authorization: `Bearer ${token}`,
      "content-type": "application/json",
      "user-agent": "groupie-tracker-jira-sync"
    },
    body: JSON.stringify({ query, variables: { owner, name } })
  });

  if (response.errors?.length) {
    throw new Error(`GraphQL GitHub : ${response.errors.map((error) => error.message).join("; ")}`);
  }

  const repositoryState = response.data?.repository;
  if (!repositoryState) throw new Error("Le dépôt GitHub est introuvable ou inaccessible.");

  if (repositoryState.issues.pageInfo.hasNextPage) {
    throw new Error("Plus de 100 tickets GitHub : synchronisation arrêtée pour éviter un état incomplet.");
  }
  if (repositoryState.pullRequests.pageInfo.hasNextPage) {
    throw new Error("Plus de 100 PR ouvertes : synchronisation arrêtée pour éviter un état incomplet.");
  }
  const truncatedPullRequest = repositoryState.pullRequests.nodes.find(
    (pullRequest) => pullRequest.closingIssuesReferences.pageInfo.hasNextPage
  );
  if (truncatedPullRequest) {
    throw new Error(`La PR #${truncatedPullRequest.number} référence plus de 20 tickets : état incomplet.`);
  }
  return repositoryState;
}

export function validateJiraBaseUrl(baseUrl) {
  const base = new URL(baseUrl);
  const hasUnsafeParts = base.protocol !== "https:" || base.username || base.password || base.search || base.hash;
  const classicEndpoint = base.hostname.endsWith(".atlassian.net") && base.pathname === "/";
  const scopedEndpoint = base.hostname === "api.atlassian.com" &&
    /^\/ex\/jira\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\/?$/i.test(base.pathname);

  if (hasUnsafeParts || (!classicEndpoint && !scopedEndpoint)) {
    throw new Error("JIRA_BASE_URL doit être l’URL HTTPS du site Atlassian ou l’URL officielle d’un jeton Jira scoped.");
  }
  if (!base.pathname.endsWith("/")) base.pathname += "/";
  return base;
}

export function jiraEndpoint(baseUrl, path) {
  const base = validateJiraBaseUrl(baseUrl);
  return new URL(String(path).replace(/^\/+/, ""), base);
}

function jiraClient(baseUrl, email, token) {
  const base = validateJiraBaseUrl(baseUrl);
  const authorization = `Basic ${Buffer.from(`${email}:${token}`).toString("base64")}`;

  return async (path, options = {}, retryOptions = {}) => jsonRequest(jiraEndpoint(base, path), {
    ...options,
    headers: {
      accept: "application/json",
      authorization,
      "content-type": "application/json",
      ...options.headers
    }
  }, fetch, retryOptions);
}

async function transitionJiraIssue({ jira, entry, target }) {
  const transitionPath = `/rest/api/3/issue/${entry.jira}/transitions`;

  for (let attempt = 0; attempt < 4; attempt += 1) {
    try {
      await jira(transitionPath, {
        method: "POST",
        body: JSON.stringify({ transition: { id: target.transitionId } })
      }, { maximumAttempts: 1 });
      return;
    } catch (error) {
      if (!(error instanceof RequestError) || !error.retryable) throw error;

      const refreshed = await jira(`/rest/api/3/issue/${entry.jira}?fields=status`);
      if (refreshed.fields.status.id === target.statusId) return;
      if (attempt === 3 || error.retryAfterMs > MAXIMUM_RETRY_DELAY_MS) throw error;

      const delay = error.retryAfterMs ?? Math.min(1000 * 2 ** attempt, 10_000);
      await sleep(delay);
    }
  }
}

export async function synchronizeEntry({ entry, issuesByNumber, pullRequestsByIssue, jira, dryRun }) {
  const target = deriveJiraTarget(entry, issuesByNumber, pullRequestsByIssue);
  const current = await jira(`/rest/api/3/issue/${entry.jira}?fields=status,labels`);
  const currentStatusId = current.fields.status.id;
  const currentLabels = current.fields.labels ?? [];
  const desiredLabels = buildManagedLabels(currentLabels, entry, issuesByNumber, target);
  const labelUpdates = buildLabelUpdates(currentLabels, desiredLabels);
  const labelsChanged = labelUpdates.length > 0;
  const statusChanged = currentStatusId !== target.statusId;

  if (!dryRun && labelsChanged) {
    await jira(`/rest/api/3/issue/${entry.jira}`, {
      method: "PUT",
      body: JSON.stringify({ update: { labels: labelUpdates } })
    });
  }

  if (!dryRun && statusChanged) {
    await transitionJiraIssue({ jira, entry, target });
  }

  return {
    jira: entry.jira,
    github: entry.githubIssues.map((number) => `#${number}`).join(", "),
    target: target.label,
    changed: labelsChanged || statusChanged,
    action: dryRun && (labelsChanged || statusChanged) ? "simulation" : labelsChanged || statusChanged ? "mis à jour" : "inchangé"
  };
}

async function writeStepSummary(lines) {
  if (process.env.GITHUB_STEP_SUMMARY) {
    await appendFile(process.env.GITHUB_STEP_SUMMARY, `${lines.join("\n")}\n`);
  }
}

function requireEnvironment(name) {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`Configuration manquante : ${name}`);
  return value;
}

async function run() {
  const repository = requireEnvironment("GITHUB_REPOSITORY");
  const githubToken = requireEnvironment("GITHUB_TOKEN");
  const jiraBaseUrl = requireEnvironment("JIRA_BASE_URL");
  const jiraEmail = requireEnvironment("JIRA_EMAIL");
  const jiraToken = requireEnvironment("JIRA_API_TOKEN");
  const dryRun = process.env.DRY_RUN?.toLowerCase() === "true";

  const mappingUrl = new URL("../jira-map.json", import.meta.url);
  const mapping = JSON.parse(await readFile(mappingUrl, "utf8"));
  validateMapping(mapping);

  const github = await loadGitHubState(repository, githubToken);
  const issuesByNumber = issueLookup(github.issues.nodes);
  const pullRequestsByIssue = pullRequestLookup(github.pullRequests.nodes);
  const jira = jiraClient(jiraBaseUrl, jiraEmail, jiraToken);
  const results = [];
  const failures = [];

  for (const entry of mapping.mappings) {
    try {
      results.push(await synchronizeEntry({
        entry,
        issuesByNumber,
        pullRequestsByIssue,
        jira,
        dryRun
      }));
    } catch (error) {
      failures.push({ jira: entry.jira, message: safeMessage(error.message) });
    }
  }

  const changed = results.filter((result) => result.changed).length;
  const lines = [
    `## Synchronisation GitHub → Jira ${failures.length ? "❌" : dryRun ? "🧪" : "✅"}`,
    "",
    `- Mode : **${dryRun ? "simulation" : "écriture"}**`,
    `- Tickets Jira attendus : **${mapping.mappings.length}**`,
    `- Tickets traités : **${results.length}**`,
    `- Changements : **${changed}**`,
    `- Échecs : **${failures.length}**`,
    `- GitHub sans équivalent Jira : **${Object.keys(mapping.unmappedGitHubIssues).map((number) => `#${number}`).join(", ")}**`,
    "",
    "| Jira | GitHub | État cible | Résultat |",
    "|---|---|---|---|",
    ...results.map((result) => `| ${result.jira} | ${result.github} | ${result.target} | ${result.action} |`)
  ];

  if (failures.length) {
    lines.push("", "### Échecs", ...failures.map((failure) => `- ${failure.jira}: ${failure.message}`));
  }

  await writeStepSummary(lines);
  console.log(lines.join("\n"));

  if (failures.length) {
    const error = new Error(`${failures.length} ticket(s) Jira n’ont pas pu être synchronisés.`);
    error.summaryWritten = true;
    throw error;
  }
}

const invokedDirectly = process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href;
if (invokedDirectly) {
  run().catch(async (error) => {
    const message = error instanceof Error ? error.message : String(error);
    console.error(message);
    if (!error?.summaryWritten) {
      await writeStepSummary(["## Synchronisation GitHub → Jira ❌", "", `- ${safeMessage(message)}`]);
    }
    process.exitCode = 1;
  });
}

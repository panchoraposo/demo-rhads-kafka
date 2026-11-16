'use strict';

// Scorecard DORA collector for this demo. Delivery is a Tekton PipelineRun,
// lead time is the GitLab commit time of the image tag, and a failed run is
// an incident closed by the next successful run of the same component.

const { createBackendModule, coreServices } = require('@backstage/backend-plugin-api');
const {
  scorecardCollectorsExtensionPoint,
} = require('/opt/app-root/src/dynamic-plugins-root/red-hat-developer-hub-backstage-plugin-scorecard-backend/node_modules/@red-hat-developer-hub/backstage-plugin-scorecard-node');
const { z } = require('zod');

const DEPLOYMENTS_ID = 'tekton:doraDeployments';
const PULL_REQUESTS_ID = 'tekton:doraDeploymentPullRequests';
const INCIDENTS_ID = 'tekton:doraIncidents';

const deploymentsInputSchema = z
  .object({
    from: z.string().datetime(),
    to: z.string().datetime(),
  })
  .passthrough();

const deploymentsOutputSchema = z
  .object({
    deployments: z.array(
      z
        .object({
          id: z.string().min(1),
          commitSha: z.string().min(1),
          environment: z.string().optional(),
          createdAt: z.string().datetime(),
          result: z.enum(['success', 'failure', '']),
        })
        .passthrough(),
    ),
  })
  .strict();

const pullRequestsInputSchema = z
  .object({
    baseCommitSha: z.string().min(1),
    headCommitSha: z.string().min(1),
  })
  .passthrough();

const pullRequestsOutputSchema = z
  .object({
    pullRequests: z.array(
      z
        .object({
          id: z.string().min(1),
          firstCommitAt: z.string().datetime(),
        })
        .passthrough(),
    ),
  })
  .strict();

const incidentsInputSchema = z
  .object({
    from: z.string().datetime(),
    to: z.string().datetime(),
    updatedSince: z.string().datetime(),
  })
  .passthrough();

const incidentsOutputSchema = z
  .object({
    incidents: z.array(
      z
        .object({
          id: z.string(),
          createdAt: z.string().datetime(),
          updatedAt: z.string().datetime(),
          resolutionAt: z.string().datetime().nullable(),
        })
        .passthrough(),
    ),
  })
  .strict();

function iso(value) {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return date.toISOString();
}

function inWindow(stamp, from, to) {
  const time = new Date(stamp).getTime();
  return time >= new Date(from).getTime() && time <= new Date(to).getTime();
}

function componentId(entity) {
  return (
    entity?.metadata?.annotations?.['backstage.io/kubernetes-id'] ||
    entity?.metadata?.name ||
    ''
  );
}

function projectSlug(entity) {
  const raw = entity?.metadata?.annotations?.['gitlab.com/project-slug'] || '';
  if (!raw) {
    return '';
  }
  if (raw.startsWith('http://') || raw.startsWith('https://')) {
    try {
      return new URL(raw).pathname.replace(/^\//, '').replace(/\.git$/, '');
    } catch {
      return '';
    }
  }
  return raw.replace(/\.git$/, '');
}

function environmentOf(namespace) {
  const suffix = String(namespace || '').split('-').pop();
  if (suffix === 'prod' || suffix === 'production') {
    return 'production';
  }
  return suffix || 'production';
}

function param(run, name) {
  const found = (run.spec?.params || []).find(item => item.name === name);
  return found && found.value != null ? String(found.value) : '';
}

function commitShaOf(run) {
  const tag = param(run, 'image-tag');
  if (/^[0-9a-f]{7,64}$/i.test(tag)) {
    return tag;
  }
  return run.metadata?.uid || run.metadata?.name || 'unknown';
}

function resultOf(run) {
  const condition = (run.status?.conditions || []).find(item => item.type === 'Succeeded');
  if (!condition) {
    return '';
  }
  if (condition.status === 'True') {
    return 'success';
  }
  if (condition.status === 'False' && condition.reason !== 'Running') {
    return 'failure';
  }
  return '';
}

function belongsTo(run, id) {
  if (!id) {
    return false;
  }
  const labels = run.metadata?.labels || {};
  if (labels['backstage.io/kubernetes-id'] === id) {
    return true;
  }
  if (labels['app.kubernetes.io/name'] === id) {
    return true;
  }
  return String(run.metadata?.namespace || '').startsWith(`${id}-`);
}

let pipelineCache = { at: 0, items: [] };

async function kubeJson(apiPath) {
  const token = (process.env.KUBERNETES_SA_TOKEN || '').trim();
  if (!token) {
    throw new Error('KUBERNETES_SA_TOKEN is empty; the Tekton collector cannot list PipelineRuns');
  }
  const response = await fetch(`https://kubernetes.default.svc${apiPath}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) {
    const body = await response.text();
    throw new Error(
      `Kubernetes ${apiPath} returned ${response.status}: ${body.slice(0, 240)}`,
    );
  }
  return response.json();
}

async function listPipelineRuns() {
  if (pipelineCache.items.length && Date.now() - pipelineCache.at < 20000) {
    return pipelineCache.items;
  }
  const items = [];
  let token = '';
  do {
    const query = token
      ? `/apis/tekton.dev/v1/pipelineruns?limit=200&continue=${encodeURIComponent(token)}`
      : '/apis/tekton.dev/v1/pipelineruns?limit=200';
    const page = await kubeJson(query);
    items.push(...(page.items || []));
    token = page.metadata?.continue || '';
  } while (token);
  pipelineCache = { at: Date.now(), items };
  return items;
}

function finishedRuns(runs, id) {
  return runs
    .filter(run => belongsTo(run, id))
    .map(run => {
      const completed = run.status?.completionTime;
      if (!completed) {
        return null;
      }
      return {
        id: `${run.metadata.namespace}/${run.metadata.name}`,
        commitSha: commitShaOf(run),
        environment: environmentOf(run.metadata.namespace),
        createdAt: iso(completed),
        result: resultOf(run),
        namespace: run.metadata.namespace,
        name: run.metadata.name,
      };
    })
    .filter(Boolean)
    .sort((left, right) => new Date(left.createdAt) - new Date(right.createdAt));
}

function gitlabBase() {
  const host = (process.env.GITLAB_HOST || '').trim().replace(/\/$/, '');
  if (!host) {
    throw new Error('GITLAB_HOST is empty; the Tekton collector cannot read commit times');
  }
  return host.startsWith('http://') || host.startsWith('https://') ? host : `https://${host}`;
}

const commitDateCache = new Map();

async function commitDate(slug, sha) {
  const key = `${slug}@${sha}`;
  if (commitDateCache.has(key)) {
    return commitDateCache.get(key);
  }
  const token = (process.env.GITLAB_TOKEN || '').trim();
  if (!token) {
    throw new Error('GITLAB_TOKEN is empty; the Tekton collector cannot read commit times');
  }
  const url = `${gitlabBase()}/api/v4/projects/${encodeURIComponent(slug)}/repository/commits/${encodeURIComponent(sha)}`;
  const response = await fetch(url, { headers: { 'PRIVATE-TOKEN': token } });
  if (!response.ok) {
    throw new Error(`GitLab commit ${sha} in ${slug} returned ${response.status}`);
  }
  const body = await response.json();
  const stamp = iso(body.authored_date || body.committed_date || body.created_at);
  if (!stamp) {
    throw new Error(`GitLab commit ${sha} in ${slug} has no timestamp`);
  }
  commitDateCache.set(key, stamp);
  return stamp;
}

async function commitsInRange(slug, baseSha, headSha) {
  if (baseSha === headSha) {
    return [{ id: headSha, firstCommitAt: await commitDate(slug, headSha) }];
  }
  const token = (process.env.GITLAB_TOKEN || '').trim();
  if (!token) {
    throw new Error('GITLAB_TOKEN is empty; the Tekton collector cannot read commit times');
  }
  const url =
    `${gitlabBase()}/api/v4/projects/${encodeURIComponent(slug)}/repository/compare` +
    `?from=${encodeURIComponent(baseSha)}&to=${encodeURIComponent(headSha)}`;
  const response = await fetch(url, { headers: { 'PRIVATE-TOKEN': token } });
  if (!response.ok) {
    throw new Error(
      `GitLab compare ${baseSha}..${headSha} in ${slug} returned ${response.status}`,
    );
  }
  const body = await response.json();
  const commits = Array.isArray(body.commits) ? body.commits : [];
  if (commits.length === 0) {
    return [{ id: headSha, firstCommitAt: await commitDate(slug, headSha) }];
  }
  return commits
    .map(commit => ({
      id: String(commit.id || commit.short_id || ''),
      firstCommitAt: iso(commit.authored_date || commit.committed_date || commit.created_at),
    }))
    .filter(commit => commit.id && commit.firstCommitAt);
}

class TektonDeploymentsCollector {
  constructor(logger) {
    this.logger = logger;
  }
  getCollectorId() {
    return DEPLOYMENTS_ID;
  }
  getCollectorDescription() {
    return 'Successful and failed Tekton PipelineRuns for a component';
  }
  getInputSchema() {
    return deploymentsInputSchema;
  }
  getOutputSchema() {
    return deploymentsOutputSchema;
  }
  async collect({ entity, input }) {
    const id = componentId(entity);
    const runs = finishedRuns(await listPipelineRuns(), id).filter(run =>
      inWindow(run.createdAt, input.from, input.to),
    );
    this.logger.info(
      `Tekton DORA deployments for ${id}: ${runs.length} PipelineRuns in the requested window`,
    );
    return {
      deployments: runs.map(run => ({
        id: run.id,
        commitSha: run.commitSha,
        environment: run.environment,
        createdAt: run.createdAt,
        result: run.result,
      })),
    };
  }
}

class TektonPullRequestsCollector {
  constructor(logger) {
    this.logger = logger;
  }
  getCollectorId() {
    return PULL_REQUESTS_ID;
  }
  getCollectorDescription() {
    return 'GitLab commits between two Tekton deployment SHAs';
  }
  getInputSchema() {
    return pullRequestsInputSchema;
  }
  getOutputSchema() {
    return pullRequestsOutputSchema;
  }
  async collect({ entity, input }) {
    const slug = projectSlug(entity);
    if (!slug) {
      throw new Error(
        `Component ${componentId(entity)} has no gitlab.com/project-slug annotation`,
      );
    }
    const pullRequests = await commitsInRange(slug, input.baseCommitSha, input.headCommitSha);
    this.logger.info(
      `Tekton DORA commits for ${slug} ${input.baseCommitSha.slice(0, 8)}..${input.headCommitSha.slice(0, 8)}: ${pullRequests.length}`,
    );
    return { pullRequests };
  }
}

class TektonIncidentsCollector {
  constructor(logger) {
    this.logger = logger;
  }
  getCollectorId() {
    return INCIDENTS_ID;
  }
  getCollectorDescription() {
    return 'Failed Tekton PipelineRuns, closed by the next successful run';
  }
  getInputSchema() {
    return incidentsInputSchema;
  }
  getOutputSchema() {
    return incidentsOutputSchema;
  }
  async collect({ entity, input }) {
    const id = componentId(entity);
    const runs = finishedRuns(await listPipelineRuns(), id);
    const incidents = [];
    for (let index = 0; index < runs.length; index += 1) {
      const run = runs[index];
      if (run.result !== 'failure') {
        continue;
      }
      const updatedAt = run.createdAt;
      if (new Date(updatedAt).getTime() < new Date(input.updatedSince).getTime()) {
        continue;
      }
      if (!inWindow(run.createdAt, input.from, input.to)) {
        continue;
      }
      const restored = runs
        .slice(index + 1)
        .find(candidate => candidate.result === 'success');
      incidents.push({
        id: run.id,
        createdAt: run.createdAt,
        updatedAt,
        resolutionAt: restored ? restored.createdAt : null,
      });
    }
    this.logger.info(`Tekton DORA incidents for ${id}: ${incidents.length} failed PipelineRuns`);
    return { incidents };
  }
}

const scorecardModuleTektonDora = createBackendModule({
  pluginId: 'scorecard',
  moduleId: 'tekton-dora',
  register(reg) {
    reg.registerInit({
      deps: {
        collectors: scorecardCollectorsExtensionPoint,
        logger: coreServices.logger,
      },
      async init({ collectors, logger }) {
        collectors.addCollector(
          new TektonDeploymentsCollector(logger),
          new TektonPullRequestsCollector(logger),
          new TektonIncidentsCollector(logger),
        );
        logger.info(
          'Registered Tekton DORA collectors tekton:doraDeployments, tekton:doraDeploymentPullRequests, tekton:doraIncidents',
        );
      },
    });
  },
});

module.exports = {
  default: scorecardModuleTektonDora,
};

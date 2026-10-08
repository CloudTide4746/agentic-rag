const { spawnSync } = require('node:child_process');

function parseArgs(argv) {
  const result = {};
  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index];
    if (!token.startsWith('--')) continue;
    const key = token.slice(2);
    const value = argv[index + 1];
    if (!value || value.startsWith('--')) {
      result[key] = true;
    } else {
      result[key] = value;
      index += 1;
    }
  }
  return result;
}

function redisOptions(args) {
  const password = args.password || process.env.REDIS_PASSWORD;
  return {
    binary: args['redis-cli'] || process.env.REDIS_CLI || 'redis-cli',
    host: args.host || process.env.REDIS_HOST || 'localhost',
    port: Number(args.port || process.env.REDIS_PORT || 6379),
    db: Number(args.db || process.env.REDIS_DB || 0),
    password,
  };
}

function runRedis(commandArgs, options) {
  const environment = { ...process.env };
  if (options.password) environment.REDISCLI_AUTH = options.password;
  const result = spawnSync(
    options.binary,
    ['--raw', '-h', options.host, '-p', String(options.port), '-n', String(options.db), ...commandArgs],
    { encoding: 'utf8', env: environment },
  );
  if (result.error) throw new Error(`无法执行 ${options.binary}: ${result.error.message}`);
  if (result.status !== 0) throw new Error((result.stderr || result.stdout || 'redis-cli 执行失败').trim());
  return result.stdout.trim();
}

function scanKeys(options, pattern, maxKeys) {
  const output = runRedis(['--scan', '--pattern', pattern, '--count', '1000'], options);
  const keys = output ? output.split(/\r?\n/).filter(Boolean) : [];
  return { keys: keys.slice(0, maxKeys), truncated: keys.length > maxKeys };
}

function readTtls(keys, options) {
  const snapshot = new Map();
  for (const key of keys) {
    const value = Number(runRedis(['PTTL', key], options));
    snapshot.set(key, value);
  }
  return snapshot;
}

function normalizeCacheKey(key) {
  return key.replace(/(?<=:)[0-9]+(?=(:|$))/g, '*');
}

function analyzeExpiryDistribution(entries, windowSeconds) {
  const buckets = new Map();
  let noExpiryCount = 0;
  let missingCount = 0;
  for (const { ttlMs } of entries) {
    if (ttlMs === -1) {
      noExpiryCount += 1;
      continue;
    }
    if (ttlMs < 0) {
      missingCount += 1;
      continue;
    }
    const startSeconds = Math.floor(ttlMs / 1000 / windowSeconds) * windowSeconds;
    buckets.set(startSeconds, (buckets.get(startSeconds) || 0) + 1);
  }
  const windows = [...buckets.entries()]
    .map(([startSeconds, keyCount]) => ({ startSeconds, endSeconds: startSeconds + windowSeconds - 1, keyCount }))
    .sort((left, right) => left.startSeconds - right.startSeconds);
  return { windows, noExpiryCount, missingCount };
}

function findRebuiltKeys(before, after) {
  return [...after.entries()]
    .filter(([key, ttl]) => before.has(key) && ttl >= 0 && ttl > before.get(key))
    .map(([key]) => key)
    .sort();
}

function familyCounts(keys) {
  const counts = new Map();
  for (const key of keys) {
    const family = normalizeCacheKey(key);
    counts.set(family, (counts.get(family) || 0) + 1);
  }
  return [...counts.entries()]
    .map(([family, keyCount]) => ({ family, keyCount }))
    .sort((left, right) => right.keyCount - left.keyCount || left.family.localeCompare(right.family));
}

function sleep(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

function buildTaskListUrl(apiBase, projectID, page, size) {
  const url = new URL('/api/v1/tasks', apiBase.endsWith('/') ? apiBase : `${apiBase}/`);
  url.searchParams.set('project_id', String(projectID));
  url.searchParams.set('page', String(page));
  url.searchParams.set('size', String(size));
  return url.toString();
}

function summarizeResponses(results) {
  const durations = results.map((result) => result.durationMs).sort((left, right) => left - right);
  const percentile = (ratio) => durations.length === 0 ? 0 : durations[Math.min(durations.length - 1, Math.ceil(durations.length * ratio) - 1)];
  return {
    total: results.length,
    success: results.filter((result) => result.ok).length,
    failed: results.filter((result) => !result.ok).length,
    minMs: durations[0] || 0,
    p50Ms: percentile(0.5),
    p95Ms: percentile(0.95),
    maxMs: durations.at(-1) || 0,
  };
}

function parseMySQLQuestions(output) {
  const match = output.match(/^Questions\s+(\d+)$/m);
  if (!match) throw new Error(`无法解析 MySQL Questions 输出: ${output.trim()}`);
  return Number(match[1]);
}

function parseDotenv(content) {
  const values = {};
  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    const separator = line.indexOf('=');
    if (separator < 1) continue;
    const key = line.slice(0, separator).trim();
    let value = line.slice(separator + 1).trim();
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1);
    }
    values[key] = value;
  }
  return values;
}

module.exports = {
  analyzeExpiryDistribution,
  familyCounts,
  findRebuiltKeys,
  normalizeCacheKey,
  parseArgs,
  readTtls,
  redisOptions,
  runRedis,
  scanKeys,
  sleep,
  buildTaskListUrl,
  summarizeResponses,
  parseMySQLQuestions,
  parseDotenv,
};

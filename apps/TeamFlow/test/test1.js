const test = require('node:test');
const assert = require('node:assert/strict');

const {
  analyzeExpiryDistribution,
  normalizeCacheKey,
  findRebuiltKeys,
  buildTaskListUrl,
  summarizeResponses,
  parseMySQLQuestions,
  parseDotenv,
} = require('../scripts/cache_diagnostics');

test('groups expiring keys into configurable avalanche windows', () => {
  const result = analyzeExpiryDistribution(
    [
      { key: 'task:list:project:1:page:1:size:10', ttlMs: 5_000 },
      { key: 'task:list:project:2:page:1:size:10', ttlMs: 25_000 },
      { key: 'project:1', ttlMs: 65_000 },
      { key: 'user:1', ttlMs: -1 },
    ],
    30,
  );

  assert.deepEqual(result.windows, [
    { startSeconds: 0, endSeconds: 29, keyCount: 2 },
    { startSeconds: 60, endSeconds: 89, keyCount: 1 },
  ]);
  assert.equal(result.noExpiryCount, 1);
});

test('normalizes dynamic IDs to cache-family names', () => {
  assert.equal(
    normalizeCacheKey('task:list:project:12:page:4:size:25'),
    'task:list:project:*:page:*:size:*',
  );
  assert.equal(normalizeCacheKey('project:stats:9'), 'project:stats:*');
});

test('finds keys whose TTL increased during observation', () => {
  const rebuilt = findRebuiltKeys(
    new Map([
      ['project:stats:1', 5_000],
      ['task:list:project:1:page:1:size:10', 30_000],
    ]),
    new Map([
      ['project:stats:1', 59_000],
      ['task:list:project:1:page:1:size:10', 29_000],
    ]),
  );

  assert.deepEqual(rebuilt, ['project:stats:1']);
});

test('builds a ListTask URL from the requested project and pagination', () => {
  assert.equal(
    buildTaskListUrl('http://localhost:8080/api/v1', 9, 2, 25),
    'http://localhost:8080/api/v1/tasks?project_id=9&page=2&size=25',
  );
});

test('summarizes successful and failed ListTask responses', () => {
  assert.deepEqual(
    summarizeResponses([
      { ok: true, durationMs: 10 },
      { ok: true, durationMs: 30 },
      { ok: false, durationMs: 50 },
    ]),
    { total: 3, success: 2, failed: 1, minMs: 10, p50Ms: 30, p95Ms: 50, maxMs: 50 },
  );
});

test('parses MySQL global Questions status output', () => {
  assert.equal(parseMySQLQuestions('Questions\t12345\n'), 12345);
});

test('parses quoted dotenv values without overriding later environment handling', () => {
  assert.deepEqual(parseDotenv('DB_PASSWORD="secret"\nAPP_ENV=development\n# comment'), {
    DB_PASSWORD: 'secret',
    APP_ENV: 'development',
  });
});

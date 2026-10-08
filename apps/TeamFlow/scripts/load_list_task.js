#!/usr/bin/env node
const { spawnSync } = require('node:child_process');
const { existsSync, readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const { buildTaskListUrl, parseArgs, parseDotenv, parseMySQLQuestions, summarizeResponses } = require('./cache_diagnostics');

function loadProjectEnv() {
  const envPath = resolve(__dirname, '..', '.env');
  if (!existsSync(envPath)) return;
  for (const [key, value] of Object.entries(parseDotenv(readFileSync(envPath, 'utf8')))) {
    if (process.env[key] === undefined) process.env[key] = value;
  }
}

function positiveInteger(value, option) {
  const number = Number(value);
  if (!Number.isInteger(number) || number < 1) throw new Error(`${option} 必须是正整数`);
  return number;
}

async function requestOnce(url, token) {
  const start = performance.now();
  try {
    const response = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
    return { ok: response.ok, status: response.status, durationMs: Math.round(performance.now() - start) };
  } catch (error) {
    return { ok: false, status: 'NETWORK_ERROR', durationMs: Math.round(performance.now() - start), error: error.message };
  }
}

function readMySQLQuestions(args) {
  const environment = { ...process.env };
  const password = args['db-password'] || process.env.DB_PASSWORD;
  if (password) environment.MYSQL_PWD = password;
  const binary = args['mysql-bin'] || process.env.MYSQL_BIN || 'mysql';
  const result = spawnSync(binary, [
    '-N', '-B',
    '-h', args['db-host'] || process.env.DB_HOST || 'localhost',
    '-P', String(args['db-port'] || process.env.DB_PORT || 3306),
    '-u', args['db-user'] || process.env.DB_USER || 'root',
    args['db-name'] || process.env.DB_NAME || 'teamflow',
    '-e', "SHOW GLOBAL STATUS LIKE 'Questions'",
  ], { encoding: 'utf8', env: environment });
  if (result.error) throw new Error(`无法执行 ${binary}: ${result.error.message}`);
  if (result.status !== 0) throw new Error((result.stderr || result.stdout || 'mysql 执行失败').trim());
  return parseMySQLQuestions(result.stdout);
}

async function main() {
  loadProjectEnv();
  const args = parseArgs(process.argv.slice(2));
  if (args.help) {
    console.log('用法: API_TOKEN=<JWT> PROJECT_ID=<项目ID> node scripts/load_list_task.js [--requests 100] [--concurrency 20] [--page 1] [--size 10] [--api-base http://localhost:8080] [--db-metrics]');
    return;
  }
  const token = process.env.API_TOKEN;
  const projectID = args['project-id'] || process.env.PROJECT_ID;
  if (!token) throw new Error('请设置 API_TOKEN');
  if (!projectID) throw new Error('请设置 PROJECT_ID 或 --project-id');
  const requests = positiveInteger(args.requests || 100, '--requests');
  const concurrency = positiveInteger(args.concurrency || 20, '--concurrency');
  const page = positiveInteger(args.page || 1, '--page');
  const size = positiveInteger(args.size || 10, '--size');
  const apiBase = args['api-base'] || process.env.API_BASE || 'http://localhost:8080';
  const url = buildTaskListUrl(apiBase, projectID, page, size);

  const questionsBefore = args['db-metrics'] ? readMySQLQuestions(args) : null;

  console.log(`请求 ${url}`);
  console.log(`总请求 ${requests}，并发 ${concurrency}`);
  const results = [];
  let nextIndex = 0;
  async function worker() {
    while (nextIndex < requests) {
      nextIndex += 1;
      results.push(await requestOnce(url, token));
    }
  }
  await Promise.all(Array.from({ length: Math.min(concurrency, requests) }, worker));
  const summary = summarizeResponses(results);
  console.log(JSON.stringify(summary, null, 2));
  if (questionsBefore !== null) {
    const questionsAfter = readMySQLQuestions(args);
    console.log(`MySQL GLOBAL Questions: ${questionsBefore} -> ${questionsAfter}（差值 ${questionsAfter - questionsBefore}）`);
  }
  const failed = results.filter((item) => !item.ok);
  if (failed.length > 0) console.log(`失败状态: ${[...new Set(failed.map((item) => item.status))].join(', ')}`);
}

main().catch((error) => {
  console.error(`压测失败: ${error.message}`);
  process.exitCode = 1;
});

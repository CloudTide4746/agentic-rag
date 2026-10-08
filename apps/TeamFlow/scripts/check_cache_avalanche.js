#!/usr/bin/env node
const {
  analyzeExpiryDistribution,
  familyCounts,
  parseArgs,
  readTtls,
  redisOptions,
  scanKeys,
} = require('./cache_diagnostics');

function printUsage() {
  console.log('用法: node scripts/check_cache_avalanche.js [--host localhost] [--port 6379] [--db 0] [--pattern "*"] [--max-keys 2000] [--window-seconds 30]');
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.help) return printUsage();
  const options = redisOptions(args);
  const pattern = args.pattern || '*';
  const maxKeys = Number(args['max-keys'] || 2000);
  const windowSeconds = Number(args['window-seconds'] || 30);
  if (!Number.isInteger(maxKeys) || maxKeys < 1 || !Number.isInteger(windowSeconds) || windowSeconds < 1) {
    throw new Error('--max-keys 与 --window-seconds 必须是正整数');
  }

  const { keys, truncated } = scanKeys(options, pattern, maxKeys);
  const ttls = readTtls(keys, options);
  const entries = keys.map((key) => ({ key, ttlMs: ttls.get(key) }));
  const report = analyzeExpiryDistribution(entries, windowSeconds);
  const alertThreshold = Math.max(10, Math.ceil(keys.length * 0.1));

  console.log(`Redis ${options.host}:${options.port}/${options.db}，模式 ${pattern}`);
  console.log(`扫描到 ${keys.length} 个 key${truncated ? `（已按 --max-keys=${maxKeys} 截断）` : ''}`);
  console.log(`无 TTL: ${report.noExpiryCount}；扫描期间消失: ${report.missingCount}`);
  console.log(`\n到期窗口（${windowSeconds} 秒；>= ${alertThreshold} 个标记为高风险）：`);
  if (report.windows.length === 0) console.log('  没有带 TTL 的 key。');
  for (const item of report.windows) {
    const marker = item.keyCount >= alertThreshold ? '  <== 高风险集中到期' : '';
    console.log(`  ${item.startSeconds}s-${item.endSeconds}s: ${item.keyCount} 个${marker}`);
  }
  console.log('\n缓存族数量：');
  for (const item of familyCounts(keys).slice(0, 20)) console.log(`  ${item.family}: ${item.keyCount}`);
}

try {
  main();
} catch (error) {
  console.error(`检查失败: ${error.message}`);
  process.exitCode = 1;
}

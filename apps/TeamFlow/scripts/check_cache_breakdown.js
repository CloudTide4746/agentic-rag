#!/usr/bin/env node
const {
  familyCounts,
  findRebuiltKeys,
  parseArgs,
  readTtls,
  redisOptions,
  runRedis,
  scanKeys,
  sleep,
} = require('./cache_diagnostics');

function printUsage() {
  console.log('用法: node scripts/check_cache_breakdown.js [--host localhost] [--port 6379] [--db 0] [--pattern "*"] [--max-keys 2000] [--observe-seconds 10] [--hotkeys]');
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.help) return printUsage();
  const options = redisOptions(args);
  const pattern = args.pattern || '*';
  const maxKeys = Number(args['max-keys'] || 2000);
  const observeSeconds = Number(args['observe-seconds'] || 10);
  if (!Number.isInteger(maxKeys) || maxKeys < 1 || !Number.isInteger(observeSeconds) || observeSeconds < 1) {
    throw new Error('--max-keys 与 --observe-seconds 必须是正整数');
  }

  const { keys, truncated } = scanKeys(options, pattern, maxKeys);
  const before = readTtls(keys, options);
  const noExpiry = keys.filter((key) => before.get(key) === -1);
  console.log(`Redis ${options.host}:${options.port}/${options.db}，模式 ${pattern}`);
  console.log(`首轮采样 ${keys.length} 个 key${truncated ? '（结果已截断）' : ''}，等待 ${observeSeconds} 秒观察重建。`);
  await sleep(observeSeconds * 1000);
  const after = readTtls(keys, options);
  const rebuilt = findRebuiltKeys(before, after);

  console.log(`\n观察期内 TTL 回升（可能发生缓存重建）的 key: ${rebuilt.length}`);
  if (rebuilt.length === 0) {
    console.log('  未观察到。注意：这不代表没有击穿，只代表本次窗口没有捕捉到重建。');
  } else {
    for (const item of familyCounts(rebuilt)) console.log(`  ${item.family}: ${item.keyCount}`);
    console.log('  示例:');
    for (const key of rebuilt.slice(0, 20)) console.log(`    ${key}`);
  }
  console.log(`\n无 TTL 的 key: ${noExpiry.length}`);
  for (const key of noExpiry.slice(0, 20)) console.log(`  ${key}`);

  if (args.hotkeys) {
    console.log('\nredis-cli --hotkeys 输出：');
    console.log(runRedis(['--hotkeys'], options) || '  Redis 未返回热点 key。通常需要配置 allkeys-lfu 或 volatile-lfu 才有有效结果。');
  } else {
    console.log('\n提示：加 --hotkeys 可读取 Redis 的 LFU 热点统计；若未启用 LFU，结果不可靠。');
  }
}

main().catch((error) => {
  console.error(`检查失败: ${error.message}`);
  process.exitCode = 1;
});

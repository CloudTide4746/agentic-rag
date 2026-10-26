/** @format */

const lanes = [
  {
    name: "向量召回",
    mark: "VECTOR",
    color: "cyan",
    hits: [
      ["内部知识库接入指南", "0.92"],
      ["平台开发规范", "0.87"],
    ],
  },
  {
    name: "关键词召回",
    mark: "KEYWORD",
    color: "violet",
    hits: [
      ["RAG 检索设计", "12.8"],
      ["平台开发规范", "9.4"],
    ],
  },
  {
    name: "数据库召回",
    mark: "DATABASE",
    color: "amber",
    hits: [["业务数据源示例", "—"]],
  },
];

export default function PlaygroundPage() {
  return (
    <div className='mx-auto max-w-5xl px-4 py-6 sm:px-7 sm:py-8'>
      <header className='mb-5'>
        <p className='font-mono text-[10px] tracking-[0.2em] text-cyan-300'>
          RETRIEVAL PLAYGROUND
        </p>
        <h1 className='mt-2 text-xl font-semibold text-slate-100'>检索测试</h1>
        <p className='mt-2 text-xs leading-5 text-slate-400'>
          展示多路召回、融合排序和质量评估的页面分区；检索功能尚未接入。
        </p>
      </header>
      <section
        aria-label='检索查询预览'
        className='rounded-xl border border-slate-800 bg-[#0b111d] p-3 sm:p-4'
      >
        <div className='flex flex-col gap-2 sm:flex-row'>
          <label htmlFor='playground-query' className='sr-only'>
            检索查询（布局预览）
          </label>
          <input
            id='playground-query'
            disabled
            value='如何设计混合检索与文档入库状态追踪？'
            readOnly
            className='h-11 min-w-0 flex-1 rounded-lg border border-slate-700 bg-[#080d16] px-3 text-xs text-slate-300 disabled:opacity-75'
          />
          <button
            type='button'
            disabled
            className='rounded-lg border border-cyan-800 bg-cyan-950/50 px-4 py-2 text-xs text-cyan-100 opacity-60'
          >
            ⌕ 执行检索
          </button>
        </div>
        <div className='mt-3 flex flex-wrap items-center gap-2 text-[10px] text-slate-500'>
          <span>示例查询</span>
          <span className='rounded-full border border-slate-700 px-2 py-1'>
            混合检索
          </span>
          <span className='rounded-full border border-slate-700 px-2 py-1'>
            文档处理
          </span>
          <span className='ml-auto font-mono'>TOP_K = 5</span>
        </div>
      </section>
      <section aria-labelledby='recall-title' className='mt-6'>
        <div className='mb-3 flex items-center gap-2'>
          <span className='font-mono text-[10px] text-cyan-300'>01</span>
          <h2
            id='recall-title'
            className='text-sm font-semibold text-slate-100'
          >
            多路召回
          </h2>
          <span className='font-mono text-[9px] text-slate-500'>
            并行召回 · 静态样例
          </span>
        </div>
        <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3'>
          {lanes.map((lane) => (
            <article
              key={lane.mark}
              className='rounded-xl border border-slate-800 bg-[#0b111d] p-3.5'
            >
              <div className='flex items-center justify-between'>
                <div>
                  <p className='font-mono text-[9px] tracking-wider text-slate-500'>
                    {lane.mark}
                  </p>
                  <h3 className='mt-1 text-xs font-semibold text-slate-200'>
                    {lane.name}
                  </h3>
                </div>
                <span
                  className={`size-2 rounded-full ${lane.color === "cyan" ? "bg-cyan-300" : lane.color === "violet" ? "bg-violet-300" : "bg-amber-300"}`}
                  aria-label='示例召回完成'
                />
              </div>
              <ul className='mt-3 space-y-2'>
                {lane.hits.map(([title, score]) => (
                  <li
                    key={title}
                    className='rounded-lg border border-slate-800 bg-slate-900/60 p-2.5'
                  >
                    <div className='flex justify-between gap-2'>
                      <span className='truncate text-[11px] text-slate-300'>
                        {title}
                      </span>
                      <span className='font-mono text-[10px] text-cyan-200'>
                        {score}
                      </span>
                    </div>
                    <p className='mt-1 line-clamp-1 text-[10px] text-slate-500'>
                      检索片段摘要仅用于展示列表密度与信息层级。
                    </p>
                  </li>
                ))}
              </ul>
            </article>
          ))}
        </div>
      </section>
      <section aria-labelledby='fusion-title' className='mt-6'>
        <div className='mb-3 flex items-center gap-2'>
          <span className='font-mono text-[10px] text-cyan-300'>02</span>
          <h2
            id='fusion-title'
            className='text-sm font-semibold text-slate-100'
          >
            融合排名
          </h2>
          <span className='font-mono text-[9px] text-slate-500'>
            RRF · Top 5
          </span>
        </div>
        <div className='overflow-x-auto rounded-xl border border-slate-800'>
          <table className='w-full min-w-[480px] text-left text-xs'>
            <caption className='sr-only'>融合排名静态示例</caption>
            <thead className='bg-slate-900 text-[9px] tracking-wider text-slate-500'>
              <tr>
                <th className='px-3 py-2.5'>RANK</th>
                <th className='px-3 py-2.5'>片段</th>
                <th className='px-3 py-2.5 text-right'>RRF</th>
                <th className='px-3 py-2.5 text-right'>来源</th>
              </tr>
            </thead>
            <tbody>
              {[
                ["#1", "平台开发规范 · 入库任务", "0.0318", "向量 / 关键词"],
                ["#2", "内部知识库接入指南", "0.0306", "向量"],
                ["#3", "RAG 检索设计", "0.0289", "关键词"],
              ].map((row) => (
                <tr
                  key={row[0]}
                  className='border-t border-slate-800 text-slate-300'
                >
                  <td className='px-3 py-3 font-mono text-cyan-200'>
                    {row[0]}
                  </td>
                  <td className='px-3 py-3'>{row[1]}</td>
                  <td className='px-3 py-3 text-right font-mono'>{row[2]}</td>
                  <td className='px-3 py-3 text-right text-[10px] text-slate-500'>
                    {row[3]}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
      <section aria-labelledby='grader-title' className='mt-6'>
        <div className='mb-3 flex items-center gap-2'>
          <span className='font-mono text-[10px] text-cyan-300'>03</span>
          <h2
            id='grader-title'
            className='text-sm font-semibold text-slate-100'
          >
            质量评估
          </h2>
        </div>
        <article className='flex flex-wrap items-center gap-4 rounded-xl border border-slate-800 bg-[#0b111d] p-4'>
          <div className='grid size-14 shrink-0 place-items-center rounded-full border-4 border-emerald-400/70 font-mono text-sm text-emerald-200'>
            8.4
          </div>
          <div className='min-w-0 flex-1'>
            <span className='rounded-full border border-emerald-900 bg-emerald-950/40 px-2 py-1 font-mono text-[9px] text-emerald-200'>
              PASS · 示例状态
            </span>
            <p className='mt-2 text-xs leading-5 text-slate-400'>
              融合结果与查询主题相关。该分数和说明为静态演示，并非实时评估。
            </p>
          </div>
        </article>
      </section>
    </div>
  );
}

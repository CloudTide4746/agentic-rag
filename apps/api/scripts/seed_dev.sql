-- =============================================================================
-- 独立开发库种子数据（agentic_rag_dev）
--
-- 依据：apps/api/migrations/0001_teamflow_core.up.sql
--       apps/api/migrations/0002_chat_platform_sessions.up.sql
-- 所有权：ADR-0001 —— Go 独占 users/teams/team_members/projects/project_members/
--        tasks/chat_platform_sessions。Python Agent 的表（chat_session、chat_message、
--        agent_decision_log、知识库与切片）不写入本文件，Go 不跨库读写。
--
-- 幂等：显式主键 + ON DUPLICATE KEY UPDATE，可重复执行。
-- 前置：先按顺序执行 migrations/*.up.sql。
-- 口令：全部种子用户为 DevPassw0rd!（bcrypt cost 10），仅用于本地开发。
--
-- 覆盖范围（便于前端联调）：
--   users 6 个（含 1 个禁用账号）、teams 3 个、projects 6 个（含 archived）、
--   tasks 16 个（4 状态 × 4 优先级、有/无负责人、有/无截止时间）、
--   chat_platform_sessions 6 个（active / pending / failed）。
-- =============================================================================

SET NAMES utf8mb4;

-- -----------------------------------------------------------------------------
-- users
-- -----------------------------------------------------------------------------
INSERT INTO users (id, created_at, updated_at, deleted_at, username, email, password, nickname, avatar, status) VALUES
  (1, '2026-08-01 09:00:00.000', '2026-10-20 18:30:00.000', NULL, 'alice', 'alice@example.dev',
   '$2a$10$t375YNnrf70yZqn29ghi8u9pc6drW4H.394haakLjMS0.9Qcue3xu', 'Alice Chen', '', 1),
  (2, '2026-08-01 09:05:00.000', '2026-10-19 10:12:00.000', NULL, 'bob', 'bob@example.dev',
   '$2a$10$uYLd/x9d3uuWro5bhwXYO.jTuTia0QF2XsHdY9Ryj6YrTS2svdG9q', 'Bob Liu', '', 1),
  (3, '2026-08-02 14:20:00.000', '2026-10-18 09:40:00.000', NULL, 'carol', 'carol@example.dev',
   '$2a$10$Ht8fqvfce3esnR08tfJNPuedYPVOvBrLVC7EQ3dbgB4WMJbT.sdie', 'Carol Wang', '', 1),
  (4, '2026-08-05 11:00:00.000', '2026-10-17 16:05:00.000', NULL, 'dave', 'dave@example.dev',
   '$2a$10$kxq3mivDj8tCDfi/1LTuteXwx1/xqsSFDgi0cgL/CnOEUl3dwBX7.', 'Dave Zhang', '', 1),
  (5, '2026-08-12 08:45:00.000', '2026-10-16 13:20:00.000', NULL, 'erin', 'erin@example.dev',
   '$2a$10$oymUyLUE1lygsu3wcKFcneGGRNxJFct2dzb0qqqedRDQms4E3j6QO', 'Erin Sun', '', 1),
  (6, '2026-09-03 15:10:00.000', '2026-10-15 11:55:00.000', NULL, 'frank', 'frank@example.dev',
   '$2a$10$PO2D9rAgb2kmb./ZnWJolekAY3Lm.ClBCpKcRiuPn9qv75GRSWp3m', 'Frank Wu', '', 0)
ON DUPLICATE KEY UPDATE
  updated_at = VALUES(updated_at), nickname = VALUES(nickname), status = VALUES(status);

-- -----------------------------------------------------------------------------
-- teams
-- -----------------------------------------------------------------------------
INSERT INTO teams (id, created_at, updated_at, deleted_at, name, description, owner_id, avatar) VALUES
  (1, '2026-08-01 09:10:00.000', '2026-10-20 18:30:00.000', NULL,
   'Platform Engineering', '负责 Go API、统一鉴权、发布链路与平台基础设施。', 1, ''),
  (2, '2026-08-02 14:25:00.000', '2026-10-19 10:12:00.000', NULL,
   'Data Intelligence', '负责 RAG 检索、文档切片、引用质量与离线评测。', 2, ''),
  (3, '2026-09-03 15:15:00.000', '2026-10-15 11:55:00.000', NULL,
   'Mobile Experience', '负责移动端协作视图与共享设计系统。', 3, '')
ON DUPLICATE KEY UPDATE
  updated_at = VALUES(updated_at), description = VALUES(description), owner_id = VALUES(owner_id);

-- -----------------------------------------------------------------------------
-- team_members（复合主键 team_id + user_id，role 受 CHECK 约束）
-- -----------------------------------------------------------------------------
INSERT INTO team_members (team_id, user_id, role, joined_at) VALUES
  (1, 1, 'owner',  '2026-08-01 09:10:00.000'),
  (1, 2, 'admin',  '2026-08-03 10:00:00.000'),
  (1, 4, 'member', '2026-08-05 11:10:00.000'),
  (2, 2, 'owner',  '2026-08-02 14:25:00.000'),
  (2, 3, 'admin',  '2026-08-02 15:00:00.000'),
  (2, 5, 'member', '2026-08-12 08:50:00.000'),
  (3, 3, 'owner',  '2026-09-03 15:15:00.000'),
  (3, 6, 'member', '2026-09-03 16:00:00.000')
ON DUPLICATE KEY UPDATE role = VALUES(role), joined_at = VALUES(joined_at);

-- -----------------------------------------------------------------------------
-- projects（active / archived；start_date <= end_date 受 CHECK 约束）
-- -----------------------------------------------------------------------------
INSERT INTO projects (id, created_at, updated_at, deleted_at, name, description, team_id, owner_id, status, start_date, end_date) VALUES
  (1, '2026-08-01 09:20:00.000', '2026-10-20 18:30:00.000', NULL,
   'Agentic RAG 平台 v1', '打通 React SPA、Go API 与 Python Agent 的首版知识库问答链路。',
   1, 1, 'active', '2026-08-01 00:00:00.000', '2026-12-31 00:00:00.000'),
  (2, '2026-09-01 10:00:00.000', '2026-10-18 14:00:00.000', NULL,
   '统一鉴权与审计', 'JWT + Redis 会话、资源归属校验与请求审计日志。',
   1, 1, 'active', '2026-09-01 00:00:00.000', '2026-11-30 00:00:00.000'),
  (3, '2026-07-15 09:00:00.000', '2026-10-19 10:12:00.000', NULL,
   '检索质量提升', '优化混合检索权重与重排策略，降低无关引用比例。',
   2, 2, 'active', '2026-07-15 00:00:00.000', '2026-10-31 00:00:00.000'),
  (4, '2026-08-10 13:30:00.000', '2026-10-17 09:00:00.000', NULL,
   '切片与评测基线', '建立文档切片规则与检索评测基线数据集。',
   2, 3, 'active', '2026-08-10 00:00:00.000', '2026-12-15 00:00:00.000'),
  (5, '2026-09-20 09:45:00.000', '2026-10-15 11:55:00.000', NULL,
   '移动端协作视图', '移动端任务看板与审批流的最小可用实现。',
   3, 3, 'active', '2026-09-20 00:00:00.000', '2027-01-31 00:00:00.000'),
  (6, '2026-03-01 09:00:00.000', '2026-06-30 18:00:00.000', NULL,
   '旧版前端归档', '归档旧 Vue 前端并整理迁移说明，已结束。',
   1, 1, 'archived', '2026-03-01 00:00:00.000', '2026-06-30 00:00:00.000')
ON DUPLICATE KEY UPDATE
  updated_at = VALUES(updated_at), status = VALUES(status),
  start_date = VALUES(start_date), end_date = VALUES(end_date);

-- -----------------------------------------------------------------------------
-- project_members（任务读写授权依赖本表，需与 team_members 对齐）
-- -----------------------------------------------------------------------------
INSERT INTO project_members (project_id, user_id, role, joined_at) VALUES
  (1, 1, 'owner',  '2026-08-01 09:20:00.000'),
  (1, 2, 'admin',  '2026-08-03 10:05:00.000'),
  (1, 4, 'member', '2026-08-05 11:15:00.000'),
  (2, 1, 'owner',  '2026-09-01 10:00:00.000'),
  (2, 4, 'member', '2026-09-02 09:30:00.000'),
  (3, 2, 'owner',  '2026-07-15 09:00:00.000'),
  (3, 3, 'admin',  '2026-07-15 09:30:00.000'),
  (3, 5, 'member', '2026-08-12 08:55:00.000'),
  (4, 3, 'owner',  '2026-08-10 13:30:00.000'),
  (4, 5, 'member', '2026-08-12 09:10:00.000'),
  (5, 3, 'owner',  '2026-09-20 09:45:00.000'),
  (5, 6, 'member', '2026-09-20 10:00:00.000'),
  (6, 1, 'owner',  '2026-03-01 09:00:00.000')
ON DUPLICATE KEY UPDATE role = VALUES(role), joined_at = VALUES(joined_at);

-- -----------------------------------------------------------------------------
-- tasks（4 种状态 × 4 种优先级；有/无负责人；有/无截止时间）
-- -----------------------------------------------------------------------------
INSERT INTO tasks (id, created_at, updated_at, deleted_at, title, description, project_id, assignee_id, creator_id, status, priority, due_date, version) VALUES
  (1,  '2026-08-02 10:00:00.000', '2026-10-20 18:30:00.000', NULL,
   'Go API 统一响应信封', '成功与失败响应统一为 data/request_id 与 error/request_id。', 1, 1, 1, 'done', 'high', '2026-08-20 00:00:00.000', 3),
  (2,  '2026-08-04 11:30:00.000', '2026-10-19 16:00:00.000', NULL,
   '聊天会话双层映射', '按 ADR-0003 建立平台会话与 Agent 会话的映射表与生命周期。', 1, 2, 1, 'in_progress', 'high', '2026-11-05 00:00:00.000', 5),
  (3,  '2026-08-06 09:15:00.000', '2026-10-18 10:20:00.000', NULL,
   'SSE 流式透传', 'Go 逐帧透传 Agent 的 text/event-stream，禁止缓冲。', 1, 4, 1, 'in_progress', 'high', '2026-11-12 00:00:00.000', 2),
  (4,  '2026-08-08 14:00:00.000', '2026-10-16 15:45:00.000', NULL,
   '会话列表分页与排序', '按 updated_at 倒序分页，返回 total/page/page_size。', 1, 4, 2, 'review', 'medium', '2026-10-30 00:00:00.000', 1),
  (5,  '2026-08-11 10:40:00.000', '2026-10-14 09:00:00.000', NULL,
   '团队列表页接入真实接口', '替换静态占位，展示团队名称、负责人与更新时间。', 1, NULL, 1, 'todo', 'medium', '2026-11-20 00:00:00.000', 0),
  (6,  '2026-08-13 16:20:00.000', '2026-10-12 11:00:00.000', NULL,
   '任务详情页并发版本提示', '版本冲突时提示刷新，避免覆盖他人修改。', 1, 4, 2, 'todo', 'low', NULL, 0),
  (7,  '2026-09-02 09:30:00.000', '2026-10-18 14:00:00.000', NULL,
   'JWT 与 Redis 会话 2 小时对齐', '令牌有效期与会话 TTL 保持一致，避免令牌有效但会话失效。', 2, 1, 1, 'done', 'high', '2026-09-20 00:00:00.000', 4),
  (8,  '2026-09-05 10:10:00.000', '2026-10-17 16:05:00.000', NULL,
   '资源归属校验补齐', '团队、项目、任务的读写均校验成员关系。', 2, 4, 1, 'in_progress', 'high', '2026-11-01 00:00:00.000', 2),
  (9,  '2026-09-09 13:00:00.000', '2026-10-15 10:30:00.000', NULL,
   '请求 ID 与审计日志', '错误响应附带 request_id，内部错误详情只进日志。', 2, NULL, 1, 'todo', 'medium', '2026-11-25 00:00:00.000', 0),
  (10, '2026-07-16 09:20:00.000', '2026-10-19 10:12:00.000', NULL,
   '混合检索权重调优', '调整向量与关键词检索的融合权重并记录指标。', 3, 2, 2, 'in_progress', 'high', '2026-10-28 00:00:00.000', 6),
  (11, '2026-07-20 15:40:00.000', '2026-10-10 09:00:00.000', NULL,
   '重排模型接入实验', '对比不同重排模型对引用准确率的影响。', 3, 3, 2, 'review', 'medium', '2026-11-08 00:00:00.000', 3),
  (12, '2026-08-14 11:05:00.000', '2026-10-11 14:30:00.000', NULL,
   '无关引用过滤规则', '过滤低分引用并记录被过滤原因供离线分析。', 3, 5, 3, 'todo', 'medium', NULL, 0),
  (13, '2026-08-11 09:00:00.000', '2026-10-17 09:00:00.000', NULL,
   '切片粒度基线', '按标题层级与段落长度定义切片规则并输出样例。', 4, 3, 3, 'done', 'high', '2026-09-15 00:00:00.000', 2),
  (14, '2026-08-18 14:10:00.000', '2026-10-16 11:20:00.000', NULL,
   '评测数据集整理', '整理 200 条问答样本作为检索评测基线。', 4, 5, 3, 'in_progress', 'medium', '2026-11-30 00:00:00.000', 1),
  (15, '2026-09-21 10:00:00.000', '2026-10-15 11:55:00.000', NULL,
   '移动端任务看板布局', '窄屏下三列看板降级为单列滚动。', 5, 6, 3, 'todo', 'medium', '2026-12-10 00:00:00.000', 0),
  (16, '2026-09-24 15:30:00.000', '2026-10-13 09:40:00.000', NULL,
   '设计令牌对齐', '颜色与间距令牌与 Web 端设计系统保持一致。', 5, 3, 3, 'todo', 'none', NULL, 0)
ON DUPLICATE KEY UPDATE
  updated_at = VALUES(updated_at), title = VALUES(title), description = VALUES(description),
  assignee_id = VALUES(assignee_id), status = VALUES(status), priority = VALUES(priority),
  due_date = VALUES(due_date), version = VALUES(version);

-- -----------------------------------------------------------------------------
-- chat_platform_sessions（Go 独占映射；public_id 对外，agent_session_id 仅内部）
-- -----------------------------------------------------------------------------
INSERT INTO chat_platform_sessions (id, created_at, updated_at, deleted_at, public_id, owner_id, agent_session_id, title, status, last_error_code, idempotency_key) VALUES
  (1, '2026-10-10 09:00:00.000', '2026-10-20 18:30:00.000', NULL,
   'cs_dev_alice_001', 1, 'agent_sess_1001', 'RAG 检索延迟怎么排查', 'active', NULL, 'idem_dev_0001'),
  (2, '2026-10-12 10:15:00.000', '2026-10-19 16:20:00.000', NULL,
   'cs_dev_alice_002', 1, 'agent_sess_1002', '知识库文档切片策略', 'active', NULL, 'idem_dev_0002'),
  (3, '2026-10-14 14:00:00.000', '2026-10-18 09:10:00.000', NULL,
   'cs_dev_alice_003', 1, NULL, 'Agent 服务超时对账', 'pending', 'AGENT_TIMEOUT', 'idem_dev_0003'),
  (4, '2026-10-15 11:30:00.000', '2026-10-17 15:45:00.000', NULL,
   'cs_dev_bob_001', 2, 'agent_sess_2001', '重排模型对比结论', 'active', NULL, 'idem_dev_0004'),
  (5, '2026-10-16 09:45:00.000', '2026-10-16 10:05:00.000', NULL,
   'cs_dev_bob_002', 2, NULL, '评测指标口径确认', 'failed', 'AGENT_REJECTED', 'idem_dev_0005'),
  (6, '2026-10-17 13:20:00.000', '2026-10-18 17:30:00.000', NULL,
   'cs_dev_carol_001', 3, 'agent_sess_3001', '移动端看板交互方案', 'active', NULL, 'idem_dev_0006')
ON DUPLICATE KEY UPDATE
  updated_at = VALUES(updated_at), title = VALUES(title), status = VALUES(status),
  agent_session_id = VALUES(agent_session_id), last_error_code = VALUES(last_error_code);

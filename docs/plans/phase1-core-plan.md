# Phase 1: Core Implementation Plan (Rev.2 — Post Plan Review)

## 목표
Symphony 포크 → Anarchy Core 변환. Linear 의존 제거, PostgreSQL 영속, Claude Code 런타임 추가.

## Plan Review 반영사항
- Tracker.Postgres 어댑터 추가 (Linear 교체의 핵심 seam)
- Issue 구조체 → Task 호환 구조체 마이그레이션 전략 명시
- PromptBuilder, DynamicTool, Config.Schema 변환 포함
- Oban을 Phase 1에서 제거 (Phase 2로 이연)
- Task.Supervisor 패턴 유지 (DynamicSupervisor 아님)
- WorkspaceManager는 신규 코드로 명시 (Symphony workspace.ex와 다름)
- WORKFLOW.md 유지하되 스키마 수정 (tracker → postgres, codex → runtime 섹션)
- 구현 순서 재배치: 데이터 레이어 → 트래커 → 오케스트레이터 → 런타임

## 변경 요약

### 1. Symphony 포크 + 리네이밍
- `symphony/elixir/` → 프로젝트 루트에 Elixir 프로젝트 생성
- 모듈명: `SymphonyElixir.*` → `Anarchy.*`
- 웹 모듈: `SymphonyElixirWeb.*` → `AnarchyWeb.*`
- OTP app: `:symphony_elixir` → `:anarchy`
- mix.exs: 의존성 추가 (ecto_sql, postgrex)
- 제거 대상: Linear client/adapter, memory tracker, DynamicTool(linear_graphql)

### 2. Config.Schema 변환
Symphony 스키마를 Anarchy용으로 수정:
- `tracker` 섹션: Linear 필드(api_key, project_slug, endpoint) → Postgres 필드(database_url)
- `codex` 섹션 유지 + `claude_code` 섹션 추가 (command, model, budget 등)
- WORKFLOW.md 패턴 유지 (hot-reload, WorkflowStore 그대로)

### 3. PostgreSQL 스키마 + Ecto 마이그레이션
spec 테이블 5개:
- `projects` — 프로젝트 관리
- `designs` — 설계서 (프로젝트당 N개)
- `project_assignments` — PM 할당
- `tasks` — 태스크 (CE 루프 상태, depends_on UUID[])
- `agent_sessions` — 에이전트 세션 영속

인덱스: tasks(project_id, status), tasks depends_on GIN, agent_sessions(project_id), agent_sessions(status)

### 4. Ecto 스키마 모듈
- `Anarchy.Schemas.Project`
- `Anarchy.Schemas.Design`
- `Anarchy.Schemas.Task`
- `Anarchy.Schemas.AgentSession`
- `Anarchy.Schemas.ProjectAssignment`

### 5. Tracker.Postgres 어댑터 (핵심)
Symphony의 Tracker behaviour를 구현하는 PostgreSQL 어댑터:
```elixir
defmodule Anarchy.Tracker.Postgres do
  @behaviour Anarchy.Tracker
  # fetch_candidate_tasks/0 — pending 상태, depends_on 충족된 태스크
  # fetch_tasks_by_states/1
  # fetch_task_states_by_ids/1
  # update_task_state/2
end
```
- Symphony의 Issue 구조체 → Anarchy Task 구조체로 통합
- Orchestrator가 Tracker behaviour를 통해 접근 (기존 패턴 유지)

### 6. Issue → Task 구조체 통합
- `Linear.Issue` 구조체 제거
- Ecto `Task` 스키마가 Orchestrator/AgentRunner에서 직접 사용
- PromptBuilder: `issue.*` 템플릿 변수 → `task.*` 변수로 변환

### 7. PromptBuilder 변환
- Solid 템플릿 변수: `issue.identifier` → `task.id`, `issue.title` → `task.title`
- 역할별 system prompt 지원 추가 (role_prompt_path 참조)

### 8. AgentProtocol behaviour + 런타임
```elixir
@callback start_session(opts :: map()) :: {:ok, session_id, pid()}
@callback resume_session(session_id) :: {:ok, pid()}
@callback send_prompt(pid(), prompt) :: :ok
@callback stop_session(pid()) :: :ok
```
구현체:
- `Anarchy.Runtime.ClaudeCode` — Claude Code CLI (`-p` + `stream-json`) Port 기반
- `Anarchy.Runtime.Codex` — 기존 AppServer 유지 (리뷰용)

### 9. SessionManager
- GenServer: 세션 ID 매핑, 재개, 복구
- DB 연동: `agent_sessions` 테이블 CRUD
- 세션 만료 시 `resume_context` 기반 복구
- `--resume` 플래그로 Claude Code 세션 재개

### 10. Orchestrator 변환
- `Tracker.Postgres` 어댑터를 통한 DB 폴링
- Task.Supervisor 패턴 유지 (Symphony 그대로)
- depends_on 평가: 의존 태스크 completed 여부 확인 후 디스패치
- 기존 재시도/백오프/reconciliation 패턴 유지

### 11. Supervision Tree
```
Anarchy.Application
├── Phoenix.PubSub
├── Anarchy.Repo (NEW — Ecto PostgreSQL)
├── {Task.Supervisor, name: Anarchy.TaskSupervisor} (유지)
├── Anarchy.SessionManager (NEW)
├── Anarchy.WorkflowStore (유지)
├── Anarchy.Orchestrator (변환)
├── Anarchy.HttpServer (유지)
└── Anarchy.StatusDashboard (유지)
```

### 12. 파일 구조
```
lib/anarchy/
├── application.ex            (변환)
├── repo.ex                   (신규)
├── orchestrator.ex           (변환 — DB 폴링)
├── session_manager.ex        (신규)
├── agent_runner.ex           (변환 — 듀얼 런타임)
├── prompt_builder.ex         (변환 — task.* 변수)
├── tracker.ex                (변환 — behaviour 유지)
├── tracker/
│   └── postgres.ex           (신규 — 핵심 어댑터)
├── runtime/
│   ├── agent_protocol.ex     (신규)
│   ├── claude_code.ex        (신규)
│   └── codex.ex              (변환 — AppServer 기반)
├── schemas/
│   ├── project.ex            (신규)
│   ├── design.ex             (신규)
│   ├── task.ex               (신규)
│   ├── agent_session.ex      (신규)
│   └── project_assignment.ex (신규)
├── config.ex                 (변환)
├── config/
│   └── schema.ex             (변환 — Linear→Postgres)
├── workspace.ex              (유지 — 디렉토리 기반)
├── path_safety.ex            (유지)
├── workflow.ex               (유지)
└── workflow_store.ex         (유지)

lib/anarchy_web/              (SymphonyElixirWeb → AnarchyWeb 리네이밍)
├── ... (기존 구조 유지)

priv/repo/migrations/
└── 001_create_anarchy_tables.exs (신규)
```

## 구현 순서 (수정됨)
1. Symphony 복사 + 모듈 리네이밍
2. mix.exs 의존성 추가 (ecto_sql, postgrex)
3. config 파일 업데이트 (Repo 설정)
4. Config.Schema 변환 (Linear 필드 → Postgres 필드)
5. Ecto Repo + 마이그레이션 생성
6. Ecto 스키마 모듈 작성
7. Tracker behaviour 리네이밍 + Tracker.Postgres 어댑터
8. Issue 구조체 → Task 구조체 통합
9. PromptBuilder 변환 (issue.* → task.*)
10. Orchestrator DB 폴링 변환 (Tracker.Postgres 사용)
11. AgentProtocol behaviour 정의
12. Claude Code Worker 구현
13. Codex Worker 리팩토링
14. AgentRunner 듀얼 런타임 지원
15. SessionManager 구현
16. Application supervision tree 업데이트
17. DynamicTool 제거 (linear_graphql)
18. 기본 테스트

## 제외 사항 (Phase 2+)
- Oban (DAG 태스크 큐) — Phase 2
- WorkflowEngine (CE 상태 머신) — Phase 2
- RoleLoader (agency-agents) — Phase 2
- CE Review 서브에이전트 — Phase 2
- Git worktree 관리 (WorkspaceManager) — Phase 2
- Architect Chat UI — Phase 3
- PM 에이전트 자동 분해 — Phase 3

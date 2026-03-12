# Anarchy: AI 개발 오케스트레이터 설계서

**Symphony 포크 기반 멀티 에이전트 개발 자동화 플랫폼**

> 설계서 하나 던지면 알아서 돌아가는 개발 환경.
> 사장은 방향 설정 + 컨펌만 한다.

---

## 1. 개요

### 1.1 목적

Anarchy는 OpenAI Symphony를 포크하여, **설계서 기반 반자동 소프트웨어 개발**을 실현하는 오케스트레이터다. 사용자(사장)가 Architect 에이전트와 대화해서 설계서를 만들고, 컨펌하면 PM 에이전트가 태스크로 분해하고, 역할별 에이전트가 자율적으로 구현한다.

### 1.2 핵심 원칙

- **사장은 설계자하고만 대화한다** — 나머지는 알아서 돌아감
- **프로젝트 내 세션은 절대 끊기지 않는다** — 세션 ID 영속
- **만드는 놈과 검증하는 놈을 분리한다** — Claude Code(구현) + Codex(리뷰)
- **매 태스크마다 학습이 축적된다** — Compound Engineering 루프
- **외부 서비스 의존 제로** — Linear ❌, SaaS ❌, Docker ❌, 내부 DB로 전부 처리

### 1.3 사전 요구사항

Anarchy가 관리하지 않는 외부 의존성. 사용자가 직접 설치해야 한다.

| 의존성 | 용도 | 비고 |
|--------|------|------|
| Elixir/OTP 1.17+ | Anarchy Core 런타임 | BEAM VM 포함 |
| PostgreSQL 16+ | 프로젝트/태스크/세션 영속 | 호스트 직접 설치 |
| Git 2.20+ | 소스 관리, worktree | |
| Claude Code CLI | 구현 런타임 | Anthropic 구독 필요 |
| Codex CLI | 리뷰 런타임 | OpenAI 구독 필요 |
| Node.js 20+ | Claude Code/Codex 실행 환경 | |

---

## 2. 아키텍처

### 2.1 시스템 구성

```
┌─────────────────────────────────────────────────────────────┐
│                    Phoenix LiveView Dashboard                │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌──────────┐ │
│  │ Architect  │ │ Plan       │ │ Project    │ │ Agent    │ │
│  │ Chat UI    │ │ Editor     │ │ Dashboard  │ │ Monitor  │ │
│  └─────┬──────┘ └─────┬──────┘ └─────┬──────┘ └────┬─────┘ │
└────────┼──────────────┼──────────────┼─────────────┼────────┘
         │              │              │             │
         ▼              ▼              ▼             ▼
┌─────────────────────────────────────────────────────────────┐
│                   Anarchy Core (Elixir/OTP)                  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                 Orchestrator GenServer                 │   │
│  │  DB 폴링 → 태스크 디스패치 → 상태 관리 → 재시도      │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │                                    │
│  ┌──────────┐  ┌───────┴───────┐  ┌──────────────────────┐ │
│  │ Workflow  │  │ Agent         │  │ Session              │ │
│  │ Engine    │  │ Supervisor    │  │ Manager              │ │
│  │ (CE Loop) │  │ (DynamicSup) │  │ (세션 ID 영속)       │ │
│  └──────────┘  └───────┬───────┘  └──────────────────────┘ │
│                        │                                     │
│         ┌──────────────┼──────────────┐                     │
│         ▼              ▼              ▼                     │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐             │
│  │ Claude Code│ │ Claude Code│ │ Codex      │             │
│  │ Worker     │ │ Worker     │ │ Worker     │             │
│  │ (Port)     │ │ (Port)     │ │ (Port)     │             │
│  └────────────┘ └────────────┘ └────────────┘             │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              PostgreSQL (Ecto)                         │   │
│  │  projects │ tasks │ agent_sessions │ designs          │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
         │              │              │
         ▼              ▼              ▼
  ┌────────────┐ ┌────────────┐ ┌────────────┐
  │ Git Repo   │ │ Worktree A │ │ Worktree B │
  │ (bare/원본)│ │ (에이전트1)│ │ (에이전트2)│
  └────────────┘ └────────────┘ └────────────┘
```

### 2.2 기술 스택

| 레이어 | 기술 | 이유 |
|--------|------|------|
| 오케스트레이션 | Elixir/OTP | Symphony 포크 기반, BEAM 동시성, Supervisor tree |
| 웹 대시보드 | Phoenix LiveView | Symphony 기존 UI 확장, 실시간 스트리밍 |
| DB | PostgreSQL + Ecto | Symphony는 인메모리였으나 영속 필요 |
| 작업 큐 | Oban (Pro) | DAG 기반 태스크 의존성, 재시도, 스케줄링 |
| 구현 런타임 | Claude Code CLI | `-p` 모드 + `stream-json` → Elixir Port |
| 리뷰 런타임 | Codex CLI | App-Server 프로토콜 → Elixir Port |
| 역할 프롬프트 | agency-agents | `.md` 기반 역할별 성격/워크플로 정의 |
| 워크플로 | Compound Engineering | Plan→Work→Review→Compound 루프 |
| VCS | Git (worktree) | 에이전트별 독립 브랜치 격리 |

### 2.3 OTP Supervision Tree

```
Anarchy.Application
├── Phoenix.Endpoint (웹 서버)
├── Phoenix.PubSub (실시간 이벤트 브로드캐스트)
├── Anarchy.Repo (Ecto PostgreSQL)
├── Oban (태스크 큐)
├── Anarchy.Orchestrator (중앙 GenServer)
│   ├── DB 폴링 루프
│   ├── 태스크 디스패치 로직
│   └── 전체 상태 관리
├── Anarchy.AgentSupervisor (DynamicSupervisor)
│   ├── ClaudeCode.Worker (GenServer + Port) x N
│   └── Codex.Worker (GenServer + Port) x N
├── Anarchy.SessionManager (GenServer)
│   └── 세션 ID 매핑, 재개, 복구
├── Anarchy.WorkspaceManager (GenServer)
│   └── Git worktree 생성/정리
└── Anarchy.WorkflowEngine (GenStateMachine)
    └── CE 루프 상태 머신
```

---

## 3. 워크플로

### 3.1 전체 흐름

```
Phase 1: 설계
  사장 ↔ Architect 에이전트 대화
  → 설계서 생성
  → 웹 UI에서 편집/수정/대화
  → 사장 컨펌

Phase 2: 분해
  PM 에이전트가 설계서 수령
  → 규모 판단 → 필요시 sub-PM 생성
  → 태스크 분해 → DB에 저장

Phase 3: 실행 (태스크별 CE 루프)
  각 태스크에 대해:
  ┌─→ Plan (Claude Code)
  │     → Plan Review (Codex) — 가벼운 구조/방향 확인
  │       ├── OK → 다음
  │       └── 수정 필요 → Plan으로
  │
  │   Work (Claude Code)
  │     → CE Review (Claude Code 서브에이전트 병렬)
  │       ├── Non-critical → 다음
  │       └── Critical → Work로 돌아감
  │     → Code Review (Codex) — 최종 검증
  │       ├── OK → 다음
  │       └── Critical → Work로 돌아감
  │
  │   Compound (Claude Code)
  │     → 학습 기록 → docs/solutions/에 저장
  └─── 다음 태스크의 Plan에 학습 자동 주입

Phase 4: 랜딩
  PR 생성 → 사장 리뷰/승인 → 머지
```

### 3.2 태스크별 CE 루프 상태 머신

```elixir
# 상태 전이
:idle → :planning → :plan_reviewing → :working → :ce_reviewing
  → :code_reviewing → :compounding → :completed

# Critical 발견 시 롤백
:ce_reviewing   → (critical) → :working
:code_reviewing → (critical) → :working

# Plan Review 수정 요청
:plan_reviewing → (revision) → :planning
```

### 3.3 역할별 런타임 할당

| 역할 | 런타임 | 모델 | 용도 |
|------|--------|------|------|
| architect | Claude Code | Opus | 설계서 생성, 사장과 대화 |
| pm | Claude Code | Sonnet | 태스크 분해, 진행 관리 |
| developer | Claude Code | Sonnet | 코드 구현 |
| plan_reviewer | Codex | - | 플랜 리뷰 (가벼운 수준) |
| ce_reviewer | Claude Code | Sonnet | CE 병렬 리뷰 (서브에이전트) |
| code_reviewer | Codex | - | 최종 코드 리뷰 |

---

## 4. 데이터 모델

### 4.1 ERD

```
projects 1──N designs
projects 1──N project_assignments
projects 1──N tasks
tasks    1──N agent_sessions
tasks    N──N tasks (depends_on)
tasks    0──1 tasks (parent_task_id, sub-PM 계층)
```

### 4.2 테이블 정의

```sql
-- 프로젝트
CREATE TABLE projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  description TEXT,
  status VARCHAR(50) DEFAULT 'active',
  -- active | paused | completed | archived
  repo_url VARCHAR(500),
  base_branch VARCHAR(255) DEFAULT 'main',
  config JSONB DEFAULT '{}',
  inserted_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);

-- 설계서 (프로젝트당 N개 가능)
CREATE TABLE designs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
  title VARCHAR(500) NOT NULL,
  content_md TEXT NOT NULL,
  status VARCHAR(50) DEFAULT 'draft',
  -- draft | reviewing | confirmed | superseded
  version INTEGER DEFAULT 1,
  confirmed_at TIMESTAMP,
  inserted_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);

-- PM 할당
CREATE TABLE project_assignments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
  role VARCHAR(50) NOT NULL,
  scope VARCHAR(500),
  agent_config JSONB DEFAULT '{}',
  inserted_at TIMESTAMP NOT NULL
);

-- 태스크
CREATE TABLE tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
  design_id UUID REFERENCES designs(id),
  parent_task_id UUID REFERENCES tasks(id),
  pm_assignment_id UUID REFERENCES project_assignments(id),
  title VARCHAR(500) NOT NULL,
  description TEXT,
  role VARCHAR(50) NOT NULL,
  -- architect | pm | developer | plan_reviewer | ce_reviewer | code_reviewer
  status VARCHAR(50) DEFAULT 'pending',
  -- pending | assigned | planning | plan_reviewing | working |
  -- ce_reviewing | code_reviewing | compounding | completed | failed
  priority INTEGER DEFAULT 5,
  depends_on UUID[] DEFAULT '{}',
  attempt INTEGER DEFAULT 0,
  max_attempts INTEGER DEFAULT 3,
  pr_url VARCHAR(500),
  branch VARCHAR(255),
  result JSONB,
  learnings TEXT,
  inserted_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);

-- 에이전트 세션
CREATE TABLE agent_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id UUID REFERENCES tasks(id),
  project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
  agent_type VARCHAR(50) NOT NULL,
  -- claude_code | codex
  session_id VARCHAR(255) NOT NULL UNIQUE,
  -- Claude Code conversation ID 또는 Codex thread ID
  role_prompt_path VARCHAR(500),
  workspace_path VARCHAR(500),
  branch VARCHAR(255),
  last_commit_sha VARCHAR(40),
  status VARCHAR(50) DEFAULT 'active',
  -- active | paused | resuming | completed | failed
  pause_reason VARCHAR(100),
  -- null | rate_limit | server_restart | manual | session_expired
  resume_context JSONB,
  -- 세션 만료 시 복구용: 마지막 상태 요약, 진행 사항 등
  started_at TIMESTAMP NOT NULL,
  last_active_at TIMESTAMP,
  paused_at TIMESTAMP,
  ended_at TIMESTAMP
);

-- 인덱스
CREATE INDEX idx_tasks_project_status ON tasks(project_id, status);
CREATE INDEX idx_tasks_depends ON tasks USING GIN(depends_on);
CREATE INDEX idx_sessions_project ON agent_sessions(project_id);
CREATE INDEX idx_sessions_status ON agent_sessions(status);
```

---

## 5. 에이전트 런타임

### 5.1 프로토콜 추상화

```elixir
defmodule Anarchy.AgentProtocol do
  @doc "에이전트 런타임 공통 인터페이스"
  @callback start_session(opts :: map()) :: {:ok, session_id :: String.t(), pid()}
  @callback resume_session(session_id :: String.t()) :: {:ok, pid()}
  @callback send_prompt(pid(), prompt :: String.t()) :: :ok
  @callback stop_session(pid()) :: :ok
end
```

### 5.2 Claude Code Worker

```elixir
defmodule Anarchy.Runtime.ClaudeCode do
  @behaviour Anarchy.AgentProtocol
  use GenServer

  def start_session(opts) do
    args = build_args(opts)
    port = Port.open(
      {:spawn_executable, claude_executable()},
      [:binary, :exit_status, :use_stdio, args: args]
    )
    session_id = opts[:session_id] || generate_session_id()
    {:ok, session_id, self()}
  end

  def resume_session(session_id) do
    args = ["--resume", session_id, "--output-format", "stream-json"]
    port = Port.open(
      {:spawn_executable, claude_executable()},
      [:binary, :exit_status, :use_stdio, args: args]
    )
    {:ok, self()}
  end

  # Port에서 수신한 스트림을 PubSub으로 브로드캐스트
  def handle_info({port, {:data, data}}, state) do
    messages = parse_stream_json(data)
    for msg <- messages do
      Phoenix.PubSub.broadcast(
        Anarchy.PubSub,
        "agent:#{state.session_id}",
        {:agent_output, state.session_id, msg}
      )
    end
    {:noreply, state}
  end

  defp build_args(opts) do
    base = ["-p", opts[:prompt],
            "--output-format", "stream-json",
            "--model", opts[:model] || "sonnet"]

    base
    |> maybe_add("--resume", opts[:session_id])
    |> maybe_add("--system-prompt", opts[:system_prompt])
    |> maybe_add("--max-budget-usd", opts[:budget])
    |> maybe_add("--allowedTools", opts[:allowed_tools])
    |> maybe_add("--agents", opts[:agents_json])
  end

  defp claude_executable do
    System.find_executable("claude") || raise "Claude Code CLI not found. Install: npm install -g @anthropic-ai/claude-code"
  end
end
```

### 5.3 Codex Worker

```elixir
defmodule Anarchy.Runtime.Codex do
  @behaviour Anarchy.AgentProtocol
  use GenServer

  @doc "App-Server 프로토콜 (JSONL over stdio)"

  def start_session(opts) do
    port = Port.open(
      {:spawn_executable, codex_executable()},
      [:binary, :exit_status, :use_stdio,
       args: ["app-server", "--cwd", opts[:workspace_path]]]
    )
    send_jsonrpc(port, "initialize", %{})
    thread_id = send_jsonrpc(port, "thread/start", %{})
    {:ok, thread_id, self()}
  end

  def resume_session(thread_id) do
    # ... Port 시작 후
    send_jsonrpc(port, "thread/resume", %{threadId: thread_id})
    {:ok, self()}
  end

  defp send_jsonrpc(port, method, params) do
    msg = Jason.encode!(%{method: method, id: next_id(), params: params})
    Port.command(port, msg <> "\n")
  end
end
```

### 5.4 세션 영속성 규칙

```
프로젝트 내 규칙:
├── 에이전트 1개 = 세션 1개 (프로젝트 수명 동안 유지)
├── 중단 시 → session_id 보존, status: paused
├── 재개 시 → 같은 session_id로 --resume
├── 세션 만료 (불가항력) → 새 세션 + resume_context 주입
│   resume_context = {
│     "summary": "Go Operator CreateVM 구현 중, gRPC 연동 남음",
│     "last_commit": "abc123",
│     "files_modified": ["pkg/operator/vm_controller.go"],
│     "remaining_work": ["gRPC 서버 연동", "E2E 테스트"]
│   }
└── 워크스페이스(git worktree)는 항상 유지 → 코드는 안 날아감
```

---

## 6. 워크스페이스 관리

### 6.1 Git Worktree 기반 격리

```elixir
defmodule Anarchy.WorkspaceManager do
  @workspace_root Application.compile_env(:anarchy, :workspace_root, "~/anarchy/workspaces")

  def create(project_id, agent_id, base_branch \\ "main") do
    path = workspace_path(project_id, agent_id)
    branch = "anarchy/#{project_id}/#{agent_id}"

    System.cmd("git", ["worktree", "add", "-b", branch, path, base_branch],
      cd: repo_path(project_id))

    {:ok, %{path: path, branch: branch}}
  end

  def cleanup(project_id, agent_id) do
    path = workspace_path(project_id, agent_id)
    System.cmd("git", ["worktree", "remove", "--force", path],
      cd: repo_path(project_id))
  end

  defp workspace_path(project_id, agent_id) do
    Path.join([@workspace_root, project_id, agent_id])
  end
end
```

### 6.2 머지 전략

```
에이전트별 독립 브랜치에서 작업
  → 완료 순서대로 FIFO 머지 큐
  → 충돌 시:
    1. git merge --no-commit 시도
    2. 자동 해결 가능 → 자동 머지
    3. 불가능 → AI 보조 해결 시도
    4. 그래도 불가 → 사장에게 에스컬레이션
```

---

## 7. 역할 프롬프트 (agency-agents)

### 7.1 로딩 구조

```
priv/agency-agents/
├── engineering/
│   ├── architect.md          ← 설계자
│   ├── senior-developer.md   ← 시니어 개발자
│   ├── qa-engineer.md        ← QA
│   └── code-reviewer.md      ← 코드 리뷰어
├── management/
│   └── project-manager.md    ← PM
└── custom/
    ├── go-engineer.md        ← Go 전문 (커스텀)
    └── elixir-engineer.md    ← Elixir 전문 (커스텀)
```

### 7.2 동적 로딩

```elixir
defmodule Anarchy.RoleLoader do
  @agents_path "priv/agency-agents"

  def load(role) do
    path = find_role_file(role)
    File.read!(path)
  end

  def spawn_with_role(role, task, opts \\ %{}) do
    system_prompt = load(role)
    runtime = runtime_for(role)
    runtime.start_session(%{
      prompt: task.description,
      system_prompt: system_prompt,
      model: model_for(role),
      workspace_path: opts[:workspace_path]
    })
  end

  defp runtime_for(role) when role in ~w(plan_reviewer code_reviewer),
    do: Anarchy.Runtime.Codex
  defp runtime_for(_role),
    do: Anarchy.Runtime.ClaudeCode

  defp model_for(:architect), do: "opus"
  defp model_for(_), do: "sonnet"
end
```

---

## 8. Compound Engineering 통합

### 8.1 CE 루프 상태 머신

```elixir
defmodule Anarchy.WorkflowEngine do
  use GenStateMachine

  # Plan 단계
  def handle_event(:internal, :run_plan, :planning, data) do
    {:ok, _sid, pid} = RoleLoader.spawn_with_role(:developer, data.task, %{
      prompt: render_plan_prompt(data.task, data.learnings)
    })
    {:keep_state, %{data | current_worker: pid}}
  end

  # Plan Review (Codex, 가벼운 수준)
  def handle_event(:internal, :run_plan_review, :plan_reviewing, data) do
    {:ok, _sid, pid} = RoleLoader.spawn_with_role(:plan_reviewer, data.task, %{
      prompt: "이 플랜의 구조와 방향만 확인하라. 코드 수준 리뷰는 불필요."
    })
    {:keep_state, %{data | current_worker: pid}}
  end

  # Work 단계
  def handle_event(:internal, :run_work, :working, data) do
    {:ok, _sid, pid} = RoleLoader.spawn_with_role(:developer, data.task)
    {:keep_state, %{data | current_worker: pid}}
  end

  # CE Review (Claude Code 서브에이전트 병렬)
  def handle_event(:internal, :run_ce_review, :ce_reviewing, data) do
    {:ok, _sid, pid} = Anarchy.Runtime.ClaudeCode.start_session(%{
      prompt: render_ce_review_prompt(data.task),
      agents_json: load_ce_review_agents()
      # security, performance, architecture, data-integrity 등
    })
    {:keep_state, %{data | current_worker: pid}}
  end

  # CE Review 결과 처리
  def handle_event(:info, {:review_complete, result}, :ce_reviewing, data) do
    if has_critical?(result) do
      {:next_state, :working, %{data | feedback: result},
       [{:next_event, :internal, :run_work}]}
    else
      {:next_state, :code_reviewing, data,
       [{:next_event, :internal, :run_code_review}]}
    end
  end

  # Code Review (Codex, 최종 검증)
  def handle_event(:info, {:review_complete, result}, :code_reviewing, data) do
    if has_critical?(result) do
      {:next_state, :working, %{data | feedback: result},
       [{:next_event, :internal, :run_work}]}
    else
      {:next_state, :compounding, data,
       [{:next_event, :internal, :run_compound}]}
    end
  end

  # Compound (학습 기록)
  def handle_event(:internal, :run_compound, :compounding, data) do
    {:ok, _sid, pid} = RoleLoader.spawn_with_role(:developer, data.task, %{
      prompt: "이번 태스크에서 배운 것을 docs/solutions/에 문서화하라."
    })
    {:keep_state, %{data | current_worker: pid}}
  end

  # Compound 완료 → 학습을 DB에 저장
  def handle_event(:info, {:compound_complete, learnings}, :compounding, data) do
    Anarchy.Repo.update_task(data.task, %{
      status: "completed",
      learnings: learnings
    })
    {:next_state, :completed, data}
  end
end
```

### 8.2 CE Review 에이전트 구성

```json
{
  "security-sentinel": {
    "description": "보안 취약점 검출",
    "prompt": "인증, 인가, 입력 검증, SQL 인젝션, XSS를 집중 검사하라",
    "tools": ["Read", "Grep", "Glob"],
    "model": "sonnet"
  },
  "performance-oracle": {
    "description": "성능 병목 분석",
    "prompt": "N+1 쿼리, 불필요한 할당, O(n²) 알고리즘을 찾아라",
    "tools": ["Read", "Grep", "Glob"],
    "model": "sonnet"
  },
  "architecture-strategist": {
    "description": "아키텍처 준수 확인",
    "prompt": "설계서의 아키텍처 원칙이 지켜졌는지 확인하라",
    "tools": ["Read", "Grep", "Glob"],
    "model": "sonnet"
  }
}
```

---

## 9. 다중 프로젝트 관리

### 9.1 구조

```
사장
├── 프로젝트 A: "클라우드 플랫폼"
│   ├── 설계서 1: Phase 1 (Go Operator)
│   ├── 설계서 2: Phase 2 (OVN + Ceph)
│   └── PM 에이전트 → 규모 판단
│       ├── Sub-PM: 백엔드 → 태스크 10개
│       └── Sub-PM: 인프라 → 태스크 5개
│
├── 프로젝트 B: "이 오케스트레이터 자체"
│   ├── 설계서 1: 코어
│   └── PM 에이전트 혼자 처리 → 태스크 8개
│
└── 프로젝트 C: 뭐든
    └── ...
```

### 9.2 PM 자동 확장

```elixir
defmodule Anarchy.PMAgent do
  def decompose(design) do
    # 1. 설계서 분석
    analysis = analyze_scope(design)

    # 2. 규모에 따라 PM 구조 결정
    cond do
      analysis.estimated_tasks < 10 ->
        # PM 혼자 처리
        create_tasks_directly(design)

      analysis.estimated_tasks < 30 ->
        # 파트별 sub-PM 생성
        for part <- analysis.parts do
          create_sub_pm(design.project_id, part)
        end

      true ->
        # 대규모: 팀 리드급 sub-PM + 하위 sub-PM
        create_pm_hierarchy(design)
    end
  end
end
```

---

## 10. Symphony 포크 변경 사항

### 10.1 유지하는 것 (Symphony 그대로)

- Elixir/OTP 프로젝트 구조
- Phoenix LiveView 대시보드 기반
- 에이전트 라이프사이클 관리 패턴
- Codex App-Server 프로토콜 클라이언트
- 워크스페이스 격리 개념
- WORKFLOW.md 기반 설정 (확장)
- 지수 백오프 재시도 로직
- 로깅/관측 패턴

### 10.2 교체하는 것

| Symphony 원본 | Anarchy |
|--------------|---------|
| Linear GraphQL 폴러 | PostgreSQL DB 폴러 |
| 인메모리 상태 관리 | Ecto/PostgreSQL 영속 |
| 단일 프로젝트 | 다중 프로젝트 |
| 이슈당 단일 에이전트 | 역할별 에이전트 + 계층 |
| Codex 전용 | Claude Code + Codex 하이브리드 |
| 역할 구분 없음 | agency-agents 기반 역할 시스템 |
| 워크플로 없음 | CE 루프 (Plan→Review→Work→Compound) |
| 수동 이슈 등록 | Architect 대화 → PM 자동 분해 |

### 10.3 추가하는 것

- Architect 채팅 UI (사장 ↔ 설계자 대화)
- Plan 편집/확인/컨펌 화면
- 프로젝트 목록/관리
- 세션 영속성 관리 (SessionManager)
- Claude Code 런타임 어댑터
- 프로토콜 추상화 레이어 (AgentProtocol behaviour)
- Oban 기반 태스크 DAG 스케줄링
- CE 워크플로 상태 머신
- agency-agents 역할 로더
- Git worktree 관리자

---

## 11. UI (Phoenix LiveView)

### 11.1 화면 구성

```
/ ─────────────── 프로젝트 목록 (전체 현황)
/projects/:id ─── 프로젝트 상세 (태스크 현황, 에이전트 상태)
/designs/:id ──── 설계서 뷰/편집
/chat/:project ── Architect 채팅 (설계서 생성/수정)
/tasks/:id ────── 태스크 상세 (CE 루프 진행, 로그)
/agents ────────── 에이전트 모니터링 (전체 세션 목록)
```

### 11.2 실시간 스트리밍 패턴

```elixir
defmodule AnarchyWeb.AgentMonitorLive do
  use AnarchyWeb, :live_view

  def mount(%{"id" => project_id}, _session, socket) do
    if connected?(socket) do
      Phoenix.PubSub.subscribe(Anarchy.PubSub, "project:#{project_id}")
    end
    tasks = Anarchy.Repo.list_tasks(project_id)
    {:ok, assign(socket, project_id: project_id, tasks: tasks)}
  end

  def handle_info({:agent_output, session_id, msg}, socket) do
    {:noreply, stream_insert(socket, :agent_logs, %{
      id: System.unique_integer(),
      session_id: session_id,
      content: msg,
      timestamp: DateTime.utc_now()
    })}
  end

  def handle_info({:task_status_changed, task_id, new_status}, socket) do
    {:noreply, update(socket, :tasks, fn tasks ->
      Enum.map(tasks, fn
        %{id: ^task_id} = t -> %{t | status: new_status}
        t -> t
      end)
    end)}
  end
end
```

---

## 12. 에러 핸들링과 복구

### 12.1 에이전트 실패 시

```
에이전트 프로세스 크래시
  → OTP Supervisor가 자동 재시작
  → SessionManager가 세션 ID로 재개 시도
  → 성공 → 이어서 작업
  → 실패 → resume_context로 새 세션 + 컨텍스트 주입
  → 3회 실패 → 태스크 status: failed, 사장에게 알림
```

### 12.2 재시도 전략

```elixir
# Symphony 기본 + 확장
@retry_delays [10_000, 20_000, 40_000, 80_000, 160_000, 300_000]
# 10초 → 20초 → 40초 → 80초 → 160초 → 5분 (상한)
```

---

## 13. 디렉토리 구조

```
anarchy/
├── SPEC.md                          ← 이 설계서
├── config/
│   ├── config.exs
│   ├── dev.exs
│   ├── prod.exs
│   └── runtime.exs
├── lib/
│   ├── anarchy/
│   │   ├── application.ex           ← OTP Application
│   │   ├── repo.ex                  ← Ecto Repo
│   │   ├── orchestrator.ex          ← 중앙 GenServer
│   │   ├── session_manager.ex       ← 세션 영속성
│   │   ├── workspace_manager.ex     ← Git worktree
│   │   ├── workflow_engine.ex       ← CE 상태 머신
│   │   ├── role_loader.ex           ← agency-agents 로더
│   │   ├── runtime/
│   │   │   ├── agent_protocol.ex    ← behaviour 정의
│   │   │   ├── claude_code.ex       ← Claude Code Worker
│   │   │   └── codex.ex             ← Codex Worker
│   │   ├── schemas/
│   │   │   ├── project.ex
│   │   │   ├── design.ex
│   │   │   ├── task.ex
│   │   │   ├── agent_session.ex
│   │   │   └── project_assignment.ex
│   │   └── workers/                  ← Oban Workers
│   │       ├── architect_worker.ex
│   │       ├── pm_worker.ex
│   │       ├── code_worker.ex
│   │       └── review_worker.ex
│   └── anarchy_web/
│       ├── router.ex
│       ├── live/
│       │   ├── project_list_live.ex
│       │   ├── project_detail_live.ex
│       │   ├── architect_chat_live.ex
│       │   ├── design_editor_live.ex
│       │   ├── task_detail_live.ex
│       │   └── agent_monitor_live.ex
│       └── components/
├── priv/
│   ├── repo/migrations/
│   └── agency-agents/               ← 역할 프롬프트
│       ├── engineering/
│       ├── management/
│       └── custom/
├── test/
└── mix.exs
```

---

## 14. 구현 로드맵

### Phase 1: 코어 (2주)

- [ ] Symphony 포크 + 빌드 확인
- [ ] PostgreSQL 스키마 + Ecto 마이그레이션
- [ ] AgentProtocol behaviour + Claude Code Worker
- [ ] SessionManager (세션 영속성)
- [ ] 기본 Orchestrator (DB 폴링 → 디스패치)

### Phase 2: 워크플로 (2주)

- [ ] WorkflowEngine (CE 상태 머신)
- [ ] Codex Worker (App-Server 프로토콜)
- [ ] agency-agents 통합 + RoleLoader
- [ ] CE Review 서브에이전트 병렬 실행
- [ ] Git worktree 관리

### Phase 3: 설계자 대화 (1주)

- [ ] Architect 채팅 UI (LiveView)
- [ ] 설계서 편집/컨펌 화면
- [ ] PM 에이전트 태스크 자동 분해

### Phase 4: 대시보드 (1주)

- [ ] 프로젝트 목록/상세 화면
- [ ] 에이전트 모니터링 (실시간 스트리밍)
- [ ] 태스크 상세 (CE 루프 진행 시각화)

### Phase 5: 안정화 (1주)

- [ ] 에러 핸들링/재시도 하드닝
- [ ] 머지 충돌 처리
- [ ] 다중 프로젝트 동시 실행 테스트

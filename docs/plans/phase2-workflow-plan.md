# Phase 2: Workflow Engine Implementation Plan

## 목표
CE 루프 상태 머신, 역할 프롬프트 로더, Git worktree 기반 워크스페이스 관리, Oban 백그라운드 잡 큐 추가.

## 구현 순서

### 1. Dependencies (mix.exs)
- `gen_state_machine ~> 3.0` — WorkflowEngine 상태 머신
- `oban ~> 2.18` — 백그라운드 잡 큐

### 2. WorkflowEngine (GenStateMachine)
CE 루프 상태 머신:
```
:idle → :planning → :plan_reviewing → :working → :ce_reviewing
  → :code_reviewing → :compounding → :completed
```
- Critical 발견 시 롤백: :ce_reviewing → :working, :code_reviewing → :working
- Plan Review 수정: :plan_reviewing → :planning
- 각 상태에서 적절한 런타임(Claude Code/Codex) 호출

### 3. RoleLoader
- `priv/agency-agents/` 디렉토리에서 역할 프롬프트 로드
- 역할 → 런타임 매핑 (plan_reviewer/code_reviewer → Codex, 나머지 → Claude Code)
- 역할 → 모델 매핑 (architect → opus, 나머지 → sonnet)

### 4. WorkspaceManager (Git worktree)
- 기존 Workspace.ex(디렉토리 기반)와 별도 모듈
- `git worktree add` / `git worktree remove` 관리
- 브랜치 네이밍: `anarchy/{project_id}/{task_id}`
- 에이전트별 독립 브랜치 격리

### 5. Oban Config + Workers
- Oban 마이그레이션 추가
- CELoopWorker — WorkflowEngine의 각 단계를 Oban 잡으로 실행
- Supervision tree에 Oban 추가

### 6. Sample Agency-Agents
```
priv/agency-agents/
├── engineering/
│   ├── architect.md
│   ├── senior-developer.md
│   └── code-reviewer.md
├── management/
│   └── project-manager.md
└── custom/
    └── .gitkeep
```

### 7. Supervision Tree 업데이트
```
Anarchy.Application
├── Anarchy.Repo
├── Phoenix.PubSub
├── Task.Supervisor
├── Oban                    (NEW)
├── Anarchy.WorkflowStore
├── Anarchy.SessionManager
├── Anarchy.Orchestrator
├── Anarchy.HttpServer
└── Anarchy.StatusDashboard
```

### 8. Config.Schema 확장
- `roles` 섹션: agency-agents 경로 설정
- WorkflowEngine은 Config에서 CE 루프 설정 로드

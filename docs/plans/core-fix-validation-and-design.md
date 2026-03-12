# Anarchy Core Fix Validation And Remaining Design

Source spec: `C:/Users/popod/Downloads/anarchy-core-fix-spec.md`

## Goal

Validate the current `core` implementation against the downloaded core-fix spec and define the remaining design work needed to make the runtime actually behave as intended.

This document is intentionally implementation-oriented. It is not a changelog. It records:

1. What the downloaded spec requires
2. What the current code actually does
3. What remains to be changed before implementation work starts

## Validation Summary

| Area | Spec intent | Current state | Status |
|------|-------------|---------------|--------|
| ClaudeCode runtime | Split into `run_once` and interactive session modes | Still a single `-p`-based GenServer with `send_prompt/2` writing to stdin | Not done |
| Architect chat | Real persistent Claude conversation with streaming | Still spawns one background task per message and calls `RoleLoader.execute_role/4` | Not done |
| Orchestrator dispatch | Route developer/reviewer work through `WorkflowEngine` CE loop | Still dispatches all tasks through `AgentRunner.run/3` | Not done |
| RoleLoader Claude execution | Use one-shot runtime API and return real text result | Still starts Claude session then calls deprecated `send_prompt/2` path | Not done |
| PM async decomposition | Kick off PM decomposition asynchronously and notify project LiveViews | UI still calls synchronous `PMAgent.decompose/1`; async API not present | Not done |
| Streaming cleanup | Separate interactive chat streaming from CE step streaming | Streaming exists in places, but topic ownership and producer/consumer model are inconsistent | Partial |

## Current Evidence

### 1. ClaudeCode runtime is still single-mode

Current runtime:

- `start_session/1` starts a GenServer immediately
- `build_args/1` always includes `-p`
- `send_prompt/2` writes to stdin after process start

That conflicts with the fix spec because the one-shot `-p` mode is being treated like an interactive stdin session.

Relevant files:

- `core/lib/anarchy/runtime/claude_code.ex`
- `core/lib/anarchy/role_loader.ex`

### 2. Architect chat is not backed by a live interactive Claude session

Current `ArchitectChatLive`:

- creates a new DB session row on mount
- spawns a background process per user message
- calls `run_architect/2`
- `run_architect/2` delegates to `RoleLoader.execute_role/4`

This means each message is effectively a one-shot invocation rather than a persistent project-scoped architect conversation.

Relevant file:

- `core/lib/anarchy_web/live/architect_chat_live.ex`

### 3. Orchestrator still bypasses CE loop

The fix spec requires developer/reviewer work to run through `WorkflowEngine`.

Current orchestrator still does:

- `Task.Supervisor.start_child(...)`
- `AgentRunner.run(task, recipient, attempt: attempt)`

The CE loop exists, but it is only started from UI/manual paths, not from the main orchestration path.

Relevant files:

- `core/lib/anarchy/orchestrator.ex`
- `core/lib/anarchy/workers/ce_loop_worker.ex`
- `core/lib/anarchy_web/live/task_detail_live.ex`

### 4. RoleLoader Claude execution path is still based on the broken session model

Current `execute_claude_code_role/5`:

- starts `ClaudeCode.start_session/1`
- calls `ClaudeCode.send_prompt/2`
- waits on process exit
- returns `:ok` on normal completion instead of guaranteed text output

This is exactly the path the fix spec tries to replace with a one-shot `run_once` execution API.

Relevant file:

- `core/lib/anarchy/role_loader.ex`

### 5. PM decomposition is still synchronous and still wired to template decomposition in UI

Current project UI:

- `confirm_design` calls `PMAgent.decompose/1`
- `decompose_design` calls `PMAgent.decompose/1`
- `decompose_with_agent/1` exists but is not the main UI path
- no `decompose_async/1` exists

Relevant files:

- `core/lib/anarchy/pm_agent.ex`
- `core/lib/anarchy_web/live/project_detail_live.ex`

### 6. Streaming model is still inconsistent

Current state:

- Claude runtime broadcasts on `agent:<session_id>`
- Architect chat subscribes to `architect:<project_id>` but does not actually receive streamed architect payloads on that topic
- `AgentMapLive` subscribes to `agents:<project_id>` even though no matching producer is visible in the runtime path

Relevant files:

- `core/lib/anarchy/runtime/claude_code.ex`
- `core/lib/anarchy_web/live/architect_chat_live.ex`
- `core/lib/anarchy_web/live/agent_map_live.ex`
- `core/lib/anarchy_web/live/agent_monitor_live.ex`

## Required Design Changes

### 1. Split Claude runtime into explicit modes

Introduce two separate execution paths in `Anarchy.Runtime.ClaudeCode`.

#### 1.1 One-shot mode for CE steps

Add:

- `run_once/1`

Contract:

- input: prompt, model, system prompt, workspace path, session id, timeout, optional stream callback
- output: `{:ok, text_output}` or `{:error, reason}`

Rules:

- one-shot mode owns `-p`
- caller must not send additional stdin prompts after startup
- return value must be final assistant text, not `:ok`

#### 1.2 Interactive mode for architect chat

Add:

- `start_interactive/1`
- `send_message/2`

Rules:

- no `-p`
- process remains alive across multiple user turns
- all incremental output is broadcast on `agent:<session_id>`
- session exit must be observable by the LiveView

#### 1.3 AgentProtocol compatibility

Keep `AgentProtocol` for existing direct-run paths, but treat it as the minimal compatibility layer.

Rules:

- CE loop should use `run_once/1` directly
- architect chat should use `start_interactive/1` and `send_message/2`
- `send_prompt/2` should no longer be the primary public API for Claude

### 2. Rebuild ArchitectChatLive around a project-scoped interactive session

Architect chat should own one interactive Claude session per project chat session.

Design:

1. On mount, load/create a stable architect session id for the project
2. Start or resume interactive Claude session
3. Store pid and session id in socket assigns
4. On user submit, call `ClaudeCode.send_message/2`
5. Receive streamed messages from `agent:<session_id>`
6. Persist final assistant message into chat history in the LiveView state

Required behavior:

- no background `spawn(fn -> run_architect(...) end)` wrapper
- no `RoleLoader.execute_role/4` for architect chat turns
- the same session must handle follow-up questions

### 3. Route orchestrated work through WorkflowEngine by default

Main orchestration path should choose between direct mode and CE-loop mode.

Dispatch matrix:

- `architect`, `pm`: direct mode is acceptable
- `developer`, `ce_reviewer`, `plan_reviewer`, `code_reviewer`: CE loop mode

Required orchestrator changes:

1. Replace unconditional `AgentRunner.run/3` dispatch with a dispatch decision
2. Start `WorkflowEngine` directly for CE-loop roles
3. Track running entry metadata with a `type` field
4. Handle `:DOWN` differently for direct workers and CE loop workers
5. Keep retry semantics consistent across both paths

### 4. Make RoleLoader Claude execution one-shot and text-returning

`execute_claude_code_role/5` must switch to `ClaudeCode.run_once/1`.

Required behavior:

- always return extracted text output when execution succeeds
- surface runtime failure as an exception or tagged error
- optionally stream to `agent:<session_id>` during execution

This change is necessary before WorkflowEngine review/classification logic can be trusted.

### 5. Add asynchronous PM decomposition path

Add `PMAgent.decompose_async/1`.

Design:

1. Spawn background decomposition process
2. Run `decompose_with_agent/1`
3. Broadcast result on `project:<project_id>`
4. Let project LiveViews refresh tasks/design state on completion

UI behavior:

- `confirm_design` should trigger async decomposition
- flash message should indicate decomposition started
- UI should refresh on `{:tasks_created, design_id, tasks}`
- UI should surface `{:pm_error, design_id, reason}`

### 6. Standardize streaming topics

Use a single topic ownership model.

Recommended rules:

- `agent:<session_id>`: raw runtime output for one live agent session
- `project:<project_id>`: project lifecycle events such as tasks created, task status changed, PM errors
- `notifications`: owner-facing alerts

Do not keep `agents:<project_id>` unless a real broadcaster is introduced.

### 7. Tighten output classification assumptions

Even after the runtime split, WorkflowEngine should classify results conservatively.

Rules:

- `:ok` is not a valid review payload for review stages
- plan/code/CE review stages should require explicit output text
- missing or empty output should be treated as retry or revision-needed, not approval

## Recommended File Changes

### Runtime

- `core/lib/anarchy/runtime/claude_code.ex`
- `core/lib/anarchy/runtime/agent_protocol.ex`

### Orchestration

- `core/lib/anarchy/orchestrator.ex`
- `core/lib/anarchy/workflow_engine.ex`
- `core/lib/anarchy/role_loader.ex`

### UI

- `core/lib/anarchy_web/live/architect_chat_live.ex`
- `core/lib/anarchy_web/live/project_detail_live.ex`
- `core/lib/anarchy_web/live/agent_map_live.ex`
- `core/lib/anarchy_web/live/agent_monitor_live.ex`

### PM flow

- `core/lib/anarchy/pm_agent.ex`

## Acceptance Criteria

### Claude runtime

- CE loop steps can call one-shot Claude execution without `send_prompt/2`
- architect chat can hold a session open and accept multiple user turns

### Architect chat

- second user message continues the same architect conversation
- assistant output streams incrementally into the chat UI

### Orchestrator

- developer tasks enter `WorkflowEngine` from the main polling loop
- architect and PM tasks still run directly

### Role execution

- `RoleLoader.execute_role/4` returns text for Claude-backed one-shot roles
- review stages receive non-empty review payloads

### PM decomposition

- design confirmation starts async PM decomposition
- project UI updates when tasks are created

### Streaming

- no LiveView subscribes to dead topics
- every subscribed topic has a matching producer in runtime code

## Out Of Scope

This document does not propose code changes for:

- broader phase 6 feature completion
- full session-history UI
- diff browser UI
- deeper PM graph decomposition strategy

Those remain separate concerns after the core-fix items above are resolved.

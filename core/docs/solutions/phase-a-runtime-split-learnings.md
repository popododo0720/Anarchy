# Learnings: Phase A — Runtime Split & Protocol Alignment

Date: 2026-03-12

## Changes

### claude_code.ex — Runtime split
- `run_once/1`: One-shot blocking execution via Port, returns `{:ok, text}` | `{:error, reason}`
- `start_interactive/1`: GenServer-based persistent session for multi-turn chat (Phase B용)
- `collect_output/4`: `remaining` buffer 패턴 — O(n) 파싱 (NOT `acc <> data` re-parse)
- `kill_port_process/1`: Port.close만으로는 OS 프로세스 미종료 → os_pid + kill -9 / taskkill
- `validate_run_once_opts/1`: null byte injection, dash-prefix flag injection 방지

### workflow_engine.ex — Classification & error visibility
- `classify_review_result/1`: `{:ok, text}` 튜플 처리 + `:ok` → `:revision_needed` (fail-closed)
- rescue `_ -> :ok` → `error -> Logger.error(...)` — 무음 실패 제거
- `persist_learnings`: task.id path traversal 방지 (regex sanitize)
- 불필요 map clause 4개 제거 (실제 caller는 `{:ok, text}`만 반환)

### agent_runner.ex — run_once migration
- `run_claude_code_turns`: start_session + send_prompt → `ClaudeCode.run_once/1` 직접 호출
- `_update_recipient` unused 처리

### role_loader.ex — Codex system_prompt skip
- Codex role은 system_prompt 사용 안 함 → 로딩 자체를 건너뜀
- `execute_codex_role/3` (was /4) — unused param 제거

## CE Review Findings & Fixes

### Security (CRITICAL)
1. **Port orphan on timeout** — `Port.close/1`은 pipe만 닫고 OS 프로세스는 살아있음. `kill_port_process/1`로 os_pid 기반 강제 종료. 크로스플랫폼 (win32: taskkill, unix: kill -9).
2. **CLI argument injection** — `spawn_executable`은 shell 없이 argv 직접 전달하므로 shell injection은 불가하나, null byte와 flag-like value는 검증 필요. `:model`, `:workspace_path`에 dash-prefix 차단.

### Security (HIGH)
3. **Error swallowing** — `rescue _ -> :ok` 패턴이 `update_task_status`, `finalize_session`, `persist_learnings`에서 에러를 완전히 무시. 모두 `Logger.error` 추가.
4. **Path traversal** — `persist_learnings`에서 `task.id`가 파일명에 직접 사용됨. `../` 포함 시 workspace 외부 쓰기 가능. regex로 영숫자+_- 외 문자 치환.

### Architecture
5. **finalize_session 중복** — `:completed` clause와 catch-all이 동일 → `:completed` clause 제거.
6. **classify_review_result map clauses** — 실제 호출경로에서 map을 반환하지 않음 → 4개 clause 제거.
7. **task_context 중복** — `%TaskSchema{}` clause는 plain map clause에 이미 포함 → struct clause 제거.

## Patterns & Lessons

1. **Port.close ≠ process kill** — BEAM Port는 pipe closure만 수행. OS 프로세스 정리는 os_pid + kill 필수. `terminate/2`에서도 동일 적용.
2. **spawn_executable = no shell injection** — 하지만 null byte와 flag injection은 여전히 가능. 특히 model/path 같은 non-prompt 파라미터에 대한 검증 필요.
3. **rescue에서 반드시 로깅** — `rescue _ -> :ok`는 프로덕션에서 디버깅 불가능한 원인. 최소 Logger.error.
4. **파일명에 user input 금지** — task.id 같은 외부 데이터가 파일명에 들어가면 path traversal 위험. regex sanitize 필수.
5. **fail-closed default** — 리뷰 결과가 파싱 불가능하면 `:revision_needed` 반환. `:ok`나 nil도 동일 처리.
6. **O(n) buffer 패턴** — stream parsing에서 `remaining <> new_data`만 파싱하고, 전체 history를 re-parse하지 않음. `collect_output`의 `remaining` 패턴이 정답.

## Code Review 추가 수정

### CRITICAL — @behaviour/@impl 누락
- `ClaudeCode`에 `@behaviour AgentProtocol` 미선언 → 컴파일러 callback 검증 불가
- `handle_info` exit_status clause, 두 번째 `terminate` clause에 `@impl GenServer` 누락
- 5개 protocol function에 `@impl AgentProtocol` 추가

### HIGH — DOWN handler double-fire
- `spawn_role_worker`의 worker가 `{:worker_complete, ...}` 전송 후 종료 → DOWN + worker_complete 이중 수신
- `demonitor_worker/1` helper: `Process.demonitor(ref, [:flush])` — 모든 `:worker_complete` handler에서 호출
- `:flush` 옵션이 이미 큐에 있는 DOWN 메시지도 제거

### CRITICAL — 호출자 미갱신 (PMAgent, ArchitectChatLive)
- `RoleLoader.execute_role/4`가 Claude Code path에서 `{:ok, text}` 반환하도록 변경됐으나, PMAgent와 ArchitectChatLive는 plain string만 처리
- PMAgent: `parse_agent_output({:ok, text})` clause 추가
- ArchitectChatLive: `extract_response_text({:ok, text}, _)` clause 추가

### 교훈
7. **@behaviour 선언 필수** — 컴파일 타임에 callback 누락/타입 불일치를 잡을 수 있음
8. **spawn_monitor + message = demonitor 필수** — worker가 직접 메시지를 보내는 패턴에서 DOWN과 이중 수신 방지
9. **return type 변경 시 모든 호출자 확인** — `execute_role/4`의 반환타입이 바뀌면 직접 호출하는 모든 모듈 검토 필수

# Phase 1: Core — Learnings

## Symphony Fork 전략
- Symphony는 `Task.Supervisor` + monitored tasks 패턴 사용 (DynamicSupervisor 아님)
- `Tracker` behaviour가 핵심 seam — 어댑터만 교체하면 Orchestrator 변경 최소화
- 1500줄 Orchestrator는 수술적 변경으로 접근 (전면 재작성 금지)

## N+1 쿼리 방지 패턴
- `Enum.filter(&db_query/1)` 패턴은 N+1의 전형
- 해결: 모든 dependency ID를 수집 → 단일 배치 쿼리 → MapSet으로 in-memory 필터
- `fetch_candidate_tasks`에서 2쿼리로 축소 (pending tasks + completed deps)

## Port 프로세스 관리
- `Port.open({:spawn_executable, ...})` 사용 시 반드시 `terminate/2`에서 `Port.close/1` 호출
- 없으면 OS 프로세스 leak → 장기 실행 시 치명적
- `:line` 모드는 stream-json과 호환 안됨 — raw binary로 받아서 `\n` 파싱

## Issue → Task 마이그레이션 체크리스트
- struct 필드 차이 확인 (`identifier` 필드 없음 → workspace 충돌)
- 모든 `.identifier` 접근 패턴 grep 필수
- Workspace, PromptBuilder, AppServer, Config 전부 확인
- 템플릿 변수도 확인 (Solid strict_variables 모드)

## Config.Schema 주의사항
- `normalize_issue_state` → `normalize_task_state` 전면 교체 시 호출 사이트 전부 확인
- WORKFLOW.md 핫리로드 유지 — 기존 WorkflowStore 패턴 건드리지 않음
- dev/test 전용 DB 자격증명은 `config_env()` 가드로 분리, prod는 runtime.exs

## Ecto.Enum vs validate_inclusion
- `Ecto.Enum`의 `values:` 옵션이 이미 cast 시 validation 수행
- `validate_inclusion`은 중복 — 제거해도 안전

## Clause 순서 (함수 패턴 매칭)
- `start_session(workspace)` vs `start_session(%{workspace: _})` — arity 같으면 guard 필수
- `when is_binary(workspace)` 추가하지 않으면 map이 잘못된 clause로 dispatch

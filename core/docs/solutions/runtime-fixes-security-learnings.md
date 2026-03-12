# Learnings: Runtime Fixes + Security Hardening

Date: 2026-03-12

## 변경 내용
- `skip_permissions?/0` rescue를 fail-open(true) → fail-closed(false)로 수정
- `ClaudeCode.skip_permissions` schema default를 true → false로 변경
- `warn_headless_codex_policy/0` 추가: 헤드리스 모드에서 안전하지 않은 Codex approval_policy 경고
- StatusDashboard ANSI 호환성: `IO.ANSI.enabled?()` + `strip_ansi/1` fallback

## CE Review에서 발견된 이슈
1. **CRITICAL: fail-open rescue** — `skip_permissions?/0`의 rescue가 `true` 반환 → 설정 파싱 실패 시 permission 우회됨. `false`로 변경 + Logger.warning 추가.
2. **MEDIUM: 타입 불일치** — `policy != "never"` 비교가 default map config(`%{"reject" => ...}`)에서 항상 true → 불필요한 경고 발생. `headless_safe_policy?/1` multi-clause로 해결.
3. **LOW: bare rescue** — `rescue _ -> :ok`가 에러를 완전히 무시. `Logger.debug`로 최소 로깅 추가.
4. **LOW: Elixir 관용구** — `if not` → `unless` 변경.

## 패턴 & 교훈
1. **Fail-closed default** — 보안 관련 rescue는 반드시 제한적 기본값 사용. `rescue _ -> true`는 절대 안 됨.
2. **StringOrMap 타입 주의** — Ecto `StringOrMap` 필드는 비교 시 string과 map 모두 고려해야 함. `!= "string"` 비교만으로는 map case를 놓침.
3. **Multi-clause guard > 조건문 체이닝** — `headless_safe_policy?/1`처럼 pattern matching multi-clause가 조건 분기보다 확장성 좋음.
4. **rescue에서 최소 로깅** — `rescue _ -> :ok`는 디버깅 불가능. 최소한 `Logger.debug` 사용.
5. **Schema default는 보안 기본값** — `skip_permissions: true`가 default면 모든 신규 설치에서 보안 우회됨. 명시적 opt-in이 원칙.

## Architecture Review 추가 권장사항 (향후 작업)
- StatusDashboard 활성화 메커니즘 통합 (ANARCHY_TUI env vs observability.dashboard_enabled config — 하나로 통일)
- `Config.settings!()` vs `Config.settings()` 프로젝트 전체 컨벤션 수립
- `normalize_status_lines/1` no-op 제거, `keyword_override/2` → `Keyword.get/2` 교체

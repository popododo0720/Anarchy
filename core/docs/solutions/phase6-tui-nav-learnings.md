# Learnings: TUI 비활성화 + 글로벌 네비게이션 추가

Date: 2026-03-12

## 변경 내용
- StatusDashboard TUI를 ANARCHY_TUI=1 환경변수로 opt-in 전환
- 전체 LiveView에 글로벌 네비게이션 바 추가 (live_session)
- CSS 미정의 변수 수정, 중복 셀렉터 정리

## CE Review에서 발견된 이슈
1. **CRITICAL: app 레이아웃이 DashboardLive에만 적용됨** — live_session으로 해결
2. **Moderate: `<a href>` 대신 `<.link navigate>` 사용 필요** — 풀 페이지 리로드 방지
3. **Moderate: CSS 변수 `--fg`, `--bg`, `--surface`, `--border` 미정의** — :root에 alias 추가
4. **Low: ANARCHY_TUI 중복 체크** — tui_enabled?/0 헬퍼로 통합

## 패턴 & 교훈
1. **live_session으로 레이아웃 중앙 관리** — 개별 LiveView에서 layout 지정하지 말고 router의 live_session 블록에서 한번에
2. **CSS 변수 alias** — Phase별로 다른 변수명 쓰면 미정의 버그 발생. :root에 alias 정의해서 통일
3. **Phoenix LiveView 네비게이션** — `<a href>` 대신 `<.link navigate>` 써야 WebSocket 유지됨
4. **env var 게이팅** — 개발용 TUI는 기본 off, 명시적 opt-in이 맞음

## CE 워크플로우 교훈
- 이 변경에서 CE 루프를 3번 건너뛰다 지적받음
- 원인: "작은 변경"이라 판단하면 프로세스 스킵하는 기본 행동
- 수정: 코드 크기와 무관하게 모든 변경에 CE 루프 적용

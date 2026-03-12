---
title: "Phase B: Orchestrator Dispatch + PubSub + Async PM + Interactive Chat"
date: 2026-03-13
tags: [orchestrator, pubsub, async, liveview, transaction-boundary, atom-safety]
severity: mixed (CRITICAL + HIGH)
---

# Phase B Learnings

## 1. Transaction Boundary — LLM Calls Outside, DB Writes Inside

**Problem**: `PMAgent.decompose/1` wrapped the entire flow (including LLM call) in
`Repo.transaction`. LLM calls take 30-120+ seconds, holding a DB pool connection the
entire time. Under load this starves the pool.

**Fix**: `decompose_async/1` runs LLM call first, then wraps only `create_tasks_from_specs`
in `Repo.transaction`.

**Rule**: Never hold a DB transaction open across an external API/LLM call.

## 2. Atom Exhaustion via `String.to_atom`

**Problem**: `Tracker.Postgres.to_existing_atom/1` used `String.to_atom/1`. Atoms are
never garbage collected. Any external input routed through this path (e.g., malformed
task status from API) creates permanent atoms → eventual VM crash.

**Fix**: Compile-time `@status_lookup` map (`Map.new(@valid_statuses, ...)`), looked up
with `Map.get/2`. Unknown strings return `nil` instead of creating atoms.

**Rule**: Never convert untrusted strings to atoms. Use a compile-time whitelist map.

## 3. PubSub Subscription Leak in LiveView

**Problem**: `AgentMapLive.handle_event("select_agent", ...)` subscribed to a new
`"agent:#{session_id}"` topic each click without unsubscribing the previous one.
Each leaked subscription accumulates a process message handler.

**Fix**: Track `subscribed_agent_topic` in assigns, unsubscribe before re-subscribing.

**Rule**: In LiveView, always unsubscribe from the previous topic before subscribing
to a new one when the subscription target can change.

## 4. PubSub Single-Source Broadcast

**Problem**: Task status changes were broadcast from multiple call sites, risking
duplicate or inconsistent events.

**Fix**: Only `Tracker.Postgres.update_task_state/2` broadcasts `:task_status_changed`.
All callers go through the tracker. Broadcast uses `updated_task.status` (atom from DB)
instead of raw `new_state` parameter (could be string or atom).

**Rule**: Single source of truth for state-change broadcasts. Use the persisted value,
not the input value.

## 5. `String.to_integer` Crash on LLM Output

**Problem**: `PMAgent.parse_task_block/1` used `String.to_integer` on LLM-generated
"PRIORITY:" values. LLMs sometimes output "3 (high)" or non-numeric text → crash.

**Fix**: `Integer.parse/1` with pattern match — `{n, _} -> n; :error -> 5`.

**Rule**: Never use `String.to_integer` on external/LLM input. Use `Integer.parse`
with a fallback.

## 6. Dead Code: `decompose_with_agent/1`

**Problem**: After adding `decompose_async/1`, the synchronous `decompose_with_agent/1`
had zero callers. Dead code increases maintenance burden and confuses readers.

**Fix**: Removed entirely.

**Rule**: After adding a replacement function, grep for callers of the old one and
remove it if unused.

## 7. `connected?/1` Guard in `handle_event`

**Problem**: `AgentMapLive` wrapped PubSub subscribe in `if connected?(socket)` inside
`handle_event`. LiveView events only fire on connected sockets — the guard is always true.

**Fix**: Removed the dead guard.

**Rule**: `connected?/1` is only meaningful in `mount/3`. Events and `handle_info` always
run on connected sockets.

## 8. Error Info Leak in Flash Messages

**Problem**: `ProjectDetailLive` rendered `inspect(reason)` in user-facing flash messages.
Internal error details (stack traces, Ecto changesets) leaked to the browser.

**Fix**: Generic user-facing messages ("Decomposition failed. Check server logs.").
Full error logged server-side with `Logger.error`.

**Rule**: Never expose `inspect(error)` in user-facing UI. Log details server-side,
show generic messages to users.

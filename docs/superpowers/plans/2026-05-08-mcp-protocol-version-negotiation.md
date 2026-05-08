# MCP Protocol Version Negotiation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Postiz MCP client work across MCP server versions by negotiating a mutually supported protocol version during initialization.

**Architecture:** Keep negotiation inside `internal/tools/postiz.go`, next to the existing MCP session code. The client tries known MCP versions newest-first, parses unsupported-version errors for a supported list, caches the accepted version with the session ID, and retries a tool call once after a session-expiration response.

**Tech Stack:** Go standard library HTTP/JSON, existing `httptest` tests in `internal/tools/tools_test.go`, existing Backlog.md workflow.

---

### Task 1: Negotiate MCP Protocol Version During Initialize

**Files:**
- Modify: `internal/tools/postiz.go`
- Test: `internal/tools/tools_test.go`

- [ ] **Step 1: Write the failing fallback test**

Add a test that has the mock server reject the first initialize request with `Unsupported protocol version` and supported versions containing `2025-06-18`, then accept the retry and assert the final `tools/call` uses `MCP-Protocol-Version: 2025-06-18`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tools`

Expected: FAIL because the client currently uses one protocol version and does not retry initialize.

- [ ] **Step 3: Implement version negotiation**

Replace the single `mcpProtocolVersion` constant with an ordered version list. Add state for `protocolVersion`, attempt initialize with each candidate, parse JSON-RPC unsupported-version errors, and store the accepted version from the initialize result.

- [ ] **Step 4: Run focused tests**

Run: `go test ./internal/tools`

Expected: PASS.

### Task 2: Reinitialize After Expired Session

**Files:**
- Modify: `internal/tools/postiz.go`
- Test: `internal/tools/tools_test.go`

- [ ] **Step 1: Write the failing expired-session retry test**

Add a test where the first `tools/call` with a valid session receives HTTP 404. Assert the client clears session state, sends a second `initialize`, and retries the same tool call with the new session.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tools`

Expected: FAIL because the client currently returns the 404 instead of reinitializing.

- [ ] **Step 3: Implement one retry on session expiry**

In `callMCPTool`, if a tool call returns HTTP 404, clear the cached session and negotiated protocol version, initialize again, and retry the same request once.

- [ ] **Step 4: Run full verification**

Run: `go test ./internal/tools` and `go test ./...`

Expected: PASS.


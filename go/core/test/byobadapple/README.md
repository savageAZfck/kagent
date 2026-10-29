# Bad Apple BYO shim

Exposes Bad Apple as an A2A agent inside kagent. The shim serves A2A on
:8080 and forwards each message to Bad Apple's MCP JSON-RPC endpoint
(`tools/call` → `inference`), returning her reply as the completed task's
agent message.

## Deploy

1. Make the daemon's MCP endpoint reachable from the cluster. It binds
   `localhost:9879` on the Mac; over WireGuard just point
   `BADAPPLE_RPC_URL` at the peer address, on the same LAN use the Mac's IP
   plus a forward (e.g. `socat TCP-LISTEN:19879,fork,reuseaddr TCP:127.0.0.1:9879`).
2. Apply `deploy/badapple-byoharness.yaml` (set the real host in
   `BADAPPLE_RPC_URL`).
3. `kagent agent session create --agent badapple --request-id <id>` then
   `kagent agent invoke --session <id> --task "..."` — upstream A2A supplies
   the task context and the shim runs as a Substrate actor with
   `/run/kagent/identity` injected.

## Image

`savageazfck/badapple-shim` — built from `go/core/test/byobadapple/main.go`.
The agent card advertises kagent HITL + usage extensions and streaming.

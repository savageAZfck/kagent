# badapple-wrap

Wrap work for running **Bad Apple** (sovereign on-device AGI organism,
macOS / Apple Silicon) as an actor under kagent.

## Status: signs of life achieved

Verified on a local `kind` cluster (kagent `0.10.4`, K8s `v1.37`):

- `badapple-mcp` runs on the host Mac serving the legacy MCP SSE transport
  (`GET /sse` + `POST /message`) on `127.0.0.1:9879`.
- The `RemoteMCPServer` in this directory registers her endpoint via
  `host.docker.internal` — reconcile goes green (`Accepted: True`) and
  kagent discovers her tools over the wire:
  `runtime_status`, `invoke_tool`, `list_models`.
- Whole control plane (17 pods) runs in ~1.2 GiB of a 4 GiB colima VM,
  alongside her resident daemon on a 16 GB MacBook Air.

## Bug found during the wrap (fixed in Bad Apple, not kagent)

Her MCP server answered `notifications/initialized` with a `-32601`
JSON-RPC error on the SSE stream. Per spec, notifications must never
receive a response — the stray error poisoned the mcp-go stream reader
and `tools/list` failed as `invalid request`. Fixed by returning no
response for `notifications/*` and bare `initialized` with no id.

## What the real wrap needs

`type: BYO` upstream expects a container serving A2A on :8080. Bad Apple
cannot be containerized (MLX, Neural Engine, Secure Enclave are
macOS/hardware-bound), so the actor-side integration is either:

1. a thin **A2A shim** container that forwards to her host endpoint
   (works upstream today), or
2. whatever the fork's actor contract defines for an external/
   endpoint-registered actor — TBD from the fork.

Direction of control: kagent manages placement/routing/lifecycle;
Bad Apple's organs (consent, verdicts, ledger, kill path) stay sovereign
on her hardware, reached over mTLS (socket stands in locally).

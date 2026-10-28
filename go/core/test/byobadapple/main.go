// Package main is the Bad Apple A2A shim: it exposes the Bad Apple
// organism (running natively on the host Mac) as a BYO agent over the
// A2A protocol on :8080. Inbound A2A messages are forwarded to Bad
// Apple's MCP JSON-RPC endpoint as inference calls and her response is
// emitted as the completed task's agent message.
//
// Backend: BADAPPLE_RPC_URL (default http://host.docker.internal:9879/rpc).
package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"net/http"
	"os"
	"time"

	a2atype "github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"

	"github.com/kagent-dev/kagent/go/adk/pkg/app"
	"github.com/kagent-dev/kagent/go/adk/pkg/config"
	"github.com/kagent-dev/kagent/go/pkg/logging"
)

type executor struct {
	backend string
	client  *http.Client
}

func messageText(m *a2atype.Message) string {
	if m == nil {
		return ""
	}
	var out string
	for _, p := range m.Parts {
		if t, ok := p.Content.(a2atype.Text); ok {
			out += string(t)
		}
	}
	return out
}

type rpcReq struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

type rpcErr struct {
	Message string `json:"message"`
}
type rpcResp struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcErr         `json:"error"`
}

func (e executor) callBadApple(prompt string) (string, error) {
	req := rpcReq{
		JSONRPC: "2.0", ID: 1, Method: "tools/call",
		Params: map[string]interface{}{
			"name":      "inference",
			"arguments": map[string]interface{}{"prompt": prompt, "max_tokens": 256},
		},
	}
	body, _ := json.Marshal(req)
	resp, err := e.client.Post(e.backend, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("bad apple unreachable: %w", err)
	}
	defer resp.Body.Close()
	var out rpcResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("bad response: %w", err)
	}
	if out.Error != nil {
		return "", fmt.Errorf("bad apple error: %s", out.Error.Message)
	}
	return string(out.Result), nil
}

func (e executor) Execute(_ context.Context, request *a2asrv.ExecutorContext) iter.Seq2[a2atype.Event, error] {
	return func(yield func(a2atype.Event, error) bool) {
		if !yield(a2atype.NewSubmittedTask(request, request.Message), nil) {
			return
		}
		text := messageText(request.Message)
		var msg *a2atype.Message
		if text == "" {
			msg = a2atype.NewMessage(a2atype.MessageRoleAgent, a2atype.NewTextPart("bad apple: empty message"))
		} else if resp, err := e.callBadApple(text); err != nil {
			msg = a2atype.NewMessage(a2atype.MessageRoleAgent, a2atype.NewTextPart(err.Error()))
		} else {
			msg = a2atype.NewMessage(a2atype.MessageRoleAgent, a2atype.NewTextPart(resp))
		}
		msg.ContextID, msg.TaskID = request.ContextID, request.TaskID
		yield(a2atype.NewStatusUpdateEvent(request, a2atype.TaskStateCompleted, msg), nil)
	}
}

func (executor) Cancel(context.Context, *a2asrv.ExecutorContext) iter.Seq2[a2atype.Event, error] {
	return func(func(a2atype.Event, error) bool) {}
}

func main() {
	// /config/config.json is injected by the kagent BYO harness; tolerate
	// its absence for standalone runs (ProcessConfig handles nil).
	_ = config.MaterializeFromEnv("/config")
	compiled, _ := config.LoadAgentConfig("/config/config.json")
	runtime := config.ProcessConfig(compiled)

	logger, err := logging.New(os.Stderr, cmp.Or(runtime.LogLevel, "info"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	if runtime.ControlPlaneUrl == "" {
		runtime.ControlPlaneUrl = cmp.Or(os.Getenv("KAGENT_CONTROL_PLANE_URL"), "http://kagent-controller.kagent.svc:8083")
	}
	backend := cmp.Or(os.Getenv("BADAPPLE_RPC_URL"), "http://host.docker.internal:9879/rpc")

	_ = logging.IntoContext(context.Background(), logger)
	application, err := app.New(app.AppConfig{
		ControlPlaneURL: runtime.ControlPlaneUrl, MaxContentLength: runtime.MaxContentLength,
		AgentCard: a2atype.AgentCard{Name: "badapple", Version: "v1", Capabilities: a2atype.AgentCapabilities{Streaming: true}},
		Port:      fmt.Sprint(cmp.Or(runtime.GetPort(), 8080)), AppName: "badapple", Logger: logger,
	}, executor{backend: backend, client: &http.Client{Timeout: 300 * time.Second}})
	if err != nil {
		logger.Error("failed to create A2A application", "error", err)
		return
	}
	logger.Info("badapple shim up", "backend", backend)
	if err := application.Run(); err != nil {
		logger.Error("a2a application stopped", "error", err)
	}
}

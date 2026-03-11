package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 types
// ---------------------------------------------------------------------------

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ---------------------------------------------------------------------------
// MCP schema helpers
// ---------------------------------------------------------------------------

type ToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolResult struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

var tools = []ToolInfo{
	{
		Name:        "query",
		Description: "Execute a PromQL instant query",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "PromQL query expression",
				},
			},
			"required": []string{"query"},
		},
	},
	{
		Name:        "query_range",
		Description: "Execute a PromQL range query",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "PromQL query expression",
				},
				"start": map[string]any{
					"type":        "string",
					"description": "Start time (RFC3339 or relative like -1h)",
				},
				"end": map[string]any{
					"type":        "string",
					"description": "End time (RFC3339 or relative like now)",
				},
				"step": map[string]any{
					"type":        "string",
					"description": "Query resolution step",
					"default":     "15s",
				},
			},
			"required": []string{"query", "start", "end"},
		},
	},
	{
		Name:        "list_targets",
		Description: "List all scrape targets and their health status",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		Name:        "list_alerts",
		Description: "List active Prometheus alerts",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
}

// ---------------------------------------------------------------------------
// Request handlers
// ---------------------------------------------------------------------------

func handleInitialize(req JSONRPCRequest) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "prometheus-sensor",
				"version": "0.1.0",
			},
		},
	}
}

func handleToolsList(req JSONRPCRequest) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"tools": tools,
		},
	}
}

func handleToolsCall(req JSONRPCRequest) JSONRPCResponse {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return errorResponse(req.ID, -32602, "invalid params: "+err.Error())
	}

	switch params.Name {
	case "query":
		return stubResult(req.ID, "query: not yet implemented")
	case "query_range":
		return stubResult(req.ID, "query_range: not yet implemented")
	case "list_targets":
		return stubResult(req.ID, "list_targets: not yet implemented")
	case "list_alerts":
		return stubResult(req.ID, "list_alerts: not yet implemented")
	default:
		return errorResponse(req.ID, -32601, fmt.Sprintf("unknown tool: %s", params.Name))
	}
}

func stubResult(id any, msg string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: ToolResult{
			Content: []TextContent{{Type: "text", Text: msg}},
		},
	}
}

func errorResponse(id any, code int, msg string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: msg},
	}
}

// ---------------------------------------------------------------------------
// Main loop — stdio JSON-RPC
// ---------------------------------------------------------------------------

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := errorResponse(nil, -32700, "parse error: "+err.Error())
			_ = encoder.Encode(resp)
			continue
		}

		var resp JSONRPCResponse
		switch req.Method {
		case "initialize":
			resp = handleInitialize(req)
		case "notifications/initialized":
			continue // no response for notifications
		case "tools/list":
			resp = handleToolsList(req)
		case "tools/call":
			resp = handleToolsCall(req)
		case "ping":
			resp = JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
		default:
			resp = errorResponse(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
		}

		_ = encoder.Encode(resp)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "stdin read error: %v\n", err)
		os.Exit(1)
	}
}

package mcp

import (
	"encoding/json"

	"github.com/prasenjit-net/mcp-gateway/proxy"
)

const MCPVersion = "2024-11-05"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type Notification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

type ClientCapabilities struct{}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Tool struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	InputSchema  map[string]interface{} `json:"inputSchema"`
	OutputSchema map[string]interface{} `json:"outputSchema,omitempty"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []proxy.MCPContent `json:"content"`
	IsError bool               `json:"isError,omitempty"`
}

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"` // base64 for binary
}

type ListResourcesParams struct {
	Cursor string `json:"cursor,omitempty"`
}

type ListResourcesResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

type ListResourceTemplatesResult struct {
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

type ReadResourceParams struct {
	URI string `json:"uri"`
}

type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

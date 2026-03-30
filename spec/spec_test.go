package spec_test

import (
	"testing"

	"github.com/prasenjit-net/mcp-gateway/spec"
)

const petStoreSpec = `{
  "openapi": "3.0.0",
  "info": {"title": "Pet Store", "version": "1.0.0"},
  "paths": {
    "/pets": {
      "get": {
        "operationId": "listPets",
        "summary": "List all pets",
        "parameters": [
          {"name": "limit", "in": "query", "schema": {"type": "integer"}}
        ],
        "responses": {"200": {"description": "ok"}}
      },
      "post": {
        "operationId": "createPet",
        "summary": "Create a pet",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "name": {"type": "string"},
                  "tag":  {"type": "string"}
                },
                "required": ["name"]
              }
            }
          }
        },
        "responses": {"201": {"description": "created"}}
      }
    },
    "/pets/{petId}": {
      "get": {
        "operationId": "getPet",
        "summary": "Get a pet",
        "parameters": [
          {"name": "petId", "in": "path", "required": true, "schema": {"type": "string"}}
        ],
        "responses": {"200": {"description": "ok"}}
      },
      "delete": {
        "operationId": "deletePet",
        "summary": "Delete a pet",
        "parameters": [
          {"name": "petId", "in": "path", "required": true, "schema": {"type": "string"}}
        ],
        "responses": {"204": {"description": "deleted"}}
      }
    }
  }
}`

func TestParseValidSpec(t *testing.T) {
	parsed, err := spec.Parse([]byte(petStoreSpec))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected non-nil parsed spec")
	}
	if parsed.Doc == nil {
		t.Fatal("expected non-nil doc")
	}
}

func TestParseInvalidSpec(t *testing.T) {
	_, err := spec.Parse([]byte(`{invalid json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseEmptyPaths(t *testing.T) {
	minimalSpec := `{"openapi":"3.0.0","info":{"title":"Min","version":"1.0"},"paths":{}}`
	parsed, err := spec.Parse([]byte(minimalSpec))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	tools, ops, err := spec.ExtractTools("s1", "Min", "https://api.example.com", parsed, false, false, nil, false)
	if err != nil {
		t.Fatalf("ExtractTools error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected 0 tools from empty paths, got %d", len(tools))
	}
	if len(ops) != 0 {
		t.Errorf("expected 0 ops from empty paths, got %d", len(ops))
	}
}

func TestExtractToolsFromPetStore(t *testing.T) {
	parsed, err := spec.Parse([]byte(petStoreSpec))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	tools, ops, err := spec.ExtractTools("spec-1", "Pet Store", "https://api.example.com", parsed, false, false, nil, false)
	if err != nil {
		t.Fatalf("ExtractTools: %v", err)
	}
	if len(tools) != 4 {
		t.Errorf("expected 4 tools (listPets, createPet, getPet, deletePet), got %d", len(tools))
	}
	if len(ops) != 4 {
		t.Errorf("expected 4 operations, got %d", len(ops))
	}

	toolMap := make(map[string]*spec.ToolDefinition)
	for _, t := range tools {
		toolMap[t.OperationID] = t
	}

	listPets, ok := toolMap["listPets"]
	if !ok {
		t.Fatal("listPets tool not found")
	}
	if listPets.Method != "GET" {
		t.Errorf("listPets.Method = %q, want GET", listPets.Method)
	}
	if listPets.PathTemplate != "/pets" {
		t.Errorf("listPets.PathTemplate = %q, want /pets", listPets.PathTemplate)
	}
	if listPets.Upstream != "https://api.example.com" {
		t.Errorf("listPets.Upstream = %q", listPets.Upstream)
	}

	getPet, ok := toolMap["getPet"]
	if !ok {
		t.Fatal("getPet tool not found")
	}
	if getPet.PathTemplate != "/pets/{petId}" {
		t.Errorf("getPet.PathTemplate = %q", getPet.PathTemplate)
	}
}

func TestExtractToolsPassthroughFlags(t *testing.T) {
	parsed, _ := spec.Parse([]byte(petStoreSpec))
	tools, _, err := spec.ExtractTools("s1", "Test", "https://api.example.com", parsed,
		true, true, []string{"X-Custom"}, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools {
		if !tool.PassthroughAuth {
			t.Errorf("tool %q should have PassthroughAuth=true", tool.Name)
		}
		if !tool.PassthroughCookies {
			t.Errorf("tool %q should have PassthroughCookies=true", tool.Name)
		}
		if !tool.MTLSEnabled {
			t.Errorf("tool %q should have MTLSEnabled=true", tool.Name)
		}
	}
}

func TestExtractToolsInputSchema(t *testing.T) {
	parsed, _ := spec.Parse([]byte(petStoreSpec))
	tools, _, _ := spec.ExtractTools("s1", "Test", "https://api.example.com", parsed, false, false, nil, false)

	toolMap := make(map[string]*spec.ToolDefinition)
	for _, t := range tools {
		toolMap[t.OperationID] = t
	}

	createPet, ok := toolMap["createPet"]
	if !ok {
		t.Fatal("createPet not found")
	}
	if createPet.InputSchema == nil {
		t.Fatal("createPet.InputSchema should not be nil")
	}
}

func TestExtractToolsOperationRecords(t *testing.T) {
	parsed, _ := spec.Parse([]byte(petStoreSpec))
	_, ops, err := spec.ExtractTools("spec-1", "Pet Store", "https://api.example.com", parsed, false, false, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range ops {
		if op.SpecID != "spec-1" {
			t.Errorf("op.SpecID = %q, want spec-1", op.SpecID)
		}
		if op.ID == "" {
			t.Error("op.ID should not be empty")
		}
		if op.Method == "" {
			t.Error("op.Method should not be empty")
		}
	}
}

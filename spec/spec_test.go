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

const schemaMetadataSpec = `{
  "openapi": "3.0.0",
  "info": {"title": "Schema Metadata", "version": "1.0.0"},
  "paths": {
    "/projects": {
      "get": {
        "operationId": "listProjects",
        "summary": "List projects",
        "description": "Returns the available projects.",
        "parameters": [
          {
            "name": "limit",
            "in": "query",
            "description": "Maximum number of projects to return",
            "example": 20,
            "schema": {
              "type": "integer",
              "minimum": 1,
              "maximum": 100,
              "default": 20
            }
          }
        ],
        "responses": {
          "200": {
            "description": "Paginated list of projects",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "data": {
                      "type": "array",
                      "items": {
                        "type": "object",
                        "properties": {
                          "id": {"type": "string", "description": "Project identifier"}
                        },
                        "required": ["id"]
                      }
                    },
                    "total": {"type": "integer", "description": "Total count"}
                  },
                  "required": ["data", "total"]
                }
              }
            }
          }
        }
      },
      "post": {
        "operationId": "createProject",
        "summary": "Create project",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "description": "Project creation payload",
                "required": ["name"],
                "properties": {
                  "name": {
                    "type": "string",
                    "description": "Unique display name",
                    "minLength": 1,
                    "maxLength": 100
                  },
                  "colour": {
                    "type": "string",
                    "description": "Project colour",
                    "pattern": "^#[0-9a-fA-F]{6}$"
                  }
                }
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Created project",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "id": {"type": "string"},
                    "name": {"type": "string"}
                  },
                  "required": ["id", "name"]
                }
              }
            }
          }
        }
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

func TestExtractToolsPreservesSchemaMetadata(t *testing.T) {
	parsed, err := spec.Parse([]byte(schemaMetadataSpec))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	tools, _, err := spec.ExtractTools("spec-2", "Schema Metadata", "https://api.example.com", parsed, false, false, nil, false)
	if err != nil {
		t.Fatalf("ExtractTools: %v", err)
	}

	toolMap := make(map[string]*spec.ToolDefinition, len(tools))
	for _, tool := range tools {
		toolMap[tool.OperationID] = tool
	}

	listProjects := toolMap["listProjects"]
	if listProjects == nil {
		t.Fatal("listProjects tool not found")
	}

	limitProp := listProjects.InputSchema["properties"].(map[string]interface{})["limit"].(map[string]interface{})
	if got := limitProp["description"]; got != "Maximum number of projects to return" {
		t.Fatalf("limit description = %v, want parameter description", got)
	}
	if got := limitProp["minimum"]; got != float64(1) {
		t.Fatalf("limit minimum = %v, want 1", got)
	}
	if got := limitProp["maximum"]; got != float64(100) {
		t.Fatalf("limit maximum = %v, want 100", got)
	}
	if got := limitProp["default"]; got != float64(20) {
		t.Fatalf("limit default = %v, want 20", got)
	}
	if got := limitProp["example"]; got != float64(20) {
		t.Fatalf("limit example = %v, want 20", got)
	}

	if listProjects.OutputSchema == nil {
		t.Fatal("listProjects.OutputSchema should not be nil")
	}
	if got := listProjects.OutputSchema["description"]; got != "Paginated list of projects" {
		t.Fatalf("output description = %v, want response description", got)
	}

	outputProps := listProjects.OutputSchema["properties"].(map[string]interface{})
	totalProp := outputProps["total"].(map[string]interface{})
	if got := totalProp["description"]; got != "Total count" {
		t.Fatalf("output total description = %v, want Total count", got)
	}

	createProject := toolMap["createProject"]
	if createProject == nil {
		t.Fatal("createProject tool not found")
	}
	if got := createProject.InputSchema["description"]; got != "Project creation payload" {
		t.Fatalf("input schema description = %v, want request body description", got)
	}

	createProps := createProject.InputSchema["properties"].(map[string]interface{})
	nameProp := createProps["name"].(map[string]interface{})
	if got := nameProp["minLength"]; got != uint64(1) {
		t.Fatalf("name minLength = %v, want 1", got)
	}
	if got := nameProp["maxLength"]; got != uint64(100) {
		t.Fatalf("name maxLength = %v, want 100", got)
	}

	colourProp := createProps["colour"].(map[string]interface{})
	if got := colourProp["pattern"]; got != "^#[0-9a-fA-F]{6}$" {
		t.Fatalf("colour pattern = %v, want hex pattern", got)
	}
}

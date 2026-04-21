package spec

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/prasenjit-net/mcp-gateway/store"
)

type ToolDefinition struct {
	Name               string
	Description        string
	InputSchema        map[string]interface{}
	OutputSchema       map[string]interface{}
	OperationID        string
	SpecID             string
	Method             string
	PathTemplate       string
	Upstream           string
	PassthroughAuth    bool
	PassthroughCookies bool
	PassthroughHeaders []string
	MTLSEnabled        bool
}

func ExtractTools(specID, specName, upstream string, parsed *ParsedSpec, passthroughAuth bool, passthroughCookies bool, passthroughHeaders []string, mtlsEnabled bool) ([]*ToolDefinition, []*store.OperationRecord, error) {
	var tools []*ToolDefinition
	var ops []*store.OperationRecord

	doc := parsed.Doc
	if doc.Paths == nil {
		return tools, ops, nil
	}

	type entry struct {
		method string
		op     *openapi3.Operation
		path   string
	}

	for pathStr, pathItem := range doc.Paths.Map() {
		if pathItem == nil {
			continue
		}

		var entries []entry
		if pathItem.Get != nil {
			entries = append(entries, entry{"GET", pathItem.Get, pathStr})
		}
		if pathItem.Post != nil {
			entries = append(entries, entry{"POST", pathItem.Post, pathStr})
		}
		if pathItem.Put != nil {
			entries = append(entries, entry{"PUT", pathItem.Put, pathStr})
		}
		if pathItem.Patch != nil {
			entries = append(entries, entry{"PATCH", pathItem.Patch, pathStr})
		}
		if pathItem.Delete != nil {
			entries = append(entries, entry{"DELETE", pathItem.Delete, pathStr})
		}
		if pathItem.Head != nil {
			entries = append(entries, entry{"HEAD", pathItem.Head, pathStr})
		}

		for _, e := range entries {
			op := e.op
			toolName := op.OperationID
			if toolName == "" {
				toolName = sanitizeName(e.method + "_" + e.path)
			}

			description := op.Summary
			if description == "" {
				description = op.Description
			}

			allParams := make(openapi3.Parameters, 0)
			allParams = append(allParams, pathItem.Parameters...)
			allParams = append(allParams, op.Parameters...)

			var bodySchema map[string]interface{}
			var bodyRequired bool
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				bodyRequired = op.RequestBody.Value.Required
				if mt, ok := op.RequestBody.Value.Content["application/json"]; ok && mt.Schema != nil && mt.Schema.Value != nil {
					bodySchema = schemaToMap(mt.Schema.Value)
				}
			}

			inputSchema := buildInputSchema(allParams, bodySchema, bodyRequired)
			outputSchema := buildOutputSchema(op.Responses)

			var tags []string
			if op.Tags != nil {
				tags = op.Tags
			}

			tool := &ToolDefinition{
				Name:               toolName,
				Description:        description,
				InputSchema:        inputSchema,
				OutputSchema:       outputSchema,
				OperationID:        toolName,
				SpecID:             specID,
				Method:             e.method,
				PathTemplate:       e.path,
				Upstream:           upstream,
				PassthroughAuth:    passthroughAuth,
				PassthroughCookies: passthroughCookies,
				PassthroughHeaders: passthroughHeaders,
				MTLSEnabled:        mtlsEnabled,
			}
			tools = append(tools, tool)

			opID := fmt.Sprintf("%s-%s-%s", specID, strings.ToLower(e.method), sanitizeName(e.path))
			opRec := &store.OperationRecord{
				ID:          opID,
				SpecID:      specID,
				OperationID: toolName,
				Method:      e.method,
				Path:        e.path,
				Summary:     op.Summary,
				Description: op.Description,
				Tags:        tags,
				Enabled:     true,
			}
			ops = append(ops, opRec)
		}
	}

	return tools, ops, nil
}

func sanitizeName(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.Trim(s, "_")
	return s
}

func buildInputSchema(params openapi3.Parameters, bodySchema map[string]interface{}, bodyRequired bool) map[string]interface{} {
	properties := map[string]interface{}{}
	required := []string{}

	for _, paramRef := range params {
		if paramRef == nil || paramRef.Value == nil {
			continue
		}
		p := paramRef.Value
		propSchema := parameterToMap(p)
		properties[p.Name] = propSchema
		if p.Required || p.In == "path" {
			required = appendUniqueString(required, p.Name)
		}
	}

	if bodySchema != nil {
		if bodyProps, ok := bodySchema["properties"].(map[string]interface{}); ok {
			for k, v := range bodyProps {
				properties[k] = v
			}
			if bodyRequired, ok := bodySchema["required"].([]interface{}); ok {
				for _, r := range bodyRequired {
					if rs, ok := r.(string); ok {
						required = appendUniqueString(required, rs)
					}
				}
			}
			if bodyRequired, ok := bodySchema["required"].([]string); ok {
				for _, r := range bodyRequired {
					required = appendUniqueString(required, r)
				}
			}
		} else {
			properties["body"] = bodySchema
			if bodyRequired {
				required = appendUniqueString(required, "body")
			}
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if bodyDescription, ok := bodySchema["description"].(string); ok && bodyDescription != "" {
		schema["description"] = bodyDescription
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func buildOutputSchema(responses *openapi3.Responses) map[string]interface{} {
	if responses == nil || responses.Len() == 0 {
		return nil
	}

	type responseCandidate struct {
		priority int
		response *openapi3.Response
	}

	candidates := make([]responseCandidate, 0, responses.Len()+1)
	for status, respRef := range responses.Map() {
		if respRef == nil || respRef.Value == nil {
			continue
		}
		code, err := strconv.Atoi(status)
		if err != nil || code < 200 || code >= 300 {
			continue
		}
		candidates = append(candidates, responseCandidate{priority: code, response: respRef.Value})
	}
	if defaultRef := responses.Default(); defaultRef != nil && defaultRef.Value != nil {
		candidates = append(candidates, responseCandidate{priority: 1000, response: defaultRef.Value})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].priority < candidates[j].priority
	})

	for _, candidate := range candidates {
		schema := responseSchemaToMap(candidate.response)
		if schema != nil {
			return schema
		}
	}

	return nil
}

func responseSchemaToMap(resp *openapi3.Response) map[string]interface{} {
	if resp == nil || len(resp.Content) == 0 {
		return nil
	}

	if mt, ok := resp.Content["application/json"]; ok && mt != nil && mt.Schema != nil && mt.Schema.Value != nil {
		schema := schemaToMap(mt.Schema.Value)
		if desc := derefString(resp.Description); desc != "" {
			if _, exists := schema["description"]; !exists {
				schema["description"] = desc
			}
		}
		return schema
	}

	contentTypes := make([]string, 0, len(resp.Content))
	for contentType := range resp.Content {
		contentTypes = append(contentTypes, contentType)
	}
	sort.Strings(contentTypes)

	for _, contentType := range contentTypes {
		mt := resp.Content[contentType]
		if mt == nil || mt.Schema == nil || mt.Schema.Value == nil {
			continue
		}
		schema := schemaToMap(mt.Schema.Value)
		if desc := derefString(resp.Description); desc != "" {
			if _, exists := schema["description"]; !exists {
				schema["description"] = desc
			}
		}
		return schema
	}

	return nil
}

func parameterToMap(p *openapi3.Parameter) map[string]interface{} {
	var schema map[string]interface{}
	if p.Schema != nil && p.Schema.Value != nil {
		schema = schemaToMap(p.Schema.Value)
	} else {
		schema = map[string]interface{}{"type": "string"}
	}

	if p.Description != "" {
		schema["description"] = p.Description
	}
	if p.Example != nil {
		if _, exists := schema["example"]; !exists {
			schema["example"] = p.Example
		}
	}

	return schema
}

func schemaToMap(s *openapi3.Schema) map[string]interface{} {
	if s == nil {
		return map[string]interface{}{"type": "string"}
	}
	m := map[string]interface{}{}
	if s.Type != nil {
		types := *s.Type
		if len(types) == 1 {
			m["type"] = types[0]
		} else if len(types) > 1 {
			m["type"] = []string(types)
		}
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if s.Title != "" {
		m["title"] = s.Title
	}
	if s.Format != "" {
		m["format"] = s.Format
	}
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}
	if s.Default != nil {
		m["default"] = s.Default
	}
	if s.Example != nil {
		m["example"] = s.Example
	}
	if s.Nullable {
		m["nullable"] = true
	}
	if s.ReadOnly {
		m["readOnly"] = true
	}
	if s.WriteOnly {
		m["writeOnly"] = true
	}
	if s.Deprecated {
		m["deprecated"] = true
	}
	if s.UniqueItems {
		m["uniqueItems"] = true
	}
	if s.ExclusiveMin {
		m["exclusiveMinimum"] = true
	}
	if s.ExclusiveMax {
		m["exclusiveMaximum"] = true
	}
	if s.Min != nil {
		m["minimum"] = *s.Min
	}
	if s.Max != nil {
		m["maximum"] = *s.Max
	}
	if s.MultipleOf != nil {
		m["multipleOf"] = *s.MultipleOf
	}
	if s.MinLength > 0 {
		m["minLength"] = s.MinLength
	}
	if s.MaxLength != nil {
		m["maxLength"] = *s.MaxLength
	}
	if s.Pattern != "" {
		m["pattern"] = s.Pattern
	}
	if s.MinItems > 0 {
		m["minItems"] = s.MinItems
	}
	if s.MaxItems != nil {
		m["maxItems"] = *s.MaxItems
	}
	if s.MinProps > 0 {
		m["minProperties"] = s.MinProps
	}
	if s.MaxProps != nil {
		m["maxProperties"] = *s.MaxProps
	}
	if s.Properties != nil {
		props := map[string]interface{}{}
		for k, v := range s.Properties {
			if v != nil && v.Value != nil {
				props[k] = schemaToMap(v.Value)
			}
		}
		m["properties"] = props
	}
	if s.Items != nil && s.Items.Value != nil {
		m["items"] = schemaToMap(s.Items.Value)
	}
	if len(s.OneOf) > 0 {
		m["oneOf"] = schemaRefsToMaps(s.OneOf)
	}
	if len(s.AnyOf) > 0 {
		m["anyOf"] = schemaRefsToMaps(s.AnyOf)
	}
	if len(s.AllOf) > 0 {
		m["allOf"] = schemaRefsToMaps(s.AllOf)
	}
	if s.Not != nil && s.Not.Value != nil {
		m["not"] = schemaToMap(s.Not.Value)
	}
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}
	return m
}

func schemaRefsToMaps(refs openapi3.SchemaRefs) []interface{} {
	items := make([]interface{}, 0, len(refs))
	for _, ref := range refs {
		if ref == nil || ref.Value == nil {
			continue
		}
		items = append(items, schemaToMap(ref.Value))
	}
	return items
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

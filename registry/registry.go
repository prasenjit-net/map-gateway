package registry

import (
	"sync"

	"github.com/prasenjit-net/mcp-gateway/spec"
	"github.com/prasenjit-net/mcp-gateway/store"
)

type Registry struct {
	mu          sync.RWMutex
	tools       map[string]*spec.ToolDefinition
	resources   []*store.ResourceRecord
	subscribers []chan struct{}
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*spec.ToolDefinition),
	}
}

func (r *Registry) RebuildAll(tools []*spec.ToolDefinition) {
	r.mu.Lock()
	newMap := make(map[string]*spec.ToolDefinition, len(tools))
	for _, t := range tools {
		newMap[t.Name] = t
	}
	r.tools = newMap
	subs := r.subscribers
	r.mu.Unlock()

	r.notifySubscribers(subs)
}

func (r *Registry) Get(name string) (*spec.ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) List() []*spec.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*spec.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

func (r *Registry) Subscribe() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	ch := make(chan struct{}, 1)
	r.subscribers = append(r.subscribers, ch)
	return ch
}

func (r *Registry) Unsubscribe(ch <-chan struct{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, sub := range r.subscribers {
		if sub == ch {
			r.subscribers = append(r.subscribers[:i], r.subscribers[i+1:]...)
			return
		}
	}
}

func (r *Registry) notifySubscribers(subs []chan struct{}) {
	for _, ch := range subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (r *Registry) RebuildResources(resources []*store.ResourceRecord) {
	r.mu.Lock()
	r.resources = resources
	subs := r.subscribers
	r.mu.Unlock()
	r.notifySubscribers(subs)
}

func (r *Registry) ListStaticResources() []*store.ResourceRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*store.ResourceRecord
	for _, res := range r.resources {
		if !res.IsTemplate {
			out = append(out, res)
		}
	}
	return out
}

func (r *Registry) ListTemplateResources() []*store.ResourceRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*store.ResourceRecord
	for _, res := range r.resources {
		if res.IsTemplate {
			out = append(out, res)
		}
	}
	return out
}

func (r *Registry) GetResourceByID(id string) (*store.ResourceRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, res := range r.resources {
		if res.ID == id {
			return res, true
		}
	}
	return nil, false
}

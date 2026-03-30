package registry_test

import (
	"testing"
	"time"

	"github.com/prasenjit-net/mcp-gateway/registry"
	"github.com/prasenjit-net/mcp-gateway/spec"
	"github.com/prasenjit-net/mcp-gateway/store"
)

func tools(names ...string) []*spec.ToolDefinition {
	out := make([]*spec.ToolDefinition, len(names))
	for i, n := range names {
		out[i] = &spec.ToolDefinition{Name: n}
	}
	return out
}

func TestRebuildAllAndGet(t *testing.T) {
	r := registry.NewRegistry()
	r.RebuildAll(tools("tool-a", "tool-b"))

	if _, ok := r.Get("tool-a"); !ok {
		t.Error("tool-a not found")
	}
	if _, ok := r.Get("tool-b"); !ok {
		t.Error("tool-b not found")
	}
	if _, ok := r.Get("tool-c"); ok {
		t.Error("tool-c should not exist")
	}
}

func TestRebuildAllReplacesTools(t *testing.T) {
	r := registry.NewRegistry()
	r.RebuildAll(tools("old-tool"))
	r.RebuildAll(tools("new-tool"))

	if _, ok := r.Get("old-tool"); ok {
		t.Error("old-tool should have been replaced")
	}
	if _, ok := r.Get("new-tool"); !ok {
		t.Error("new-tool should exist")
	}
}

func TestList(t *testing.T) {
	r := registry.NewRegistry()
	r.RebuildAll(tools("a", "b", "c"))
	list := r.List()
	if len(list) != 3 {
		t.Errorf("List() len = %d, want 3", len(list))
	}
}

func TestListEmpty(t *testing.T) {
	r := registry.NewRegistry()
	list := r.List()
	if len(list) != 0 {
		t.Errorf("List() should be empty on new registry, got %d", len(list))
	}
}

func TestSubscribeReceivesNotification(t *testing.T) {
	r := registry.NewRegistry()
	ch := r.Subscribe()
	defer r.Unsubscribe(ch)

	go r.RebuildAll(tools("notify-tool"))

	select {
	case <-ch:
		// good
	case <-time.After(time.Second):
		t.Error("did not receive registry notification within 1s")
	}
}

func TestUnsubscribeStopsNotifications(t *testing.T) {
	r := registry.NewRegistry()
	ch := r.Subscribe()
	r.Unsubscribe(ch)

	// Drain any pending notifications
	for len(ch) > 0 {
		<-ch
	}

	r.RebuildAll(tools("after-unsub"))

	select {
	case <-ch:
		t.Error("should not receive notification after unsubscribe")
	case <-time.After(50 * time.Millisecond):
		// good — no notification
	}
}

func TestMultipleSubscribers(t *testing.T) {
	r := registry.NewRegistry()
	ch1 := r.Subscribe()
	ch2 := r.Subscribe()
	defer r.Unsubscribe(ch1)
	defer r.Unsubscribe(ch2)

	r.RebuildAll(tools("multi"))

	for _, ch := range []<-chan struct{}{ch1, ch2} {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Error("subscriber did not receive notification")
		}
	}
}

// ── Resources ─────────────────────────────────────────────────────────────────

func TestRebuildResources(t *testing.T) {
	r := registry.NewRegistry()
	resources := []*store.ResourceRecord{
		{ID: "r1", Name: "Resource 1", IsTemplate: false},
		{ID: "r2", Name: "Template 1", IsTemplate: true},
		{ID: "r3", Name: "Resource 2", IsTemplate: false},
	}
	r.RebuildResources(resources)

	statics := r.ListStaticResources()
	if len(statics) != 2 {
		t.Errorf("ListStaticResources() = %d, want 2", len(statics))
	}

	templates := r.ListTemplateResources()
	if len(templates) != 1 {
		t.Errorf("ListTemplateResources() = %d, want 1", len(templates))
	}
}

func TestGetResourceByID(t *testing.T) {
	r := registry.NewRegistry()
	r.RebuildResources([]*store.ResourceRecord{
		{ID: "abc", Name: "My Resource"},
	})
	res, ok := r.GetResourceByID("abc")
	if !ok {
		t.Fatal("resource abc not found")
	}
	if res.Name != "My Resource" {
		t.Errorf("Name = %q", res.Name)
	}
	_, ok = r.GetResourceByID("notexist")
	if ok {
		t.Error("should not find nonexistent resource")
	}
}

func TestRebuildResourcesNotifiesSubscribers(t *testing.T) {
	r := registry.NewRegistry()
	ch := r.Subscribe()
	defer r.Unsubscribe(ch)

	go r.RebuildResources([]*store.ResourceRecord{{ID: "x"}})

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Error("no notification for RebuildResources")
	}
}

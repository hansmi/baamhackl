package watchmantrigger

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/hansmi/baamhackl/internal/config"
)

type fakeClient struct {
	mu         sync.Mutex
	configured map[string]map[string]struct{}
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		configured: map[string]map[string]struct{}{},
	}
}

func (c *fakeClient) Ping(ctx context.Context) error {
	panic("not implemented")
}

func (c *fakeClient) WatchSet(ctx context.Context, root string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configured[root] == nil {
		c.configured[root] = map[string]struct{}{}
	}

	return nil
}

func (c *fakeClient) Recrawl(ctx context.Context, root string) error {
	if watch := c.configured[root]; watch == nil {
		return fmt.Errorf("watch on root %q not set", root)
	}

	return nil
}

func (c *fakeClient) TriggerSet(ctx context.Context, root string, args any) error {
	name := args.(map[string]any)["name"].(string)

	c.mu.Lock()
	defer c.mu.Unlock()

	if watch := c.configured[root]; watch == nil {
		return fmt.Errorf("watch on root %q not set", root)
	} else if _, ok := watch[name]; ok {
		return fmt.Errorf("trigger %q already set on root %q", name, root)
	} else {
		watch[name] = struct{}{}
	}

	return nil
}

func (c *fakeClient) TriggerDel(ctx context.Context, root, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if watch := c.configured[root]; watch == nil {
		return fmt.Errorf("watch on root %q not set", root)
	} else if _, ok := watch[name]; !ok {
		return fmt.Errorf("no trigger named %q on root %q", name, root)
	} else {
		delete(watch, name)
	}

	return nil
}

func (c *fakeClient) ShutdownServer(context.Context) error {
	return nil
}

func TestGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := newFakeClient()
	g := Group{
		Client: client,
	}

	num := 0

	for _, count := range []int{0, 5, 20, maxConcurrent} {
		var handlers []*config.Handler

		path := t.TempDir()

		for i := 0; i < count; i++ {
			handlers = append(handlers, &config.Handler{
				Name: fmt.Sprintf("handler%d", num),
				Path: path,
			})
			num++
		}

		if err := g.SetAll(ctx, handlers); err != nil {
			t.Errorf("SetAll() failed: %v", err)
		}

		defer func() {
			client.mu.Lock()
			defer client.mu.Unlock()

			if got := len(client.configured[path]); got != 0 {
				t.Errorf("DeleteAll() didn't delete all triggers, %d remain", got)
			}
		}()

		if err := g.RecrawlAll(ctx); err != nil {
			t.Errorf("RecrawlAll() failed: %v", err)
		}
	}

	if err := g.DeleteAll(ctx); err != nil {
		t.Errorf("DeleteAll() failed: %v", err)
	}
}

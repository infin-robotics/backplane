package backplane_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/infin-robotics/backplane"
)

func TestOnMessageObservesBeforeFanout(t *testing.T) {
	type message struct{ ID int }
	observed := make(chan struct{})
	publisher := func(_ context.Context, out chan<- message) error {
		out <- message{42}
		return nil
	}
	subscriber := func(_ context.Context, in <-chan message) error {
		for value := range in {
			select {
			case <-observed:
			default:
				t.Error("subscriber received message before observation")
			}
			if value.ID != 42 {
				t.Errorf("received %v", value)
			}
		}
		return nil
	}
	bp, err := backplane.New(publisher, subscriber, subscriber)
	if err != nil {
		t.Fatal(err)
	}
	wantSender := bp.Graph().Nodes[0]
	calls := 0
	ctx := context.WithValue(context.Background(), contextKey{}, "marker")
	err = bp.Run(ctx, backplane.OnMessage(func(ctx context.Context, sender backplane.Node, topic reflect.Type, value any) {
		calls++
		if sender != wantSender {
			t.Errorf("sender = %+v, want %+v", sender, wantSender)
		}
		if topic != reflect.TypeFor[message]() || value != (message{42}) {
			t.Errorf("topic/value = %v/%v", topic, value)
		}
		if ctx.Value(contextKey{}) != "marker" {
			t.Error("missing run context value")
		}
		close(observed)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("observed %d publications, want 1", calls)
	}
}

func TestOptionsAreRunScopedAndFunctionsRemainResources(t *testing.T) {
	type callback = func(context.Context, backplane.Node, reflect.Type, any)
	var resourceCalls, firstCalls, lastCalls int
	resource := callback(func(context.Context, backplane.Node, reflect.Type, any) { resourceCalls++ })
	bp, err := backplane.New(func(ctx context.Context, fn callback, out chan<- int) error {
		fn(ctx, backplane.Node{}, reflect.TypeFor[int](), 7)
		out <- 7
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	first := backplane.OnMessage(func(context.Context, backplane.Node, reflect.Type, any) { firstCalls++ })
	last := backplane.OnMessage(func(context.Context, backplane.Node, reflect.Type, any) { lastCalls++ })
	for _, args := range [][]any{
		{first, resource, last},
		{resource},
		{first, backplane.OnMessage(nil), resource},
	} {
		if err := bp.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
	}
	if resourceCalls != 3 || firstCalls != 0 || lastCalls != 1 {
		t.Fatalf("resource/first/last calls = %d/%d/%d", resourceCalls, firstCalls, lastCalls)
	}
}

func TestOnMessageIdentifiesDuplicateModulesAndDeclaredInterfaceTopic(t *testing.T) {
	publisher := func(_ context.Context, out chan<- any) error { out <- "hello"; return nil }
	bp, err := backplane.New(publisher, publisher)
	if err != nil {
		t.Fatal(err)
	}
	nodes := bp.Graph().Nodes
	seen := map[backplane.Node]int{}
	err = bp.Run(context.Background(), backplane.OnMessage(func(_ context.Context, sender backplane.Node, topic reflect.Type, value any) {
		seen[sender]++
		if topic != reflect.TypeFor[any]() || value != "hello" {
			t.Errorf("topic/value = %v/%v", topic, value)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 2 || seen[nodes[0]] != 1 || seen[nodes[1]] != 1 {
		t.Fatalf("senders = %v", seen)
	}
}

func TestOnMessageCanOverlapAcrossTopics(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		arrived := make(chan struct{}, 2)
		release := make(chan struct{})
		bp, err := backplane.New(func(_ context.Context, out chan<- int) error { out <- 1; return nil },
			func(_ context.Context, out chan<- string) error { out <- "two"; return nil })
		if err != nil {
			t.Fatal(err)
		}
		var runErr error
		go func() {
			runErr = bp.Run(context.Background(), backplane.OnMessage(func(context.Context, backplane.Node, reflect.Type, any) {
				arrived <- struct{}{}
				<-release
			}))
		}()
		synctest.Wait()
		count := len(arrived)
		close(release)
		synctest.Wait()
		if count != 2 {
			t.Fatalf("only %d topics entered callback concurrently", count)
		}
		if runErr != nil {
			t.Fatal(runErr)
		}
	})
}

func TestOnMessageDrainsDuringShutdownAndRunWaitsForHandler(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		release := make(chan struct{})
		var releaseOnce sync.Once
		defer releaseOnce.Do(func() { close(release) })
		entered := make(chan struct{})
		bp, err := backplane.New(func(ctx context.Context, out chan<- int) error {
			<-ctx.Done()
			out <- 42 // Bare sends must still unwind during shutdown.
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		returned := make(chan error, 1)
		calls := 0
		go func() {
			returned <- bp.Run(ctx, backplane.OnMessage(func(ctx context.Context, _ backplane.Node, _ reflect.Type, value any) {
				calls++
				if !errors.Is(ctx.Err(), context.Canceled) || value != 42 {
					t.Errorf("context/value = %v/%v", ctx.Err(), value)
				}
				close(entered)
				<-release
			}))
		}()
		cancel()
		synctest.Wait()
		select {
		case <-entered:
		default:
			t.Fatal("drained publication was not observed")
		}
		select {
		case err := <-returned:
			t.Fatalf("Run returned before handler: %v", err)
		default:
		}
		releaseOnce.Do(func() { close(release) })
		synctest.Wait()
		if err := <-returned; !errors.Is(err, context.Canceled) {
			t.Fatalf("Run = %v", err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d", calls)
		}
	})
}

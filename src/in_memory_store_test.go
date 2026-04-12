package main

import (
	"os"
	"testing"
	"time"

	"sleuth/internal/ringbuffer"
)

// newStoreService builds a service with an initialized history buffer,
// suitable for use in store tests.
func newStoreService(id uint, name string) Service {
	s := Service{
		ID:             id,
		Name:           name,
		Address:        "localhost:8080",
		ProtocolString: "TCP",
		Timer:          30,
		History:        ringbuffer.NewRingBuffer[EventData](10),
	}
	s.protocol = NewProtocol(s)
	return s
}

// newTestStore returns a fresh in-memory store pre-loaded with the given services.
// noHistory=true prevents loading from any .sleuth.bin on disk.
func newTestStore(services ...Service) *InMemoryStore {
	store, _ := NewInMemoryStore(true)
	for _, s := range services {
		store.AddService(s)
	}
	return store
}

// cdTemp changes the working directory to a temp dir for the duration of the
// test. Required for Save/Load tests since they write .sleuth.bin relative to cwd.
func cdTemp(t *testing.T) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(t.TempDir())
}

// ---- AddService ----

func TestAddService_Single(t *testing.T) {
	store := newTestStore()
	store.AddService(newStoreService(1, "Alpha"))

	services := *store.GetServices()
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].ID != 1 {
		t.Errorf("expected id=1, got %d", services[0].ID)
	}
}

func TestAddService_Multiple(t *testing.T) {
	store := newTestStore(
		newStoreService(1, "Alpha"),
		newStoreService(2, "Beta"),
		newStoreService(3, "Gamma"),
	)
	if len(*store.GetServices()) != 3 {
		t.Errorf("expected 3 services, got %d", len(*store.GetServices()))
	}
}

// ---- GetServiceByID ----

func TestGetServiceByID_Found(t *testing.T) {
	store := newTestStore(newStoreService(42, "Target"))

	s, err := store.GetServiceByID(42)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if s.Name != "Target" {
		t.Errorf("expected name=Target, got %q", s.Name)
	}
}

func TestGetServiceByID_NotFound(t *testing.T) {
	store := newTestStore(newStoreService(1, "Alpha"))

	_, err := store.GetServiceByID(99)
	if err == nil {
		t.Error("expected error for missing id, got nil")
	}
}

func TestGetServiceByID_EmptyStore(t *testing.T) {
	store := newTestStore()
	_, err := store.GetServiceByID(1)
	if err == nil {
		t.Error("expected error for empty store, got nil")
	}
}

// ---- EventUpdate ----

func TestEventUpdate_UpdatesFields(t *testing.T) {
	store := newTestStore(newStoreService(1, "Alpha"))

	ts := time.Now()
	err := store.EventUpdate(EventData{ServiceID: 1, Status: true, Timestamp: ts})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s, _ := store.GetServiceByID(1)
	if !s.Status {
		t.Error("expected status=true after update")
	}
	if !s.LastUpdate.Equal(ts) {
		t.Errorf("expected LastUpdate=%v, got %v", ts, s.LastUpdate)
	}
}

func TestEventUpdate_PushesHistory(t *testing.T) {
	store := newTestStore(newStoreService(1, "Alpha"))

	store.EventUpdate(EventData{ServiceID: 1, Status: true, Timestamp: time.Now()})
	store.EventUpdate(EventData{ServiceID: 1, Status: false, Timestamp: time.Now()})

	s, _ := store.GetServiceByID(1)
	if s.History.GetSize() != 2 {
		t.Errorf("expected 2 history entries, got %d", s.History.GetSize())
	}
}

func TestEventUpdate_RecalculatesUptime(t *testing.T) {
	store := newTestStore(newStoreService(1, "Alpha"))

	// 3 up, 1 down → 75% uptime
	for range 3 {
		store.EventUpdate(EventData{ServiceID: 1, Status: true, Timestamp: time.Now()})
	}
	store.EventUpdate(EventData{ServiceID: 1, Status: false, Timestamp: time.Now()})

	s, _ := store.GetServiceByID(1)
	if s.Uptime != 75.0 {
		t.Errorf("expected uptime=75.0, got %.2f", s.Uptime)
	}
}

func TestEventUpdate_UnknownService(t *testing.T) {
	store := newTestStore(newStoreService(1, "Alpha"))

	err := store.EventUpdate(EventData{ServiceID: 99, Status: true, Timestamp: time.Now()})
	if err == nil {
		t.Error("expected error for unknown service id, got nil")
	}
}

// ---- ReconcileServices ----

func TestReconcile_AddsNewServices(t *testing.T) {
	store := newTestStore()
	store.ReconcileServices([]Service{
		newStoreService(1, "Alpha"),
		newStoreService(2, "Beta"),
	})

	if len(*store.GetServices()) != 2 {
		t.Errorf("expected 2 services after reconcile, got %d", len(*store.GetServices()))
	}
}

func TestReconcile_RemovesStaleServices(t *testing.T) {
	store := newTestStore(
		newStoreService(1, "Alpha"),
		newStoreService(2, "Beta"),
	)
	// Config only has Alpha now
	store.ReconcileServices([]Service{newStoreService(1, "Alpha")})

	services := *store.GetServices()
	if len(services) != 1 {
		t.Fatalf("expected 1 service after reconcile, got %d", len(services))
	}
	if services[0].ID != 1 {
		t.Errorf("expected service id=1 to survive, got id=%d", services[0].ID)
	}
}

func TestReconcile_UpdatesConfigFields(t *testing.T) {
	store := newTestStore(newStoreService(1, "OldName"))

	updated := newStoreService(1, "NewName")
	updated.Address = "newhost:9090"
	updated.Timer = 60
	store.ReconcileServices([]Service{updated})

	s, _ := store.GetServiceByID(1)
	if s.Name != "NewName" {
		t.Errorf("expected Name=NewName, got %q", s.Name)
	}
	if s.Address != "newhost:9090" {
		t.Errorf("expected Address=newhost:9090, got %q", s.Address)
	}
	if s.Timer != 60 {
		t.Errorf("expected Timer=60, got %d", s.Timer)
	}
}

func TestReconcile_PreservesRuntimeState(t *testing.T) {
	store := newTestStore(newStoreService(1, "Alpha"))

	// Simulate some runtime history
	ts := time.Now()
	store.EventUpdate(EventData{ServiceID: 1, Status: true, Timestamp: ts})
	store.EventUpdate(EventData{ServiceID: 1, Status: true, Timestamp: ts.Add(time.Second)})

	historyBefore := *store.GetServices()
	sizeBefore := historyBefore[0].History.GetSize()

	// Reconcile with an updated name — runtime state must survive
	updated := newStoreService(1, "Alpha Renamed")
	store.ReconcileServices([]Service{updated})

	s, _ := store.GetServiceByID(1)
	if s.Name != "Alpha Renamed" {
		t.Errorf("expected Name updated, got %q", s.Name)
	}
	if s.History.GetSize() != sizeBefore {
		t.Errorf("expected history size=%d to be preserved, got %d", sizeBefore, s.History.GetSize())
	}
	if s.Uptime == 0 {
		t.Error("expected uptime to be preserved after reconcile")
	}
}

func TestReconcile_RenumberedID(t *testing.T) {
	// Service previously had id=1, now the config uses id=2
	store := newTestStore(newStoreService(1, "Alpha"))
	store.EventUpdate(EventData{ServiceID: 1, Status: true, Timestamp: time.Now()})

	store.ReconcileServices([]Service{newStoreService(2, "Alpha")})

	services := *store.GetServices()
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].ID != 2 {
		t.Errorf("expected new id=2, got %d", services[0].ID)
	}
	// Fresh service — no history carried over from old id
	if services[0].History.GetSize() != 0 {
		t.Error("expected fresh history for renumbered service")
	}
}

func TestReconcile_EmptyConfig_RemovesAll(t *testing.T) {
	store := newTestStore(
		newStoreService(1, "Alpha"),
		newStoreService(2, "Beta"),
	)
	store.ReconcileServices([]Service{})

	if len(*store.GetServices()) != 0 {
		t.Errorf("expected empty store after reconcile with no config, got %d", len(*store.GetServices()))
	}
}

func TestReconcile_NoOp_WhenUnchanged(t *testing.T) {
	svc := newStoreService(1, "Alpha")
	store := newTestStore(svc)
	store.EventUpdate(EventData{ServiceID: 1, Status: true, Timestamp: time.Now()})

	store.ReconcileServices([]Service{svc})

	services := *store.GetServices()
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].History.GetSize() != 1 {
		t.Error("expected history preserved on no-op reconcile")
	}
}

// ---- Save / Load ----

func TestSaveLoad_RoundTrip(t *testing.T) {
	cdTemp(t)

	original := newTestStore(
		newStoreService(1, "Alpha"),
		newStoreService(2, "Beta"),
	)
	if err := original.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, _ := NewInMemoryStore(false) // noHistory=false → loads from disk
	services := *loaded.GetServices()

	if len(services) != 2 {
		t.Fatalf("expected 2 services after load, got %d", len(services))
	}

	ids := map[uint]string{1: "Alpha", 2: "Beta"}
	for _, s := range services {
		expectedName, ok := ids[s.ID]
		if !ok {
			t.Errorf("unexpected service id=%d after load", s.ID)
		} else if s.Name != expectedName {
			t.Errorf("id=%d: expected name=%q, got %q", s.ID, expectedName, s.Name)
		}
		if s.protocol == nil {
			t.Errorf("id=%d: protocol not restored after load", s.ID)
		}
	}
}

func TestSaveLoad_PreservesHistory(t *testing.T) {
	cdTemp(t)

	store := newTestStore(newStoreService(1, "Alpha"))
	store.EventUpdate(EventData{ServiceID: 1, Status: true, Timestamp: time.Now()})
	store.EventUpdate(EventData{ServiceID: 1, Status: false, Timestamp: time.Now()})

	if err := store.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, _ := NewInMemoryStore(false)
	s, err := loaded.GetServiceByID(1)
	if err != nil {
		t.Fatalf("service not found after load: %v", err)
	}
	if s.History.GetSize() != 2 {
		t.Errorf("expected 2 history entries after load, got %d", s.History.GetSize())
	}
}

func TestLoad_NoHistory_IgnoresBin(t *testing.T) {
	cdTemp(t)

	// Write a bin file, then load with noHistory=true
	original := newTestStore(newStoreService(1, "Alpha"))
	original.Save()

	store, _ := NewInMemoryStore(true) // should ignore .sleuth.bin
	if len(*store.GetServices()) != 0 {
		t.Error("expected empty store when noHistory=true, even if .sleuth.bin exists")
	}
}

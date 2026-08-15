package fileio

import "testing"

func TestGroupEventsByTurn(t *testing.T) {
	events := []Civ5ReplayEvent{
		{Turn: 1, TypeId: 0, Text: "a"},
		{Turn: 2, TypeId: 1, Text: "b"},
		{Turn: 1, TypeId: 2, Text: "c"},
	}

	grouped := GroupEventsByTurn(events)

	if len(grouped) != 2 {
		t.Fatalf("GroupEventsByTurn() returned %d turns, want 2", len(grouped))
	}
	if len(grouped[1]) != 2 {
		t.Errorf("turn 1 has %d events, want 2", len(grouped[1]))
	}
	if len(grouped[2]) != 1 {
		t.Errorf("turn 2 has %d events, want 1", len(grouped[2]))
	}
	if grouped[1][0].Text != "a" || grouped[1][1].Text != "c" {
		t.Errorf("turn 1 events out of order: %+v", grouped[1])
	}
}

func TestGroupEventsByTurnEmpty(t *testing.T) {
	grouped := GroupEventsByTurn(nil)
	if len(grouped) != 0 {
		t.Errorf("GroupEventsByTurn(nil) = %v, want empty map", grouped)
	}
}

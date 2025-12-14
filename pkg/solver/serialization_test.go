package solver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/behrlich/poker-solver/pkg/notation"
)

func TestStrategyProfile_ToJSON(t *testing.T) {
	// Create a simple strategy profile
	sp := NewStrategyProfile()

	actions := []notation.Action{
		{Type: notation.Check, Amount: 0},
		{Type: notation.Bet, Amount: 5.0},
	}

	strat := sp.GetOrCreate("Kh9s4c||>BTN|AsAd", actions)
	strat.RegretSum = []float64{1.5, -0.5}
	strat.StrategySum = []float64{10.0, 5.0}

	// Serialize to JSON
	data, err := sp.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Check that it's valid JSON
	if len(data) == 0 {
		t.Error("JSON output is empty")
	}

	t.Logf("Serialized JSON:\n%s", string(data))
}

func TestStrategyProfile_RoundTrip(t *testing.T) {
	// Create original strategy profile
	original := NewStrategyProfile()

	// Add multiple strategies
	actions1 := []notation.Action{
		{Type: notation.Check, Amount: 0},
		{Type: notation.Bet, Amount: 5.0},
	}
	strat1 := original.GetOrCreate("Kh9s4c||>BTN|AsAd", actions1)
	strat1.RegretSum = []float64{1.5, -0.5}
	strat1.StrategySum = []float64{10.0, 5.0}

	actions2 := []notation.Action{
		{Type: notation.Fold, Amount: 0},
		{Type: notation.Call, Amount: 0},
		{Type: notation.Raise, Amount: 15.0},
	}
	strat2 := original.GetOrCreate("Kh9s4c|b5.0|>BB|QdQh", actions2)
	strat2.RegretSum = []float64{-2.0, 3.0, 1.0}
	strat2.StrategySum = []float64{2.0, 8.0, 10.0}

	// Serialize
	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Deserialize
	restored, err := FromJSON(data)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Verify number of strategies
	if restored.NumInfoSets() != original.NumInfoSets() {
		t.Errorf("Expected %d infosets, got %d", original.NumInfoSets(), restored.NumInfoSets())
	}

	// Verify first strategy
	restoredStrat1, exists := restored.Get("Kh9s4c||>BTN|AsAd")
	if !exists {
		t.Fatal("First strategy not found after deserialization")
	}

	if len(restoredStrat1.Actions) != len(strat1.Actions) {
		t.Errorf("Expected %d actions, got %d", len(strat1.Actions), len(restoredStrat1.Actions))
	}

	for i := range strat1.Actions {
		if restoredStrat1.Actions[i].Type != strat1.Actions[i].Type {
			t.Errorf("Action %d type mismatch: expected %v, got %v",
				i, strat1.Actions[i].Type, restoredStrat1.Actions[i].Type)
		}
		if restoredStrat1.Actions[i].Amount != strat1.Actions[i].Amount {
			t.Errorf("Action %d amount mismatch: expected %.1f, got %.1f",
				i, strat1.Actions[i].Amount, restoredStrat1.Actions[i].Amount)
		}
	}

	for i := range strat1.RegretSum {
		if restoredStrat1.RegretSum[i] != strat1.RegretSum[i] {
			t.Errorf("RegretSum[%d] mismatch: expected %.2f, got %.2f",
				i, strat1.RegretSum[i], restoredStrat1.RegretSum[i])
		}
	}

	for i := range strat1.StrategySum {
		if restoredStrat1.StrategySum[i] != strat1.StrategySum[i] {
			t.Errorf("StrategySum[%d] mismatch: expected %.2f, got %.2f",
				i, strat1.StrategySum[i], restoredStrat1.StrategySum[i])
		}
	}

	// Verify second strategy
	restoredStrat2, exists := restored.Get("Kh9s4c|b5.0|>BB|QdQh")
	if !exists {
		t.Fatal("Second strategy not found after deserialization")
	}

	if len(restoredStrat2.Actions) != 3 {
		t.Errorf("Expected 3 actions in second strategy, got %d", len(restoredStrat2.Actions))
	}
}

func TestStrategyProfile_SaveAndLoad(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "strategy.json")

	// Create and save strategy profile
	original := NewStrategyProfile()
	actions := []notation.Action{
		{Type: notation.Check, Amount: 0},
		{Type: notation.Bet, Amount: 10.0},
		{Type: notation.Bet, Amount: 20.0},
	}
	strat := original.GetOrCreate("test-infoset", actions)
	strat.RegretSum = []float64{5.0, -2.0, 3.0}
	strat.StrategySum = []float64{100.0, 50.0, 150.0}

	// Save to file
	err := original.SaveToFile(filename)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// Load from file
	restored, err := LoadFromFile(filename)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify
	if restored.NumInfoSets() != 1 {
		t.Errorf("Expected 1 infoset, got %d", restored.NumInfoSets())
	}

	restoredStrat, exists := restored.Get("test-infoset")
	if !exists {
		t.Fatal("Strategy not found after load")
	}

	// Check average strategies match
	originalAvg := strat.GetAverageStrategy()
	restoredAvg := restoredStrat.GetAverageStrategy()

	for i := range originalAvg {
		diff := originalAvg[i] - restoredAvg[i]
		if diff < -0.0001 || diff > 0.0001 {
			t.Errorf("Average strategy mismatch at action %d: expected %.4f, got %.4f",
				i, originalAvg[i], restoredAvg[i])
		}
	}
}

func TestLoadFromFile_NonExistent(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/to/file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}

func TestFromJSON_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{"invalid": json}`)
	_, err := FromJSON(invalidJSON)
	if err == nil {
		t.Error("Expected error when deserializing invalid JSON")
	}
}

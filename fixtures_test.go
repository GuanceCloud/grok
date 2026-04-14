package grok

import (
	"encoding/json"
	"os"
	"testing"
)

type datakitFixtureCase struct {
	Collector      string                       `json:"collector"`
	PackageDir     string                       `json:"package_dir"`
	PipelineSource string                       `json:"pipeline_source"`
	ExampleSource  string                       `json:"example_source"`
	Pipelines      map[string]string            `json:"pipelines"`
	Examples       map[string]map[string]string `json:"examples"`
}

func loadDatakitFixtureCases(t testing.TB) []datakitFixtureCase {
	t.Helper()

	raw, err := os.ReadFile("testdata/datakit_pipeline_cases.json")
	if err != nil {
		t.Fatal(err)
	}

	var cases []datakitFixtureCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

func TestDatakitFixtureCases(t *testing.T) {
	cases := loadDatakitFixtureCases(t)
	if len(cases) == 0 {
		t.Fatal("expected datakit fixture cases")
	}

	for _, c := range cases {
		if c.Collector == "" {
			t.Fatal("fixture collector is empty")
		}
		if c.PackageDir == "" || c.PipelineSource == "" || c.ExampleSource == "" {
			t.Fatalf("fixture %s missing source metadata", c.Collector)
		}
		if len(c.Pipelines) == 0 {
			t.Fatalf("fixture %s missing pipelines", c.Collector)
		}
		if len(c.Examples) == 0 {
			t.Fatalf("fixture %s missing examples", c.Collector)
		}
	}
}

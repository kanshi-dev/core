package service

import (
	"bytes"
	"errors"
	"testing"

	pprof "github.com/google/pprof/profile"
)

func TestBuildFlamegraph(t *testing.T) {
	rootFn := &pprof.Function{ID: 1, Name: "main.work"}
	leafFn := &pprof.Function{ID: 2, Name: "main.hash"}
	rootLoc := &pprof.Location{ID: 1, Line: []pprof.Line{{Function: rootFn}}}
	leafLoc := &pprof.Location{ID: 2, Line: []pprof.Line{{Function: leafFn}}}
	p := &pprof.Profile{
		SampleType: []*pprof.ValueType{{Type: "samples", Unit: "count"}, {Type: "cpu", Unit: "nanoseconds"}},
		Sample: []*pprof.Sample{
			{Location: []*pprof.Location{leafLoc, rootLoc}, Value: []int64{2, 20}},
			{Location: []*pprof.Location{rootLoc}, Value: []int64{1, 10}},
		},
	}

	graph, err := buildFlamegraph(p, "cpu")
	if err != nil {
		t.Fatal(err)
	}
	if graph.SampleType != "cpu" || graph.Unit != "nanoseconds" || graph.Total != 30 {
		t.Fatalf("unexpected graph metadata: %+v", graph)
	}
	if len(graph.Root.Children) != 1 || graph.Root.Children[0].Name != "main.work" || graph.Root.Children[0].Value != 30 {
		t.Fatalf("unexpected root: %+v", graph.Root)
	}
	if got := graph.Root.Children[0].Children; len(got) != 1 || got[0].Name != "main.hash" || got[0].Value != 20 {
		t.Fatalf("unexpected leaf: %+v", got)
	}
	if _, err := buildFlamegraph(p, "missing"); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("unknown sample type error = %v", err)
	}
}

func TestParseCompressedFlamegraph(t *testing.T) {
	p := &pprof.Profile{SampleType: []*pprof.ValueType{{Type: "samples", Unit: "count"}}}
	var data bytes.Buffer
	if err := p.Write(&data); err != nil {
		t.Fatal(err)
	}
	graph, err := parseFlamegraph(data.Bytes(), "")
	if err != nil {
		t.Fatal(err)
	}
	if graph.SampleType != "samples" || graph.Total != 0 || len(graph.Root.Children) != 0 {
		t.Fatalf("unexpected empty graph: %+v", graph)
	}
	if _, err := parseFlamegraph([]byte("not pprof"), ""); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("malformed profile error = %v", err)
	}
}

func TestBuildFlamegraphNodeLimit(t *testing.T) {
	locations := make([]*pprof.Location, maxFlamegraphNodes)
	for i := range locations {
		locations[i] = &pprof.Location{Address: uint64(i + 1)}
	}
	p := &pprof.Profile{SampleType: []*pprof.ValueType{{Type: "samples", Unit: "count"}}, Sample: []*pprof.Sample{{Location: locations, Value: []int64{1}}}}
	if _, err := buildFlamegraph(p, ""); !errors.Is(err, ErrProfileTooLarge) {
		t.Fatalf("node limit error = %v", err)
	}
}

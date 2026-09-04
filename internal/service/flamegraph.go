package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	pprof "github.com/google/pprof/profile"
)

const maxFlamegraphNodes = 20_000

var ErrProfileTooLarge = errors.New("profile exceeds 20000 flamegraph nodes")

type Flamegraph struct {
	SampleType string         `json:"sampleType"`
	Unit       string         `json:"unit"`
	Total      int64          `json:"total"`
	Root       FlamegraphNode `json:"root"`
}

type FlamegraphNode struct {
	Name     string           `json:"name"`
	Value    int64            `json:"value"`
	Children []FlamegraphNode `json:"children"`
}

type flameNode struct {
	name     string
	value    int64
	children map[string]*flameNode
}

func (s *ProfilesService) Flamegraph(ctx context.Context, id, sampleType string) (Flamegraph, error) {
	row, err := s.get(ctx, id)
	if err != nil {
		return Flamegraph{}, err
	}
	if row.ProfileType == "trace" || row.State != "completed" || len(row.Artifact) == 0 {
		return Flamegraph{}, fmt.Errorf("%w: capture is not a completed pprof profile", ErrInvalidProfile)
	}
	return parseFlamegraph(row.Artifact, sampleType)
}

func parseFlamegraph(data []byte, sampleType string) (Flamegraph, error) {
	p, err := pprof.ParseData(data)
	if err != nil {
		return Flamegraph{}, fmt.Errorf("%w: malformed pprof data", ErrInvalidProfile)
	}
	return buildFlamegraph(p, sampleType)
}

func buildFlamegraph(p *pprof.Profile, requested string) (Flamegraph, error) {
	if len(p.SampleType) == 0 {
		return Flamegraph{}, fmt.Errorf("%w: profile has no sample types", ErrInvalidProfile)
	}
	index := 0
	if requested != "" {
		index = -1
		for i, sample := range p.SampleType {
			if sample.Type == requested {
				index = i
				break
			}
		}
		if index < 0 {
			return Flamegraph{}, fmt.Errorf("%w: unknown sample type %q", ErrInvalidProfile, requested)
		}
	}

	root := &flameNode{name: "root", children: map[string]*flameNode{}}
	nodes := 1
	for _, sample := range p.Sample {
		if index >= len(sample.Value) {
			return Flamegraph{}, fmt.Errorf("%w: sample value is missing", ErrInvalidProfile)
		}
		value := sample.Value[index]
		root.value += value
		parent := root
		for i := len(sample.Location) - 1; i >= 0; i-- {
			name := locationName(sample.Location[i])
			child := parent.children[name]
			if child == nil {
				nodes++
				if nodes > maxFlamegraphNodes {
					return Flamegraph{}, ErrProfileTooLarge
				}
				child = &flameNode{name: name, children: map[string]*flameNode{}}
				parent.children[name] = child
			}
			child.value += value
			parent = child
		}
	}

	typ := p.SampleType[index]
	return Flamegraph{SampleType: typ.Type, Unit: typ.Unit, Total: root.value, Root: exportFlameNode(root)}, nil
}

func locationName(location *pprof.Location) string {
	for _, line := range location.Line {
		if line.Function != nil && line.Function.Name != "" {
			return line.Function.Name
		}
	}
	return fmt.Sprintf("0x%x", location.Address)
}

func exportFlameNode(node *flameNode) FlamegraphNode {
	out := FlamegraphNode{Name: node.name, Value: node.value, Children: make([]FlamegraphNode, 0, len(node.children))}
	for _, child := range node.children {
		out.Children = append(out.Children, exportFlameNode(child))
	}
	sort.Slice(out.Children, func(i, j int) bool {
		if out.Children[i].Value == out.Children[j].Value {
			return out.Children[i].Name < out.Children[j].Name
		}
		return out.Children[i].Value > out.Children[j].Value
	})
	return out
}

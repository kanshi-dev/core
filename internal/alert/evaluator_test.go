package alert

import "testing"

func TestDecide(t *testing.T) {
	cases := []struct {
		name        string
		priorFiring bool
		breaching   bool
		want        transition
	}{
		{"quiet stays quiet", false, false, noop},
		{"breach fires once", false, true, fire},
		{"sustained breach does not duplicate", true, true, noop},
		{"recovery resolves once", true, false, resolve},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := decide(c.priorFiring, c.breaching); got != c.want {
				t.Fatalf("decide(%v, %v) = %v, want %v", c.priorFiring, c.breaching, got, c.want)
			}
		})
	}
}

func TestBreach(t *testing.T) {
	cases := []struct {
		comparator    string
		value, thresh float64
		want          bool
	}{
		{"gt", 91, 90, true},
		{"gt", 90, 90, false},
		{"gt", 50, 90, false},
		{"lt", 5, 10, true},
		{"lt", 10, 10, false},
		{"lt", 50, 10, false},
	}
	for _, c := range cases {
		if got := breach(c.comparator, c.value, c.thresh); got != c.want {
			t.Errorf("breach(%q, %v, %v) = %v, want %v", c.comparator, c.value, c.thresh, got, c.want)
		}
	}
}

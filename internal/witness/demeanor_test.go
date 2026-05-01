package witness_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jerkeyray/hearsay/internal/kase"
	"github.com/jerkeyray/hearsay/internal/witness"
)

func TestParseDemeanor(t *testing.T) {
	for _, d := range kase.AllDemeanors {
		got, ok := kase.ParseDemeanor(string(d))
		if !ok || got != d {
			t.Errorf("ParseDemeanor(%q) = (%q,%v), want (%q,true)", d, got, ok, d)
		}
	}
	if _, ok := kase.ParseDemeanor("excited"); ok {
		t.Errorf("ParseDemeanor accepted unknown state")
	}
}

func TestDemeanorTool_FiresSetter(t *testing.T) {
	var got kase.Demeanor
	tt := witness.DemeanorTool(func(d kase.Demeanor) { got = d })

	in := witness.DemeanorInput{State: string(kase.DemeanorDefensive)}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out, err := tt.Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got != kase.DemeanorDefensive {
		t.Errorf("setter got %q, want %q", got, kase.DemeanorDefensive)
	}
	var ack witness.DemeanorOutput
	if err := json.Unmarshal(out, &ack); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !ack.Ack {
		t.Error("expected ack=true")
	}
}

func TestDemeanorTool_RejectsUnknownState(t *testing.T) {
	tt := witness.DemeanorTool(func(kase.Demeanor) {})
	raw, _ := json.Marshal(witness.DemeanorInput{State: "ecstatic"})
	if _, err := tt.Execute(context.Background(), raw); err == nil {
		t.Error("expected error for unknown demeanor")
	}
}

func TestStubDriver_PopulatesDemeanor(t *testing.T) {
	d := witness.NewStubDriver()
	cases := []struct {
		tech kase.Technique
		want kase.Demeanor
	}{
		{kase.Directly, kase.DemeanorEngaged},
		{kase.MomentBefore, kase.DemeanorUncomfortable},
		{kase.PushBack, kase.DemeanorDefensive},
	}
	for _, tc := range cases {
		resp, err := d.Respond(context.Background(), "the bag", tc.tech, nil)
		if err != nil {
			t.Fatalf("Respond %s: %v", tc.tech.Label(), err)
		}
		if resp.Demeanor != tc.want {
			t.Errorf("technique %s: demeanor = %q, want %q", tc.tech.Label(), resp.Demeanor, tc.want)
		}
	}
}

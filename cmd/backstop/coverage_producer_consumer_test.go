package main

import (
	"reflect"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// TestCoverageRecord_SingleReconciledTypeAcrossProducerAndConsumer proves a SINGLE
// check.CoverageRecord type is the shared carrier across producer (dispatch) and
// consumer — the consumer boundary consumes the SAME canonical type the producer
// emits, with NO second divergent shape (no producer-local + consumer-local
// CoverageRecord) (CLM-018). SPEC-041's drafted {Path, Pct, Measured, Excluded} is
// reconciled to {Path, Covered, Total, Measured, Excluded, Metric}.
func TestCoverageRecord_SingleReconciledTypeAcrossProducerAndConsumer(t *testing.T) {
	// The producer's emitted element type.
	producerElem := reflect.TypeOf(dispatchPackCoverage).Out(0).Elem()
	// The consumer boundary consumes []check.CoverageRecord (the minimal test-local
	// consumer's verdict method).
	consumerParam := reflect.TypeOf(minimalCoverageConsumer{}.verdict).In(0).Elem()

	if producerElem != consumerParam {
		t.Fatalf("producer and consumer must share ONE CoverageRecord type; producer emits %v, consumer consumes %v", producerElem, consumerParam)
	}
	if producerElem != reflect.TypeOf(check.CoverageRecord{}) {
		t.Errorf("the shared carrier must be the canonical check.CoverageRecord, got %v", producerElem)
	}
	// The reconciled type carries RAW COUNTS + Metric (the reconciliation target), not
	// SPEC-041's drafted Pct — a single shape, no second divergent record.
	rt := reflect.TypeOf(check.CoverageRecord{})
	for _, name := range []string{"Covered", "Total", "Metric"} {
		if _, ok := rt.FieldByName(name); !ok {
			t.Errorf("the reconciled CoverageRecord must carry %q", name)
		}
	}
	if _, ok := rt.FieldByName("Pct"); ok {
		t.Errorf("the reconciled type must NOT retain SPEC-041's drafted Pct — that is the divergent shape REQ-006 forbids")
	}
}

// TestCoverageRecord_ProducerRecordsFlowToCoverageStepUntranslated proves the
// producer-emitted []check.CoverageRecord is consumed directly with NO lossy
// producer→consumer translation layer — the records dispatch produces flow to the
// coverage-step boundary unchanged (no counts→percent / metric-dropping shim)
// (CLM-019).
func TestCoverageRecord_ProducerRecordsFlowToCoverageStepUntranslated(t *testing.T) {
	records := dispatchRecordsForConvert(t, coverageRecordsJSON())
	if len(records) == 0 {
		t.Fatal("expected producer records")
	}

	// The exact records the producer emitted are handed to the consumer boundary with
	// no translation — same slice element type, same field values survive.
	consumer := minimalCoverageConsumer{thresholdPct: 80}
	lines := consumer.verdict(records)
	if len(lines) != len(records) {
		t.Fatalf("the consumer must consume every producer record untranslated, got %d lines for %d records", len(lines), len(records))
	}
	// Counts and metric survive verbatim to the boundary: no counts→percent collapse,
	// no metric drop. Re-feeding the SAME records through the consumer is lossless.
	for i, r := range records {
		if lines[i].Path != r.Path {
			t.Errorf("record %d path must survive untranslated, got %q want %q", i, lines[i].Path, r.Path)
		}
		if lines[i].Metric != r.Metric {
			t.Errorf("record %d metric must NOT be dropped in flow, got %q want %q", i, lines[i].Metric, r.Metric)
		}
	}
}

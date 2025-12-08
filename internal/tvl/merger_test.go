package tvl

import (
	"reflect"
	"sort"
	"testing"

	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

func TestMergeProtocolLists_AutoOnly(t *testing.T) {
	auto := []string{"alpha", "beta"}
	got := MergeProtocolLists(auto, nil)

	if got == nil {
		t.Fatalf("expected non-nil slice")
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 protocols, got %d", len(got))
	}

	for _, p := range got {
		if p.Source != "auto" || p.SimpleTVSRatio != 1.0 || p.IsOngoing || p.IntegrationDate != nil {
			t.Fatalf("auto defaults not applied: %+v", p)
		}
		if p.DocsProof == nil || *p.DocsProof != autoDocsProof(p.Slug) {
			t.Fatalf("docs proof not generated for %s", p.Slug)
		}
	}
}

func TestMergeProtocolLists_CustomOnly(t *testing.T) {
	date := int64(1700000000)
	custom := []models.CustomProtocol{{
		Slug:           "gamma",
		IsOngoing:      false,
		Live:           true,
		Date:           &date,
		SimpleTVSRatio: 0.5,
	}}

	got := MergeProtocolLists(nil, custom)

	if len(got) != 1 {
		t.Fatalf("expected 1 protocol, got %d", len(got))
	}

	p := got[0]
	if p.Source != "custom" || p.IsOngoing != false || p.SimpleTVSRatio != 0.5 {
		t.Fatalf("custom metadata not preserved: %+v", p)
	}
	if p.IntegrationDate == nil || *p.IntegrationDate != date {
		t.Fatalf("integration_date not preserved: %+v", p)
	}
}

func TestMergeProtocolLists_CustomNilDateStaysNil(t *testing.T) {
	custom := []models.CustomProtocol{{
		Slug:           "nil-date",
		IsOngoing:      true,
		Live:           true,
		SimpleTVSRatio: 0.9,
		Date:           nil,
	}}

	got := MergeProtocolLists(nil, custom)

	if len(got) != 1 {
		t.Fatalf("expected 1 protocol, got %d", len(got))
	}

	p := got[0]
	if p.IntegrationDate != nil {
		t.Fatalf("expected IntegrationDate nil, got %v", p.IntegrationDate)
	}
	if p.SimpleTVSRatio != 0.9 || p.Source != "custom" {
		t.Fatalf("unexpected metadata: %+v", p)
	}
}

func TestMergeProtocolLists_CustomOverridesAuto(t *testing.T) {
	auto := []string{"dup", "unique"}
	docs := "https://docs/dup"
	custom := []models.CustomProtocol{{
		Slug:           "dup",
		IsOngoing:      false,
		Live:           true,
		SimpleTVSRatio: 0.25,
		DocsProof:      &docs,
	}}

	got := MergeProtocolLists(auto, custom)

	if len(got) != 2 {
		t.Fatalf("expected 2 protocols after merge, got %d", len(got))
	}

	merged := findBySlug(t, got, "dup")
	if merged.Source != "custom" || merged.SimpleTVSRatio != 0.25 || merged.DocsProof == nil || *merged.DocsProof != docs {
		t.Fatalf("custom entry did not overwrite auto: %+v", merged)
	}

	unique := findBySlug(t, got, "unique")
	if unique.Source != "auto" {
		t.Fatalf("auto entry altered unexpectedly: %+v", unique)
	}
}

func TestMergeProtocolLists_Sorting(t *testing.T) {
	auto := []string{"delta", "alpha"}
	custom := []models.CustomProtocol{{
		Slug:           "charlie",
		IsOngoing:      true,
		Live:           true,
		SimpleTVSRatio: 1,
	}}

	got := MergeProtocolLists(auto, custom)

	slugs := []string{got[0].Slug, got[1].Slug, got[2].Slug}
	expected := []string{"alpha", "charlie", "delta"}
	if !reflect.DeepEqual(slugs, expected) {
		t.Fatalf("slice not sorted: got %v, want %v", slugs, expected)
	}
}

func TestMergeProtocolLists_EmptyInputs(t *testing.T) {
	got := MergeProtocolLists(nil, nil)

	if got == nil {
		t.Fatalf("expected empty slice, got nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected zero length slice, got %d", len(got))
	}
}

func TestMergeProtocolLists_NilDocsProofForCustom(t *testing.T) {
	custom := []models.CustomProtocol{{
		Slug:           "omega",
		IsOngoing:      true,
		Live:           true,
		SimpleTVSRatio: 0.8,
	}}

	got := MergeProtocolLists(nil, custom)

	p := got[0]
	if p.DocsProof != nil {
		t.Fatalf("expected custom docs proof nil, got %v", p.DocsProof)
	}
}

func findBySlug(t *testing.T, protocols []models.MergedProtocol, slug string) models.MergedProtocol {
	t.Helper()
	sort.SliceStable(protocols, func(i, j int) bool { return protocols[i].Slug < protocols[j].Slug })
	for _, p := range protocols {
		if p.Slug == slug {
			return p
		}
	}
	t.Fatalf("protocol %s not found", slug)
	return models.MergedProtocol{}
}

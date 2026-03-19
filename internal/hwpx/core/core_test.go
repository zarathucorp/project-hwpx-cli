package core

import (
	"slices"
	"testing"
)

func TestInspectEntriesAddsRiskHintsForComplexDocuments(t *testing.T) {
	entries := map[string][]byte{
		"mimetype":              []byte("application/hwp+zip"),
		"version.xml":           []byte(`<version appVersion="1" hwpxVersion="1"/>`),
		"Contents/header.xml":   []byte(`<hh:head xmlns:hh="http://www.hancom.co.kr/hwpml/2011/head" secCnt="2"><hh:styles>TOC Heading toc 1</hh:styles></hh:head>`),
		"Contents/content.hpf":  []byte(`<package><metadata><title>Complex</title></metadata><manifest><item id="header" href="Contents/header.xml" media-type="application/xml"></item><item id="section0" href="Contents/section0.xml" media-type="application/xml"></item><item id="section1" href="Contents/section1.xml" media-type="application/xml"></item></manifest><spine><itemref idref="header"></itemref><itemref idref="section0"></itemref><itemref idref="section1"></itemref></spine></package>`),
		"Contents/section0.xml": []byte(`<hs:sec xmlns:hs="http://www.hancom.co.kr/hwpml/2011/section" xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"><hp:p><hp:run><hp:header></hp:header><hp:pageNum></hp:pageNum><hp:tbl><hp:tr><hp:tc><hp:cellSpan rowSpan="2" colSpan="1"></hp:cellSpan></hp:tc></hp:tr></hp:tbl><hp:t>목차</hp:t></hp:run></hp:p></hs:sec>`),
		"Contents/section1.xml": []byte(`<hs:sec xmlns:hs="http://www.hancom.co.kr/hwpml/2011/section" xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"><hp:p><hp:run><hp:rect></hp:rect></hp:run></hp:p></hs:sec>`),
	}

	report, err := inspectEntries(entries)
	if err != nil {
		t.Fatalf("inspect entries: %v", err)
	}
	if !report.Valid {
		t.Fatalf("expected valid report: %+v", report.Errors)
	}
	if report.RenderSafe {
		t.Fatalf("complex report should not be render-safe: %+v", report)
	}

	for _, risk := range []string{"section-risk", "toc-risk", "table-risk", "layout-risk"} {
		if !slices.Contains(report.RiskHints, risk) {
			t.Fatalf("expected risk %q in %+v", risk, report.RiskHints)
		}
	}
	if report.RiskSignals["sectionCount"] != 2 {
		t.Fatalf("expected section count signal, got %+v", report.RiskSignals)
	}
	if report.RiskSignals["mergedCellCount"] == 0 {
		t.Fatalf("expected merged cell signal, got %+v", report.RiskSignals)
	}
}

package appcore

import (
	"strings"
	"testing"

	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

func TestAcceptExtensionProjectsRejectsInvalidPayloads(t *testing.T) {
	events := make(chan Event, 1)
	svc := New(Options{OnEvent: func(event Event) { events <- event }})

	results := svc.AcceptExtensionProjects([]appstructs.ExtensionProjectPayload{
		{PlatformCode: "xx", Slug: "bad"},
		{PlatformCode: "mr", Slug: "   "},
	})
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	for i, result := range results {
		if !result.Skipped {
			t.Fatalf("result[%d] skipped = false, want true", i)
		}
		if result.Reason == "" {
			t.Fatalf("result[%d] has no reason", i)
		}
	}
	if !strings.Contains(results[0].Reason, "invalid platform") {
		t.Fatalf("result[0] reason = %q", results[0].Reason)
	}
	if !strings.Contains(results[1].Reason, "empty project id and slug") {
		t.Fatalf("result[1] reason = %q", results[1].Reason)
	}
	select {
	case event := <-events:
		t.Fatalf("unexpected event for rejected payloads: %#v", event)
	default:
	}
}

func TestAcceptExtensionProjectsReturnsStableResultCount(t *testing.T) {
	svc := New(Options{})
	if results := svc.AcceptExtensionProjects(nil); len(results) != 0 {
		t.Fatalf("nil payload results = %d, want 0", len(results))
	}
}

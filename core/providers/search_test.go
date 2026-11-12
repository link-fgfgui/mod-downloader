// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package providers

import (
	"errors"
	"strings"
	"testing"

	"github.com/link-fgfgui/mod-downloader/core/models"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

type fakeSearchProvider struct {
	name      string
	exactErr  error
	searchErr error
}

func (p fakeSearchProvider) Name() string { return p.name }

func (p fakeSearchProvider) ExactSearch(appstructs.SearchModsRequest) ([]models.ModProject, error) {
	return nil, p.exactErr
}

func (p fakeSearchProvider) Search(appstructs.SearchModsRequest) ([]models.ModProject, error) {
	return nil, p.searchErr
}

func (p fakeSearchProvider) ListVersions(string, projectVersionFilter) ([]models.ModVersion, error) {
	return nil, nil
}

func TestSearchModsAggregatesProviderErrorsOnFinalUpdate(t *testing.T) {
	sources := []modProvider{
		fakeSearchProvider{name: "broken", exactErr: errors.New("exact boom"), searchErr: errors.New("text boom")},
		fakeSearchProvider{name: "healthy"},
	}
	var updates []appview.SearchModsUpdate
	searchMods(sources, appstructs.SearchModsRequest{RequestID: "req-1", Query: "sodium"}, func(update appview.SearchModsUpdate) {
		updates = append(updates, update)
	})

	if len(updates) == 0 {
		t.Fatal("search emitted no updates")
	}
	final := updates[len(updates)-1]
	if final.RequestID != "req-1" {
		t.Fatalf("request id = %q", final.RequestID)
	}
	if final.Loading {
		t.Fatal("final update still loading")
	}
	for _, want := range []string{"broken: exact boom", "broken: text boom"} {
		if !strings.Contains(final.Error, want) {
			t.Fatalf("final error = %q, want substring %q", final.Error, want)
		}
	}
	for _, update := range updates[:len(updates)-1] {
		if update.Error != "" {
			t.Fatalf("intermediate update exposed error = %q", update.Error)
		}
	}
}

func TestSearchModsTextAggregatesProviderErrors(t *testing.T) {
	sources := []modProvider{
		fakeSearchProvider{name: "broken", searchErr: errors.New("text boom")},
		fakeSearchProvider{name: "healthy"},
	}
	var updates []appview.SearchModsUpdate
	searchModsText(sources, appstructs.SearchModsRequest{RequestID: "req-2", Query: "sodium"}, func(update appview.SearchModsUpdate) {
		updates = append(updates, update)
	})

	final := updates[len(updates)-1]
	if final.Loading {
		t.Fatal("final update still loading")
	}
	if !strings.Contains(final.Error, "broken: text boom") {
		t.Fatalf("final error = %q", final.Error)
	}
}

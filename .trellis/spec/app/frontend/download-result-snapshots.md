# Download Result Snapshots

## Local Result Import Contract

When a local source such as a favorite list needs to appear in the Download
page result list, use `useDownloadSearchStore().showResults(results)` rather
than assigning `searchResults` from a view.

The action must:

- accept `models.ModProject[]`-compatible values;
- replace the current result snapshot and clear `searchText`;
- invalidate remote-search request/pagination/loading state;
- clear old `downloadStates` before refreshing them for the active target tuple;
- reset any selected result/version overlay left by the previous snapshot.

Local imports are snapshots, not remote searches: they must not call
`SearchMods`, and they must set `hasMoreResults` to false. A later normal text
search is responsible for replacing the snapshot with remote results.

```ts
await downloadStore.showResults(favoriteItems.map(projectFromFavorite));
await router.push({ name: "Download" });
```

Keep source identity in each result (`id`, `platform`, and `projectId`) so
version lookup, install-state refresh, and download actions use the same
project reference as ordinary search results.

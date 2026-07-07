export namespace appcore {
	
	export class FavoriteBulkAddRequest {
	    targetListIds: string[];
	    mods: database.FavoriteMod[];
	
	    static createFrom(source: any = {}) {
	        return new FavoriteBulkAddRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetListIds = source["targetListIds"];
	        this.mods = this.convertValues(source["mods"], database.FavoriteMod);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FavoriteBulkOperationResult {
	    added: number;
	    updated: number;
	    skipped: number;
	    errors?: string[];
	
	    static createFrom(source: any = {}) {
	        return new FavoriteBulkOperationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.added = source["added"];
	        this.updated = source["updated"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	    }
	}
	export class FavoriteListCopyRequest {
	    sourceListId: string;
	    targetListId: string;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteListCopyRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceListId = source["sourceListId"];
	        this.targetListId = source["targetListId"];
	    }
	}
	export class FavoriteMigrationConflict {
	    source: database.FavoriteModEntry;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteMigrationConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = this.convertValues(source["source"], database.FavoriteModEntry);
	        this.reason = source["reason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FavoriteMigrationMatch {
	    source: database.FavoriteModEntry;
	    target: database.FavoriteMod;
	    project: models.ModProject;
	    version: models.ModVersion;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteMigrationMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = this.convertValues(source["source"], database.FavoriteModEntry);
	        this.target = this.convertValues(source["target"], database.FavoriteMod);
	        this.project = this.convertValues(source["project"], models.ModProject);
	        this.version = this.convertValues(source["version"], models.ModVersion);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FavoriteMigrationPreview {
	    sourceListId: string;
	    targetListId: string;
	    matched: FavoriteMigrationMatch[];
	    conflicts: FavoriteMigrationConflict[];
	    errors?: string[];
	
	    static createFrom(source: any = {}) {
	        return new FavoriteMigrationPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceListId = source["sourceListId"];
	        this.targetListId = source["targetListId"];
	        this.matched = this.convertValues(source["matched"], FavoriteMigrationMatch);
	        this.conflicts = this.convertValues(source["conflicts"], FavoriteMigrationConflict);
	        this.errors = source["errors"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FavoriteMigrationApplyResult {
	    applied: boolean;
	    preview: FavoriteMigrationPreview;
	    result: FavoriteBulkOperationResult;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteMigrationApplyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.applied = source["applied"];
	        this.preview = this.convertValues(source["preview"], FavoriteMigrationPreview);
	        this.result = this.convertValues(source["result"], FavoriteBulkOperationResult);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class FavoriteMigrationRequest {
	    sourceListId: string;
	    targetListId: string;
	    minecraftVersion: string;
	    modLoader: string;
	    ignoreConflicts?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteMigrationRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceListId = source["sourceListId"];
	        this.targetListId = source["targetListId"];
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
	        this.ignoreConflicts = source["ignoreConflicts"];
	    }
	}

}

export namespace database {
	
	export class FavoriteList {
	    id: string;
	    groupId?: string;
	    name: string;
	    iconKind?: string;
	    iconValue?: string;
	    iconUrl?: string;
	    pinned?: boolean;
	    createdAt: number;
	    updatedAt: number;
	    sortOrder: number;
	    system?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.groupId = source["groupId"];
	        this.name = source["name"];
	        this.iconKind = source["iconKind"];
	        this.iconValue = source["iconValue"];
	        this.iconUrl = source["iconUrl"];
	        this.pinned = source["pinned"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.sortOrder = source["sortOrder"];
	        this.system = source["system"];
	    }
	}
	export class FavoriteListRef {
	    id: string;
	    parentListId: string;
	    childListId: string;
	    createdAt: number;
	    updatedAt: number;
	    sortOrder: number;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteListRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.parentListId = source["parentListId"];
	        this.childListId = source["childListId"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.sortOrder = source["sortOrder"];
	    }
	}
	export class FavoriteModEntry {
	    id: string;
	    listId: string;
	    platform: string;
	    modId: string;
	    versionId?: string;
	    minecraftVersion?: string;
	    modLoader?: string;
	    title?: string;
	    slug?: string;
	    iconUrl?: string;
	    description?: string;
	    categories?: string[];
	    createdAt: number;
	    updatedAt: number;
	    sourceListId?: string;
	    sourceListName?: string;
	    referenced?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteModEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.listId = source["listId"];
	        this.platform = source["platform"];
	        this.modId = source["modId"];
	        this.versionId = source["versionId"];
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
	        this.title = source["title"];
	        this.slug = source["slug"];
	        this.iconUrl = source["iconUrl"];
	        this.description = source["description"];
	        this.categories = source["categories"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.sourceListId = source["sourceListId"];
	        this.sourceListName = source["sourceListName"];
	        this.referenced = source["referenced"];
	    }
	}
	export class FavoriteListContents {
	    listId: string;
	    mods: FavoriteModEntry[];
	    refs?: FavoriteListRef[];
	
	    static createFrom(source: any = {}) {
	        return new FavoriteListContents(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listId = source["listId"];
	        this.mods = this.convertValues(source["mods"], FavoriteModEntry);
	        this.refs = this.convertValues(source["refs"], FavoriteListRef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class FavoriteMod {
	    id: string;
	    listId: string;
	    platform: string;
	    modId: string;
	    versionId?: string;
	    minecraftVersion?: string;
	    modLoader?: string;
	    title?: string;
	    slug?: string;
	    iconUrl?: string;
	    description?: string;
	    categories?: string[];
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteMod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.listId = source["listId"];
	        this.platform = source["platform"];
	        this.modId = source["modId"];
	        this.versionId = source["versionId"];
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
	        this.title = source["title"];
	        this.slug = source["slug"];
	        this.iconUrl = source["iconUrl"];
	        this.description = source["description"];
	        this.categories = source["categories"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	
	export class PinnedMod {
	    id: string;
	    platform: string;
	    modId: string;
	    versionId: string;
	    minecraftVersion: string;
	    modLoader: string;
	
	    static createFrom(source: any = {}) {
	        return new PinnedMod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.modId = source["modId"];
	        this.versionId = source["versionId"];
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
	    }
	}

}

export namespace main {
	
	export class AppPreferences {
	    theme: string;
	    animationMode: string;
	    animationEnabled: boolean;
	    animationDurationMultiplier: number;
	
	    static createFrom(source: any = {}) {
	        return new AppPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.animationMode = source["animationMode"];
	        this.animationEnabled = source["animationEnabled"];
	        this.animationDurationMultiplier = source["animationDurationMultiplier"];
	    }
	}
	export class SaveAnimationSettingsRequest {
	    animationMode: string;
	    animationEnabled: boolean;
	    animationDurationMultiplier: number;
	
	    static createFrom(source: any = {}) {
	        return new SaveAnimationSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.animationMode = source["animationMode"];
	        this.animationEnabled = source["animationEnabled"];
	        this.animationDurationMultiplier = source["animationDurationMultiplier"];
	    }
	}
	export class SaveApiKeysRequest {
	    curseforgeApiKey: string;
	    modrinthApiKey: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveApiKeysRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.curseforgeApiKey = source["curseforgeApiKey"];
	        this.modrinthApiKey = source["modrinthApiKey"];
	    }
	}
	export class SettingsView {
	    theme: string;
	    animationMode: string;
	    animationEnabled: boolean;
	    animationDurationMultiplier: number;
	    minecraftDir: string;
	    cacheDir: string;
	    cachePath: string;
	    hasCurseforgeKey: boolean;
	    curseforgeKeyMask: string;
	    hasModrinthKey: boolean;
	    modrinthKeyMask: string;
	
	    static createFrom(source: any = {}) {
	        return new SettingsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.animationMode = source["animationMode"];
	        this.animationEnabled = source["animationEnabled"];
	        this.animationDurationMultiplier = source["animationDurationMultiplier"];
	        this.minecraftDir = source["minecraftDir"];
	        this.cacheDir = source["cacheDir"];
	        this.cachePath = source["cachePath"];
	        this.hasCurseforgeKey = source["hasCurseforgeKey"];
	        this.curseforgeKeyMask = source["curseforgeKeyMask"];
	        this.hasModrinthKey = source["hasModrinthKey"];
	        this.modrinthKeyMask = source["modrinthKeyMask"];
	    }
	}

}

export namespace models {
	
	export class ModDependency {
	    id?: string;
	    platformVersionId?: string;
	    projectId: string;
	    versionId?: string;
	    type?: string;
	
	    static createFrom(source: any = {}) {
	        return new ModDependency(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platformVersionId = source["platformVersionId"];
	        this.projectId = source["projectId"];
	        this.versionId = source["versionId"];
	        this.type = source["type"];
	    }
	}
	export class ModProject {
	    id: string;
	    platform: string;
	    projectId: string;
	    slug: string;
	    title: string;
	    icon: string;
	    iconUrl: string;
	    description: string;
	    downloads: number;
	    categories?: string[];
	    updatedAt: number;
	    cachedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new ModProject(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.projectId = source["projectId"];
	        this.slug = source["slug"];
	        this.title = source["title"];
	        this.icon = source["icon"];
	        this.iconUrl = source["iconUrl"];
	        this.description = source["description"];
	        this.downloads = source["downloads"];
	        this.categories = source["categories"];
	        this.updatedAt = source["updatedAt"];
	        this.cachedAt = source["cachedAt"];
	    }
	}
	export class ModVersion {
	    id: string;
	    platform: string;
	    projectId: string;
	    versionId: string;
	    name: string;
	    version: string;
	    fileName: string;
	    downloadUrl: string;
	    sha1: string;
	    publishedAt: number;
	    downloads: number;
	    gameVersions: string[];
	    loaders: string[];
	    dependencies?: ModDependency[];
	    modIds?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ModVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.projectId = source["projectId"];
	        this.versionId = source["versionId"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.fileName = source["fileName"];
	        this.downloadUrl = source["downloadUrl"];
	        this.sha1 = source["sha1"];
	        this.publishedAt = source["publishedAt"];
	        this.downloads = source["downloads"];
	        this.gameVersions = source["gameVersions"];
	        this.loaders = source["loaders"];
	        this.dependencies = this.convertValues(source["dependencies"], ModDependency);
	        this.modIds = source["modIds"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace structs {
	
	export class DownloadQueueItem {
	    id: string;
	    status: string;
	    title: string;
	    fileName: string;
	    versionId: string;
	    platform: string;
	    minecraftVersion: string;
	    modLoader: string;
	    cancelable: boolean;
	    retryable: boolean;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadQueueItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.status = source["status"];
	        this.title = source["title"];
	        this.fileName = source["fileName"];
	        this.versionId = source["versionId"];
	        this.platform = source["platform"];
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
	        this.cancelable = source["cancelable"];
	        this.retryable = source["retryable"];
	        this.reason = source["reason"];
	    }
	}
	export class DownloadQueueState {
	    active: boolean;
	    pending: number;
	    running: number;
	    items?: DownloadQueueItem[];
	
	    static createFrom(source: any = {}) {
	        return new DownloadQueueState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.pending = source["pending"];
	        this.running = source["running"];
	        this.items = this.convertValues(source["items"], DownloadQueueItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DownloadStatesRequest {
	    results: models.ModProject[];
	    minecraftVersion: string;
	    modLoader: string;
	    targetDir?: string;
	    instanceId?: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadStatesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], models.ModProject);
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
	        this.targetDir = source["targetDir"];
	        this.instanceId = source["instanceId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class JijModInfo {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new JijModInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class LocalModBatchOperationRequest {
	    paths: string[];
	    action: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalModBatchOperationRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paths = source["paths"];
	        this.action = source["action"];
	    }
	}
	export class ModDownloadButtonState {
	    key: string;
	    status: string;
	    disabled: boolean;
	    loading: boolean;
	    icon: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new ModDownloadButtonState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.status = source["status"];
	        this.disabled = source["disabled"];
	        this.loading = source["loading"];
	        this.icon = source["icon"];
	        this.color = source["color"];
	    }
	}
	export class ModDownloadRequest {
	    projectId: string;
	    result: models.ModProject;
	    minecraftVersion: string;
	    modLoader: string;
	    versionId?: string;
	    targetDir?: string;
	    instanceId?: string;
	
	    static createFrom(source: any = {}) {
	        return new ModDownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.result = this.convertValues(source["result"], models.ModProject);
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
	        this.versionId = source["versionId"];
	        this.targetDir = source["targetDir"];
	        this.instanceId = source["instanceId"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModDownloadResult {
	    queued: boolean;
	    skipped: boolean;
	    reason: string;
	    fileName: string;
	    versionId: string;
	
	    static createFrom(source: any = {}) {
	        return new ModDownloadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.queued = source["queued"];
	        this.skipped = source["skipped"];
	        this.reason = source["reason"];
	        this.fileName = source["fileName"];
	        this.versionId = source["versionId"];
	    }
	}
	export class ModInfo {
	    id: string;
	    name: string;
	    version: string;
	    description: string;
	    fileName: string;
	    path: string;
	    sha1?: string;
	    onlineName?: string;
	    onlinePlatform?: string;
	    onlineProjectId?: string;
	    onlineSlug?: string;
	    iconUrl?: string;
	    categories?: string[];
	    enabled: boolean;
	    jijMods?: JijModInfo[];
	
	    static createFrom(source: any = {}) {
	        return new ModInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	        this.fileName = source["fileName"];
	        this.path = source["path"];
	        this.sha1 = source["sha1"];
	        this.onlineName = source["onlineName"];
	        this.onlinePlatform = source["onlinePlatform"];
	        this.onlineProjectId = source["onlineProjectId"];
	        this.onlineSlug = source["onlineSlug"];
	        this.iconUrl = source["iconUrl"];
	        this.categories = source["categories"];
	        this.enabled = source["enabled"];
	        this.jijMods = this.convertValues(source["jijMods"], JijModInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ModVersionPinRequest {
	    platform: string;
	    modId: string;
	    versionId: string;
	    minecraftVersion: string;
	    modLoader: string;
	
	    static createFrom(source: any = {}) {
	        return new ModVersionPinRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.modId = source["modId"];
	        this.versionId = source["versionId"];
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
	    }
	}
	export class SearchModsRequest {
	    requestId: string;
	    query: string;
	    version: string;
	    modLoader: string;
	    offset: number;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchModsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestId = source["requestId"];
	        this.query = source["query"];
	        this.version = source["version"];
	        this.modLoader = source["modLoader"];
	        this.offset = source["offset"];
	        this.limit = source["limit"];
	    }
	}
	export class VersionInfo {
	    name: string;
	    id: string;
	    minecraftVersion: string;
	    modLoader: string;
	    mods?: ModInfo[];
	
	    static createFrom(source: any = {}) {
	        return new VersionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.id = source["id"];
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
	        this.mods = this.convertValues(source["mods"], ModInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}


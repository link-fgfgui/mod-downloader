export namespace appcore {
	
	export class FavoriteBulkAddRequest {
	    targetListIds: string[];
	    mods: storage.FavoriteMod[];
	
	    static createFrom(source: any = {}) {
	        return new FavoriteBulkAddRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetListIds = source["targetListIds"];
	        this.mods = this.convertValues(source["mods"], storage.FavoriteMod);
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
	    source: storage.FavoriteModEntry;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteMigrationConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = this.convertValues(source["source"], storage.FavoriteModEntry);
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
	    source: storage.FavoriteModEntry;
	    target: storage.FavoriteMod;
	    project: models.ModProject;
	    version: models.ModVersion;
	
	    static createFrom(source: any = {}) {
	        return new FavoriteMigrationMatch(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = this.convertValues(source["source"], storage.FavoriteModEntry);
	        this.target = this.convertValues(source["target"], storage.FavoriteMod);
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

export namespace main {
	
	export class AppPreferences {
	    theme: string;
	    language: string;
	    animationMode: string;
	    animationEnabled: boolean;
	    animationDurationMultiplier: number;
	
	    static createFrom(source: any = {}) {
	        return new AppPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.language = source["language"];
	        this.animationMode = source["animationMode"];
	        this.animationEnabled = source["animationEnabled"];
	        this.animationDurationMultiplier = source["animationDurationMultiplier"];
	    }
	}
	export class ExportFavoritePackwizResult {
	    path: string;
	    canceled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExportFavoritePackwizResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.canceled = source["canceled"];
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
	export class SaveMCIMSettingsRequest {
	    mcimEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SaveMCIMSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mcimEnabled = source["mcimEnabled"];
	    }
	}
	export class SaveNetworkSettingsRequest {
	    fileConcurrency: number;
	    concurrentDownloads: number;
	    adaptiveFileConcurrency: boolean;
	    targetDownloadRateMiB: number;
	    verifySha1: boolean;
	    requestsPerSecond: number;
	
	    static createFrom(source: any = {}) {
	        return new SaveNetworkSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileConcurrency = source["fileConcurrency"];
	        this.concurrentDownloads = source["concurrentDownloads"];
	        this.adaptiveFileConcurrency = source["adaptiveFileConcurrency"];
	        this.targetDownloadRateMiB = source["targetDownloadRateMiB"];
	        this.verifySha1 = source["verifySha1"];
	        this.requestsPerSecond = source["requestsPerSecond"];
	    }
	}
	export class SaveSimpleModeSettingsRequest {
	    simpleMode: boolean;

	    static createFrom(source: any = {}) {
	        return new SaveSimpleModeSettingsRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.simpleMode = source["simpleMode"];
	    }
	}
	export class SaveUnusedDependencyCleanupSettingsRequest {
	    autoScanUnusedDependencies: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SaveUnusedDependencyCleanupSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autoScanUnusedDependencies = source["autoScanUnusedDependencies"];
	    }
	}
	export class SettingsView {
	    theme: string;
	    language: string;
	    simpleMode: boolean;
	    animationMode: string;
	    animationEnabled: boolean;
	    animationDurationMultiplier: number;
	    autoScanUnusedDependencies: boolean;
	    mcimEnabled: boolean;
	    minecraftDir: string;
	    cacheDir: string;
	    cachePath: string;
	    hasCurseforgeKey: boolean;
	    curseforgeKeyMask: string;
	    hasModrinthKey: boolean;
	    modrinthKeyMask: string;
	    fileConcurrency: number;
	    concurrentDownloads: number;
	    adaptiveFileConcurrency: boolean;
	    targetDownloadRateMiB: number;
	    verifySha1: boolean;
	    requestsPerSecond: number;
	
	    static createFrom(source: any = {}) {
	        return new SettingsView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.language = source["language"];
	        this.simpleMode = source["simpleMode"];
	        this.animationMode = source["animationMode"];
	        this.animationEnabled = source["animationEnabled"];
	        this.animationDurationMultiplier = source["animationDurationMultiplier"];
	        this.autoScanUnusedDependencies = source["autoScanUnusedDependencies"];
	        this.mcimEnabled = source["mcimEnabled"];
	        this.minecraftDir = source["minecraftDir"];
	        this.cacheDir = source["cacheDir"];
	        this.cachePath = source["cachePath"];
	        this.hasCurseforgeKey = source["hasCurseforgeKey"];
	        this.curseforgeKeyMask = source["curseforgeKeyMask"];
	        this.hasModrinthKey = source["hasModrinthKey"];
	        this.modrinthKeyMask = source["modrinthKeyMask"];
	        this.fileConcurrency = source["fileConcurrency"];
	        this.concurrentDownloads = source["concurrentDownloads"];
	        this.adaptiveFileConcurrency = source["adaptiveFileConcurrency"];
	        this.targetDownloadRateMiB = source["targetDownloadRateMiB"];
	        this.verifySha1 = source["verifySha1"];
	        this.requestsPerSecond = source["requestsPerSecond"];
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
	    fileSize: number;
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
	        this.fileSize = source["fileSize"];
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

export namespace storage {
	
	export class FavoriteList {
	    id: string;
	    groupId?: string;
	    name: string;
	    minecraftVersion?: string;
	    modLoader?: string;
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
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
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
	export class UsageStats {
	    downloadsCompleted: number;
	    modsEnabled: number;
	    modsDisabled: number;
	    modsDeleted: number;
	    favoritesAdded: number;
	    packwizExports: number;
	
	    static createFrom(source: any = {}) {
	        return new UsageStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.downloadsCompleted = source["downloadsCompleted"];
	        this.modsEnabled = source["modsEnabled"];
	        this.modsDisabled = source["modsDisabled"];
	        this.modsDeleted = source["modsDeleted"];
	        this.favoritesAdded = source["favoritesAdded"];
	        this.packwizExports = source["packwizExports"];
	    }
	}

}

export namespace structs {
	
	export class BatchDownloadRequest {
	    results: models.ModProject[];
	    minecraftVersion: string;
	    modLoader: string;
	    targetDir?: string;
	    instanceId?: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchDownloadRequest(source);
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
	export class IncompatibleLocalMod {
	    path: string;
	    fileName: string;
	    sha1: string;
	
	    static createFrom(source: any = {}) {
	        return new IncompatibleLocalMod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.sha1 = source["sha1"];
	    }
	}
	export class IncompatibleConflict {
	    projectKey: string;
	    projectTitle: string;
	    incompatibleProjectKey: string;
	    incompatibleTitle: string;
	    incompatibleVersionId?: string;
	    paths?: IncompatibleLocalMod[];
	
	    static createFrom(source: any = {}) {
	        return new IncompatibleConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectKey = source["projectKey"];
	        this.projectTitle = source["projectTitle"];
	        this.incompatibleProjectKey = source["incompatibleProjectKey"];
	        this.incompatibleTitle = source["incompatibleTitle"];
	        this.incompatibleVersionId = source["incompatibleVersionId"];
	        this.paths = this.convertValues(source["paths"], IncompatibleLocalMod);
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
	export class BatchIncompatibleConflict {
	    key: string;
	    title: string;
	    result: models.ModProject;
	    conflicts?: IncompatibleConflict[];
	
	    static createFrom(source: any = {}) {
	        return new BatchIncompatibleConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.title = source["title"];
	        this.result = this.convertValues(source["result"], models.ModProject);
	        this.conflicts = this.convertValues(source["conflicts"], IncompatibleConflict);
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
	export class BatchIncompatibleAnalysis {
	    conflicts?: BatchIncompatibleConflict[];
	
	    static createFrom(source: any = {}) {
	        return new BatchIncompatibleAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conflicts = this.convertValues(source["conflicts"], BatchIncompatibleConflict);
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
	
	export class CachedDependencyCandidate {
	    modId: string;
	    platform: string;
	    projectId: string;
	    versionId: string;
	    title?: string;
	
	    static createFrom(source: any = {}) {
	        return new CachedDependencyCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modId = source["modId"];
	        this.platform = source["platform"];
	        this.projectId = source["projectId"];
	        this.versionId = source["versionId"];
	        this.title = source["title"];
	    }
	}
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
	    bytesComplete: number;
	    totalBytes: number;
	
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
	        this.bytesComplete = source["bytesComplete"];
	        this.totalBytes = source["totalBytes"];
	    }
	}
	export class OptionalDependencyCandidate {
	    projectKey: string;
	    platform: string;
	    projectId: string;
	    title: string;
	    versionId?: string;
	    status: string;
	    disabled: boolean;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new OptionalDependencyCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectKey = source["projectKey"];
	        this.platform = source["platform"];
	        this.projectId = source["projectId"];
	        this.title = source["title"];
	        this.versionId = source["versionId"];
	        this.status = source["status"];
	        this.disabled = source["disabled"];
	        this.reason = source["reason"];
	    }
	}
	export class OptionalDependencyReminder {
	    id: string;
	    mainProjectKey: string;
	    mainTitle: string;
	    mainVersionId: string;
	    minecraftVersion: string;
	    modLoader: string;
	    dependencies?: OptionalDependencyCandidate[];
	
	    static createFrom(source: any = {}) {
	        return new OptionalDependencyReminder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.mainProjectKey = source["mainProjectKey"];
	        this.mainTitle = source["mainTitle"];
	        this.mainVersionId = source["mainVersionId"];
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
	        this.dependencies = this.convertValues(source["dependencies"], OptionalDependencyCandidate);
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
	export class DownloadQueueState {
	    active: boolean;
	    pending: number;
	    running: number;
	    messageCount: number;
	    bytesComplete: number;
	    totalBytes: number;
	    bytesPerSecond: number;
	    items?: DownloadQueueItem[];
	    optionalReminders?: OptionalDependencyReminder[];
	
	    static createFrom(source: any = {}) {
	        return new DownloadQueueState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.pending = source["pending"];
	        this.running = source["running"];
	        this.messageCount = source["messageCount"];
	        this.bytesComplete = source["bytesComplete"];
	        this.totalBytes = source["totalBytes"];
	        this.bytesPerSecond = source["bytesPerSecond"];
	        this.items = this.convertValues(source["items"], DownloadQueueItem);
	        this.optionalReminders = this.convertValues(source["optionalReminders"], OptionalDependencyReminder);
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
	export class LocalModDependency {
	    modId: string;
	    type?: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalModDependency(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modId = source["modId"];
	        this.type = source["type"];
	    }
	}
	export class LocalModDependencyMod {
	    path: string;
	    fileName?: string;
	    name?: string;
	    modIds?: string[];
	
	    static createFrom(source: any = {}) {
	        return new LocalModDependencyMod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.name = source["name"];
	        this.modIds = source["modIds"];
	    }
	}
	export class LocalModDependencyImpact {
	    modId: string;
	    affectedMods: LocalModDependencyMod[];
	    disabledProviders: LocalModDependencyMod[];
	    candidates?: CachedDependencyCandidate[];
	
	    static createFrom(source: any = {}) {
	        return new LocalModDependencyImpact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modId = source["modId"];
	        this.affectedMods = this.convertValues(source["affectedMods"], LocalModDependencyMod);
	        this.disabledProviders = this.convertValues(source["disabledProviders"], LocalModDependencyMod);
	        this.candidates = this.convertValues(source["candidates"], CachedDependencyCandidate);
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
	
	export class LocalModDisableImpactRequest {
	    paths: string[];
	    action: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalModDisableImpactRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paths = source["paths"];
	        this.action = source["action"];
	    }
	}
	export class LocalModDisableImpactResult {
	    impacts: LocalModDependencyImpact[];
	
	    static createFrom(source: any = {}) {
	        return new LocalModDisableImpactResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.impacts = this.convertValues(source["impacts"], LocalModDependencyImpact);
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
	export class ModDownloadButtonState {
	    key: string;
	    status: string;
	    conflictFileName?: string;
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
	        this.conflictFileName = source["conflictFileName"];
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
	    cfFingerprint?: number;
	    onlineName?: string;
	    onlinePlatform?: string;
	    onlineProjectId?: string;
	    onlineSlug?: string;
	    onlineVersion?: string;
	    onlineVersionId?: string;
	    onlineFileName?: string;
	    onlineMetadataLoading?: boolean;
	    iconUrl?: string;
	    categories?: string[];
	    enabled: boolean;
	    jijMods?: JijModInfo[];
	    dependencies?: LocalModDependency[];
	
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
	        this.cfFingerprint = source["cfFingerprint"];
	        this.onlineName = source["onlineName"];
	        this.onlinePlatform = source["onlinePlatform"];
	        this.onlineProjectId = source["onlineProjectId"];
	        this.onlineSlug = source["onlineSlug"];
	        this.onlineVersion = source["onlineVersion"];
	        this.onlineVersionId = source["onlineVersionId"];
	        this.onlineFileName = source["onlineFileName"];
	        this.onlineMetadataLoading = source["onlineMetadataLoading"];
	        this.iconUrl = source["iconUrl"];
	        this.categories = source["categories"];
	        this.enabled = source["enabled"];
	        this.jijMods = this.convertValues(source["jijMods"], JijModInfo);
	        this.dependencies = this.convertValues(source["dependencies"], LocalModDependency);
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
	
	
	export class RestoreCachedDependencyRequest {
	    modId: string;
	    platform: string;
	    projectId: string;
	    versionId: string;
	
	    static createFrom(source: any = {}) {
	        return new RestoreCachedDependencyRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modId = source["modId"];
	        this.platform = source["platform"];
	        this.projectId = source["projectId"];
	        this.versionId = source["versionId"];
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
	export class UnusedDependencyCandidate {
	    path: string;
	    fileName: string;
	    modIds: string[];
	    name?: string;
	    onlineName?: string;
	    evidence?: string[];
	
	    static createFrom(source: any = {}) {
	        return new UnusedDependencyCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.modIds = source["modIds"];
	        this.name = source["name"];
	        this.onlineName = source["onlineName"];
	        this.evidence = source["evidence"];
	    }
	}
	export class UnusedDependencyScanRequest {
	    excludedPaths?: string[];
	
	    static createFrom(source: any = {}) {
	        return new UnusedDependencyScanRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.excludedPaths = source["excludedPaths"];
	    }
	}
	export class UnusedDependencyScanResult {
	    candidates: UnusedDependencyCandidate[];
	
	    static createFrom(source: any = {}) {
	        return new UnusedDependencyScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.candidates = this.convertValues(source["candidates"], UnusedDependencyCandidate);
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

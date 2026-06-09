export namespace database {
	
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
	
	    static createFrom(source: any = {}) {
	        return new AppPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	    }
	}

}

export namespace structs {
	
	export class DownloadQueueState {
	    active: boolean;
	    pending: number;
	    running: number;
	
	    static createFrom(source: any = {}) {
	        return new DownloadQueueState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.pending = source["pending"];
	        this.running = source["running"];
	    }
	}
	export class SearchModResult {
	    id: string;
	    platform: string;
	    title: string;
	    icon: string;
	    iconUrl: string;
	    description: string;
	    downloads: number;
	    slug: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchModResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.title = source["title"];
	        this.icon = source["icon"];
	        this.iconUrl = source["iconUrl"];
	        this.description = source["description"];
	        this.downloads = source["downloads"];
	        this.slug = source["slug"];
	    }
	}
	export class DownloadStatesRequest {
	    results: SearchModResult[];
	    minecraftVersion: string;
	    modLoader: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadStatesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], SearchModResult);
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
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
	    disabled: boolean;
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
	        this.icon = source["icon"];
	        this.color = source["color"];
	    }
	}
	export class ModDownloadRequest {
	    projectId: string;
	    result: SearchModResult;
	    minecraftVersion: string;
	    modLoader: string;
	
	    static createFrom(source: any = {}) {
	        return new ModDownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.result = this.convertValues(source["result"], SearchModResult);
	        this.minecraftVersion = source["minecraftVersion"];
	        this.modLoader = source["modLoader"];
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
	    enabled: boolean;
	
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
	        this.enabled = source["enabled"];
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
	export class ProjectDependency {
	    projectId: string;
	    versionId?: string;
	    type?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectDependency(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.versionId = source["versionId"];
	        this.type = source["type"];
	    }
	}
	export class ProjectVersionResult {
	    id: string;
	    platform: string;
	    projectId: string;
	    name: string;
	    version: string;
	    fileName: string;
	    downloadUrl: string;
	    sha1: string;
	    publishedAt: number;
	    downloads: number;
	    gameVersions: string[];
	    loaders: string[];
	    dependencies?: ProjectDependency[];
	
	    static createFrom(source: any = {}) {
	        return new ProjectVersionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.platform = source["platform"];
	        this.projectId = source["projectId"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.fileName = source["fileName"];
	        this.downloadUrl = source["downloadUrl"];
	        this.sha1 = source["sha1"];
	        this.publishedAt = source["publishedAt"];
	        this.downloads = source["downloads"];
	        this.gameVersions = source["gameVersions"];
	        this.loaders = source["loaders"];
	        this.dependencies = this.convertValues(source["dependencies"], ProjectDependency);
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


export namespace main {
	
	export class ConfigLocation {
	    path: string;
	    mode: string;
	    exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConfigLocation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.mode = source["mode"];
	        this.exists = source["exists"];
	    }
	}
	export class Preferences {
	    minimizeToTray: boolean;
	    startSharingOnLaunch: boolean;
	    launchAtStartup: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Preferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.minimizeToTray = source["minimizeToTray"];
	        this.startSharingOnLaunch = source["startSharingOnLaunch"];
	        this.launchAtStartup = source["launchAtStartup"];
	    }
	}
	export class SourceState {
	    name: string;
	    path: string;
	    available: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.available = source["available"];
	        this.error = source["error"];
	    }
	}
	export class DesktopState {
	    status: runtime.Status;
	    sources: SourceState[];
	    port: number;
	    logLevel: string;
	    preferences: Preferences;
	    config: ConfigLocation;
	    version: string;
	    firstRun: boolean;
	    canStart: boolean;
	    loadError?: string;
	
	    static createFrom(source: any = {}) {
	        return new DesktopState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = this.convertValues(source["status"], runtime.Status);
	        this.sources = this.convertValues(source["sources"], SourceState);
	        this.port = source["port"];
	        this.logLevel = source["logLevel"];
	        this.preferences = this.convertValues(source["preferences"], Preferences);
	        this.config = this.convertValues(source["config"], ConfigLocation);
	        this.version = source["version"];
	        this.firstRun = source["firstRun"];
	        this.canStart = source["canStart"];
	        this.loadError = source["loadError"];
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

export namespace runtime {
	
	export class Status {
	    state: string;
	    address?: string;
	    port?: number;
	    urls?: string[];
	    // Go type: time
	    startedAt?: any;
	    lastError?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.address = source["address"];
	        this.port = source["port"];
	        this.urls = source["urls"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.lastError = source["lastError"];
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


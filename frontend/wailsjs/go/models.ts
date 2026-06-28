export namespace config {
	
	export class TotpAccount {
	    id: string;
	    label: string;
	    issuer: string;
	
	    static createFrom(source: any = {}) {
	        return new TotpAccount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.issuer = source["issuer"];
	    }
	}

}

export namespace github {
	
	export class PR {
	    repo: string;
	    number: number;
	    title: string;
	    author: string;
	    url: string;
	    draft: boolean;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new PR(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repo = source["repo"];
	        this.number = source["number"];
	        this.title = source["title"];
	        this.author = source["author"];
	        this.url = source["url"];
	        this.draft = source["draft"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Result {
	    prs: PR[];
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prs = this.convertValues(source["prs"], PR);
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
	export class WorkflowRun {
	    repo: string;
	    name: string;
	    status: string;
	    conclusion: string;
	    branch: string;
	    event: string;
	    url: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkflowRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repo = source["repo"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.conclusion = source["conclusion"];
	        this.branch = source["branch"];
	        this.event = source["event"];
	        this.url = source["url"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class WorkflowResult {
	    runs: WorkflowRun[];
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new WorkflowResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.runs = this.convertValues(source["runs"], WorkflowRun);
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

}

export namespace main {
	
	export class MFACodeEntry {
	    id: string;
	    label: string;
	    issuer: string;
	    code: string;
	    seconds: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new MFACodeEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.issuer = source["issuer"];
	        this.code = source["code"];
	        this.seconds = source["seconds"];
	        this.error = source["error"];
	    }
	}
	export class MFACodes {
	    locked: boolean;
	    entries: MFACodeEntry[];
	    secondsUntilLock: number;
	
	    static createFrom(source: any = {}) {
	        return new MFACodes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.locked = source["locked"];
	        this.entries = this.convertValues(source["entries"], MFACodeEntry);
	        this.secondsUntilLock = source["secondsUntilLock"];
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
	export class MFAStatus {
	    hasPin: boolean;
	    unlocked: boolean;
	    accountCount: number;
	    secondsUntilLock: number;
	
	    static createFrom(source: any = {}) {
	        return new MFAStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasPin = source["hasPin"];
	        this.unlocked = source["unlocked"];
	        this.accountCount = source["accountCount"];
	        this.secondsUntilLock = source["secondsUntilLock"];
	    }
	}
	export class Settings {
	    locationName: string;
	    units: string;
	    theme: string;
	    githubToken: string;
	    githubRepos: string[];
	    githubLogin: string;
	    teamsSource: string;
	    teamsClientId: string;
	    teamsTenantId: string;
	    teamsFavorites: string[];
	    views: string[];
	    configPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.locationName = source["locationName"];
	        this.units = source["units"];
	        this.theme = source["theme"];
	        this.githubToken = source["githubToken"];
	        this.githubRepos = source["githubRepos"];
	        this.githubLogin = source["githubLogin"];
	        this.teamsSource = source["teamsSource"];
	        this.teamsClientId = source["teamsClientId"];
	        this.teamsTenantId = source["teamsTenantId"];
	        this.teamsFavorites = source["teamsFavorites"];
	        this.views = source["views"];
	        this.configPath = source["configPath"];
	    }
	}

}

export namespace sysstats {
	
	export class Process {
	    name: string;
	    cpu: number;
	    mem: number;
	
	    static createFrom(source: any = {}) {
	        return new Process(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.cpu = source["cpu"];
	        this.mem = source["mem"];
	    }
	}
	export class Stats {
	    cpuPercent: number;
	    memPercent: number;
	    memUsedGB: number;
	    memTotalGB: number;
	    batteryPct: number;
	    batteryState: string;
	    uptimeHours: number;
	    hostname: string;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpuPercent = source["cpuPercent"];
	        this.memPercent = source["memPercent"];
	        this.memUsedGB = source["memUsedGB"];
	        this.memTotalGB = source["memTotalGB"];
	        this.batteryPct = source["batteryPct"];
	        this.batteryState = source["batteryState"];
	        this.uptimeHours = source["uptimeHours"];
	        this.hostname = source["hostname"];
	    }
	}

}

export namespace teams {
	
	export class Chat {
	    id: string;
	    name: string;
	    preview: string;
	    from: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new Chat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.preview = source["preview"];
	        this.from = source["from"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class Result {
	    unreadChats: Chat[];
	    totalUnread: number;
	    needsLogin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unreadChats = this.convertValues(source["unreadChats"], Chat);
	        this.totalUnread = source["totalUnread"];
	        this.needsLogin = source["needsLogin"];
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

export namespace weather {
	
	export class Data {
	    location: string;
	    label: string;
	    description: string;
	    temp: number;
	    unit: string;
	    feelsLike: number;
	    humidity: number;
	    windSpeed: number;
	    code: number;
	    isDay: boolean;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Data(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.location = source["location"];
	        this.label = source["label"];
	        this.description = source["description"];
	        this.temp = source["temp"];
	        this.unit = source["unit"];
	        this.feelsLike = source["feelsLike"];
	        this.humidity = source["humidity"];
	        this.windSpeed = source["windSpeed"];
	        this.code = source["code"];
	        this.isDay = source["isDay"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}


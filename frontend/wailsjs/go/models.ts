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
	    views: string[];
	    configPath: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.locationName = source["locationName"];
	        this.units = source["units"];
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


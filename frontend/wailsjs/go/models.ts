export namespace desktop {
	
	export class Account {
	    id: string;
	    provider: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.provider = source["provider"];
	        this.label = source["label"];
	    }
	}
	export class UsageWindow {
	    kind: string;
	    used: number;
	    resetsAt: string;
	
	    static createFrom(source: any = {}) {
	        return new UsageWindow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.used = source["used"];
	        this.resetsAt = source["resetsAt"];
	    }
	}
	export class AccountUsage {
	    id: string;
	    windows: UsageWindow[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new AccountUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.windows = this.convertValues(source["windows"], UsageWindow);
	        this.error = source["error"];
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
	export class KeyDraft {
	    apiKey: string;
	    baseUrl: string;
	    prefix: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyDraft(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	        this.baseUrl = source["baseUrl"];
	        this.prefix = source["prefix"];
	    }
	}
	export class ModelDraft {
	    name: string;
	    alias: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelDraft(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.alias = source["alias"];
	    }
	}
	export class OpenAIDraft {
	    name: string;
	    baseUrl: string;
	    apiKeys: string[];
	    models: ModelDraft[];
	
	    static createFrom(source: any = {}) {
	        return new OpenAIDraft(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.baseUrl = source["baseUrl"];
	        this.apiKeys = source["apiKeys"];
	        this.models = this.convertValues(source["models"], ModelDraft);
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
	export class ServiceSettings {
	    listenMode: string;
	    customHost: string;
	    port: number;
	    clientApiKeys: string[];
	    proxyUrl: string;
	    routingStrategy: string;
	    debug: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServiceSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listenMode = source["listenMode"];
	        this.customHost = source["customHost"];
	        this.port = source["port"];
	        this.clientApiKeys = source["clientApiKeys"];
	        this.proxyUrl = source["proxyUrl"];
	        this.routingStrategy = source["routingStrategy"];
	        this.debug = source["debug"];
	    }
	}
	export class Status {
	    running: boolean;
	    action: string;
	    address: string;
	    allInterfaces: boolean;
	    restartRequired: boolean;
	    savedAddress: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.action = source["action"];
	        this.address = source["address"];
	        this.allInterfaces = source["allInterfaces"];
	        this.restartRequired = source["restartRequired"];
	        this.savedAddress = source["savedAddress"];
	        this.error = source["error"];
	    }
	}

}


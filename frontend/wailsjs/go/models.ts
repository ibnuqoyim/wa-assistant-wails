export namespace whatsapp {
	
	export class AutoReplyConfig {
	    enabled: boolean;
	    ai_provider: string;
	    openai_api_key: string;
	    openai_model: string;
	    ollama_url: string;
	    ollama_model: string;
	    whitelist_numbers: string[];
	    system_prompt: string;
	    response_delay: number;
	    max_response_length: number;
	
	    static createFrom(source: any = {}) {
	        return new AutoReplyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.ai_provider = source["ai_provider"];
	        this.openai_api_key = source["openai_api_key"];
	        this.openai_model = source["openai_model"];
	        this.ollama_url = source["ollama_url"];
	        this.ollama_model = source["ollama_model"];
	        this.whitelist_numbers = source["whitelist_numbers"];
	        this.system_prompt = source["system_prompt"];
	        this.response_delay = source["response_delay"];
	        this.max_response_length = source["max_response_length"];
	    }
	}
	export class Chat {
	    id: string;
	    name: string;
	    last: string;
	    time: string;
	    unread: number;
	    isGroup: boolean;
	    avatar?: string;
	
	    static createFrom(source: any = {}) {
	        return new Chat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.last = source["last"];
	        this.time = source["time"];
	        this.unread = source["unread"];
	        this.isGroup = source["isGroup"];
	        this.avatar = source["avatar"];
	    }
	}
	export class ConnectionStatus {
	    isConnected: boolean;
	    deviceId?: string;
	    pushName?: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isConnected = source["isConnected"];
	        this.deviceId = source["deviceId"];
	        this.pushName = source["pushName"];
	    }
	}
	export class Contact {
	    jid: string;
	    name: string;
	    phoneNumber: string;
	    pushName: string;
	    businessName: string;
	    profilePicUrl: string;
	    isGroup: boolean;
	    isBusiness: boolean;
	    lastSeen: string;
	
	    static createFrom(source: any = {}) {
	        return new Contact(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.phoneNumber = source["phoneNumber"];
	        this.pushName = source["pushName"];
	        this.businessName = source["businessName"];
	        this.profilePicUrl = source["profilePicUrl"];
	        this.isGroup = source["isGroup"];
	        this.isBusiness = source["isBusiness"];
	        this.lastSeen = source["lastSeen"];
	    }
	}
	export class GroupInfo {
	    name: string;
	    description: string;
	    owner: string;
	    // Go type: time
	    createdAt: any;
	    memberCount: number;
	
	    static createFrom(source: any = {}) {
	        return new GroupInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.owner = source["owner"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.memberCount = source["memberCount"];
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
	export class ContactInfo {
	    jid: string;
	    name: string;
	    phoneNumber: string;
	    pushName: string;
	    businessName: string;
	    profilePicUrl: string;
	    status: string;
	    isGroup: boolean;
	    isBusiness: boolean;
	    isBlocked: boolean;
	    lastSeen: string;
	    groupInfo?: GroupInfo;
	
	    static createFrom(source: any = {}) {
	        return new ContactInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.phoneNumber = source["phoneNumber"];
	        this.pushName = source["pushName"];
	        this.businessName = source["businessName"];
	        this.profilePicUrl = source["profilePicUrl"];
	        this.status = source["status"];
	        this.isGroup = source["isGroup"];
	        this.isBusiness = source["isBusiness"];
	        this.isBlocked = source["isBlocked"];
	        this.lastSeen = source["lastSeen"];
	        this.groupInfo = this.convertValues(source["groupInfo"], GroupInfo);
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
	
	export class Message {
	    id: string;
	    chatId: string;
	    author: string;
	    text: string;
	    time: string;
	    mine: boolean;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.chatId = source["chatId"];
	        this.author = source["author"];
	        this.text = source["text"];
	        this.time = source["time"];
	        this.mine = source["mine"];
	        this.type = source["type"];
	    }
	}
	export class StoryConfig {
	    backgroundColor?: string;
	    font?: string;
	    fontSize?: number;
	    textColor?: string;
	    duration?: number;
	
	    static createFrom(source: any = {}) {
	        return new StoryConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backgroundColor = source["backgroundColor"];
	        this.font = source["font"];
	        this.fontSize = source["fontSize"];
	        this.textColor = source["textColor"];
	        this.duration = source["duration"];
	    }
	}
	export class TaskContent {
	    text?: string;
	    mediaPath?: string;
	    mediaType?: string;
	    caption?: string;
	    variables?: Record<string, string>;
	    statusType?: string;
	    storyConfig?: StoryConfig;
	
	    static createFrom(source: any = {}) {
	        return new TaskContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.mediaPath = source["mediaPath"];
	        this.mediaType = source["mediaType"];
	        this.caption = source["caption"];
	        this.variables = source["variables"];
	        this.statusType = source["statusType"];
	        this.storyConfig = this.convertValues(source["storyConfig"], StoryConfig);
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
	export class ScheduledTask {
	    id: string;
	    name: string;
	    type: string;
	    status: string;
	    cronExpr: string;
	    recipients: string[];
	    content: TaskContent;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    // Go type: time
	    nextRun?: any;
	    // Go type: time
	    lastRun?: any;
	    runCount: number;
	    maxRuns?: number;
	    isActive: boolean;
	    errorMsg?: string;
	
	    static createFrom(source: any = {}) {
	        return new ScheduledTask(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.cronExpr = source["cronExpr"];
	        this.recipients = source["recipients"];
	        this.content = this.convertValues(source["content"], TaskContent);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.nextRun = this.convertValues(source["nextRun"], null);
	        this.lastRun = this.convertValues(source["lastRun"], null);
	        this.runCount = source["runCount"];
	        this.maxRuns = source["maxRuns"];
	        this.isActive = source["isActive"];
	        this.errorMsg = source["errorMsg"];
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


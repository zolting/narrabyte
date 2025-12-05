export namespace git {
	
	export class Repository {
	    Storer: any;
	
	    static createFrom(source: any = {}) {
	        return new Repository(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Storer = source["Storer"];
	    }
	}

}

export namespace models {
	
	export class AppSettings {
	    ID: number;
	    Version: number;
	    Theme: string;
	    Locale: string;
	    DefaultModelKey: string;
	    UpdatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Version = source["Version"];
	        this.Theme = source["Theme"];
	        this.Locale = source["Locale"];
	        this.DefaultModelKey = source["DefaultModelKey"];
	        this.UpdatedAt = source["UpdatedAt"];
	    }
	}
	export class BranchInfo {
	    name: string;
	    // Go type: time
	    lastCommitDate: any;
	
	    static createFrom(source: any = {}) {
	        return new BranchInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.lastCommitDate = this.convertValues(source["lastCommitDate"], null);
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
	export class ChatMessage {
	    role: string;
	    content: string;
	    createdAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class DocChangedFile {
	    path: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new DocChangedFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.status = source["status"];
	    }
	}
	export class DocGenerationResult {
	    branch: string;
	    targetBranch: string;
	    docsBranch: string;
	    docsInCodeRepo: boolean;
	    files: DocChangedFile[];
	    diff: string;
	    summary: string;
	    chatMessages?: ChatMessage[];
	
	    static createFrom(source: any = {}) {
	        return new DocGenerationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.branch = source["branch"];
	        this.targetBranch = source["targetBranch"];
	        this.docsBranch = source["docsBranch"];
	        this.docsInCodeRepo = source["docsInCodeRepo"];
	        this.files = this.convertValues(source["files"], DocChangedFile);
	        this.diff = source["diff"];
	        this.summary = source["summary"];
	        this.chatMessages = this.convertValues(source["chatMessages"], ChatMessage);
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
	export class GenerationSession {
	    ID: number;
	    ProjectID: number;
	    SourceBranch: string;
	    TargetBranch: string;
	    Provider: string;
	    ModelKey: string;
	    DocsBranch: string;
	    MessagesJSON: string;
	    ChatMessagesJSON: string;
	    // Go type: time
	    CreatedAt: any;
	    // Go type: time
	    UpdatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new GenerationSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.ProjectID = source["ProjectID"];
	        this.SourceBranch = source["SourceBranch"];
	        this.TargetBranch = source["TargetBranch"];
	        this.Provider = source["Provider"];
	        this.ModelKey = source["ModelKey"];
	        this.DocsBranch = source["DocsBranch"];
	        this.MessagesJSON = source["MessagesJSON"];
	        this.ChatMessagesJSON = source["ChatMessagesJSON"];
	        this.CreatedAt = this.convertValues(source["CreatedAt"], null);
	        this.UpdatedAt = this.convertValues(source["UpdatedAt"], null);
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
	export class LLMModel {
	    key: string;
	    displayName: string;
	    apiName: string;
	    providerId: string;
	    providerName: string;
	    reasoningEffort?: string;
	    thinking?: boolean;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LLMModel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.displayName = source["displayName"];
	        this.apiName = source["apiName"];
	        this.providerId = source["providerId"];
	        this.providerName = source["providerName"];
	        this.reasoningEffort = source["reasoningEffort"];
	        this.thinking = source["thinking"];
	        this.enabled = source["enabled"];
	    }
	}
	export class LLMModelGroup {
	    providerId: string;
	    providerName: string;
	    models: LLMModel[];
	
	    static createFrom(source: any = {}) {
	        return new LLMModelGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerId = source["providerId"];
	        this.providerName = source["providerName"];
	        this.models = this.convertValues(source["models"], LLMModel);
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
	export class RepoLink {
	    ID: number;
	    DocumentationRepo: string;
	    CodebaseRepo: string;
	    ProjectName: string;
	    DocumentationBaseBranch: string;
	    index: number;
	
	    static createFrom(source: any = {}) {
	        return new RepoLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.DocumentationRepo = source["DocumentationRepo"];
	        this.CodebaseRepo = source["CodebaseRepo"];
	        this.ProjectName = source["ProjectName"];
	        this.DocumentationBaseBranch = source["DocumentationBaseBranch"];
	        this.index = source["index"];
	    }
	}
	export class RepoLinkOrderUpdate {
	    ID: number;
	    Index: number;
	
	    static createFrom(source: any = {}) {
	        return new RepoLinkOrderUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Index = source["Index"];
	    }
	}
	export class Template {
	    id: number;
	    name: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new Template(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.content = source["content"];
	    }
	}

}

export namespace services {
	
	export class DirectoryValidationResult {
	    isValid: boolean;
	    errorCode: string;
	
	    static createFrom(source: any = {}) {
	        return new DirectoryValidationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isValid = source["isValid"];
	        this.errorCode = source["errorCode"];
	    }
	}
	export class SessionInfo {
	    projectId: number;
	    sourceBranch: string;
	    targetBranch: string;
	    modelKey: string;
	    provider: string;
	    docsBranch: string;
	    inTab: boolean;
	    isRunning: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.sourceBranch = source["sourceBranch"];
	        this.targetBranch = source["targetBranch"];
	        this.modelKey = source["modelKey"];
	        this.provider = source["provider"];
	        this.docsBranch = source["docsBranch"];
	        this.inTab = source["inTab"];
	        this.isRunning = source["isRunning"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}


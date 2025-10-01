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
	    files: DocChangedFile[];
	    diff: string;
	    summary: string;
	
	    static createFrom(source: any = {}) {
	        return new DocGenerationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.branch = source["branch"];
	        this.files = this.convertValues(source["files"], DocChangedFile);
	        this.diff = source["diff"];
	        this.summary = source["summary"];
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

}


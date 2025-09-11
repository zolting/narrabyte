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
	export class RepoLink {
	    ID: number;
	    DocumentationRepo: string;
	    CodebaseRepo: string;
	    ProjectName: string;
	
	    static createFrom(source: any = {}) {
	        return new RepoLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.DocumentationRepo = source["DocumentationRepo"];
	        this.CodebaseRepo = source["CodebaseRepo"];
	        this.ProjectName = source["ProjectName"];
	    }
	}

}


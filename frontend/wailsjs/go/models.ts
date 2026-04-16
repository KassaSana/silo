export namespace blocker {
	
	export class BlockedAttempt {
	    name: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new BlockedAttempt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.timestamp = source["timestamp"];
	    }
	}

}

export namespace session {
	
	export class Session {
	    id: string;
	    workspace_id: string;
	    workspace_name: string;
	    task_description: string;
	    first_step: string;
	    commit_message: string;
	    lock_type: string;
	    lock_chars: number;
	    lock_text: string;
	    duration_planned: number;
	    started_at: string;
	    status: string;
	    breach_attempts: number;
	    exceptions: string[];
	    obsidian_vault: string;
	    obsidian_note: string;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workspace_id = source["workspace_id"];
	        this.workspace_name = source["workspace_name"];
	        this.task_description = source["task_description"];
	        this.first_step = source["first_step"];
	        this.commit_message = source["commit_message"];
	        this.lock_type = source["lock_type"];
	        this.lock_chars = source["lock_chars"];
	        this.lock_text = source["lock_text"];
	        this.duration_planned = source["duration_planned"];
	        this.started_at = source["started_at"];
	        this.status = source["status"];
	        this.breach_attempts = source["breach_attempts"];
	        this.exceptions = source["exceptions"];
	        this.obsidian_vault = source["obsidian_vault"];
	        this.obsidian_note = source["obsidian_note"];
	    }
	}

}

export namespace stats {
	
	export class Session {
	    id: string;
	    workspace_id: string;
	    workspace_name: string;
	    task_description: string;
	    commit_message: string;
	    duration_actual: number;
	    duration_planned: number;
	    started_at: string;
	    completed_at: string;
	    status: string;
	    breach_attempts: number;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workspace_id = source["workspace_id"];
	        this.workspace_name = source["workspace_name"];
	        this.task_description = source["task_description"];
	        this.commit_message = source["commit_message"];
	        this.duration_actual = source["duration_actual"];
	        this.duration_planned = source["duration_planned"];
	        this.started_at = source["started_at"];
	        this.completed_at = source["completed_at"];
	        this.status = source["status"];
	        this.breach_attempts = source["breach_attempts"];
	    }
	}
	export class Summary {
	    today_minutes: number;
	    week_minutes: number;
	    streak_days: number;
	    total_sessions: number;
	    total_focus_minutes: number;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.today_minutes = source["today_minutes"];
	        this.week_minutes = source["week_minutes"];
	        this.streak_days = source["streak_days"];
	        this.total_sessions = source["total_sessions"];
	        this.total_focus_minutes = source["total_focus_minutes"];
	    }
	}

}

export namespace workspace {
	
	export class Template {
	    name: string;
	    description: string;
	    apps: string[];
	    sites: string[];
	
	    static createFrom(source: any = {}) {
	        return new Template(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.apps = source["apps"];
	        this.sites = source["sites"];
	    }
	}
	export class Workspace {
	    id: string;
	    name: string;
	    allowed_apps: string[];
	    allowed_sites: string[];
	    obsidian_vault: string;
	    obsidian_note: string;
	    template_source: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Workspace(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.allowed_apps = source["allowed_apps"];
	        this.allowed_sites = source["allowed_sites"];
	        this.obsidian_vault = source["obsidian_vault"];
	        this.obsidian_note = source["obsidian_note"];
	        this.template_source = source["template_source"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}

}


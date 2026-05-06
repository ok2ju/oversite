export namespace main {
	
	export class Demo {
	    id: number;
	    map_name: string;
	    file_path: string;
	    file_size: number;
	    status: string;
	    total_ticks: number;
	    tick_rate: number;
	    duration_secs: number;
	    match_date: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Demo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.map_name = source["map_name"];
	        this.file_path = source["file_path"];
	        this.file_size = source["file_size"];
	        this.status = source["status"];
	        this.total_ticks = source["total_ticks"];
	        this.tick_rate = source["tick_rate"];
	        this.duration_secs = source["duration_secs"];
	        this.match_date = source["match_date"];
	        this.created_at = source["created_at"];
	    }
	}
	export class PaginationMeta {
	    total: number;
	    page: number;
	    per_page: number;
	
	    static createFrom(source: any = {}) {
	        return new PaginationMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.per_page = source["per_page"];
	    }
	}
	export class DemoListResult {
	    data: Demo[];
	    meta: PaginationMeta;
	
	    static createFrom(source: any = {}) {
	        return new DemoListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], Demo);
	        this.meta = this.convertValues(source["meta"], PaginationMeta);
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
	export class GameEvent {
	    id: string;
	    demo_id: string;
	    round_id?: string;
	    tick: number;
	    event_type: string;
	    attacker_steam_id?: string;
	    victim_steam_id?: string;
	    weapon?: string;
	    x?: number;
	    y?: number;
	    z?: number;
	    extra_data: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new GameEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.demo_id = source["demo_id"];
	        this.round_id = source["round_id"];
	        this.tick = source["tick"];
	        this.event_type = source["event_type"];
	        this.attacker_steam_id = source["attacker_steam_id"];
	        this.victim_steam_id = source["victim_steam_id"];
	        this.weapon = source["weapon"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.z = source["z"];
	        this.extra_data = source["extra_data"];
	    }
	}
	export class HeatmapPoint {
	    x: number;
	    y: number;
	    kill_count: number;
	
	    static createFrom(source: any = {}) {
	        return new HeatmapPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	        this.kill_count = source["kill_count"];
	    }
	}
	
	export class PlayerInfo {
	    steam_id: string;
	    player_name: string;
	
	    static createFrom(source: any = {}) {
	        return new PlayerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steam_id = source["steam_id"];
	        this.player_name = source["player_name"];
	    }
	}
	export class PlayerRosterEntry {
	    steam_id: string;
	    player_name: string;
	    team_side: string;
	
	    static createFrom(source: any = {}) {
	        return new PlayerRosterEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steam_id = source["steam_id"];
	        this.player_name = source["player_name"];
	        this.team_side = source["team_side"];
	    }
	}
	export class Round {
	    id: string;
	    round_number: number;
	    start_tick: number;
	    freeze_end_tick: number;
	    end_tick: number;
	    winner_side: string;
	    win_reason: string;
	    ct_score: number;
	    t_score: number;
	    is_overtime: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Round(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.round_number = source["round_number"];
	        this.start_tick = source["start_tick"];
	        this.freeze_end_tick = source["freeze_end_tick"];
	        this.end_tick = source["end_tick"];
	        this.winner_side = source["winner_side"];
	        this.win_reason = source["win_reason"];
	        this.ct_score = source["ct_score"];
	        this.t_score = source["t_score"];
	        this.is_overtime = source["is_overtime"];
	    }
	}
	export class ScoreboardEntry {
	    steam_id: string;
	    player_name: string;
	    team_side: string;
	    kills: number;
	    deaths: number;
	    assists: number;
	    damage: number;
	    hs_kills: number;
	    rounds_played: number;
	    hs_percent: number;
	    adr: number;
	
	    static createFrom(source: any = {}) {
	        return new ScoreboardEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steam_id = source["steam_id"];
	        this.player_name = source["player_name"];
	        this.team_side = source["team_side"];
	        this.kills = source["kills"];
	        this.deaths = source["deaths"];
	        this.assists = source["assists"];
	        this.damage = source["damage"];
	        this.hs_kills = source["hs_kills"];
	        this.rounds_played = source["rounds_played"];
	        this.hs_percent = source["hs_percent"];
	        this.adr = source["adr"];
	    }
	}
	export class TickData {
	    tick: number;
	    steam_id: string;
	    x: number;
	    y: number;
	    z: number;
	    yaw: number;
	    health: number;
	    armor: number;
	    is_alive: boolean;
	    weapon?: string;
	
	    static createFrom(source: any = {}) {
	        return new TickData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tick = source["tick"];
	        this.steam_id = source["steam_id"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.z = source["z"];
	        this.yaw = source["yaw"];
	        this.health = source["health"];
	        this.armor = source["armor"];
	        this.is_alive = source["is_alive"];
	        this.weapon = source["weapon"];
	    }
	}
	export class WeaponStat {
	    weapon: string;
	    kill_count: number;
	    hs_count: number;
	
	    static createFrom(source: any = {}) {
	        return new WeaponStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.weapon = source["weapon"];
	        this.kill_count = source["kill_count"];
	        this.hs_count = source["hs_count"];
	    }
	}

}


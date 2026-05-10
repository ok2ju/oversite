export namespace main {
	
	export class AnalysisStatus {
	    demo_id: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.demo_id = source["demo_id"];
	        this.status = source["status"];
	    }
	}
	export class DamageByOpponent {
	    steam_id: string;
	    player_name: string;
	    team_side: string;
	    damage: number;
	
	    static createFrom(source: any = {}) {
	        return new DamageByOpponent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steam_id = source["steam_id"];
	        this.player_name = source["player_name"];
	        this.team_side = source["team_side"];
	        this.damage = source["damage"];
	    }
	}
	export class DamageByWeapon {
	    weapon: string;
	    damage: number;
	
	    static createFrom(source: any = {}) {
	        return new DamageByWeapon(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.weapon = source["weapon"];
	        this.damage = source["damage"];
	    }
	}
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
	export class DemoSummary {
	    id: number;
	    map_name: string;
	    file_name: string;
	    file_size: number;
	    status: string;
	    total_ticks: number;
	    tick_rate: number;
	    duration_secs: number;
	    match_date: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new DemoSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.map_name = source["map_name"];
	        this.file_name = source["file_name"];
	        this.file_size = source["file_size"];
	        this.status = source["status"];
	        this.total_ticks = source["total_ticks"];
	        this.tick_rate = source["tick_rate"];
	        this.duration_secs = source["duration_secs"];
	        this.match_date = source["match_date"];
	        this.created_at = source["created_at"];
	    }
	}
	export class DemoListResult {
	    data: DemoSummary[];
	    meta: PaginationMeta;
	
	    static createFrom(source: any = {}) {
	        return new DemoListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = this.convertValues(source["data"], DemoSummary);
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
	    headshot: boolean;
	    assister_steam_id?: string;
	    health_damage: number;
	    attacker_name: string;
	    victim_name: string;
	    attacker_team: string;
	    victim_team: string;
	    extra_data: number[];
	
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
	        this.headshot = source["headshot"];
	        this.assister_steam_id = source["assister_steam_id"];
	        this.health_damage = source["health_damage"];
	        this.attacker_name = source["attacker_name"];
	        this.victim_name = source["victim_name"];
	        this.attacker_team = source["attacker_team"];
	        this.victim_team = source["victim_team"];
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
	export class HitGroupBreakdown {
	    hit_group: number;
	    label: string;
	    damage: number;
	    hits: number;
	
	    static createFrom(source: any = {}) {
	        return new HitGroupBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hit_group = source["hit_group"];
	        this.label = source["label"];
	        this.damage = source["damage"];
	        this.hits = source["hits"];
	    }
	}
	export class PlayerHighlight {
	    steam_id: string;
	    category: string;
	    metric_name: string;
	    metric_value: number;
	
	    static createFrom(source: any = {}) {
	        return new PlayerHighlight(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steam_id = source["steam_id"];
	        this.category = source["category"];
	        this.metric_name = source["metric_name"];
	        this.metric_value = source["metric_value"];
	    }
	}
	export class TeamSummary {
	    side: string;
	    players: number;
	    avg_overall_score: number;
	    avg_trade_pct: number;
	    avg_standing_shot_pct: number;
	    avg_counter_strafe_pct: number;
	    avg_first_shot_acc_pct: number;
	    total_flash_assists: number;
	    total_smokes_kill_assist: number;
	    total_he_damage: number;
	    total_isolated_peek_deaths: number;
	    total_eco_kills: number;
	    avg_full_buy_adr: number;
	
	    static createFrom(source: any = {}) {
	        return new TeamSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.side = source["side"];
	        this.players = source["players"];
	        this.avg_overall_score = source["avg_overall_score"];
	        this.avg_trade_pct = source["avg_trade_pct"];
	        this.avg_standing_shot_pct = source["avg_standing_shot_pct"];
	        this.avg_counter_strafe_pct = source["avg_counter_strafe_pct"];
	        this.avg_first_shot_acc_pct = source["avg_first_shot_acc_pct"];
	        this.total_flash_assists = source["total_flash_assists"];
	        this.total_smokes_kill_assist = source["total_smokes_kill_assist"];
	        this.total_he_damage = source["total_he_damage"];
	        this.total_isolated_peek_deaths = source["total_isolated_peek_deaths"];
	        this.total_eco_kills = source["total_eco_kills"];
	        this.avg_full_buy_adr = source["avg_full_buy_adr"];
	    }
	}
	export class MatchInsights {
	    demo_id: string;
	    ct_summary: TeamSummary;
	    t_summary: TeamSummary;
	    standouts: PlayerHighlight[];
	
	    static createFrom(source: any = {}) {
	        return new MatchInsights(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.demo_id = source["demo_id"];
	        this.ct_summary = this.convertValues(source["ct_summary"], TeamSummary);
	        this.t_summary = this.convertValues(source["t_summary"], TeamSummary);
	        this.standouts = this.convertValues(source["standouts"], PlayerHighlight);
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
	export class MistakeEntry {
	    id: number;
	    kind: string;
	    category: string;
	    severity: number;
	    title: string;
	    suggestion: string;
	    round_number: number;
	    tick: number;
	    steam_id: string;
	    extras: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new MistakeEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.category = source["category"];
	        this.severity = source["severity"];
	        this.title = source["title"];
	        this.suggestion = source["suggestion"];
	        this.round_number = source["round_number"];
	        this.tick = source["tick"];
	        this.steam_id = source["steam_id"];
	        this.extras = source["extras"];
	    }
	}
	export class MistakeContext {
	    entry: MistakeEntry;
	    round_start_tick: number;
	    round_end_tick: number;
	    freeze_end_tick: number;
	
	    static createFrom(source: any = {}) {
	        return new MistakeContext(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry = this.convertValues(source["entry"], MistakeEntry);
	        this.round_start_tick = source["round_start_tick"];
	        this.round_end_tick = source["round_end_tick"];
	        this.freeze_end_tick = source["freeze_end_tick"];
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
	
	export class MovementStats {
	    distance_units: number;
	    avg_speed_ups: number;
	    max_speed_ups: number;
	    strafe_percent: number;
	    stationary_ratio: number;
	    walking_ratio: number;
	    running_ratio: number;
	
	    static createFrom(source: any = {}) {
	        return new MovementStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.distance_units = source["distance_units"];
	        this.avg_speed_ups = source["avg_speed_ups"];
	        this.max_speed_ups = source["max_speed_ups"];
	        this.strafe_percent = source["strafe_percent"];
	        this.stationary_ratio = source["stationary_ratio"];
	        this.walking_ratio = source["walking_ratio"];
	        this.running_ratio = source["running_ratio"];
	    }
	}
	
	export class PlayerAnalysis {
	    steam_id: string;
	    overall_score: number;
	    version: number;
	    trade_pct: number;
	    avg_trade_ticks: number;
	    crosshair_height_avg_off: number;
	    time_to_fire_ms_avg: number;
	    flick_count: number;
	    flick_hit_pct: number;
	    first_shot_acc_pct: number;
	    spray_decay_slope: number;
	    standing_shot_pct: number;
	    counter_strafe_pct: number;
	    smokes_thrown: number;
	    smokes_kill_assist: number;
	    flash_assists: number;
	    he_damage: number;
	    nades_unused: number;
	    isolated_peek_deaths: number;
	    repeated_death_zones: number;
	    full_buy_adr: number;
	    eco_kills: number;
	    extras: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new PlayerAnalysis(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steam_id = source["steam_id"];
	        this.overall_score = source["overall_score"];
	        this.version = source["version"];
	        this.trade_pct = source["trade_pct"];
	        this.avg_trade_ticks = source["avg_trade_ticks"];
	        this.crosshair_height_avg_off = source["crosshair_height_avg_off"];
	        this.time_to_fire_ms_avg = source["time_to_fire_ms_avg"];
	        this.flick_count = source["flick_count"];
	        this.flick_hit_pct = source["flick_hit_pct"];
	        this.first_shot_acc_pct = source["first_shot_acc_pct"];
	        this.spray_decay_slope = source["spray_decay_slope"];
	        this.standing_shot_pct = source["standing_shot_pct"];
	        this.counter_strafe_pct = source["counter_strafe_pct"];
	        this.smokes_thrown = source["smokes_thrown"];
	        this.smokes_kill_assist = source["smokes_kill_assist"];
	        this.flash_assists = source["flash_assists"];
	        this.he_damage = source["he_damage"];
	        this.nades_unused = source["nades_unused"];
	        this.isolated_peek_deaths = source["isolated_peek_deaths"];
	        this.repeated_death_zones = source["repeated_death_zones"];
	        this.full_buy_adr = source["full_buy_adr"];
	        this.eco_kills = source["eco_kills"];
	        this.extras = source["extras"];
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
	export class UtilityStats {
	    flashes_thrown: number;
	    smokes_thrown: number;
	    hes_thrown: number;
	    molotovs_thrown: number;
	    decoys_thrown: number;
	    flash_assists: number;
	    blind_time_inflicted_secs: number;
	    enemies_flashed: number;
	
	    static createFrom(source: any = {}) {
	        return new UtilityStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.flashes_thrown = source["flashes_thrown"];
	        this.smokes_thrown = source["smokes_thrown"];
	        this.hes_thrown = source["hes_thrown"];
	        this.molotovs_thrown = source["molotovs_thrown"];
	        this.decoys_thrown = source["decoys_thrown"];
	        this.flash_assists = source["flash_assists"];
	        this.blind_time_inflicted_secs = source["blind_time_inflicted_secs"];
	        this.enemies_flashed = source["enemies_flashed"];
	    }
	}
	export class TimingStats {
	    avg_time_to_first_contact_secs: number;
	    avg_alive_duration_secs: number;
	    time_on_site_a_secs: number;
	    time_on_site_b_secs: number;
	
	    static createFrom(source: any = {}) {
	        return new TimingStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.avg_time_to_first_contact_secs = source["avg_time_to_first_contact_secs"];
	        this.avg_alive_duration_secs = source["avg_alive_duration_secs"];
	        this.time_on_site_a_secs = source["time_on_site_a_secs"];
	        this.time_on_site_b_secs = source["time_on_site_b_secs"];
	    }
	}
	export class PlayerRoundDetail {
	    round_number: number;
	    team_side: string;
	    kills: number;
	    deaths: number;
	    assists: number;
	    damage: number;
	    hs_kills: number;
	    clutch_kills: number;
	    first_kill: boolean;
	    first_death: boolean;
	    trade_kill: boolean;
	    loadout_value: number;
	    distance_units: number;
	    alive_duration_secs: number;
	    time_to_first_contact_sec?: number;
	
	    static createFrom(source: any = {}) {
	        return new PlayerRoundDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.round_number = source["round_number"];
	        this.team_side = source["team_side"];
	        this.kills = source["kills"];
	        this.deaths = source["deaths"];
	        this.assists = source["assists"];
	        this.damage = source["damage"];
	        this.hs_kills = source["hs_kills"];
	        this.clutch_kills = source["clutch_kills"];
	        this.first_kill = source["first_kill"];
	        this.first_death = source["first_death"];
	        this.trade_kill = source["trade_kill"];
	        this.loadout_value = source["loadout_value"];
	        this.distance_units = source["distance_units"];
	        this.alive_duration_secs = source["alive_duration_secs"];
	        this.time_to_first_contact_sec = source["time_to_first_contact_sec"];
	    }
	}
	export class PlayerMatchStats {
	    steam_id: string;
	    player_name: string;
	    team_side: string;
	    rounds_played: number;
	    kills: number;
	    deaths: number;
	    assists: number;
	    damage: number;
	    hs_kills: number;
	    clutch_kills: number;
	    first_kills: number;
	    first_deaths: number;
	    opening_wins: number;
	    opening_losses: number;
	    trade_kills: number;
	    hs_percent: number;
	    adr: number;
	    damage_by_weapon: DamageByWeapon[];
	    damage_by_opponent: DamageByOpponent[];
	    rounds: PlayerRoundDetail[];
	    movement: MovementStats;
	    timings: TimingStats;
	    utility: UtilityStats;
	    hit_groups: HitGroupBreakdown[];
	
	    static createFrom(source: any = {}) {
	        return new PlayerMatchStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steam_id = source["steam_id"];
	        this.player_name = source["player_name"];
	        this.team_side = source["team_side"];
	        this.rounds_played = source["rounds_played"];
	        this.kills = source["kills"];
	        this.deaths = source["deaths"];
	        this.assists = source["assists"];
	        this.damage = source["damage"];
	        this.hs_kills = source["hs_kills"];
	        this.clutch_kills = source["clutch_kills"];
	        this.first_kills = source["first_kills"];
	        this.first_deaths = source["first_deaths"];
	        this.opening_wins = source["opening_wins"];
	        this.opening_losses = source["opening_losses"];
	        this.trade_kills = source["trade_kills"];
	        this.hs_percent = source["hs_percent"];
	        this.adr = source["adr"];
	        this.damage_by_weapon = this.convertValues(source["damage_by_weapon"], DamageByWeapon);
	        this.damage_by_opponent = this.convertValues(source["damage_by_opponent"], DamageByOpponent);
	        this.rounds = this.convertValues(source["rounds"], PlayerRoundDetail);
	        this.movement = this.convertValues(source["movement"], MovementStats);
	        this.timings = this.convertValues(source["timings"], TimingStats);
	        this.utility = this.convertValues(source["utility"], UtilityStats);
	        this.hit_groups = this.convertValues(source["hit_groups"], HitGroupBreakdown);
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
	
	export class PlayerRoundEntry {
	    steam_id: string;
	    round_number: number;
	    trade_pct: number;
	    buy_type: string;
	    money_spent: number;
	    nades_used: number;
	    nades_unused: number;
	    shots_fired: number;
	    shots_hit: number;
	    extras: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new PlayerRoundEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.steam_id = source["steam_id"];
	        this.round_number = source["round_number"];
	        this.trade_pct = source["trade_pct"];
	        this.buy_type = source["buy_type"];
	        this.money_spent = source["money_spent"];
	        this.nades_used = source["nades_used"];
	        this.nades_unused = source["nades_unused"];
	        this.shots_fired = source["shots_fired"];
	        this.shots_hit = source["shots_hit"];
	        this.extras = source["extras"];
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
	    ct_team_name: string;
	    t_team_name: string;
	
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
	        this.ct_team_name = source["ct_team_name"];
	        this.t_team_name = source["t_team_name"];
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
	    money: number;
	    has_helmet: boolean;
	    has_defuser: boolean;
	    ammo_clip: number;
	    ammo_reserve: number;
	
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
	        this.money = source["money"];
	        this.has_helmet = source["has_helmet"];
	        this.has_defuser = source["has_defuser"];
	        this.ammo_clip = source["ammo_clip"];
	        this.ammo_reserve = source["ammo_reserve"];
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


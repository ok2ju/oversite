package detectors

// WeaponInfo describes the magazine economics of one CS2 weapon. Only
// fields current detectors consume are listed; v2 rules (spray_decay,
// etc.) can grow the struct without breaking callers.
type WeaponInfo struct {
	MaxClip int  // full magazine capacity in rounds
	IsAuto  bool // sprayable rifles + SMGs true; AWP/Scout/Pistols false
}

// WeaponCatalog is the lookup table for WeaponInfo by demoinfocs
// weapon name (the same strings that appear in GameEvent.Weapon).
type WeaponCatalog map[string]WeaponInfo

// DefaultWeaponCatalog returns the static lookup used in v1. The map is
// allocated fresh on every call (no shared mutable state); callers
// typically build it once per demo and hold the value in DetectorCtx.
//
// Weapon names mirror those produced by demoinfocs v5 (the parser
// passes the engine class name through unmodified to GameEvent.Weapon).
func DefaultWeaponCatalog() WeaponCatalog {
	return WeaponCatalog{
		// Rifles
		"ak47":          {MaxClip: 30, IsAuto: true},
		"m4a1":          {MaxClip: 30, IsAuto: true},
		"m4a1_silencer": {MaxClip: 20, IsAuto: true},
		"aug":           {MaxClip: 30, IsAuto: true},
		"sg556":         {MaxClip: 30, IsAuto: true},
		"famas":         {MaxClip: 25, IsAuto: true},
		"galilar":       {MaxClip: 35, IsAuto: true},

		// Snipers
		"awp":    {MaxClip: 5, IsAuto: false},
		"ssg08":  {MaxClip: 10, IsAuto: false},
		"scar20": {MaxClip: 20, IsAuto: false},
		"g3sg1":  {MaxClip: 20, IsAuto: false},

		// SMGs
		"mp9":   {MaxClip: 30, IsAuto: true},
		"mp7":   {MaxClip: 30, IsAuto: true},
		"mp5sd": {MaxClip: 30, IsAuto: true},
		"ump45": {MaxClip: 25, IsAuto: true},
		"p90":   {MaxClip: 50, IsAuto: true},
		"bizon": {MaxClip: 64, IsAuto: true},
		"mac10": {MaxClip: 30, IsAuto: true},

		// Heavies
		"negev": {MaxClip: 150, IsAuto: true},
		"m249":  {MaxClip: 100, IsAuto: true},

		// Shotguns
		"nova":     {MaxClip: 8, IsAuto: false},
		"xm1014":   {MaxClip: 7, IsAuto: true},
		"mag7":     {MaxClip: 5, IsAuto: false},
		"sawedoff": {MaxClip: 7, IsAuto: false},

		// Pistols
		"deagle":       {MaxClip: 7, IsAuto: false},
		"fiveseven":    {MaxClip: 20, IsAuto: false},
		"tec9":         {MaxClip: 18, IsAuto: false},
		"cz75a":        {MaxClip: 12, IsAuto: true},
		"usp_silencer": {MaxClip: 12, IsAuto: false},
		"hkp2000":      {MaxClip: 13, IsAuto: false},
		"glock":        {MaxClip: 20, IsAuto: false},
		"elite":        {MaxClip: 30, IsAuto: false}, // dual berettas
		"revolver":     {MaxClip: 8, IsAuto: false},
		"p250":         {MaxClip: 13, IsAuto: false},
	}
}

// Lookup returns the WeaponInfo for name and ok=false when the weapon
// is unknown (knives, grenades, "world"). Callers treat unknown as
// "don't flag" — the surrounding detector returns no findings.
func (c WeaponCatalog) Lookup(name string) (WeaponInfo, bool) {
	if name == "" {
		return WeaponInfo{}, false
	}
	info, ok := c[name]
	return info, ok
}

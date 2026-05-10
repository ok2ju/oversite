export interface MistakeEntry {
  kind: string
  round_number: number
  tick: number
  steam_id: string
  extras: Record<string, unknown> | null
}

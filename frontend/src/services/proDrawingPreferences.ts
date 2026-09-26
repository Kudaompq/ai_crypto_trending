import {
  getChartPreferences,
  saveChartPreferences,
  type SavedChartDrawing
} from './chartPreferences'

export interface CoreOverlaySnapshot {
  id?: string
  name?: string
  points?: Array<{ timestamp?: number; value?: number }>
  styles?: Record<string, unknown>
  visible?: boolean
  lock?: boolean
  mode?: SavedChartDrawing['mode']
  totalStep?: number
}

export interface ProOverlayInput {
  id: string
  name: string
  points: Array<{ timestamp: number; value: number }>
  styles?: Record<string, unknown>
  visible?: boolean
  lock?: boolean
  mode?: SavedChartDrawing['mode']
}

function validAnchor(point: { timestamp?: number; value?: number } | undefined): point is { timestamp: number; value: number } {
  return point !== undefined && Number.isFinite(point.timestamp) && Number.isFinite(point.value)
}

export function serializeCoreOverlays(overlays: CoreOverlaySnapshot[]): SavedChartDrawing[] {
  return overlays.flatMap(overlay => {
    if (!overlay.id || !overlay.name || !Array.isArray(overlay.points)) return []
    const points = overlay.points.filter(validAnchor).map(point => ({ timestamp: point.timestamp, value: point.value }))
    const requiredPoints = overlay.totalStep && overlay.totalStep > 0 ? overlay.totalStep - 1 : 1
    if (points.length < requiredPoints) return []
    return [{
      id: overlay.id,
      name: overlay.name,
      points,
      ...(overlay.styles ? { styles: { ...overlay.styles } } : {}),
      ...(overlay.visible !== undefined ? { visible: overlay.visible } : {}),
      ...(overlay.lock !== undefined ? { lock: overlay.lock } : {}),
      ...(overlay.mode ? { mode: overlay.mode } : {})
    }]
  })
}

export function restoreProOverlays(drawings: SavedChartDrawing[]): ProOverlayInput[] {
  return drawings.map(drawing => ({
    id: drawing.id,
    name: drawing.name,
    points: drawing.points.map(point => ({ timestamp: point.timestamp, value: point.value })),
    ...(drawing.styles ? { styles: { ...drawing.styles } } : {}),
    ...(drawing.visible !== undefined ? { visible: drawing.visible } : {}),
    ...(drawing.lock !== undefined ? { lock: drawing.lock } : {}),
    ...(drawing.mode ? { mode: drawing.mode } : {})
  }))
}

export function persistProDrawings(symbol: string, overlays: CoreOverlaySnapshot[]): boolean {
  const currentPreferences = getChartPreferences(symbol)
  return saveChartPreferences(symbol, {
    ...currentPreferences,
    drawings: serializeCoreOverlays(overlays)
  })
}

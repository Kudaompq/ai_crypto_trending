import { describe, expect, it } from 'vitest'
import { patchLocaleMerge } from '../../scripts/pro-patch-utils.mjs'

describe('Pro locale patch', () => {
  it('adds custom translations without removing the bundled locale strings', () => {
    const source = `const l5 = { 'zh-CN': { indicator: '指标', timezone: '时区' } };
function Fl(e, t) {
  l5[e] = t;
}`

    const patched = patchLocaleMerge(source)
    expect(patchLocaleMerge(patched)).toBe(patched)
    const readLocales = new Function(`${patched}\nFl('zh-CN', { open_interest: 'OI（未平仓持仓量）' });\nreturn l5;`)
    const locales = readLocales() as Record<string, Record<string, string>>

    expect(locales['zh-CN']).toEqual({
      indicator: '指标',
      timezone: '时区',
      open_interest: 'OI（未平仓持仓量）'
    })
  })
})

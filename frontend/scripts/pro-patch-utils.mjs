export function patchLocaleMerge(source) {
  const current = 'function Fl(e, t) {\n  l5[e] = t;\n}'
  const replacement = 'function Fl(e, t) {\n  l5[e] = { ...(l5[e] ?? {}), ...t };\n}'
  if (source.includes(replacement)) return source
  const occurrences = source.split(current).length - 1
  if (occurrences !== 1) {
    throw new Error(`Expected one Pro locale assignment to patch, found ${occurrences}`)
  }
  return source.replace(current, replacement)
}

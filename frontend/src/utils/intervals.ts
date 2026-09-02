export interface Span {
  start: number
  end: number
}

export const mergeSpans = (spans: Span[]): Span[] => {
  const sorted = [...spans].sort((a, b) => a.start - b.start)
  const merged: Span[] = []
  for (const span of sorted) {
    const last = merged[merged.length - 1]
    if (last && span.start <= last.end) {
      last.end = Math.max(last.end, span.end)
    } else {
      merged.push({ ...span })
    }
  }
  return merged
}

export const subtractSpans = (base: Span, blocks: Span[]): Span[] => {
  let remaining: Span[] = [base]

  for (const block of blocks) {
    const next: Span[] = []
    for (const piece of remaining) {
      if (block.end <= piece.start || block.start >= piece.end) {
        next.push(piece)
        continue
      }
      if (block.start > piece.start) {
        next.push({ start: piece.start, end: block.start })
      }
      if (block.end < piece.end) {
        next.push({ start: block.end, end: piece.end })
      }
    }
    remaining = next
  }

  return remaining
}

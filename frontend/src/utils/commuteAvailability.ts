import { mergeSpans, type Span } from './intervals.ts'

export interface CommuteEngagement extends Span {
  branchId: number
}

export const commuteSpansForBranch = (
  engagements: CommuteEngagement[],
  targetBranchId: number | null,
  commuteMinutes: number | null
): Span[] => {
  if (targetBranchId === null || !commuteMinutes || commuteMinutes < 0) return []

  const buffer = commuteMinutes * 60 * 1000
  const spans: Span[] = []
  for (const engagement of engagements) {
    if (engagement.branchId === targetBranchId) continue
    spans.push({ start: engagement.start - buffer, end: engagement.start })
    spans.push({ start: engagement.end, end: engagement.end + buffer })
  }
  return mergeSpans(spans)
}

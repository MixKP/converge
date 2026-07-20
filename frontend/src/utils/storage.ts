export function getValidatedArray<T>(key: string, validator: (value: unknown) => value is T): T[] {
  const stored = localStorage.getItem(key)
  if (!stored) return []

  try {
    const parsed = JSON.parse(stored)
    if (Array.isArray(parsed)) {
      return parsed.filter(validator)
    }
    return []
  } catch {
    return []
  }
}

export function setItem(key: string, value: unknown): void {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch (error) {
    console.error(`Failed to save to localStorage (key: ${key}):`, error)
  }
}

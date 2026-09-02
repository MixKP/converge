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

export function getValidated<T>(key: string, validator: (value: unknown) => value is T): T | null {
  const stored = localStorage.getItem(key)
  if (!stored) return null

  try {
    const parsed = JSON.parse(stored)
    return validator(parsed) ? parsed : null
  } catch {
    return null
  }
}

export function removeItem(key: string): void {
  try {
    localStorage.removeItem(key)
  } catch (error) {
    console.error(`Failed to remove from localStorage (key: ${key}):`, error)
  }
}

export function setItem(key: string, value: unknown): void {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch (error) {
    console.error(`Failed to save to localStorage (key: ${key}):`, error)
  }
}

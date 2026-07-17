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

export function removeItem(key: string): void {
  try {
    localStorage.removeItem(key)
  } catch (error) {
    console.error(`Failed to remove from localStorage (key: ${key}):`, error)
  }
}

export function getRaw(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch (error) {
    console.error(`Failed to get from localStorage (key: ${key}):`, error)
    return null
  }
}

export function hasItem(key: string): boolean {
  try {
    return localStorage.getItem(key) !== null
  } catch {
    return false
  }
}

export function getKeys(): string[] {
  try {
    return Object.keys(localStorage)
  } catch {
    return []
  }
}

export function clearAll(): void {
  try {
    localStorage.clear()
  } catch (error) {
    console.error('Failed to clear localStorage:', error)
  }
}

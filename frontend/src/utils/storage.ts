// LocalStorage utilities with validation

/**
 * Get an array of validated items from localStorage
 * Ensures all items in the array pass validation
 *
 * @param key - The localStorage key
 * @param validator - Type guard function to validate each item
 * @returns Array of validated items or empty array if invalid/not found
 *
 * @example
 * ```ts
 * interface CartItem {
 *   id: number
 *   name: string
 * }
 *
 * const isCartItem = (value: unknown): value is CartItem => {
 *   return typeof value === 'object' && value !== null && 'id' in value && 'name' in value
 * }
 *
 * const items = getValidatedArray('myItems', isCartItem)
 * ```
 */
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

/**
 * Set an item in localStorage
 * Automatically serializes the value to JSON
 *
 * @param key - The localStorage key
 * @param value - The value to store (must be JSON-serializable)
 *
 * @example
 * ```ts
 * setItem('user', { id: 1, name: 'John' })
 * ```
 */
export function setItem(key: string, value: unknown): void {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch (error) {
    console.error(`Failed to save to localStorage (key: ${key}):`, error)
  }
}

/**
 * Remove an item from localStorage
 *
 * @param key - The localStorage key to remove
 *
 * @example
 * ```ts
 * removeItem('expiredToken')
 * ```
 */
export function removeItem(key: string): void {
  try {
    localStorage.removeItem(key)
  } catch (error) {
    console.error(`Failed to remove from localStorage (key: ${key}):`, error)
  }
}

/**
 * Get a raw item from localStorage without parsing
 * Returns the raw string value as stored
 *
 * @param key - The localStorage key
 * @returns The raw string value or null
 *
 * @example
 * ```ts
 * const rawValue = getRaw('preferences')
 * ```
 */
export function getRaw(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch (error) {
    console.error(`Failed to get from localStorage (key: ${key}):`, error)
    return null
  }
}

/**
 * Check if a key exists in localStorage
 *
 * @param key - The localStorage key to check
 * @returns True if the key exists
 *
 * @example
 * ```ts
 * if (hasItem('userSession')) {
 *   // Restore session
 * }
 * ```
 */
export function hasItem(key: string): boolean {
  try {
    return localStorage.getItem(key) !== null
  } catch {
    return false
  }
}

/**
 * Get all keys from localStorage
 *
 * @returns Array of all localStorage keys
 *
 * @example
 * ```ts
 * const allKeys = getKeys()
 * console.log('Stored items:', allKeys)
 * ```
 */
export function getKeys(): string[] {
  try {
    return Object.keys(localStorage)
  } catch {
    return []
  }
}

/**
 * Clear all items from localStorage
 * Use with caution - this removes ALL localStorage data for your domain
 *
 * @example
 * ```ts
 * // On user logout
 * clearAll()
 * ```
 */
export function clearAll(): void {
  try {
    localStorage.clear()
  } catch (error) {
    console.error('Failed to clear localStorage:', error)
  }
}

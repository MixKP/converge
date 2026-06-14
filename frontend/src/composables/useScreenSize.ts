import { ref, onMounted, onUnmounted } from 'vue'

/**
 * Screen size breakpoints in pixels
 */
export const BREAKPOINTS = {
  MOBILE: 425, // ≤425px
  TABLET: 1024, // 426px - 1023px
  DESKTOP: 1024, // ≥1024px
} as const

/**
 * Composable for reactive screen size detection
 *
 * Automatically tracks screen size and updates on window resize.
 * Cleans up event listeners on unmount.
 *
 * @example
 * ```ts
 * const { isMobile, isTablet, isDesktop, screenWidth } = useScreenSize()
 *
 * if (isMobile.value) {
 *   // Show mobile layout
 * } else if (isTablet.value) {
 *   // Show tablet layout
 * } else {
 *   // Show desktop layout
 * }
 * ```
 */
export function useScreenSize() {
  const screenWidth = ref<number>(globalThis.innerWidth)
  const isMobile = ref<boolean>(globalThis.innerWidth <= BREAKPOINTS.MOBILE)
  const isTablet = ref<boolean>(
    globalThis.innerWidth > BREAKPOINTS.MOBILE && globalThis.innerWidth < BREAKPOINTS.TABLET
  )
  const isDesktop = ref<boolean>(globalThis.innerWidth >= BREAKPOINTS.DESKTOP)

  /**
   * Update all screen size reactive values
   * Called automatically on window resize
   */
  const updateScreenSize = () => {
    const width = globalThis.innerWidth
    screenWidth.value = width
    isMobile.value = width <= BREAKPOINTS.MOBILE
    isTablet.value = width > BREAKPOINTS.MOBILE && width < BREAKPOINTS.TABLET
    isDesktop.value = width >= BREAKPOINTS.DESKTOP
  }

  // Set up resize listener on mount
  onMounted(() => {
    updateScreenSize()
    globalThis.addEventListener('resize', updateScreenSize)
  })

  // Clean up resize listener on unmount
  onUnmounted(() => {
    globalThis.removeEventListener('resize', updateScreenSize)
  })

  return {
    screenWidth,
    isMobile,
    isTablet,
    isDesktop,
    updateScreenSize,
  }
}

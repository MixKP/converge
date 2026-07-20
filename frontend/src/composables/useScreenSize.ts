import { ref, onMounted, onUnmounted } from 'vue'

const BREAKPOINTS = {
  MOBILE: 425, // ≤425px
  TABLET: 1024, // 426px - 1023px
  DESKTOP: 1024, // ≥1024px
} as const

export function useScreenSize() {
  const screenWidth = ref<number>(globalThis.innerWidth)
  const isMobile = ref<boolean>(globalThis.innerWidth <= BREAKPOINTS.MOBILE)
  const isTablet = ref<boolean>(
    globalThis.innerWidth > BREAKPOINTS.MOBILE && globalThis.innerWidth < BREAKPOINTS.TABLET
  )
  const isDesktop = ref<boolean>(globalThis.innerWidth >= BREAKPOINTS.DESKTOP)

  const updateScreenSize = () => {
    const width = globalThis.innerWidth
    screenWidth.value = width
    isMobile.value = width <= BREAKPOINTS.MOBILE
    isTablet.value = width > BREAKPOINTS.MOBILE && width < BREAKPOINTS.TABLET
    isDesktop.value = width >= BREAKPOINTS.DESKTOP
  }

  onMounted(() => {
    updateScreenSize()
    globalThis.addEventListener('resize', updateScreenSize)
  })

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

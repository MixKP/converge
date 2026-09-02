import { computed, type Ref, type WritableComputedRef } from 'vue'

export const NONE = '__none'

export function useNumberSelect(source: Ref<number | null>): WritableComputedRef<string> {
  return computed<string>({
    get: () => (source.value === null ? NONE : String(source.value)),
    set: (val) => {
      source.value = val === NONE ? null : Number(val)
    },
  })
}

export function useEnumSelect<T extends string>(
  source: Ref<T | null>
): WritableComputedRef<string> {
  return computed<string>({
    get: () => source.value ?? NONE,
    set: (val) => {
      source.value = val === NONE ? null : (val as T)
    },
  })
}

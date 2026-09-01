import { computed, type Ref, type WritableComputedRef } from 'vue'

/* shadcn's Select is string-valued while every id in the booking store is a
 * number, and SelectItem rejects an empty-string value — so `__none` stands in
 * for null. Every v3 select goes through one of these instead of scattering
 * casts across the templates. */
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

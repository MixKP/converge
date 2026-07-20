import type { z } from 'zod'
import type { TypedSchema } from 'vee-validate'

/**
 * Adapts a zod schema to vee-validate's TypedSchema interface.
 *
 * vee-validate 4.15 does not yet detect Standard Schema (`~standard`), so a raw
 * zod schema passed to `useForm({ validationSchema })` is mistaken for a
 * per-field rules map and fails. This wrapper implements the `VVTypedSchema`
 * contract vee-validate does understand — kept in-repo to avoid the
 * `@vee-validate/zod` adapter's zod-v3/v4 version-skew.
 */
export function toTypedSchema<TSchema extends z.ZodType>(
  schema: TSchema
): TypedSchema<z.input<TSchema>, z.output<TSchema>> {
  return {
    __type: 'VVTypedSchema',
    async parse(values) {
      const result = await schema.safeParseAsync(values)
      if (result.success) {
        return { value: result.data, errors: [] }
      }

      // Group issue messages by their vee-validate field path.
      const byPath = new Map<string, string[]>()
      for (const issue of result.error.issues) {
        const path = pathToString(issue.path)
        const list = byPath.get(path)
        if (list) list.push(issue.message)
        else byPath.set(path, [issue.message])
      }

      return {
        errors: Array.from(byPath, ([path, errors]) => ({ path, errors })),
      }
    },
  }
}

/** Convert a zod issue path (`['weekly', 0, 'end']`) to vee-validate's `weekly[0].end`. */
function pathToString(segments: PropertyKey[]): string {
  let out = ''
  for (const segment of segments) {
    if (typeof segment === 'number') {
      out += `[${segment}]`
    } else {
      out += out ? `.${String(segment)}` : String(segment)
    }
  }
  return out
}

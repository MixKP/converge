import type { z } from 'zod'
import type { TypedSchema } from 'vee-validate'

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

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { CalendarPlus, ChevronDown, Loader2, MapPin, Search, Settings, User } from '@lucide/vue'
  import PageLayout from '../components/PageLayout.vue'
  import {
    useAdminRoster,
    type RosterEntry,
    type RosterMode,
    type RosterStatus,
  } from '../composables/useAdminRoster'
  import { subjectTone, subjectToneSoftVar, subjectToneVar } from '../utils/subjectColor'
  import { Badge } from '@/components/ui/badge'
  import { Button } from '@/components/ui/button'
  import { Card, CardContent } from '@/components/ui/card'
  import { Input } from '@/components/ui/input'

  const { mode, search, entries, allEntries, counts, subjectTones, isLoading, loadError, load } =
    useAdminRoster()

  // One colour per subject, used everywhere a subject name is rendered on this
  // page so the dot reads as the same language in all three places. Returns
  // undefined for an unknown name, which renders no dot rather than a
  // meaningless one.
  //
  // Two steps: the pastel fill and the deeper ring that gives it an edge. A
  // bare pastel dot is ~1.5:1 against the card and all but disappears.
  const dotStyle = (subject: string): Record<string, string> | undefined => {
    const tone = subjectTone(subjectTones.value, subject)
    const fill = subjectToneSoftVar(tone)
    const ring = subjectToneVar(tone)
    if (!fill || !ring) return undefined
    return { backgroundColor: fill, borderColor: ring }
  }

  const MODES: { value: RosterMode; label: string }[] = [
    { value: 'teachers', label: 'Teachers' },
    { value: 'students', label: 'Students' },
  ]

  const expanded = ref<Set<number>>(new Set())

  const isDeactivated = (entry: RosterEntry): boolean => entry.status === 'deactivated'

  // A deactivated teacher does not open. Gating isOpen on it too means a row
  // cannot be left stuck open if an entry ever changes status under us.
  const canExpand = (entry: RosterEntry): boolean => !isDeactivated(entry)

  const isOpen = (entry: RosterEntry): boolean => canExpand(entry) && expanded.value.has(entry.id)

  const toggle = (entry: RosterEntry) => {
    if (!canExpand(entry)) return
    const next = new Set(expanded.value)
    if (next.has(entry.id)) next.delete(entry.id)
    else next.add(entry.id)
    expanded.value = next
  }

  watch(mode, () => {
    expanded.value = new Set()
  })

  const pluralize = (word: string, count: number, plural?: string): string =>
    count === 1 ? word : (plural ?? `${word}s`)

  const entryNoun = computed(() => (mode.value === 'teachers' ? 'teacher' : 'student'))
  const peopleNoun = computed(() => (mode.value === 'teachers' ? 'student' : 'teacher'))

  const statusLabel = (status: RosterStatus): string =>
    status === 'active' ? 'Active' : 'Deactivated'

  // GET /teachers?subject_id=... returns active teachers only, so a deactivated
  // teacher always comes back with an empty subject list even when subjects are
  // assigned. Saying "no subjects assigned" there would be a lie.
  const subjectsPlaceholder = (entry: RosterEntry): string => {
    if (isDeactivated(entry)) return 'Subjects hidden while deactivated'
    return mode.value === 'teachers' ? 'No subjects assigned' : 'No subjects booked yet'
  }

  const noPeopleLabel = computed(() =>
    mode.value === 'teachers'
      ? 'No students booked with this teacher this week.'
      : 'No classes booked for this student this week.'
  )

  const stats = computed(() => [
    { label: 'Teachers', value: counts.value.teachers },
    { label: 'Students', value: counts.value.students },
    { label: 'Subjects', value: counts.value.subjects },
    { label: 'Classes this week', value: counts.value.classes },
  ])

  const dayFormat = new Intl.DateTimeFormat('en-US', {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
  })
  const timeFormat = new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })

  const formatSlot = (startTime: string, endTime: string): string => {
    const start = new Date(startTime)
    if (Number.isNaN(start.getTime())) return 'Unknown time'
    const end = new Date(endTime)
    const until = Number.isNaN(end.getTime()) ? '' : `–${timeFormat.format(end)}`
    return `${dayFormat.format(start)} · ${timeFormat.format(start)}${until}`
  }

  const isPast = (endTime: string): boolean => {
    const end = new Date(endTime).getTime()
    return Number.isFinite(end) && end < Date.now()
  }

  onMounted(load)
</script>

<template>
  <PageLayout title="Admin Dashboard">
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-6 sm:px-6 lg:px-8">
      <Card>
        <CardContent>
          <dl class="grid grid-cols-2 gap-y-4 sm:grid-cols-4 sm:gap-y-0 sm:divide-x">
            <div
              v-for="stat in stats"
              :key="stat.label"
              class="flex flex-col sm:px-5 sm:first:pl-0"
            >
              <dt class="text-muted-foreground text-xs">{{ stat.label }}</dt>
              <dd class="text-2xl leading-tight font-semibold tabular-nums">{{ stat.value }}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:gap-2">
        <div class="bg-muted inline-flex w-fit shrink-0 rounded-md p-0.5" role="tablist">
          <button
            v-for="option in MODES"
            :key="option.value"
            type="button"
            role="tab"
            :aria-selected="mode === option.value"
            class="focus-visible:ring-ring/50 rounded-sm px-3 py-1.5 text-sm font-medium transition-colors outline-none focus-visible:ring-3"
            :class="
              mode === option.value
                ? 'bg-background text-foreground shadow-xs'
                : 'text-muted-foreground hover:text-foreground'
            "
            @click="mode = option.value"
          >
            {{ option.label }}
          </button>
        </div>

        <div class="relative flex-1">
          <Search
            class="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
          />
          <!-- The shared Input, not a bare <input> carrying its own copy of the
               field styles. The copy had drifted: it still had the focus halo
               the component dropped, and never had the placeholder, selection
               or disabled styling every other field in the app has. pl-9 is the
               only thing actually specific here -- it clears the icon. -->
          <Input
            v-model="search"
            type="search"
            :aria-label="`Search ${entryNoun}s`"
            class="pl-9"
            placeholder="Search by name"
          />
        </div>

        <div class="flex gap-2">
          <RouterLink to="/manage" class="flex-1 sm:flex-none">
            <Button variant="outline" class="w-full">
              <Settings />
              Manage
            </Button>
          </RouterLink>
          <RouterLink to="/booking" class="flex-1 sm:flex-none">
            <Button class="w-full">
              <CalendarPlus />
              Book a session
            </Button>
          </RouterLink>
        </div>
      </div>

      <Card v-if="isLoading">
        <CardContent class="text-muted-foreground flex items-center gap-2 py-4 text-sm">
          <Loader2 class="size-4 animate-spin" />
          Loading roster…
        </CardContent>
      </Card>

      <Card v-else-if="loadError" class="border-destructive/30">
        <CardContent class="flex flex-wrap items-center justify-between gap-3">
          <p class="text-destructive text-sm">{{ loadError }}</p>
          <Button variant="outline" @click="load">Retry</Button>
        </CardContent>
      </Card>

      <Card v-else-if="allEntries.length === 0">
        <CardContent class="text-muted-foreground text-center text-sm">
          No {{ entryNoun }}s yet.
        </CardContent>
      </Card>

      <Card v-else-if="entries.length === 0">
        <CardContent class="text-muted-foreground text-center text-sm">
          No {{ entryNoun }}s match “{{ search }}”.
        </CardContent>
      </Card>

      <template v-else>
        <Card class="gap-0 overflow-hidden py-0">
          <CardContent class="divide-border divide-y px-0">
            <div v-for="entry in entries" :key="entry.id">
              <component
                :is="canExpand(entry) ? 'button' : 'div'"
                :type="canExpand(entry) ? 'button' : undefined"
                :aria-expanded="canExpand(entry) ? isOpen(entry) : undefined"
                class="flex w-full items-start gap-3 px-4 py-3 text-left sm:px-6"
                :class="canExpand(entry) ? 'hover:bg-muted/40 transition-colors' : 'cursor-default'"
                @click="toggle(entry)"
              >
                <span class="flex h-5 shrink-0 items-center gap-2" aria-hidden="true">
                  <span class="flex w-2.5 justify-center">
                    <span
                      v-if="entry.status"
                      class="size-2.5 rounded-full"
                      :class="entry.status === 'active' ? 'bg-success' : 'bg-danger'"
                    ></span>
                  </span>

                  <span class="flex w-4 justify-center">
                    <ChevronDown
                      v-if="canExpand(entry)"
                      class="text-muted-foreground size-4 transition-transform"
                      :class="{ 'rotate-180': isOpen(entry) }"
                    />
                  </span>
                </span>

                <span class="flex min-w-0 flex-1 flex-col gap-1.5">
                  <span class="truncate text-sm font-medium">
                    {{ entry.name }}
                    <span v-if="entry.status" class="sr-only">
                      — {{ statusLabel(entry.status) }}
                    </span>
                  </span>
                  <span v-if="entry.subjects.length" class="flex flex-wrap gap-1">
                    <Badge
                      v-for="subject in entry.subjects"
                      :key="subject"
                      variant="secondary"
                      class="gap-1.5 font-normal"
                    >
                      <span
                        v-if="dotStyle(subject)"
                        class="size-2 shrink-0 rounded-full border"
                        :style="dotStyle(subject)"
                        aria-hidden="true"
                      />
                      {{ subject }}
                    </Badge>
                  </span>
                  <span v-else class="text-muted-foreground text-xs">{{
                    subjectsPlaceholder(entry)
                  }}</span>
                </span>

                <span v-if="!isDeactivated(entry)" class="flex h-5 shrink-0 items-center">
                  <span v-if="entry.classCount === 0" class="text-muted-foreground text-sm">
                    No classes
                  </span>
                  <span v-else class="text-sm tabular-nums">
                    {{ entry.people.length }}
                    <span class="text-muted-foreground">
                      {{ pluralize(peopleNoun, entry.people.length) }} ·
                    </span>
                    {{ entry.classCount }}
                    <span class="text-muted-foreground">
                      {{ pluralize('class', entry.classCount, 'classes') }}
                    </span>
                  </span>
                </span>
              </component>

              <div
                v-if="isOpen(entry)"
                class="border-border bg-muted/20 border-t px-4 py-3 sm:px-6"
              >
                <p v-if="entry.people.length === 0" class="text-muted-foreground text-sm">
                  {{ noPeopleLabel }}
                </p>

                <ul v-else class="flex flex-col gap-2">
                  <li
                    v-for="person in entry.people"
                    :key="person.id"
                    class="bg-card border-border flex flex-col gap-1.5 rounded-md border px-3 py-2"
                  >
                    <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                      <User class="text-muted-foreground size-3.5 shrink-0" />
                      <span class="text-sm font-medium">{{ person.name }}</span>
                      <Badge
                        v-for="subject in person.subjects"
                        :key="subject"
                        variant="outline"
                        class="gap-1.5 font-normal"
                      >
                        <span
                          v-if="dotStyle(subject)"
                          class="size-2 shrink-0 rounded-full border"
                          :style="dotStyle(subject)"
                          aria-hidden="true"
                        />
                        {{ subject }}
                      </Badge>
                      <span class="text-muted-foreground ml-auto text-xs tabular-nums">
                        {{ person.sessions.length }}
                        {{ pluralize('class', person.sessions.length, 'classes') }}
                      </span>
                    </div>

                    <!-- Aligned columns from sm up: the branch name varies in
                         width, so a plain flex row left the subject starting at
                         a different x on every line and the eye could not scan
                         down. subgrid makes every session share the parent's
                         three tracks. Narrow screens keep the wrapping row. -->
                    <ul
                      class="flex flex-col gap-1 pl-5.5 sm:grid sm:grid-cols-[max-content_max-content_max-content] sm:justify-start sm:gap-x-4"
                    >
                      <li
                        v-for="session in person.sessions"
                        :key="session.id"
                        class="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs sm:col-span-full sm:grid sm:grid-cols-subgrid sm:gap-x-4"
                        :class="
                          isPast(session.endTime) ? 'text-muted-foreground/60' : 'text-foreground'
                        "
                      >
                        <span class="tabular-nums">
                          {{ formatSlot(session.startTime, session.endTime) }}
                        </span>
                        <span class="text-muted-foreground flex items-center gap-1">
                          <MapPin class="size-3" />
                          {{ session.branch }}
                        </span>
                        <span class="text-muted-foreground flex items-center gap-1.5">
                          <span
                            v-if="dotStyle(session.subject)"
                            class="size-2 shrink-0 rounded-full border"
                            :style="dotStyle(session.subject)"
                            aria-hidden="true"
                          />
                          {{ session.subject }}
                        </span>
                      </li>
                    </ul>
                  </li>
                </ul>
              </div>
            </div>
          </CardContent>
        </Card>

        <div
          class="text-muted-foreground flex flex-wrap items-center justify-between gap-x-4 gap-y-1 px-1 text-xs"
        >
          <p>
            Showing {{ entries.length }} of {{ allEntries.length }}
            {{ pluralize(entryNoun, allEntries.length) }}
          </p>

          <div v-if="mode === 'teachers'" class="flex items-center gap-4">
            <span class="flex items-center gap-1.5">
              <span class="bg-success size-2 rounded-full" aria-hidden="true"></span>
              Active
            </span>
            <span class="flex items-center gap-1.5">
              <span class="bg-danger size-2 rounded-full" aria-hidden="true"></span>
              Deactivated
            </span>
          </div>
        </div>
      </template>
    </div>
  </PageLayout>
</template>

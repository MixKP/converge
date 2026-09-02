<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { CalendarPlus, ChevronDown, Loader2, MapPin, Search, Settings, User } from '@lucide/vue'
  import PageLayout from '../components/PageLayout.vue'
  import { useAdminRoster, type RosterMode } from '../composables/useAdminRoster'
  import { Badge } from '@/components/ui/badge'
  import { Button } from '@/components/ui/button'
  import { Card, CardContent } from '@/components/ui/card'
  import { Label } from '@/components/ui/label'
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from '@/components/ui/select'

  const { mode, search, entries, allEntries, totalClasses, isLoading, loadError, load } =
    useAdminRoster()

  const modeValue = computed<string>({
    get: () => mode.value,
    set: (value) => {
      mode.value = value as RosterMode
    },
  })

  const expanded = ref<Set<number>>(new Set())

  const toggle = (id: number) => {
    const next = new Set(expanded.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    expanded.value = next
  }

  watch(mode, () => {
    expanded.value = new Set()
  })

  const noun = computed(() => (mode.value === 'teachers' ? 'teacher' : 'student'))

  const formatWhen = (iso: string): string => {
    const date = new Date(iso)
    if (Number.isNaN(date.getTime())) return 'Unknown time'
    return date.toLocaleString('en-US', {
      weekday: 'short',
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
    })
  }

  onMounted(load)
</script>

<template>
  <PageLayout title="Admin Dashboard">
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-5 px-4 py-6 sm:px-6 lg:px-8">
      <div class="grid gap-3 sm:grid-cols-2">
        <RouterLink to="/manage" class="group">
          <Card class="hover:border-primary/40 h-full transition-colors">
            <CardContent class="flex items-center gap-3 py-4">
              <span class="bg-muted rounded-lg p-2">
                <Settings class="size-5" />
              </span>
              <span class="flex flex-col">
                <span class="text-sm font-medium">Manage</span>
                <span class="text-muted-foreground text-xs">
                  Teachers, branches, commute, accounts
                </span>
              </span>
            </CardContent>
          </Card>
        </RouterLink>

        <RouterLink to="/booking" class="group">
          <Card class="hover:border-primary/40 h-full transition-colors">
            <CardContent class="flex items-center gap-3 py-4">
              <span class="bg-muted rounded-lg p-2">
                <CalendarPlus class="size-5" />
              </span>
              <span class="flex flex-col">
                <span class="text-sm font-medium">Book a session</span>
                <span class="text-muted-foreground text-xs">
                  Schedule a class on behalf of a student
                </span>
              </span>
            </CardContent>
          </Card>
        </RouterLink>
      </div>
      <Card>
        <CardContent class="grid gap-4 py-4 sm:grid-cols-[14rem_1fr]">
          <div class="flex flex-col gap-2">
            <Label for="roster-mode">Show</Label>
            <Select v-model="modeValue">
              <SelectTrigger id="roster-mode" class="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="teachers">Teachers</SelectItem>
                <SelectItem value="students">Students</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="flex flex-col gap-2">
            <Label for="roster-search">Search</Label>
            <div class="relative">
              <Search
                class="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
              />
              <input
                id="roster-search"
                v-model="search"
                type="search"
                class="border-input bg-background focus-visible:border-ring focus-visible:ring-ring/50 h-9 w-full rounded-md border py-1 pr-3 pl-9 text-sm shadow-xs outline-none focus-visible:ring-3"
                :placeholder="mode === 'teachers' ? 'Search by name or subject' : 'Search by name'"
              />
            </div>
          </div>
        </CardContent>
      </Card>
      <Card v-if="isLoading">
        <CardContent class="text-muted-foreground flex items-center gap-2 py-10 text-sm">
          <Loader2 class="size-4 animate-spin" />
          Loading roster…
        </CardContent>
      </Card>

      <Card v-else-if="loadError" class="border-destructive/30">
        <CardContent class="flex items-center justify-between gap-3 py-6">
          <p class="text-destructive text-sm">{{ loadError }}</p>
          <Button variant="outline" @click="load">Retry</Button>
        </CardContent>
      </Card>

      <template v-else>
        <p class="text-muted-foreground text-sm">
          {{ entries.length }} of {{ allEntries.length }} {{ noun
          }}{{ allEntries.length === 1 ? '' : 's' }} · {{ totalClasses }} booked class{{
            totalClasses === 1 ? '' : 'es'
          }}
          in total
        </p>

        <Card v-if="entries.length === 0">
          <CardContent class="text-muted-foreground py-10 text-center text-sm">
            No {{ noun }}s match “{{ search }}”.
          </CardContent>
        </Card>

        <Card v-for="entry in entries" v-else :key="entry.id">
          <CardContent class="py-0">
            <button
              type="button"
              class="hover:bg-muted/40 -mx-6 flex w-[calc(100%+3rem)] items-center gap-3 rounded-lg px-6 py-4 text-left transition-colors"
              :aria-expanded="expanded.has(entry.id)"
              @click="toggle(entry.id)"
            >
              <ChevronDown
                class="text-muted-foreground size-4 shrink-0 transition-transform"
                :class="{ 'rotate-180': expanded.has(entry.id) }"
              />

              <span class="flex min-w-0 flex-1 flex-col gap-1">
                <span class="truncate text-sm font-medium">{{ entry.name }}</span>
                <span v-if="entry.subjects.length" class="flex flex-wrap gap-1">
                  <Badge
                    v-for="subject in entry.subjects"
                    :key="subject"
                    variant="secondary"
                    class="font-normal"
                  >
                    {{ subject }}
                  </Badge>
                </span>
                <span v-else-if="mode === 'teachers'" class="text-muted-foreground text-xs">
                  No subjects assigned
                </span>
              </span>

              <Badge :variant="entry.classes.length ? 'secondary' : 'outline'" class="shrink-0">
                {{ entry.classes.length }} class{{ entry.classes.length === 1 ? '' : 'es' }}
              </Badge>
            </button>

            <div v-if="expanded.has(entry.id)" class="border-border border-t py-3">
              <p v-if="entry.classes.length === 0" class="text-muted-foreground py-2 text-sm">
                No booked classes yet.
              </p>

              <ul v-else class="flex flex-col gap-2">
                <li
                  v-for="session in entry.classes"
                  :key="session.id"
                  class="border-border flex flex-wrap items-center gap-x-4 gap-y-1 rounded-md border px-3 py-2 text-sm"
                >
                  <span class="font-medium">{{ formatWhen(session.startTime) }}</span>
                  <Badge variant="outline" class="font-normal">{{ session.subject }}</Badge>
                  <span class="text-muted-foreground flex items-center gap-1 text-xs">
                    <MapPin class="size-3.5" />
                    {{ session.branch }}
                  </span>
                  <span class="text-muted-foreground flex items-center gap-1 text-xs">
                    <User class="size-3.5" />
                    {{ session.counterpart }}
                  </span>
                </li>
              </ul>
            </div>
          </CardContent>
        </Card>
      </template>
    </div>
  </PageLayout>
</template>

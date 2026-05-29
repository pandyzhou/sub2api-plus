<template>
  <AppLayout>
    <div class="image-studio relative flex h-[calc(100vh-4rem)] overflow-hidden bg-[#0a0a0b]">
      <!-- Background grain texture -->
      <div class="pointer-events-none absolute inset-0 z-0 opacity-[0.03]" style="background-image: url('data:image/svg+xml,%3Csvg viewBox=%220 0 256 256%22 xmlns=%22http://www.w3.org/2000/svg%22%3E%3Cfilter id=%22n%22%3E%3CfeTurbulence type=%22fractalNoise%22 baseFrequency=%220.9%22 numOctaves=%224%22 stitchTiles=%22stitch%22/%3E%3C/filter%3E%3Crect width=%22100%25%22 height=%22100%25%22 filter=%22url(%23n)%22/%3E%3C/svg%3E');" />

      <!-- ==================== Left Sidebar: Sessions ==================== -->
      <aside
        :class="[
          'relative z-10 flex flex-col border-r border-white/[0.06] bg-[#0e0e10]',
          'w-64 shrink-0 transition-all duration-300',
          mobileShowSessions
            ? 'absolute inset-0 z-30 w-full sm:relative sm:w-64'
            : 'hidden sm:flex'
        ]"
      >
        <!-- Sidebar Header -->
        <div class="flex items-center justify-between px-5 pt-5 pb-4">
          <div class="flex items-center gap-2.5">
            <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-500/10">
              <svg class="h-4 w-4 text-amber-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 2L2 7l10 5 10-5-10-5z" /><path d="M2 17l10 5 10-5" /><path d="M2 12l10 5 10-5" />
              </svg>
            </div>
            <span class="text-[13px] font-semibold tracking-wide text-white/90">画布工作台</span>
          </div>
          <button
            class="sm:hidden rounded-md p-1 text-white/40 hover:text-white/70 transition-colors"
            @click="mobileShowSessions = false"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>

        <!-- New Session -->
        <div class="px-3 pb-2">
          <button
            class="group flex w-full items-center gap-2 rounded-lg border border-white/[0.06] bg-white/[0.03] px-3 py-2 text-[13px] font-medium text-white/60 transition-all hover:border-amber-500/20 hover:bg-amber-500/[0.06] hover:text-amber-300"
            @click="handleNewSession"
          >
            <svg class="h-3.5 w-3.5 transition-transform group-hover:rotate-90" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
            {{ t('image.newSession') }}
          </button>
        </div>

        <!-- Session List -->
        <div class="flex-1 overflow-y-auto px-2 scrollbar-hide">
          <div v-if="imageStore.loading" class="flex justify-center py-10">
            <div class="h-5 w-5 animate-spin rounded-full border-2 border-amber-500/20 border-t-amber-400" />
          </div>
          <div v-else-if="sessions.length === 0" class="flex flex-col items-center py-12 text-center">
            <div class="mb-3 flex h-10 w-10 items-center justify-center rounded-xl bg-white/[0.03]">
              <svg class="h-5 w-5 text-white/20" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
              </svg>
            </div>
            <p class="text-xs text-white/25">{{ t('image.noSessions') }}</p>
          </div>
          <div v-else class="space-y-0.5">
            <div
              v-for="session in sessions"
              :key="session.id"
              :class="[
                'group relative flex cursor-pointer items-center gap-2.5 rounded-lg px-3 py-2 transition-all',
                currentSessionId === session.id
                  ? 'bg-amber-500/[0.08] text-amber-200'
                  : 'text-white/50 hover:bg-white/[0.04] hover:text-white/80'
              ]"
              @click="handleSelectSession(session)"
            >
              <div class="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-white/[0.05]">
                <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
                </svg>
              </div>
              <div class="min-w-0 flex-1">
                <p class="truncate text-[13px] leading-tight">{{ session.title || t('image.newSession') }}</p>
                <p class="mt-0.5 text-[11px] text-white/30">{{ formatTime(session.updated_at || session.created_at) }}</p>
              </div>
              <button
                class="shrink-0 rounded p-1 text-white/0 transition-all hover:text-red-400 group-hover:text-white/20"
                :title="t('image.deleteSession')"
                @click.stop="handleDeleteSession(session.id)"
              >
                <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
        </div>

        <!-- Clear History -->
        <div class="border-t border-white/[0.04] p-3">
          <button
            v-if="sessions.length > 0"
            class="flex w-full items-center justify-center gap-1.5 rounded-lg px-3 py-1.5 text-[11px] text-white/25 transition-all hover:bg-red-500/[0.06] hover:text-red-400"
            @click="handleClearHistory"
          >
            <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0" />
            </svg>
            {{ t('image.clearHistory') }}
          </button>
        </div>
      </aside>

      <!-- ==================== Main Canvas Area ==================== -->
      <div class="relative z-10 flex flex-1 flex-col overflow-hidden">

        <!-- Top Bar -->
        <header class="flex items-center gap-3 border-b border-white/[0.06] bg-[#0a0a0b] px-5 py-3">
          <!-- Mobile menu -->
          <button
            class="sm:hidden rounded-md p-1.5 text-white/50 hover:bg-white/[0.06]"
            @click="mobileShowSessions = true"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" /></svg>
          </button>

          <!-- Mode Toggle -->
          <div class="flex rounded-lg bg-white/[0.04] p-0.5">
            <button
              :class="[
                'relative rounded-md px-3.5 py-1.5 text-[12px] font-medium transition-all',
                !imageStore.editMode
                  ? 'bg-amber-500/[0.15] text-amber-300 shadow-sm shadow-amber-500/10'
                  : 'text-white/40 hover:text-white/60'
              ]"
              @click="imageStore.editMode = false"
            >
              {{ t('image.textToImage') }}
            </button>
            <button
              :class="[
                'relative rounded-md px-3.5 py-1.5 text-[12px] font-medium transition-all',
                imageStore.editMode
                  ? 'bg-amber-500/[0.15] text-amber-300 shadow-sm shadow-amber-500/10'
                  : 'text-white/40 hover:text-white/60'
              ]"
              @click="imageStore.editMode = true"
            >
              {{ t('image.imageEdit') }}
            </button>
          </div>

          <div class="flex-1" />

          <!-- Settings pills -->
          <div class="hidden items-center gap-2 md:flex">
            <button
              @click="showSettings = !showSettings"
              :class="[
                'flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-[12px] font-medium transition-all',
                showSettings
                  ? 'bg-white/[0.08] text-white/80'
                  : 'text-white/35 hover:bg-white/[0.04] hover:text-white/60'
              ]"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
              设置
            </button>

            <button
              v-if="results.length > 1"
              class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-[12px] text-white/35 transition-all hover:bg-white/[0.04] hover:text-white/60"
              @click="downloadAll"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
              {{ t('image.downloadAll') }}
            </button>
          </div>
        </header>

        <!-- Settings Panel (collapsible) -->
        <Transition name="settings-slide">
          <div v-if="showSettings" class="border-b border-white/[0.04] bg-[#0c0c0e] px-5 py-3">
            <div class="flex flex-wrap items-end gap-4">
              <div v-for="field in settingsFields" :key="field.key" class="flex flex-col gap-1.5">
                <label class="text-[11px] font-medium uppercase tracking-wider text-white/30">{{ field.label }}</label>
                <select
                  v-model="field.model.value"
                  class="rounded-lg border border-white/[0.08] bg-white/[0.04] px-3 py-1.5 text-[13px] text-white/80 transition-colors focus:border-amber-500/40 focus:outline-none focus:ring-1 focus:ring-amber-500/20 [&>option]:bg-[#1a1a1e] [&>option]:text-white/80"
                >
                  <option v-for="opt in field.options" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
              </div>
            </div>
          </div>
        </Transition>

        <!-- Edit Mode: Image Upload -->
        <Transition name="settings-slide">
          <div v-if="imageStore.editMode" class="border-b border-white/[0.04] bg-[#0c0c0e] px-5 py-4">
            <div
              :class="[
                'relative flex flex-col items-center justify-center rounded-xl border border-dashed p-5 transition-all',
                isDragging
                  ? 'border-amber-500/50 bg-amber-500/[0.04]'
                  : editPreview
                    ? 'border-white/[0.08] bg-white/[0.02]'
                    : 'border-white/[0.06] bg-white/[0.02] hover:border-white/[0.12]'
              ]"
              @dragover.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleDrop"
            >
              <div v-if="editPreview" class="relative">
                <img :src="editPreview" class="max-h-40 rounded-lg object-contain shadow-lg shadow-black/50" alt="Reference" />
                <button
                  class="absolute -right-2 -top-2 rounded-full bg-red-500/90 p-1 text-white shadow-lg hover:bg-red-600 transition-colors"
                  @click="clearEditImage"
                >
                  <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
                </button>
              </div>
              <div v-else class="text-center">
                <div class="mx-auto mb-2 flex h-10 w-10 items-center justify-center rounded-xl bg-white/[0.04]">
                  <svg class="h-5 w-5 text-white/25" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
                  </svg>
                </div>
                <p class="text-[12px] text-white/35">{{ t('image.dragOrClick') }}</p>
                <input
                  ref="fileInput"
                  type="file"
                  accept="image/*"
                  class="absolute inset-0 cursor-pointer opacity-0"
                  @change="handleFileSelect"
                />
              </div>
            </div>
          </div>
        </Transition>

        <!-- ==================== Canvas: Results ==================== -->
        <div class="flex-1 overflow-y-auto" ref="canvasRef">
          <!-- Empty State -->
          <div v-if="results.length === 0 && !imageStore.generating" class="flex h-full flex-col items-center justify-center px-6">
            <div class="relative">
              <!-- Ambient glow -->
              <div class="absolute -inset-16 rounded-full bg-amber-500/[0.04] blur-3xl" />
              <div class="relative flex h-20 w-20 items-center justify-center rounded-2xl border border-white/[0.06] bg-white/[0.03]">
                <svg class="h-8 w-8 text-amber-500/40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="12" cy="12" r="3" /><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
                </svg>
              </div>
            </div>
            <h3 class="mt-6 text-[15px] font-medium text-white/50">描述你的想象</h3>
            <p class="mt-1.5 max-w-sm text-center text-[13px] leading-relaxed text-white/25">
              在下方输入提示词，AI 将为你创造独一无二的画面
            </p>
          </div>

          <!-- Generating Skeleton -->
          <div v-if="imageStore.generating" class="p-5">
            <div class="grid gap-4" :class="gridColsClass">
              <div
                v-for="i in imageStore.settings.n"
                :key="`skel-${i}`"
                class="group relative overflow-hidden rounded-xl border border-white/[0.06] bg-white/[0.02]"
              >
                <div class="aspect-square animate-pulse bg-white/[0.04]">
                  <div class="flex h-full items-center justify-center">
                    <div class="relative">
                      <div class="h-10 w-10 animate-spin rounded-full border-2 border-amber-500/10 border-t-amber-500/50" />
                      <div class="absolute inset-0 flex items-center justify-center">
                        <div class="h-3 w-3 rounded-full bg-amber-500/30" />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Image Grid -->
          <div v-if="results.length > 0" class="p-5">
            <div class="grid gap-4" :class="gridColsClass">
              <div
                v-for="(img, index) in results"
                :key="index"
                class="group relative overflow-hidden rounded-xl border border-white/[0.06] bg-white/[0.02] transition-all duration-300 hover:border-amber-500/20 hover:shadow-lg hover:shadow-amber-500/[0.05]"
                :style="{ animationDelay: `${index * 80}ms` }"
                :class="'animate-fade-in-up'"
              >
                <div class="aspect-square cursor-pointer" @click="openLightbox(index)">
                  <img
                    :src="getImageSrc(img)"
                    :alt="img.revised_prompt || 'Generated image'"
                    class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.02]"
                    loading="lazy"
                  />
                </div>
                <!-- Hover overlay -->
                <div class="absolute inset-0 bg-gradient-to-t from-black/70 via-black/10 to-transparent opacity-0 transition-opacity duration-300 group-hover:opacity-100">
                  <div class="absolute inset-x-0 bottom-0 p-3">
                    <p v-if="img.revised_prompt" class="mb-2 line-clamp-2 text-[11px] leading-relaxed text-white/80">
                      {{ img.revised_prompt }}
                    </p>
                    <div class="flex items-center gap-2">
                      <button
                        class="flex h-7 w-7 items-center justify-center rounded-lg bg-white/10 text-white/80 backdrop-blur-sm transition-colors hover:bg-white/20"
                        :title="t('image.download')"
                        @click.stop="downloadImage(img, index)"
                      >
                        <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5M16.5 12L12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
                      </button>
                      <button
                        class="flex h-7 w-7 items-center justify-center rounded-lg bg-white/10 text-white/80 backdrop-blur-sm transition-colors hover:bg-white/20"
                        @click.stop="openLightbox(index)"
                      >
                        <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 3.75v4.5m0-4.5h4.5m-4.5 0L9 9M3.75 20.25v-4.5m0 4.5h4.5m-4.5 0L9 15M20.25 3.75h-4.5m4.5 0v4.5m0-4.5L15 9m5.25 11.25h-4.5m4.5 0v-4.5m0 4.5L15 15" /></svg>
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ==================== Prompt Input (Bottom) ==================== -->
        <div class="border-t border-white/[0.06] bg-[#0e0e10] p-4">
          <div class="mx-auto flex max-w-3xl gap-3">
            <div class="relative flex-1">
              <textarea
                v-model="prompt"
                :placeholder="t('image.promptPlaceholder')"
                rows="2"
                class="w-full resize-none rounded-xl border border-white/[0.08] bg-white/[0.04] px-4 py-3 pr-12 text-[14px] text-white/90 placeholder:text-white/25 transition-all focus:border-amber-500/30 focus:outline-none focus:ring-1 focus:ring-amber-500/15"
                @keydown.ctrl.enter="handleGenerate"
                @keydown.meta.enter="handleGenerate"
              />
              <div class="absolute bottom-2.5 right-2.5 text-[10px] text-white/20">
                Ctrl+↵
              </div>
            </div>
            <button
              :disabled="imageStore.generating || !prompt.trim()"
              :class="[
                'flex h-[60px] w-[60px] shrink-0 items-center justify-center rounded-xl transition-all duration-300',
                imageStore.generating || !prompt.trim()
                  ? 'cursor-not-allowed border border-white/[0.04] bg-white/[0.03] text-white/15'
                  : 'border border-amber-500/30 bg-amber-500/[0.15] text-amber-300 shadow-lg shadow-amber-500/10 hover:bg-amber-500/25 hover:shadow-amber-500/20'
              ]"
              @click="handleGenerate"
            >
              <svg v-if="imageStore.generating" class="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle class="opacity-20" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" />
                <path class="opacity-80" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              <svg v-else class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 2L2 7l10 5 10-5-10-5z" /><path d="M2 17l10 5 10-5" /><path d="M2 12l10 5 10-5" />
              </svg>
            </button>
          </div>
        </div>
      </div>

      <!-- ==================== Lightbox ==================== -->
      <ImageLightbox
        :visible="lightboxVisible"
        :images="results"
        :initial-index="lightboxIndex"
        @close="lightboxVisible = false"
      />

      <!-- ==================== Confirm Dialogs ==================== -->
      <Teleport to="body">
        <Transition name="dialog-fade">
          <div
            v-if="showClearConfirm"
            class="fixed inset-0 z-[9997] flex items-center justify-center bg-black/60 backdrop-blur-sm"
            @click.self="showClearConfirm = false"
          >
            <div class="mx-4 w-full max-w-sm rounded-2xl border border-white/[0.08] bg-[#161618] p-6 shadow-2xl">
              <h3 class="text-[15px] font-semibold text-white/90">{{ t('image.clearHistory') }}</h3>
              <p class="mt-2 text-[13px] text-white/45">{{ t('image.confirmClear') }}</p>
              <div class="mt-6 flex justify-end gap-3">
                <button
                  class="rounded-lg px-4 py-2 text-[13px] text-white/50 transition-colors hover:bg-white/[0.06] hover:text-white/80"
                  @click="showClearConfirm = false"
                >{{ t('common.cancel') }}</button>
                <button
                  class="rounded-lg bg-red-500/[0.15] px-4 py-2 text-[13px] font-medium text-red-400 transition-colors hover:bg-red-500/25"
                  @click="confirmClear"
                >{{ t('common.confirm') }}</button>
              </div>
            </div>
          </div>
        </Transition>
      </Teleport>

      <Teleport to="body">
        <Transition name="dialog-fade">
          <div
            v-if="showDeleteConfirm"
            class="fixed inset-0 z-[9997] flex items-center justify-center bg-black/60 backdrop-blur-sm"
            @click.self="showDeleteConfirm = false"
          >
            <div class="mx-4 w-full max-w-sm rounded-2xl border border-white/[0.08] bg-[#161618] p-6 shadow-2xl">
              <h3 class="text-[15px] font-semibold text-white/90">{{ t('image.deleteSession') }}</h3>
              <p class="mt-2 text-[13px] text-white/45">{{ t('image.confirmDelete') }}</p>
              <div class="mt-6 flex justify-end gap-3">
                <button
                  class="rounded-lg px-4 py-2 text-[13px] text-white/50 transition-colors hover:bg-white/[0.06] hover:text-white/80"
                  @click="showDeleteConfirm = false"
                >{{ t('common.cancel') }}</button>
                <button
                  class="rounded-lg bg-red-500/[0.15] px-4 py-2 text-[13px] font-medium text-red-400 transition-colors hover:bg-red-500/25"
                  @click="confirmDelete"
                >{{ t('common.confirm') }}</button>
              </div>
            </div>
          </div>
        </Transition>
      </Teleport>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useImageStore } from '@/stores/image'
import type { ImageSession, ImageResult } from '@/api/image'
import AppLayout from '@/components/layout/AppLayout.vue'
import ImageLightbox from '@/components/image/ImageLightbox.vue'

const { t } = useI18n()
const imageStore = useImageStore()

// ==================== State ====================

const prompt = ref('')
const isDragging = ref(false)
const editFile = ref<File | null>(null)
const editPreview = ref<string | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const canvasRef = ref<HTMLElement | null>(null)

const mobileShowSessions = ref(false)
const lightboxVisible = ref(false)
const lightboxIndex = ref(0)
const showClearConfirm = ref(false)
const showDeleteConfirm = ref(false)
const pendingDeleteId = ref<string | null>(null)
const showSettings = ref(false)

// ==================== Settings Config ====================

const settingsFields = computed(() => [
  {
    key: 'model',
    label: t('image.model'),
    model: computed({
      get: () => imageStore.settings.model,
      set: (v: string) => { imageStore.settings.model = v }
    }) as Ref<string>,
    options: [
      { value: 'auto', label: 'auto' },
      { value: 'gpt-image-1', label: 'gpt-image-1' },
      { value: 'dall-e-3', label: 'dall-e-3' },
      { value: 'dall-e-2', label: 'dall-e-2' }
    ]
  },
  {
    key: 'size',
    label: t('image.size'),
    model: computed({
      get: () => imageStore.settings.size,
      set: (v: string) => { imageStore.settings.size = v }
    }) as Ref<string>,
    options: [
      { value: 'auto', label: 'auto' },
      { value: '1024x1024', label: '1024×1024' },
      { value: '1536x1024', label: '1536×1024' },
      { value: '1024x1536', label: '1024×1536' }
    ]
  },
  {
    key: 'quality',
    label: t('image.quality'),
    model: computed({
      get: () => imageStore.settings.quality,
      set: (v: string) => { imageStore.settings.quality = v }
    }) as Ref<string>,
    options: [
      { value: 'auto', label: 'auto' },
      { value: 'low', label: 'low' },
      { value: 'medium', label: 'medium' },
      { value: 'high', label: 'high' }
    ]
  },
  {
    key: 'n',
    label: t('image.count'),
    model: computed({
      get: () => String(imageStore.settings.n),
      set: (v: string) => { imageStore.settings.n = Number(v) }
    }) as Ref<string>,
    options: [
      { value: '1', label: '1' },
      { value: '2', label: '2' },
      { value: '3', label: '3' },
      { value: '4', label: '4' }
    ]
  }
])

// ==================== Computed ====================

const sessions = computed(() => imageStore.sessions)
const results = computed(() => imageStore.results)
const currentSessionId = computed(() => imageStore.currentSession?.id || null)

const gridColsClass = computed(() => {
  const n = imageStore.settings.n
  if (n <= 1) return 'grid-cols-1 max-w-lg mx-auto'
  if (n === 2) return 'grid-cols-1 sm:grid-cols-2 max-w-2xl mx-auto'
  return 'grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'
})

// ==================== Methods ====================

function getImageSrc(img: ImageResult): string {
  if (img.b64_json) return `data:image/png;base64,${img.b64_json}`
  return img.url || ''
}

function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  try {
    const d = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - d.getTime()
    if (diff < 60000) return '刚刚'
    if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    const hours = String(d.getHours()).padStart(2, '0')
    const mins = String(d.getMinutes()).padStart(2, '0')
    if (d.getFullYear() === now.getFullYear()) return `${month}-${day} ${hours}:${mins}`
    return `${d.getFullYear()}-${month}-${day} ${hours}:${mins}`
  } catch {
    return dateStr
  }
}

async function handleNewSession() {
  const title = prompt.value.trim().slice(0, 50) || t('image.newSession')
  await imageStore.createAndSelectSession(title)
  mobileShowSessions.value = false
}

function handleSelectSession(session: ImageSession) {
  imageStore.selectSession(session)
  mobileShowSessions.value = false
}

function handleDeleteSession(id: string) {
  pendingDeleteId.value = id
  showDeleteConfirm.value = true
}

async function confirmDelete() {
  if (pendingDeleteId.value) await imageStore.removeSession(pendingDeleteId.value)
  pendingDeleteId.value = null
  showDeleteConfirm.value = false
}

function handleClearHistory() { showClearConfirm.value = true }
async function confirmClear() {
  await imageStore.removeAllSessions()
  showClearConfirm.value = false
}

async function handleGenerate() {
  const trimmed = prompt.value.trim()
  if (!trimmed || imageStore.generating) return
  if (!imageStore.currentSession) {
    await imageStore.createAndSelectSession(trimmed.slice(0, 50))
  }
  try {
    if (imageStore.editMode && editFile.value) {
      const fd = new FormData()
      fd.append('image', editFile.value)
      fd.append('prompt', trimmed)
      fd.append('model', imageStore.settings.model)
      fd.append('n', String(imageStore.settings.n))
      fd.append('size', imageStore.settings.size)
      fd.append('quality', imageStore.settings.quality)
      await imageStore.edit(fd)
    } else {
      await imageStore.generate(trimmed)
    }
  } catch (e) {
    console.error('Generation failed:', e)
  }
}

function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files?.[0]) setEditFile(input.files[0])
}

function handleDrop(e: DragEvent) {
  isDragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file && file.type.startsWith('image/')) setEditFile(file)
}

function setEditFile(file: File) {
  editFile.value = file
  const reader = new FileReader()
  reader.onload = (e) => { editPreview.value = e.target?.result as string }
  reader.readAsDataURL(file)
}

function clearEditImage() {
  editFile.value = null
  editPreview.value = null
  if (fileInput.value) fileInput.value.value = ''
}

function downloadImage(img: ImageResult, index: number) {
  const src = getImageSrc(img)
  const link = document.createElement('a')
  link.href = src
  link.download = `image-${Date.now()}-${index}.png`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

function downloadAll() {
  results.value.forEach((img, i) => setTimeout(() => downloadImage(img, i), i * 300))
}

function openLightbox(index: number) {
  lightboxIndex.value = index
  lightboxVisible.value = true
}

// ==================== Lifecycle ====================

onMounted(() => { imageStore.loadSessions() })
</script>

<style scoped>
@keyframes fade-in-up {
  from {
    opacity: 0;
    transform: translateY(12px) scale(0.97);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.animate-fade-in-up {
  animation: fade-in-up 0.4s ease-out both;
}

/* Settings panel slide */
.settings-slide-enter-active,
.settings-slide-leave-active {
  transition: all 0.25s ease;
  overflow: hidden;
}
.settings-slide-enter-from,
.settings-slide-leave-to {
  opacity: 0;
  max-height: 0;
  padding-top: 0;
  padding-bottom: 0;
}
.settings-slide-enter-to,
.settings-slide-leave-from {
  max-height: 200px;
}

/* Dialog fade */
.dialog-fade-enter-active,
.dialog-fade-leave-active {
  transition: opacity 0.2s ease;
}
.dialog-fade-enter-from,
.dialog-fade-leave-to {
  opacity: 0;
}

/* Scrollbar */
.scrollbar-hide::-webkit-scrollbar {
  display: none;
}
.scrollbar-hide {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>

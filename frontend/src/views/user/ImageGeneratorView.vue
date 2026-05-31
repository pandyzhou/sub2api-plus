<template>
  <AppLayout>
    <div class="relative h-[calc(100dvh-6rem)] min-h-[640px] overflow-hidden rounded-3xl border border-slate-200 bg-slate-100 shadow-xl shadow-slate-950/5 dark:border-slate-800 dark:bg-slate-950 md:h-[calc(100dvh-7rem)] lg:h-[calc(100dvh-8rem)]">
      <div class="absolute inset-0 pointer-events-none bg-[radial-gradient(circle_at_top_left,rgba(34,197,94,0.14),transparent_28%),radial-gradient(circle_at_top_right,rgba(14,165,233,0.10),transparent_30%)]" />

      <div class="relative grid h-full min-h-0 grid-cols-1 lg:grid-cols-[248px_minmax(0,1fr)] 2xl:grid-cols-[260px_minmax(0,1fr)_320px]">
        <!-- Sessions -->
        <aside
          :class="[
            'z-30 flex min-h-0 flex-col border-r border-slate-200/80 bg-white/92 backdrop-blur-xl dark:border-slate-800 dark:bg-slate-950/88',
            mobileShowSessions ? 'absolute inset-y-0 left-0 w-[min(86vw,320px)] shadow-2xl lg:relative lg:w-auto lg:shadow-none' : 'hidden lg:flex'
          ]"
        >
          <div class="flex items-center justify-between border-b border-slate-200 px-4 py-4 dark:border-slate-800">
            <div>
              <p class="text-xs font-medium uppercase tracking-[0.2em] text-emerald-600 dark:text-emerald-400">Studio</p>
              <h2 class="mt-1 text-base font-semibold text-slate-950 dark:text-white">{{ t('image.title') }}</h2>
            </div>
            <button
              class="inline-flex h-11 w-11 cursor-pointer items-center justify-center rounded-2xl text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 focus:outline-none focus:ring-2 focus:ring-emerald-400 dark:text-slate-400 dark:hover:bg-slate-900 dark:hover:text-white lg:hidden"
              aria-label="关闭会话列表"
              @click="mobileShowSessions = false"
            >
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" /></svg>
            </button>
          </div>

          <div class="border-b border-slate-200 p-3 dark:border-slate-800">
            <button class="btn btn-primary btn-sm min-h-11 w-full cursor-pointer" @click="handleNewSession">
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" /></svg>
              {{ t('image.newSession') }}
            </button>
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto px-2 py-3 scrollbar-hide">
            <div v-if="imageStore.loading" class="flex justify-center py-10"><LoadingSpinner /></div>
            <div v-else-if="sessions.length === 0" class="rounded-2xl border border-dashed border-slate-200 px-4 py-8 text-center text-sm text-slate-500 dark:border-slate-800 dark:text-slate-400">
              {{ t('image.noSessions') }}
            </div>
            <div v-else class="space-y-1">
              <button
                v-for="session in sessions"
                :key="session.id"
                :class="[
                  'group flex min-h-16 w-full cursor-pointer items-center gap-3 rounded-2xl border px-3 py-2.5 text-left transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-emerald-400',
                  currentSessionId === session.id
                    ? 'border-emerald-200 bg-emerald-50/50 shadow-sm shadow-emerald-500/8 dark:border-emerald-800/60 dark:bg-emerald-950/20'
                    : 'border-transparent hover:border-slate-200 hover:bg-slate-50 dark:hover:border-slate-800 dark:hover:bg-slate-900/70'
                ]"
                @click="handleSelectSession(session)"
              >
                <span
                  :class="[
                    'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border text-xs font-semibold',
                    currentSessionId === session.id
                      ? 'border-emerald-200 bg-white text-emerald-700 dark:border-emerald-800 dark:bg-slate-950 dark:text-emerald-300'
                      : 'border-slate-200 bg-white text-slate-500 dark:border-slate-800 dark:bg-slate-950 dark:text-slate-400'
                  ]"
                >{{ session.records?.length || 0 }}</span>
                <span class="min-w-0 flex-1">
                  <span class="line-clamp-2 text-sm font-semibold leading-snug text-slate-800 dark:text-slate-100">{{ session.title || t('image.newSession') }}</span>
                  <span class="mt-1 block text-xs text-slate-500 dark:text-slate-400">{{ formatTime(session.updated_at || session.created_at) }}</span>
                </span>
                <span
                  class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-slate-300 opacity-0 transition-all hover:bg-red-50 hover:text-red-500 group-hover:opacity-100 dark:text-slate-700 dark:hover:bg-red-950/40 dark:hover:text-red-400"
                  role="button"
                  :aria-label="t('image.deleteSession')"
                  @click.stop="handleDeleteSession(session.id)"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" /></svg>
                </span>
              </button>
            </div>
          </div>

          <div class="border-t border-slate-200 p-3 dark:border-slate-800">
            <button
              v-if="sessions.length > 0"
              class="btn btn-ghost btn-sm min-h-11 w-full cursor-pointer text-slate-500 hover:text-red-500"
              @click="handleClearHistory"
            >
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79" /></svg>
              {{ t('image.clearHistory') }}
            </button>
          </div>
        </aside>

        <button
          v-if="mobileShowSessions"
          class="absolute inset-0 z-20 bg-slate-950/40 backdrop-blur-sm lg:hidden"
          aria-label="关闭会话遮罩"
          @click="mobileShowSessions = false"
        />

        <!-- Main canvas -->
        <main class="flex min-w-0 min-h-0 flex-col">
          <header class="flex min-h-[72px] items-center gap-3 border-b border-slate-200/80 bg-white/88 px-4 backdrop-blur-xl dark:border-slate-800 dark:bg-slate-950/78 sm:px-5">
            <button
              class="inline-flex h-11 w-11 cursor-pointer items-center justify-center rounded-2xl border border-slate-200 bg-white text-slate-600 shadow-sm transition-colors hover:bg-slate-50 focus:outline-none focus:ring-2 focus:ring-emerald-400 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800 lg:hidden"
              aria-label="打开会话列表"
              @click="mobileShowSessions = true"
            >
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8"><path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" /></svg>
            </button>

            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <span class="inline-flex items-center rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-300">
                  {{ imageStore.editMode ? t('image.imageEdit') : t('image.textToImage') }}
                </span>
                <span class="hidden text-xs text-slate-500 dark:text-slate-400 sm:inline">{{ currentSession?.title || '新创作' }}</span>
              </div>
              <h1 class="mt-1 truncate text-lg font-semibold text-slate-950 dark:text-white">AI 图片创作</h1>
            </div>

            <div class="hidden items-center rounded-full border border-slate-200 bg-slate-100/80 p-0.5 dark:border-slate-800 dark:bg-slate-900/80 md:flex">
              <button
                :class="[
                  'min-h-[34px] cursor-pointer rounded-full px-3.5 text-xs font-medium transition-all focus:outline-none focus:ring-2 focus:ring-emerald-400',
                  !imageStore.editMode ? 'bg-white text-emerald-700 shadow-sm dark:bg-slate-800 dark:text-emerald-300' : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
                ]"
                @click="setMode(false)"
              >{{ t('image.textToImage') }}</button>
              <button
                :class="[
                  'min-h-[34px] cursor-pointer rounded-full px-3.5 text-xs font-medium transition-all focus:outline-none focus:ring-2 focus:ring-emerald-400',
                  imageStore.editMode ? 'bg-white text-emerald-700 shadow-sm dark:bg-slate-800 dark:text-emerald-300' : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
                ]"
                @click="setMode(true)"
              >{{ t('image.imageEdit') }}</button>
            </div>
          </header>

          <div ref="canvasRef" class="min-h-0 flex-1 overflow-y-auto scroll-smooth px-4 py-5 sm:px-6 lg:px-8">
            <section v-if="lastError" class="mx-auto mb-5 max-w-5xl rounded-3xl border border-red-200 bg-red-50 p-4 shadow-sm dark:border-red-900/70 dark:bg-red-950/30">
              <div class="flex gap-3">
                <span class="mt-0.5 inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-red-100 text-red-600 dark:bg-red-950 dark:text-red-300">
                  <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m0 3.75h.008v.008H12v-.008Zm9-4.5a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" /></svg>
                </span>
                <div class="min-w-0 flex-1">
                  <h3 class="font-semibold text-red-900 dark:text-red-100">生成没有完成</h3>
                  <p class="mt-1 whitespace-pre-wrap text-sm leading-6 text-red-700 dark:text-red-200">{{ lastError }}</p>
                  <div class="mt-3 flex flex-wrap gap-2">
                    <button class="btn btn-danger btn-sm min-h-10 cursor-pointer" :disabled="!failedPrompt || imageStore.generating" @click="retryFailedPrompt">重试上一条</button>
                    <button class="btn btn-secondary btn-sm min-h-10 cursor-pointer" @click="lastError = ''">知道了</button>
                  </div>
                </div>
              </div>
            </section>

            <section v-if="displayTurns.length === 0 && !imageStore.generating" class="mx-auto grid min-h-full max-w-6xl place-items-center py-8">
              <div class="relative w-full overflow-hidden rounded-[2rem] border border-slate-200/80 bg-white/82 p-6 text-center shadow-xl shadow-slate-950/5 backdrop-blur dark:border-slate-800/70 dark:bg-slate-900/70 sm:p-10">
                <!-- 装饰光晕 -->
                <div class="pointer-events-none absolute -top-24 -right-24 h-72 w-72 rounded-full bg-gradient-to-br from-emerald-300/20 to-sky-300/10 blur-3xl" />
                <div class="pointer-events-none absolute -bottom-20 -left-20 h-56 w-56 rounded-full bg-gradient-to-tr from-emerald-400/15 to-teal-300/8 blur-3xl" />
                <div class="pointer-events-none absolute top-1/2 left-1/2 h-40 w-40 -translate-x-1/2 -translate-y-1/2 rounded-full bg-gradient-to-br from-emerald-200/10 to-sky-200/10 blur-2xl" />

                <div class="relative">
                  <div class="mx-auto flex h-20 w-20 items-center justify-center rounded-3xl bg-gradient-to-br from-emerald-400 to-sky-500 text-white shadow-lg shadow-emerald-500/25">
                    <svg class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0 0 22.5 18.75V5.25A2.25 2.25 0 0 0 20.25 3H3.75A2.25 2.25 0 0 0 1.5 5.25v13.5A2.25 2.25 0 0 0 3.75 21Z" /></svg>
                  </div>
                  <h2 class="mt-6 text-3xl font-bold text-slate-950 dark:text-white">从一句提示词开始创作</h2>
                  <p class="mx-auto mt-3 max-w-2xl text-sm leading-6 text-slate-500 dark:text-slate-400">描述主体、风格、光线、构图和色彩，生成结果会保存在当前会话中。</p>
                  <div class="mt-7 grid grid-cols-1 gap-2.5 xl:grid-cols-2">
                    <button
                      v-for="(example, idx) in promptExamples"
                      :key="example"
                      class="group flex items-start gap-3 rounded-2xl border border-slate-200/80 bg-gradient-to-br from-white to-slate-50/80 p-3.5 text-left transition-all duration-200 hover:border-emerald-300 hover:shadow-md hover:shadow-emerald-500/8 focus:outline-none focus:ring-2 focus:ring-emerald-400 dark:border-slate-700/60 dark:from-slate-900 dark:to-slate-950/80 dark:hover:border-emerald-700/60 dark:hover:shadow-emerald-900/20"
                      @click="usePromptExample(example)"
                    >
                      <span class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-emerald-400 to-sky-400 text-xs font-bold text-white shadow-sm shadow-emerald-500/20 dark:from-emerald-500 dark:to-sky-500">{{ idx + 1 }}</span>
                      <span class="text-sm leading-5 text-slate-600 group-hover:text-emerald-700 dark:text-slate-300 dark:group-hover:text-emerald-300">{{ example }}</span>
                    </button>
                  </div>
                </div>
              </div>
            </section>

            <section v-if="displayTurns.length > 0" class="mx-auto flex w-full max-w-6xl flex-col gap-6 pb-6">
              <article
                v-for="(turn, turnIndex) in displayTurns"
                :key="turn.id"
                class="animate-fade-in"
                :style="{ animationDelay: `${Math.min(turnIndex, 6) * 45}ms` }"
              >
                <div class="flex justify-end">
                  <div class="max-w-[94%] rounded-3xl rounded-tr-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-right shadow-sm dark:border-emerald-900/70 dark:bg-emerald-950/30 sm:max-w-[78%]">
                    <div class="mb-1.5 flex flex-wrap justify-end gap-2 text-xs text-emerald-700/80 dark:text-emerald-300/80">
                      <span>#{{ turnIndex + 1 }}</span>
                      <span>{{ turn.model || imageStore.settings.model }}</span>
                      <span v-if="turn.createdAt">{{ formatTime(turn.createdAt) }}</span>
                    </div>
                    <p class="whitespace-pre-wrap text-sm leading-6 text-slate-950 dark:text-slate-100 sm:text-[15px]">{{ turn.prompt }}</p>
                  </div>
                </div>

                <div class="mt-3 rounded-3xl rounded-tl-lg border border-slate-200 bg-white/88 p-3 shadow-sm backdrop-blur dark:border-slate-800 dark:bg-slate-900/76 sm:p-4">
                  <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
                    <div class="flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-slate-200">
                      <span class="inline-flex h-9 w-9 items-center justify-center rounded-2xl bg-slate-100 text-emerald-600 dark:bg-slate-950 dark:text-emerald-300">
                        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09Z" /></svg>
                      </span>
                      <span>{{ turn.imageResults.length > 0 ? `${turn.imageResults.length} 张结果` : '提示词记录' }}</span>
                    </div>
                    <button
                      v-if="turn.imageResults.length > 1"
                      class="btn btn-ghost btn-sm min-h-10 cursor-pointer"
                      @click="downloadImages(turn.imageResults)"
                    >{{ t('image.downloadAll') }}</button>
                  </div>

                  <div v-if="turn.imageResults.length > 0" class="grid gap-4" :class="gridClassForCount(turn.imageResults.length)">
                    <figure
                      v-for="(img, imageIndex) in turn.imageResults"
                      :key="`${turn.id}-${imageIndex}`"
                      class="group relative overflow-hidden rounded-3xl border border-slate-200 bg-slate-50 shadow-sm transition-shadow hover:shadow-xl hover:shadow-slate-950/10 dark:border-slate-800 dark:bg-slate-950"
                    >
                      <button class="block aspect-square w-full cursor-zoom-in overflow-hidden text-left focus:outline-none focus:ring-2 focus:ring-emerald-400" :aria-label="`预览第 ${imageIndex + 1} 张图片`" @click="openTurnLightbox(turn.imageResults, imageIndex)">
                        <img
                          :src="getImageSrc(img)"
                          :alt="img.revised_prompt || turn.prompt || 'Generated image'"
                          class="h-full w-full object-cover transition duration-300 group-hover:scale-[1.025] group-hover:brightness-95"
                          loading="lazy"
                        />
                      </button>
                      <figcaption class="absolute inset-x-0 bottom-0 bg-gradient-to-t from-slate-950/80 via-slate-950/40 to-transparent p-3 pb-2.5 pt-10">
                        <div class="flex items-end justify-between gap-3">
                          <p class="line-clamp-2 text-xs leading-5 text-white/90 opacity-0 transition-opacity duration-200 group-hover:opacity-100">{{ img.revised_prompt || turn.prompt }}</p>
                          <div class="flex shrink-0 gap-1.5">
                            <button class="inline-flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl bg-white/18 text-white backdrop-blur transition-colors hover:bg-white/30 focus:outline-none focus:ring-2 focus:ring-white" aria-label="复制图片地址" @click.stop="copyImageUrl(img)">
                              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 0 1-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75" /><path stroke-linecap="round" stroke-linejoin="round" d="M9 3.75h10.125c.621 0 1.125.504 1.125 1.125v12.75c0 .621-.504 1.125-1.125 1.125H9A1.125 1.125 0 0 1 7.875 17.625V4.875C7.875 4.254 8.379 3.75 9 3.75Z" /></svg>
                            </button>
                            <button class="inline-flex h-9 w-9 cursor-pointer items-center justify-center rounded-xl bg-white/18 text-white backdrop-blur transition-colors hover:bg-white/30 focus:outline-none focus:ring-2 focus:ring-white" :aria-label="t('image.download')" @click.stop="downloadImage(img, imageIndex)">
                              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3" /></svg>
                            </button>
                          </div>
                        </div>
                      </figcaption>
                    </figure>
                  </div>

                  <div v-else class="rounded-3xl border border-dashed border-slate-200 bg-slate-50 px-4 py-8 text-center text-sm text-slate-500 dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-400">
                    这条历史记录只保存了提示词，未包含图片数据。重新发送或新生成后会在这里显示图片。
                  </div>
                </div>
              </article>
            </section>

            <section v-if="imageStore.generating" class="mx-auto w-full max-w-6xl pb-6">
              <div class="rounded-3xl border border-emerald-200 bg-white p-4 shadow-xl shadow-emerald-950/5 dark:border-emerald-900/60 dark:bg-slate-900/80">
                <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
                  <div class="flex items-center gap-3 text-sm text-slate-700 dark:text-slate-200">
                    <LoadingSpinner />
                    <div>
                      <p class="font-semibold">正在生成图片</p>
                      <p class="mt-0.5 text-xs text-slate-500 dark:text-slate-400">结果会追加到当前会话，生成较慢时请保持页面打开。</p>
                    </div>
                  </div>
                  <span class="rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-500 dark:bg-slate-950 dark:text-slate-400">{{ generatingElapsedLabel }}</span>
                </div>
                <div class="grid gap-4" :class="gridColsClass">
                  <div v-for="i in imageStore.settings.n" :key="`skel-${i}`" class="aspect-square overflow-hidden rounded-3xl bg-emerald-50/50 dark:bg-emerald-950/20">
                    <div class="h-full w-full animate-shimmer bg-gradient-to-r from-transparent via-emerald-200/60 to-transparent dark:via-emerald-800/20" />
                  </div>
                </div>
              </div>
            </section>
          </div>

          <footer class="border-t border-slate-200/80 bg-white/90 p-3 backdrop-blur-xl dark:border-slate-800 dark:bg-slate-950/82 sm:p-4">
            <div class="space-y-2">
              <div class="flex items-end gap-2">
                <textarea
                  v-model="prompt"
                  data-image-prompt-input="true"
                  placeholder="描述你想生成的图片：主体、风格、光线、构图、色彩……"
                  rows="2"
                  class="input min-h-[48px] flex-1 resize-none !rounded-2xl !text-slate-800 placeholder:!text-slate-400/70 dark:placeholder:!text-slate-500/60"
                  @keydown.ctrl.enter="handleGenerate"
                  @keydown.meta.enter="handleGenerate"
                />
                <button
                  type="button"
                  class="btn btn-secondary min-h-11 cursor-pointer px-3 2xl:hidden"
                  aria-label="打开设置"
                  @click="showMobileSettings = !showMobileSettings"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281M15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" /></svg>
                </button>
                <button
                  type="button"
                  :disabled="!canSubmit"
                  :class="['btn min-h-11 cursor-pointer rounded-full px-6 font-semibold shadow-md transition-all', canSubmit ? 'bg-gradient-to-r from-emerald-500 to-emerald-600 text-white shadow-emerald-500/25 hover:from-emerald-600 hover:to-emerald-700 hover:shadow-lg hover:shadow-emerald-500/30' : 'btn-secondary cursor-not-allowed opacity-50']"
                  @click="handleGenerate"
                >
                  {{ imageStore.generating ? '生成中' : (imageStore.editMode ? '编辑' : '生成') }}
                </button>
              </div>
            </div>
          </footer>
        </main>

        <!-- Right control panel -->
        <aside class="hidden min-h-0 border-l border-slate-200 bg-white/88 backdrop-blur-xl dark:border-slate-800 dark:bg-slate-950/78 2xl:flex 2xl:flex-col">
          <CreatorPanel />
        </aside>
      </div>

      <!-- Mobile settings sheet -->
      <Teleport to="body">
        <Transition name="fade">
          <div v-if="showMobileSettings" class="fixed inset-0 z-[9996] bg-slate-950/50 backdrop-blur-sm 2xl:hidden" @click.self="showMobileSettings = false">
            <div class="absolute inset-x-0 bottom-0 max-h-[82dvh] overflow-y-auto rounded-t-[2rem] border border-slate-200 bg-white p-4 shadow-2xl dark:border-slate-800 dark:bg-slate-950">
              <div class="mb-4 flex items-center justify-between">
                <h3 class="text-base font-semibold text-slate-950 dark:text-white">创作设置</h3>
                <button class="inline-flex h-11 w-11 cursor-pointer items-center justify-center rounded-2xl text-slate-500 hover:bg-slate-100 focus:outline-none focus:ring-2 focus:ring-emerald-400 dark:text-slate-400 dark:hover:bg-slate-900" aria-label="关闭设置" @click="showMobileSettings = false">
                  <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" /></svg>
                </button>
              </div>
              <CreatorPanel compact />
            </div>
          </div>
        </Transition>
      </Teleport>

      <ImageLightbox :visible="lightboxVisible" :images="lightboxImages" :initial-index="lightboxIndex" @close="lightboxVisible = false" />

      <Teleport to="body">
        <Transition name="fade">
          <div v-if="showClearConfirm" class="fixed inset-0 z-[9997] flex items-center justify-center bg-black/50 p-4" @click.self="showClearConfirm = false">
            <div class="card w-full max-w-sm p-6">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('image.clearHistory') }}</h3>
              <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">{{ t('image.confirmClear') }}</p>
              <div class="mt-6 flex justify-end gap-3">
                <button class="btn btn-secondary btn-sm min-h-10 cursor-pointer" @click="showClearConfirm = false">{{ t('common.cancel') }}</button>
                <button class="btn btn-danger btn-sm min-h-10 cursor-pointer" @click="confirmClear">{{ t('common.confirm') }}</button>
              </div>
            </div>
          </div>
        </Transition>
      </Teleport>

      <Teleport to="body">
        <Transition name="fade">
          <div v-if="showDeleteConfirm" class="fixed inset-0 z-[9997] flex items-center justify-center bg-black/50 p-4" @click.self="showDeleteConfirm = false">
            <div class="card w-full max-w-sm p-6">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('image.deleteSession') }}</h3>
              <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">{{ t('image.confirmDelete') }}</p>
              <div class="mt-6 flex justify-end gap-3">
                <button class="btn btn-secondary btn-sm min-h-10 cursor-pointer" @click="showDeleteConfirm = false">{{ t('common.cancel') }}</button>
                <button class="btn btn-danger btn-sm min-h-10 cursor-pointer" @click="confirmDelete">{{ t('common.confirm') }}</button>
              </div>
            </div>
          </div>
        </Transition>
      </Teleport>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted, defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import { useImageStore } from '@/stores/image'
import { useAppStore } from '@/stores/app'
import type { ImageSession, ImageRecord, ImageResult } from '@/api/image'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ImageLightbox from '@/components/image/ImageLightbox.vue'

const { t } = useI18n()
const imageStore = useImageStore()
const appStore = useAppStore()

const prompt = ref('')
const isDragging = ref(false)
const editFile = ref<File | null>(null)
const editPreview = ref<string | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const canvasRef = ref<HTMLElement | null>(null)
const mobileShowSessions = ref(false)
const lightboxVisible = ref(false)
const lightboxIndex = ref(0)
const lightboxImages = ref<ImageResult[]>([])
const showClearConfirm = ref(false)
const showDeleteConfirm = ref(false)
const pendingDeleteId = ref<string | null>(null)
const showMobileSettings = ref(false)
const lastError = ref('')
const failedPrompt = ref('')
const activePrompt = ref('')
const generatingStartedAt = ref<number | null>(null)
const nowTick = ref(Date.now())
let tickTimer: number | undefined

const promptExamples = [
  '一只可爱的橘猫，柔和光线，卡通插画风格',
  '未来城市夜景，霓虹灯，电影感构图，超清细节',
  '森林里的玻璃小屋，晨雾，温暖光线，治愈系插画',
  '高端咖啡产品海报，极简构图，柔和阴影，商业摄影风格'
]

const settingsFields = computed(() => [
  {
    key: 'model', label: t('image.model'), hint: '建议使用 auto，让后端选择可用模型。',
    options: [
      { value: 'auto', label: 'Auto' },
      { value: 'gpt-image-1', label: 'gpt-image-1' },
      { value: 'dall-e-3', label: 'dall-e-3' },
      { value: 'dall-e-2', label: 'dall-e-2' },
    ]
  },
  {
    key: 'size', label: t('image.size'), hint: '方图适合头像和插画，横图适合封面。',
    options: [
      { value: 'auto', label: 'Auto' },
      { value: '1024x1024', label: '1024 × 1024' },
      { value: '1536x1024', label: '1536 × 1024' },
      { value: '1024x1536', label: '1024 × 1536' },
    ]
  },
  {
    key: 'quality', label: t('image.quality'), hint: '高质量更慢，失败时可先用 auto 或 medium。',
    options: [
      { value: 'auto', label: 'Auto' },
      { value: 'low', label: 'Low' },
      { value: 'medium', label: 'Medium' },
      { value: 'high', label: 'High' },
    ]
  },
  {
    key: 'n', label: t('image.count'), hint: '多图会消耗更多额度，建议先生成 1 张。',
    options: [
      { value: '1', label: '1' },
      { value: '2', label: '2' },
      { value: '3', label: '3' },
      { value: '4', label: '4' },
    ]
  },
])

function getSettingValue(key: string): string {
  if (key === 'model') return imageStore.settings.model
  if (key === 'size') return imageStore.settings.size
  if (key === 'quality') return imageStore.settings.quality
  if (key === 'n') return String(imageStore.settings.n)
  return ''
}

function setSettingValue(key: string, value: string) {
  if (key === 'model') imageStore.settings.model = value
  else if (key === 'size') imageStore.settings.size = value
  else if (key === 'quality') imageStore.settings.quality = value
  else if (key === 'n') imageStore.settings.n = Number(value)
}

function setMode(editMode: boolean) {
  imageStore.editMode = editMode
  lastError.value = ''
}

interface DisplayTurn {
  id: string
  prompt: string
  model?: string
  createdAt?: string
  imageResults: ImageResult[]
}

function storedImageToResult(image: string): ImageResult | null {
  const value = String(image || '').trim()
  if (!value) return null
  if (value.startsWith('data:') || value.startsWith('http://') || value.startsWith('https://') || value.startsWith('/')) return { url: value }
  return { b64_json: value }
}

function recordToDisplayTurn(record: ImageRecord): DisplayTurn {
  return {
    id: record.id,
    prompt: record.prompt,
    model: record.model,
    createdAt: record.created_at,
    imageResults: (record.images || []).flatMap((image) => {
      const result = storedImageToResult(image)
      return result ? [result] : []
    })
  }
}

const sessions = computed(() => imageStore.sessions)
const results = computed(() => imageStore.results)
const currentSession = computed(() => imageStore.currentSession)
const currentSessionId = computed(() => imageStore.currentSession?.id || null)
const canSubmit = computed(() => Boolean(prompt.value.trim()) && !imageStore.generating && (!imageStore.editMode || Boolean(editFile.value)))
const activeSettingsSummary = computed(() => `${imageStore.settings.model} · ${imageStore.settings.size} · ${imageStore.settings.quality} · ${imageStore.settings.n} 张`)
const allDisplayedImages = computed(() => displayTurns.value.flatMap((turn) => turn.imageResults))

const displayTurns = computed<DisplayTurn[]>(() => {
  const session = currentSession.value
  const records = session?.records || []
  if (records.length > 0) return records.map(recordToDisplayTurn)
  if (results.value.length > 0) {
    return [{
      id: session?.id || 'current-results',
      prompt: session?.title || prompt.value || activePrompt.value || '当前生成',
      model: imageStore.settings.model,
      createdAt: session?.updated_at || session?.created_at,
      imageResults: [...results.value]
    }]
  }
  return []
})

const generatingElapsedLabel = computed(() => {
  if (!generatingStartedAt.value) return '刚开始'
  const seconds = Math.max(0, Math.floor((nowTick.value - generatingStartedAt.value) / 1000))
  if (seconds < 60) return `${seconds}s`
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
})

function gridClassForCount(count: number) {
  if (count <= 1) return 'grid-cols-1 max-w-xl'
  if (count === 2) return 'grid-cols-1 sm:grid-cols-2 max-w-4xl'
  return 'grid-cols-1 sm:grid-cols-2 xl:grid-cols-3'
}

const gridColsClass = computed(() => gridClassForCount(imageStore.settings.n))

function getImageSrc(img: ImageResult): string {
  if (img.url) return img.url
  if (img.b64_json) {
    const value = img.b64_json.trim()
    if (value.startsWith('/') || value.startsWith('http://') || value.startsWith('https://') || value.startsWith('data:')) return value
    return `data:image/png;base64,${value}`
  }
  return ''
}

function formatTime(dateStr: string): string {
  if (!dateStr) return ''
  try {
    const d = new Date(dateStr), now = new Date(), diff = now.getTime() - d.getTime()
    if (diff < 60000) return '刚刚'
    if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
    const m = String(d.getMonth() + 1).padStart(2, '0'), day = String(d.getDate()).padStart(2, '0')
    const h = String(d.getHours()).padStart(2, '0'), min = String(d.getMinutes()).padStart(2, '0')
    return d.getFullYear() === now.getFullYear() ? `${m}-${day} ${h}:${min}` : `${d.getFullYear()}-${m}-${day} ${h}:${min}`
  } catch { return dateStr }
}

function scrollCanvasToLatest(behavior: 'auto' | 'smooth' = 'smooth') {
  void nextTick(() => {
    const element = canvasRef.value
    if (!element) return
    element.scrollTo({ top: element.scrollHeight, behavior })
  })
}

function focusPromptInput() {
  void nextTick(() => {
    const input = document.querySelector<HTMLTextAreaElement>('[data-image-prompt-input="true"]')
    if (!input) return
    input.focus()
    input.scrollIntoView({ behavior: 'smooth', block: 'center' })
  })
}

function usePromptExample(example: string) {
  prompt.value = example
  lastError.value = ''
  focusPromptInput()
  appStore.showSuccess('已填入示例提示词，点击“生成”开始生图')
}

async function handleNewSession() {
  const title = prompt.value.trim().slice(0, 50) || t('image.newSession')
  await imageStore.createAndSelectSession(title)
  mobileShowSessions.value = false
  lastError.value = ''
  scrollCanvasToLatest('auto')
}

async function handleSelectSession(session: ImageSession) {
  imageStore.selectSession(session)
  mobileShowSessions.value = false
  await imageStore.loadSession(session.id)
  lastError.value = ''
  scrollCanvasToLatest('auto')
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
async function confirmClear() { await imageStore.removeAllSessions(); showClearConfirm.value = false }

function friendlyErrorMessage(error: any): string {
  const raw = String(error?.response?.data?.message || error?.message || error || '')
  if (/token_invalidated|invalidated|invalid access token|authentication token/i.test(raw)) {
    return '当前选中的 ChatGPT 账号 Token 已失效。系统会跳过异常账号，请稍后重试；如果连续失败，请在管理后台刷新或重新授权账号。'
  }
  if (/policy|safety|违反|安全|不能|无法|refused/i.test(raw)) {
    return '提示词可能触发了图片安全策略。请去掉年龄暗示、敏感描述或容易误判的词，再重新生成。'
  }
  if (/没有可用|no available/i.test(raw)) {
    return '当前没有可用的图片生成账号。请检查账号状态、额度和限流情况。'
  }
  if (/timeout|deadline|超时/i.test(raw)) {
    return '图片生成等待超时。可以稍后重试，或降低图片数量/质量。'
  }
  return raw || '图片生成失败，请稍后重试。'
}

async function handleGenerate() {
  const trimmed = prompt.value.trim()
  if (!trimmed || imageStore.generating) return
  if (imageStore.editMode && !editFile.value) {
    lastError.value = '图片编辑模式需要先上传参考图。'
    return
  }
  if (!imageStore.currentSession) await imageStore.createAndSelectSession(trimmed.slice(0, 50))

  activePrompt.value = trimmed
  generatingStartedAt.value = Date.now()
  lastError.value = ''
  failedPrompt.value = ''
  scrollCanvasToLatest('smooth')

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
      clearEditImage()
    } else {
      await imageStore.generate(trimmed)
    }
    prompt.value = ''
    activePrompt.value = ''
    scrollCanvasToLatest('smooth')
  } catch (e: any) {
    console.error('Generation failed:', e)
    failedPrompt.value = trimmed
    lastError.value = friendlyErrorMessage(e)
    appStore.showError(lastError.value, 15000)
  } finally {
    generatingStartedAt.value = null
  }
}

function retryFailedPrompt() {
  if (!failedPrompt.value) return
  prompt.value = failedPrompt.value
  void handleGenerate()
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
  if (!src) return
  const link = document.createElement('a')
  link.href = src
  link.download = `image-${Date.now()}-${index}.png`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}
function downloadImages(images: ImageResult[]) { images.forEach((img, i) => setTimeout(() => downloadImage(img, i), i * 250)) }
function downloadAll() { downloadImages(allDisplayedImages.value) }
function openTurnLightbox(images: ImageResult[], index: number) {
  lightboxImages.value = images
  lightboxIndex.value = index
  lightboxVisible.value = true
}
async function copyImageUrl(img: ImageResult) {
  const src = getImageSrc(img)
  if (!src) return
  try {
    await navigator.clipboard.writeText(new URL(src, window.location.origin).toString())
    appStore.showSuccess('图片地址已复制')
  } catch {
    appStore.showError('复制失败')
  }
}

const CreatorPanel = defineComponent({
  name: 'CreatorPanel',
  props: { compact: { type: Boolean, default: false } },
  setup() {
    return () => h('div', { class: 'flex min-h-0 flex-col gap-4 overflow-y-auto p-4' }, [
      h('section', { class: 'rounded-3xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-900/70' }, [
        h('div', { class: 'mb-3 flex items-center justify-between gap-3' }, [
          h('div', null, [
            h('h3', { class: 'font-semibold text-slate-950 dark:text-white' }, '创作模式')
          ])
        ]),
        h('div', { class: 'grid grid-cols-2 gap-2' }, [
          h('button', { class: ['min-h-11 cursor-pointer rounded-2xl text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-400', !imageStore.editMode ? 'bg-emerald-500 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-950 dark:text-slate-300 dark:hover:bg-slate-800'], onClick: () => setMode(false) }, t('image.textToImage')),
          h('button', { class: ['min-h-11 cursor-pointer rounded-2xl text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-emerald-400', imageStore.editMode ? 'bg-emerald-500 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-950 dark:text-slate-300 dark:hover:bg-slate-800'], onClick: () => setMode(true) }, t('image.imageEdit'))
        ])
      ]),

      imageStore.editMode ? h('section', { class: 'rounded-3xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-900/70' }, [
        h('h3', { class: 'font-semibold text-slate-950 dark:text-white' }, '参考图片'),
        h('div', {
          class: ['relative mt-3 flex min-h-40 cursor-pointer flex-col items-center justify-center rounded-3xl border-2 border-dashed p-4 text-center transition-colors', isDragging.value ? 'border-emerald-400 bg-emerald-50 dark:bg-emerald-950/30' : 'border-slate-200 bg-slate-50 hover:border-emerald-300 dark:border-slate-800 dark:bg-slate-950 dark:hover:border-emerald-800'],
          onDragover: (event: DragEvent) => { event.preventDefault(); isDragging.value = true },
          onDragleave: (event: DragEvent) => { event.preventDefault(); isDragging.value = false },
          onDrop: (event: DragEvent) => { event.preventDefault(); handleDrop(event) }
        }, editPreview.value ? [
          h('img', { src: editPreview.value, class: 'max-h-56 rounded-2xl object-contain shadow-md', alt: 'Reference image' }),
          h('button', { class: 'absolute right-3 top-3 inline-flex h-10 w-10 cursor-pointer items-center justify-center rounded-2xl bg-red-500 text-white shadow hover:bg-red-600 focus:outline-none focus:ring-2 focus:ring-red-300', 'aria-label': '移除参考图', onClick: clearEditImage }, '×')
        ] : [
          h('svg', { class: 'h-9 w-9 text-slate-300 dark:text-slate-600', fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'm2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159M3.75 21h16.5A2.25 2.25 0 0 0 22.5 18.75V5.25A2.25 2.25 0 0 0 20.25 3H3.75A2.25 2.25 0 0 0 1.5 5.25v13.5A2.25 2.25 0 0 0 3.75 21Z' })]),
          h('p', { class: 'mt-2 text-sm font-medium text-slate-700 dark:text-slate-200' }, '点击或拖拽上传'),
          h('p', { class: 'mt-1 text-xs text-slate-500 dark:text-slate-400' }, '编辑模式必须提供参考图'),
          h('input', { ref: fileInput, type: 'file', accept: 'image/*', class: 'absolute inset-0 cursor-pointer opacity-0', onChange: handleFileSelect })
        ])
      ]) : null,

      h('section', { class: 'rounded-3xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-900/70' }, [
        h('div', { class: 'mb-4' }, [
          h('h3', { class: 'font-semibold text-slate-950 dark:text-white' }, '生成参数'),
          h('p', { class: 'mt-1 text-xs text-slate-500 dark:text-slate-400' }, activeSettingsSummary.value)
        ]),
        h('div', { class: 'space-y-4' }, settingsFields.value.map((field) => h('label', { class: 'block' }, [
          h('span', { class: 'mb-1.5 block text-sm font-medium text-slate-700 dark:text-slate-200' }, field.label),
          h('select', { value: getSettingValue(field.key), class: 'input min-h-11 w-full !rounded-2xl', onChange: (event: Event) => setSettingValue(field.key, (event.target as HTMLSelectElement).value) }, field.options.map((opt) => h('option', { value: opt.value }, opt.label))),
          h('span', { class: 'mt-1 block text-xs leading-5 text-slate-500 dark:text-slate-400' }, field.hint)
        ])))
      ]),

      h('section', { class: 'rounded-3xl border border-emerald-200 bg-emerald-50/80 p-4 shadow-sm dark:border-emerald-900/60 dark:bg-emerald-950/20' }, [
        h('h3', { class: 'font-semibold text-emerald-950 dark:text-emerald-100' }, '提示词建议'),
        h('ul', { class: 'mt-3 space-y-2 text-sm leading-6 text-emerald-800 dark:text-emerald-200' }, [
          h('li', null, '写清主体、风格、光线、构图和色彩。'),
          h('li', null, '避免年龄暗示、性化描述和容易触发安全策略的词。'),
          h('li', null, '失败时先减少敏感词，再降低数量或质量重试。')
        ])
      ]),

      allDisplayedImages.value.length > 1 ? h('button', { class: 'btn btn-secondary min-h-11 w-full cursor-pointer', onClick: downloadAll }, t('image.downloadAll')) : null
    ])
  }
})

onMounted(() => {
  imageStore.loadSessions()
  tickTimer = window.setInterval(() => { nowTick.value = Date.now() }, 1000)
})
onUnmounted(() => {
  if (tickTimer) window.clearInterval(tickTimer)
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@keyframes fade-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}
.animate-fade-in {
  animation: fade-in 0.35s ease-out both;
}

@keyframes shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}
.animate-shimmer {
  animation: shimmer 1.6s infinite;
}

@media (prefers-reduced-motion: reduce) {
  .animate-fade-in,
  .animate-shimmer {
    animation: none;
  }
  * {
    transition-duration: 0.01ms !important;
  }
}
</style>

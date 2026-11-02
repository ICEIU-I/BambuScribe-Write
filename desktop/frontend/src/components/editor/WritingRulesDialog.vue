<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import markdown from '../../../../docs/writing-syntax.md?raw'
import { rulePages } from '../../state/writingRules'
import AppIcon from '../AppIcon.vue'
import WritingRulePreview from './WritingRulePreview.vue'
import WritingRuleText from './WritingRuleText.vue'
const props = defineProps<{ open: boolean; returnFocus?: HTMLElement; fontId?: string }>()
const emit = defineEmits<{ close: [] }>()
const dialog = ref<HTMLDialogElement>(),
  notice = ref('')
const pages = rulePages(markdown).filter((page) => page.title !== '错误与兼容'),
  index = ref(0)
const marks = ['Aa', 'x₂', 'a/b', '√', 'x̄', '=', '±']
const introductions = [
  '普通文字直接输入，特殊结构用半角符号限定范围。',
  '用 _{内容} 写下标，用 ^{内容} 写上标。',
  '用 \\frac{分子}{分母} 输入上下分式。',
  '用 \\sqrt{内容} 指定根号的覆盖范围。',
  '用 \\bar{内容} 给字符或表达式添加上横线。',
  '完整公式单独一行，用 \\[ 和 \\] 包住。',
  '希腊字母和运算符直接输入实际字符。',
]
const shortRules = [
  ['花括号限定范围，不显示；圆括号会显示。', '正文直接写，长公式另起一行。'],
  ['一个字符也要写花括号，如 `x^{2}`。', '上下标可同时使用，如 `x_{i}^{2}`。'],
  ['先写分子，再写分母；两项都不能为空。', '按实际行高铺满一行，上下留出间隙。'],
  ['横线随内容的实际宽度伸缩，高度自动适应。', '三次根写作 `\\sqrt[3]{x}`，次数在左上角。'],
  ['花括号里写多少内容，横线就覆盖多少。', '`\\bar{U}_{H}` 只覆盖 U；`\\bar{U_{H}}` 覆盖整体。'],
  ['定界符和公式写在同一行，外面不加正文。', '双击纸面公式可编辑；旧写法 `\\(…\\)` 也可用。'],
  ['分号只是标点，不表示换行。', '数值与单位间留一个空格，如 `10 mA`。'],
]
const rules = computed(() => shortRules[index.value])
const exampleIndex = ref(0)
const examples = computed(() => {
  const lines = pages[index.value].blocks
    .filter((b) => b.kind === 'code')
    .flatMap((b) => b.text!.split('\n'))
    .filter(Boolean)
  const formatted = lines.filter((line) => /[_^\\√]/.test(line))
  return index.value === 5 && formatted.length ? formatted : lines
})
function nextExample() {
  exampleIndex.value++
  notice.value = ''
}
const example = computed(() => examples.value[exampleIndex.value % examples.value.length])
async function selectPage(value: number) {
  index.value = Math.max(0, Math.min(pages.length - 1, value))
  notice.value = ''
  exampleIndex.value = 0
  await nextTick()
  dialog.value?.querySelector('.writing-rules-body')?.scrollTo(0, 0)
}
function pageKey(event: KeyboardEvent) {
  const target =
    event.key === 'ArrowLeft' || event.key === 'ArrowUp'
      ? index.value - 1
      : event.key === 'ArrowRight' || event.key === 'ArrowDown'
        ? index.value + 1
        : event.key === 'Home'
          ? 0
          : event.key === 'End'
            ? pages.length - 1
            : undefined
  if (target === undefined) return
  event.preventDefault()
  void selectPage(target).then(() =>
    dialog.value?.querySelector<HTMLButtonElement>('[role=tab][aria-selected=true]')?.focus(),
  )
}
watch(
  () => props.open,
  async (open) => {
    await nextTick()
    if (open) {
      index.value = 0
      exampleIndex.value = 0
      notice.value = ''
      dialog.value?.showModal()
      dialog.value?.querySelector('.writing-rules-body')?.scrollTo(0, 0)
    } else {
      dialog.value?.close()
      props.returnFocus?.focus()
    }
  },
)
async function copy(text: string) {
  try {
    const runtime = (
      window as unknown as { runtime?: { ClipboardSetText?: (text: string) => Promise<boolean> } }
    ).runtime
    if (runtime?.ClipboardSetText) {
      if (!(await runtime.ClipboardSetText(text))) throw new Error('copy')
    } else await navigator.clipboard.writeText(text)
    notice.value = '已复制，可粘贴到正文'
  } catch {
    notice.value = '复制失败，请选中示例后按 ⌘C / Ctrl+C'
  }
}
</script>
<template>
  <dialog
    ref="dialog"
    class="writing-rules-dialog"
    aria-labelledby="writing-rules-title"
    @cancel.prevent="emit('close')"
  >
    <aside class="rules-navigation">
      <h2 id="writing-rules-title">输入规则</h2>
      <nav role="tablist" aria-label="规则类别" aria-orientation="vertical" @keydown="pageKey">
        <button
          v-for="(page, i) in pages"
          :id="`writing-rule-tab-${i}`"
          :key="page.title"
          type="button"
          role="tab"
          :aria-selected="index === i"
          :tabindex="index === i ? 0 : -1"
          aria-controls="writing-rule-page"
          @click="selectPage(i)"
        >
          <span class="category-mark" aria-hidden="true">{{ marks[i] }}</span>
          <span class="category-label">{{ page.title }}</span>
        </button>
      </nav>
      <span class="rules-scope">正文与公式</span>
    </aside>
    <section class="rules-main">
      <header class="rules-heading">
        <div>
          <span class="rules-counter"
            >{{ String(index + 1).padStart(2, '0') }} /
            {{ String(pages.length).padStart(2, '0') }}</span
          >
          <h3>{{ pages[index].title }}</h3>
        </div>
        <button
          class="icon-button rules-close"
          aria-label="关闭输入规则"
          autofocus
          @click="emit('close')"
        >
          <AppIcon name="close" :size="18" />
        </button>
      </header>
      <div
        id="writing-rule-page"
        class="writing-rules-body"
        role="tabpanel"
        :aria-labelledby="`writing-rule-tab-${index}`"
        tabindex="0"
      >
        <p class="rules-intro">{{ introductions[index] }}</p>
        <ul class="rules-points">
          <li v-for="(line, i) in rules" :key="i"><WritingRuleText :text="line" /></li>
        </ul>
        <div class="rule-example-card">
          <div class="rule-example-label">
            <span>输入写法</span
            ><button class="rule-copy" :aria-label="`复制示例${index + 1}`" @click="copy(example)">
              <AppIcon name="copy" :size="14" />复制
            </button>
          </div>
          <pre>{{ example }}</pre>
          <div class="rule-example-label preview-label"><span>实际手写效果</span></div>
          <WritingRulePreview v-if="open" :text="example" :font-id="fontId" />
          <button v-if="examples.length > 1" class="rule-more" @click="nextExample">
            换个示例 <span>{{ (exampleIndex % examples.length) + 1 }} / {{ examples.length }}</span
            ><AppIcon name="chevronRight" :size="13" />
          </button>
        </div>
      </div>
      <footer>
        <span class="rule-notice" aria-live="polite">{{ notice }}</span>
        <div class="writing-rules-pager">
          <button
            class="icon-button"
            title="上一页"
            aria-label="上一页"
            :disabled="index === 0"
            @click="selectPage(index - 1)"
          >
            <AppIcon name="chevronLeft" :size="17" /></button
          ><span>{{ index + 1 }} / {{ pages.length }}</span
          ><button
            class="icon-button"
            title="下一页"
            aria-label="下一页"
            :disabled="index === pages.length - 1"
            @click="selectPage(index + 1)"
          >
            <AppIcon name="chevronRight" :size="17" />
          </button>
        </div>
      </footer>
    </section>
  </dialog>
</template>
<style scoped>
.writing-rules-dialog {
  width: min(800px, calc(100vw - 40px));
  height: min(590px, calc(100dvh - 64px));
  padding: 0;
  border: 1px solid var(--border-heavy);
  border-radius: 16px;
  background: var(--surface);
  color: var(--text);
  overflow: hidden;
  box-shadow: 0 24px 90px rgb(0 0 0 / 22%);
}
.writing-rules-dialog[open] {
  display: grid;
  grid-template-columns: 166px minmax(0, 1fr);
}
.writing-rules-dialog::backdrop {
  background: rgb(0 0 0 / 25%);
  backdrop-filter: blur(3px);
}
.rules-navigation {
  background: var(--sidebar);
  padding: 30px 12px 20px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  min-height: 0;
}
h2 {
  font-size: 14px;
  font-weight: 600;
  margin: 0 14px 26px;
}
nav {
  display: flex;
  flex-direction: column;
  gap: 5px;
  overflow: auto;
}
nav button {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--text-soft);
  padding: 10px 12px;
  font: inherit;
  font-size: 13px;
  text-align: left;
  cursor: pointer;
}
nav button:hover:not([aria-selected='true']) {
  background: var(--surface-soft);
}
nav button:focus {
  outline: none;
}
nav button:focus-visible {
  outline: 2px solid var(--text-soft);
  outline-offset: -2px;
}
nav button[aria-selected='true'] {
  background: var(--surface-selected);
  color: var(--text);
  font-weight: 600;
}
.category-mark {
  display: grid;
  place-items: center;
  height: 24px;
  text-align: center;
  font-family: Georgia, serif;
  font-size: 18px;
  font-weight: 400;
  line-height: 1;
  white-space: nowrap;
}
.category-label {
  line-height: 24px;
  white-space: nowrap;
}
.rules-scope {
  margin: auto 14px 0;
  padding-top: 18px;
  color: var(--muted);
  font-size: 11px;
}
.rules-main {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.rules-heading {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 26px 30px 0;
  flex-shrink: 0;
}
.rules-counter {
  font-size: 10px;
  letter-spacing: 0.12em;
  color: var(--muted);
  font-variant-numeric: tabular-nums;
}
h3 {
  font-size: 23px;
  font-weight: 600;
  margin: 8px 0 0;
  letter-spacing: -0.02em;
}
.rules-close {
  margin: -12px -16px 0 0;
  color: var(--muted);
}
.writing-rules-body {
  min-height: 0;
  overflow: auto;
  padding: 0 30px 14px;
  flex: 1;
  font-size: 13px;
  line-height: 1.7;
}
.rules-intro {
  margin: 12px 0 16px;
  color: var(--text-soft);
}
.rules-points {
  margin: 0 0 22px;
  padding-left: 17px;
}
.rules-points li {
  padding-left: 2px;
  margin: 7px 0;
}
.rules-points li::marker {
  color: var(--muted);
  font-size: 9px;
}
:deep(code),
pre {
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 12px;
  overflow-wrap: anywhere;
}
:deep(code) {
  color: var(--text);
}
.rule-example-card {
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 12px 14px 8px;
  background: var(--surface-soft);
}
.rule-example-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
  color: var(--muted);
}
.rule-copy {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 0;
  background: transparent;
  color: var(--text-soft);
  padding: 2px 0 2px 8px;
  cursor: pointer;
  font-size: 11px;
}
pre {
  margin: 12px 0 16px;
  line-height: 1.65;
  white-space: pre-wrap;
}
.preview-label {
  padding: 0 0 8px;
}
.rule-more {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 8px 0 0 auto;
  padding: 4px 0 2px 8px;
  border: 0;
  background: transparent;
  color: var(--text-soft);
  font-size: 11px;
  cursor: pointer;
}
.rule-more span {
  color: var(--muted);
  margin-left: 4px;
}
footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 54px;
  padding: 6px 24px 6px 30px;
  flex-shrink: 0;
}
.rule-notice {
  font-size: 11px;
  color: var(--text-soft);
}
.writing-rules-pager {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  color: var(--muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}
@media (max-width: 600px) {
  .writing-rules-dialog[open] {
    grid-template-columns: 120px minmax(0, 1fr);
  }
  .rules-navigation {
    padding: 24px 6px 16px;
  }
  nav button {
    gap: 5px;
    padding: 10px 6px;
    font-size: 12px;
  }
  .rules-heading {
    padding: 22px 20px 0;
  }
  .writing-rules-body {
    padding: 0 20px 10px;
  }
  footer {
    padding: 6px 14px 6px 20px;
  }
}
</style>

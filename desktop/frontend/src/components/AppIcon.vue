<script setup lang="ts">
withDefaults(defineProps<{ name: string; size?: number }>(), { size: 18 })
const paths: Record<string, string> = {
  'input-rules':
    'M13.3 3.5Q13.6 2.5 14.7 2.5H15.7Q17 2.5 16.55 3.75L12.8 14.3Q12.55 15 11.8 15H11Q9.9 15 10.25 13.9L13.3 3.5ZM11.5 19a1.9 1.9 0 1 1-3.8 0 1.9 1.9 0 0 1 3.8 0Z',
  sidebar: 'M4 3h16a1 1 0 0 1 1 1v16a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1Zm5 0v18',
  folder: 'M3 7V5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v2M3 7h18l-3 13H2L3 7Z',
  save: 'M5 3h12l4 4v14H3V3h2Zm2 0v6h10V3M7 21v-8h10v8',
  image: 'M3 3h18v18H3V3Zm0 13 5-5 5 5 3-3 5 5M15 7h.01',
  undo: 'M3 9h10a6 6 0 0 1 0 12M3 9l5-5M3 9l5 5',
  redo: 'M21 9H11a6 6 0 0 0 0 12m10-12-5-5m5 5-5 5',
  'align-left': 'M4 4h16M4 9h10M4 14h16M4 19h10',
  'align-center': 'M4 4h16M7 9h10M4 14h16M7 19h10',
  'align-right': 'M4 4h16M10 9h10M4 14h16M10 19h10',

  sun: 'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8ZM12 2v2m0 16v2M2 12h2m16 0h2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4',
  moon: 'M20.9 13A9 9 0 0 1 11 3.1 9 9 0 1 0 20.9 13Z',
  pen: 'm15 5 4 4M4 20l4-1L20 7a2.8 2.8 0 0 0-4-4L4 15v5Z',
  table: 'M3 3h18v18H3V3Zm0 6h18M3 15h18M9 3v18M15 3v18',
  text: 'M5 5h14M12 5v14M8 19h8',
  eraser: 'm4 15 9-9a2.8 2.8 0 0 1 4 0l3 3-9 9H6l-2-2Zm9 3h7',
  file: 'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6Zm0 0v6h6M8 13h8M8 17h5',
  'file-plus':
    'M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6Zm0 0v6h6M8 15h8m-4-4v8',
  printer:
    'M6 9V3h12v6M6 17H4a2 2 0 0 1-2-2v-4a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v4a2 2 0 0 1-2 2h-2M6 14h12v8H6v-8ZM18 12h.01',
  download: 'M12 3v12m-5-5 5 5 5-5M4 17v4h16v-4',
  upload: 'M12 16V4m-5 5 5-5 5 5M4 17v4h16v-4',
  chevronLeft: 'm14 6-6 6 6 6',
  chevronRight: 'm10 6 6 6-6 6',
  close: 'm6 6 12 12M6 18 18 6',
  trash: 'M3 6h18M9 6V3h6v3M5 6l1 15h12l1-15M10 10v7m4-7v7',
  copy: 'M8 8h13v13H8V8ZM16 8V3H3v13h5',
  check: 'm5 12 4 4L19 6',
  play: 'm7 4 14 8-14 8V4Z',
  pause: 'M8 4v16M16 4v16',
  stop: 'M5 5h14v14H5z',
  settings: 'M4 6h16M4 12h16M4 18h16M8 3v6M16 9v6M10 15v6',
  home: 'm3 10 9-8 9 8M5 9v12h14V9M9 21v-8h6v8',
  frame: 'M8 3H3v5m13-5h5v5M3 16v5h5m8 0h5v-5',
  wifi: 'M2 8a16 16 0 0 1 20 0M5 12a11 11 0 0 1 14 0M8.5 16a5.5 5.5 0 0 1 7 0M12 20h.01',
  minus: 'M5 12h14',
  plus: 'M5 12h14M12 5v14',
  alert: 'm12 3 10 18H2L12 3Zm0 6v5m0 3h.01',
}
</script>
<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.65"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path
      :d="paths[name] || paths.file"
      :fill="name === 'input-rules' ? 'currentColor' : undefined"
      :stroke="name === 'input-rules' ? 'none' : undefined"
    />
  </svg>
</template>

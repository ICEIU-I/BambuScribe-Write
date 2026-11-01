import { ref } from 'vue'

// One hint at a time, including when keyboard focus and the pointer differ.
export const activeHint = ref<string | null>(null)

<script setup lang="ts">
import { ruleText } from '../../state/writingRules'
defineProps<{ text: string }>()
</script>
<template>
  <component :is="part.kind" v-for="(part, i) in ruleText(text)" :key="i">{{
    part.text
  }}</component>
</template>

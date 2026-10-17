<!--
  AxiOps 标志 · Orbit A（轨道轴心）
  构成：深空渐变瓷砖 + 玻璃高光边 + 倾斜轨道环 + 平顶 A 主体 + 青色轴心节点
  小尺寸（16~32px）下仍能辨认：外轮廓是瓷砖，中心是 A 与高亮节点。
-->
<template>
  <div class="axiops-logo" :style="logoStyle" role="img" aria-label="AxiOps">
    <svg viewBox="0 0 128 128" class="logo-svg" aria-hidden="true">
      <defs>
        <linearGradient :id="`${uid}-tile`" x1="6%" y1="0%" x2="94%" y2="100%">
          <stop offset="0%" stop-color="#0A2A6B" />
          <stop offset="46%" stop-color="#1461E6" />
          <stop offset="100%" stop-color="#22C4FF" />
        </linearGradient>
        <linearGradient :id="`${uid}-glass`" x1="0%" y1="0%" x2="20%" y2="100%">
          <stop offset="0%" stop-color="#ffffff" stop-opacity="0.24" />
          <stop offset="38%" stop-color="#ffffff" stop-opacity="0.03" />
          <stop offset="100%" stop-color="#04091c" stop-opacity="0.18" />
        </linearGradient>
        <linearGradient :id="`${uid}-orbit`" x1="0%" y1="0%" x2="100%" y2="60%">
          <stop offset="0%" stop-color="#A8F6FF" stop-opacity="0.95" />
          <stop offset="70%" stop-color="#8FF3FF" stop-opacity="0.35" />
          <stop offset="100%" stop-color="#8FF3FF" stop-opacity="0.08" />
        </linearGradient>
        <linearGradient :id="`${uid}-glyph`" x1="10%" y1="0%" x2="90%" y2="100%">
          <stop offset="0%" stop-color="#ffffff" />
          <stop offset="100%" stop-color="#AFD6FF" />
        </linearGradient>
        <radialGradient :id="`${uid}-core`" cx="38%" cy="32%" r="72%">
          <stop offset="0%" stop-color="#E4FDFF" />
          <stop offset="60%" stop-color="#7BEBFF" />
          <stop offset="100%" stop-color="#25C2F5" />
        </radialGradient>
        <clipPath :id="`${uid}-clip`">
          <rect width="128" height="128" rx="31" />
        </clipPath>
      </defs>

      <g :clip-path="`url(#${uid}-clip)`">
        <rect width="128" height="128" :fill="`url(#${uid}-tile)`" />

        <ellipse
          cx="64"
          cy="64"
          rx="42"
          ry="15"
          fill="none"
          :stroke="`url(#${uid}-orbit)`"
          stroke-width="2.6"
          transform="rotate(-22 64 64)"
        />

        <path
          d="M36 97 L60.5 31 L67.5 31 L92 97"
          fill="none"
          :stroke="`url(#${uid}-glyph)`"
          stroke-width="14"
          stroke-linecap="round"
          stroke-linejoin="round"
        />
        <path d="M45 73 L83 73" stroke="#5FE3FF" stroke-width="11" stroke-linecap="round" />

        <circle cx="103" cy="48.3" r="9.6" fill="none" stroke="#ffffff" stroke-opacity="0.32" stroke-width="1.5" />
        <circle cx="103" cy="48.3" r="6" :fill="`url(#${uid}-core)`" />

        <rect width="128" height="128" :fill="`url(#${uid}-glass)`" />
      </g>

      <rect
        x="0.8"
        y="0.8"
        width="126.4"
        height="126.4"
        rx="30.4"
        fill="none"
        stroke="#ffffff"
        stroke-opacity="0.2"
        stroke-width="1.6"
      />
    </svg>
  </div>
</template>

<script lang="ts">
  // 模块级计数器：保证同页多个 logo 实例的渐变 id 互不冲突
  let logoInstanceCount = 0
</script>

<script setup lang="ts">
  defineOptions({ name: 'ArtLogo' })

  interface Props {
    /** logo 大小 */
    size?: number | string
  }

  const props = withDefaults(defineProps<Props>(), {
    size: 36
  })

  const uid = `ax-logo-${++logoInstanceCount}`

  const logoStyle = computed(() => ({
    width: `${props.size}px`,
    height: `${props.size}px`
  }))
</script>

<style scoped>
  .axiops-logo {
    display: inline-flex;
    flex: 0 0 auto;
    overflow: hidden;
    border-radius: 25%;
    box-shadow:
      0 2px 8px rgb(11 95 224 / 22%),
      inset 0 0 0 1px rgb(255 255 255 / 8%);
  }

  .logo-svg {
    display: block;
    width: 100%;
    height: 100%;
  }
</style>

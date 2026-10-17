<template>
  <div class="platform-intro-page" :style="pageStyle">
    <div class="ambient-light"></div>

    <div class="page-container">
      <section class="hero-section">
        <div class="hero-content">
          <div class="platform-meta">
            <div class="platform-icon-box">
              <ArtSvgIcon :icon="config.icon" :size="32" />
            </div>
            <span class="platform-en-name">{{ config.enName }}</span>
          </div>

          <h1 class="hero-title">{{ config.name }}</h1>
          <h2 class="hero-subtitle">{{ config.subtitle }}</h2>
          <p class="hero-desc">{{ config.description }}</p>

          <div class="stats-row">
            <div v-for="(item, index) in config.highlights" :key="index" class="stat-item">
              <div class="stat-value">
                {{ item.value
                }}<span v-if="item.suffix" class="stat-suffix">{{ item.suffix }}</span>
              </div>
              <div class="stat-label">{{ item.label }}</div>
            </div>
          </div>
        </div>

        <div class="hero-visual">
          <div class="visual-circle circle-1"></div>
          <div class="visual-circle circle-2"></div>
          <div class="glass-card">
            <div class="card-header">
              <span class="dot red"></span>
              <span class="dot yellow"></span>
              <span class="dot green"></span>
            </div>
            <div class="card-body">
              <div class="skeleton-line w-70"></div>
              <div class="skeleton-line w-50"></div>
              <div class="skeleton-line w-90"></div>
              <div class="skeleton-graph"></div>
            </div>
            <div class="floating-badge badge-1">
              <ArtSvgIcon icon="ri:cpu-line" :size="20" />
            </div>
            <div class="floating-badge badge-2">
              <ArtSvgIcon icon="ri:cloud-line" :size="20" />
            </div>
          </div>
        </div>
      </section>

      <section class="features-section">
        <div class="section-header">
          <h3 class="section-title">核心功能</h3>
          <div class="section-divider"></div>
        </div>

        <div class="features-grid">
          <div v-for="(feature, index) in config.features" :key="index" class="feature-card">
            <div class="card-glow"></div>
            <div class="feature-icon-wrapper">
              <ArtSvgIcon :icon="feature.icon" :size="24" />
            </div>
            <h4 class="feature-title">{{ feature.title }}</h4>
            <p class="feature-desc">{{ feature.description }}</p>
          </div>
        </div>
      </section>

      <section class="capabilities-section">
        <div class="section-header">
          <h3 class="section-title">能力清单</h3>
          <div class="section-divider"></div>
        </div>

        <div class="capabilities-container">
          <div class="capabilities-wrapper">
            <div v-for="(cap, index) in config.capabilities" :key="index" class="capability-tag">
              <ArtSvgIcon icon="ri:checkbox-circle-fill" class="check-icon" :size="16" />
              <span>{{ cap }}</span>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { storeToRefs } from 'pinia'
  import { usePlatformStore } from '@/store/modules/platform'
  import { getPlatformConfig, getDefaultPlatformConfig } from './config'

  defineOptions({ name: 'PlatformIntroduction' })

  const platformStore = usePlatformStore()
  const { currentPlatformId } = storeToRefs(platformStore)

  // 获取配置
  const config = computed(() => {
    const platformConfig = getPlatformConfig(currentPlatformId.value || 1)
    return platformConfig || getDefaultPlatformConfig()
  })

  // 动态样式变量
  const pageStyle = computed(() => ({
    '--theme-color': config.value.themeColor,
    '--bg-gradient': config.value.bgGradient
  }))
</script>

<style lang="scss" scoped>
  // ================= 变量定义 =================
  $border-radius-lg: 24px;
  $border-radius-md: 16px;
  $border-radius-sm: 8px;

  .platform-intro-page {
    position: relative;
    min-height: 100%;
    padding: 24px;
    background-color: var(--art-main-bg); // 使用系统底色
    overflow-x: hidden;
    font-family:
      -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  }

  // 背景光效
  .ambient-light {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 600px;
    background-image: var(--bg-gradient);
    opacity: 0.6;
    filter: blur(60px);
    pointer-events: none;
    z-index: 0;
  }

  .page-container {
    position: relative;
    z-index: 1;
    max-width: 1280px;
    margin: 0 auto;
  }

  // ================= 1. Hero 区域 =================
  .hero-section {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 60px;
    padding: 60px 0 80px;

    .hero-content {
      flex: 1;
      max-width: 600px;
      animation: slideUpFade 0.8s cubic-bezier(0.2, 0.8, 0.2, 1);
    }

    .platform-meta {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 24px;

      .platform-icon-box {
        width: 56px;
        height: 56px;
        background: #fff;
        border-radius: 16px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--theme-color);
        box-shadow: 0 10px 20px -5px rgba(0, 0, 0, 0.08);
      }

      .platform-en-name {
        font-size: 14px;
        font-weight: 600;
        color: var(--theme-color);
        text-transform: uppercase;
        letter-spacing: 1px;
        background: rgba(255, 255, 255, 0.5);
        padding: 4px 12px;
        border-radius: 20px;
        border: 1px solid rgba(0, 0, 0, 0.05);
      }
    }

    .hero-title {
      font-size: 48px;
      font-weight: 800;
      color: var(--art-text-primary);
      line-height: 1.1;
      margin-bottom: 12px;
      letter-spacing: -1px;
    }

    .hero-subtitle {
      font-size: 24px;
      font-weight: 500;
      color: var(--art-text-secondary);
      margin-bottom: 24px;
    }

    .hero-desc {
      font-size: 16px;
      line-height: 1.8;
      color: var(--art-text-gray);
      margin-bottom: 40px;
    }

    // 数据指标胶囊
    .stats-row {
      display: inline-flex;
      background: rgba(255, 255, 255, 0.6);
      backdrop-filter: blur(12px);
      border: 1px solid rgba(255, 255, 255, 0.4);
      border-radius: 20px;
      padding: 20px 32px;
      gap: 48px;
      box-shadow: 0 4px 24px -4px rgba(0, 0, 0, 0.05);

      .stat-item {
        display: flex;
        flex-direction: column;
        gap: 4px;

        .stat-value {
          font-size: 28px;
          font-weight: 700;
          color: var(--art-text-primary);
          font-feature-settings: 'tnum'; // 等宽数字

          .stat-suffix {
            font-size: 14px;
            margin-left: 2px;
            color: var(--art-text-secondary);
            font-weight: 500;
          }
        }

        .stat-label {
          font-size: 12px;
          color: var(--art-text-gray);
        }
      }
    }
  }

  // 右侧视觉图
  .hero-visual {
    flex: 1;
    height: 400px;
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    animation: float 6s ease-in-out infinite;

    .visual-circle {
      position: absolute;
      border-radius: 50%;
      filter: blur(40px);
      opacity: 0.4;

      &.circle-1 {
        width: 300px;
        height: 300px;
        background: var(--theme-color);
        top: 0;
        right: 20px;
      }

      &.circle-2 {
        width: 240px;
        height: 240px;
        background: #8b5cf6;
        bottom: 20px;
        left: 40px;
      }
    }

    // 玻璃拟态卡片
    .glass-card {
      position: relative;
      width: 380px;
      height: 260px;
      background: rgba(255, 255, 255, 0.7);
      backdrop-filter: blur(20px);
      border: 1px solid rgba(255, 255, 255, 0.6);
      border-radius: $border-radius-lg;
      box-shadow: 0 20px 40px -10px rgba(0, 0, 0, 0.1);
      padding: 24px;
      z-index: 2;

      .card-header {
        display: flex;
        gap: 8px;
        margin-bottom: 24px;

        .dot {
          width: 10px;
          height: 10px;
          border-radius: 50%;
          &.red {
            background: #ff5f56;
          }
          &.yellow {
            background: #ffbd2e;
          }
          &.green {
            background: #27c93f;
          }
        }
      }

      .card-body {
        .skeleton-line {
          height: 12px;
          background: rgba(0, 0, 0, 0.05);
          border-radius: 6px;
          margin-bottom: 12px;

          &.w-70 {
            width: 70%;
          }
          &.w-50 {
            width: 50%;
          }
          &.w-90 {
            width: 90%;
          }
        }

        .skeleton-graph {
          margin-top: 24px;
          height: 80px;
          background: linear-gradient(180deg, var(--theme-color) 0%, transparent 100%);
          opacity: 0.1;
          border-radius: 8px;
          border-top: 2px solid var(--theme-color);
        }
      }

      .floating-badge {
        position: absolute;
        width: 48px;
        height: 48px;
        background: #fff;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        box-shadow: 0 10px 20px rgba(0, 0, 0, 0.1);
        color: var(--theme-color);

        &.badge-1 {
          top: -20px;
          right: -20px;
          animation: floatBadge 4s ease-in-out infinite 1s;
        }
        &.badge-2 {
          bottom: -20px;
          left: -20px;
          animation: floatBadge 4s ease-in-out infinite;
        }
      }
    }
  }

  // ================= 2. 功能区域 (Bento Grid) =================
  .section-header {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 32px;

    .section-title {
      font-size: 20px;
      font-weight: 700;
      color: var(--art-text-primary);
      white-space: nowrap;
    }

    .section-divider {
      height: 1px;
      width: 100%;
      background: linear-gradient(90deg, var(--art-border-color) 0%, transparent 100%);
    }
  }

  .features-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 24px;
    margin-bottom: 60px;

    .feature-card {
      position: relative;
      background: #fff;
      border: 1px solid var(--art-border-dashed);
      border-radius: $border-radius-md;
      padding: 32px;
      transition: all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1);
      overflow: hidden;
      cursor: default;

      // 悬浮时的光晕效果
      .card-glow {
        position: absolute;
        top: 0;
        right: 0;
        width: 120px;
        height: 120px;
        background: var(--theme-color);
        filter: blur(80px);
        opacity: 0;
        transition: opacity 0.4s ease;
        pointer-events: none;
      }

      &:hover {
        transform: translateY(-4px);
        box-shadow: 0 12px 32px rgba(0, 0, 0, 0.06);
        border-color: transparent;

        .card-glow {
          opacity: 0.1;
        }

        .feature-icon-wrapper {
          background: var(--theme-color);
          color: #fff;
          transform: scale(1.1);
        }
      }

      .feature-icon-wrapper {
        width: 48px;
        height: 48px;
        background: var(--art-bg-light);
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: var(--theme-color);
        margin-bottom: 20px;
        transition: all 0.3s ease;
      }

      .feature-title {
        font-size: 18px;
        font-weight: 600;
        color: var(--art-text-primary);
        margin-bottom: 12px;
        position: relative;
      }

      .feature-desc {
        font-size: 14px;
        line-height: 1.6;
        color: var(--art-text-secondary);
        position: relative;
      }
    }
  }

  // ================= 3. 能力清单 =================
  .capabilities-container {
    background: linear-gradient(145deg, #ffffff 0%, #f9fafb 100%);
    border: 1px solid var(--art-border-dashed);
    border-radius: $border-radius-lg;
    padding: 40px;

    .capabilities-wrapper {
      display: flex;
      flex-wrap: wrap;
      gap: 12px;

      .capability-tag {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 10px 20px;
        background: #fff;
        border: 1px solid var(--art-border-color);
        border-radius: 50px;
        font-size: 14px;
        color: var(--art-text-primary);
        font-weight: 500;
        transition: all 0.2s ease;

        .check-icon {
          color: var(--theme-color);
          opacity: 0.8;
        }

        &:hover {
          border-color: var(--theme-color);
          background: var(--theme-color);
          color: #fff;
          transform: translateY(-2px);
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);

          .check-icon {
            color: #fff;
            opacity: 1;
          }
        }
      }
    }
  }

  // ================= 响应式 =================
  @media (max-width: 1024px) {
    .hero-section {
      flex-direction: column;
      padding-top: 40px;
      gap: 40px;
      text-align: center;

      .hero-content {
        margin: 0 auto;
      }

      .platform-meta {
        justify-content: center;
      }

      .stats-row {
        width: 100%;
        justify-content: space-around;
        gap: 12px;
        padding: 20px;
      }
    }

    .features-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 768px) {
    .hero-title {
      font-size: 32px;
    }

    .features-grid {
      grid-template-columns: 1fr;
    }

    .hero-visual {
      display: none; // 移动端隐藏复杂视觉图
    }
  }

  // 动画 Keyframes
  @keyframes slideUpFade {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes float {
    0%,
    100% {
      transform: translateY(0);
    }
    50% {
      transform: translateY(-10px);
    }
  }

  @keyframes floatBadge {
    0%,
    100% {
      transform: translateY(0);
    }
    50% {
      transform: translateY(-6px);
    }
  }
</style>

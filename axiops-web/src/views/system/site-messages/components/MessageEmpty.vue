<template>
  <div class="message-empty">
    <el-empty description="请选择一条消息查看详情">
      <template #image>
        <el-icon :size="120" color="#c0c4cc">
          <ChatLineSquare />
        </el-icon>
      </template>
      <div class="empty-tips">
        <p>💡 提示</p>
        <ul>
          <li>点击左侧消息列表查看详情</li>
          <li>使用筛选器快速查找消息</li>
          <li>标记已读以保持收件箱整洁</li>
        </ul>
      </div>
    </el-empty>
  </div>
</template>

<script setup lang="ts">
  import { ChatLineSquare } from '@element-plus/icons-vue'
</script>

<style scoped lang="scss">
  .message-empty {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 40px;

    .empty-tips {
      margin-top: 24px;
      text-align: left;
      color: #909399;
      font-size: 14px;

      p {
        font-weight: 600;
        margin-bottom: 12px;
        color: #606266;
      }

      ul {
        padding-left: 20px;

        li {
          margin-bottom: 8px;
          line-height: 1.6;
        }
      }
    }
  }
</style>

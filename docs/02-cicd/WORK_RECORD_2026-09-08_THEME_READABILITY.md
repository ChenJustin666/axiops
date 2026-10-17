# CI/CD 页面主题可读性修复记录

## 问题

项目工作区出现白底白字、边框缺失或局部内容不可读。检查发现 CI/CD 页面大量使用
`--art-bg-color`、`--art-main-bg-color`、`--art-border-color` 和
`--art-text-gray-*`，但全局主题文件没有定义这些变量，浏览器会丢弃对应 CSS 声明。

## 修改

- 在 `axiops-web/src/assets/styles/core/tailwind.css` 为亮色/暗色主题补齐统一变量别名。
- 将项目工作区根背景、服务目录背景和首次加载遮罩改为主题变量。

## 数据库与重启

- 无 SQL，不需要执行迁移。
- 不需要重启后端；前端开发服务需刷新，若使用生产构建则重新构建并重启 Web 服务。

## 验证

- `npm.cmd run build`：通过（Vite 6262 modules）。
- 浏览器验证重点：项目列表、项目工作区、服务编辑器在亮色/暗色主题下文字与背景均保持对比度。

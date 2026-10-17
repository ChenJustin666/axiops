import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'
import vueDevTools from 'vite-plugin-vue-devtools'
import viteCompression from 'vite-plugin-compression'
import Components from 'unplugin-vue-components/vite'
import AutoImport from 'unplugin-auto-import/vite'
import ElementPlus from 'unplugin-element-plus/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import tailwindcss from '@tailwindcss/vite'
// import { visualizer } from 'rollup-plugin-visualizer'

const iconifyVirtualModuleId = 'virtual:iconify-icons'
const resolvedIconifyVirtualModuleId = `\0${iconifyVirtualModuleId}`
const iconifyVirtualModulePrefix = `${iconifyVirtualModuleId}/`
const configDir = path.dirname(fileURLToPath(import.meta.url))
const resolvedIconifyVirtualModulePrefix = `\0${iconifyVirtualModulePrefix}`
const iconifyPrefixes = [
  'ri',
  'svg-spinners',
  'line-md',
  'logos',
  'simple-icons',
  'devicon',
  'skill-icons',
  'carbon',
  'mdi'
]

export default ({ mode }: { mode: string }) => {
  const root = process.cwd()
  const env = loadEnv(mode, root)
  const { VITE_VERSION, VITE_PORT, VITE_BASE_URL, VITE_API_URL } = env

  console.log(`🚀 API_URL = ${VITE_API_URL}`)
  console.log(`🚀 VERSION = ${VITE_VERSION}`)

  return defineConfig({
    define: {
      __APP_VERSION__: JSON.stringify(VITE_VERSION)
    },
    base: VITE_BASE_URL,
    server: {
      port: Number(VITE_PORT),
      proxy: {
        // New AxiOps APIs both expose /api/v1. Keep their browser routes
        // distinct and remove only the service prefix before forwarding.
        '/api/platform': {
          target: env.VITE_PLATFORM_API_TARGET || 'http://localhost:8090',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api\/platform/, '/api')
        },
        '/api/cicd': {
          target: env.VITE_CICD_API_TARGET || 'http://localhost:8091',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api\/cicd/, '/api')
        },
        '/api/portal': {
          target: 'http://localhost:8810',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, '')
        },
        '/api/manager': {
          target: 'http://localhost:8811',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, '')
        },
        '/api/workload': {
          target: 'http://localhost:8812',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, '')
        },
        '/api/console': {
          target: 'http://localhost:8818',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, '')
        },
        '/api/devops': {
          target: 'http://localhost:8813',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, '')
        },
        '/ws/v1/pod': {
          target: 'http://localhost:8818',
          changeOrigin: true,
          ws: true
        },
        '/ws/v1/site-messages': {
          target: 'http://localhost:8810',
          changeOrigin: true,
          ws: true
        }
      },
      host: true
    },
    // 路径别名
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
        '@views': resolvePath('src/views'),
        '@imgs': resolvePath('src/assets/images'),
        '@icons': resolvePath('src/assets/icons'),
        '@utils': resolvePath('src/utils'),
        '@stores': resolvePath('src/store'),
        '@styles': resolvePath('src/assets/styles')
      }
    },
    build: {
      target: 'es2015',
      outDir: 'dist',
      chunkSizeWarningLimit: 2000,
      minify: 'terser',
      terserOptions: {
        compress: {
          // 生产环境去除 console
          drop_console: true,
          // 生产环境去除 debugger
          drop_debugger: true
        }
      },
      dynamicImportVarsOptions: {
        warnOnError: true,
        exclude: [],
        include: ['src/views/**/*.vue']
      }
    },
    plugins: [
      vue(),
      iconifySubsetPlugin(root),
      tailwindcss(),
      // 自动按需导入 API
      AutoImport({
        imports: ['vue', 'vue-router', 'pinia', '@vueuse/core'],
        dts: 'src/types/import/auto-imports.d.ts',
        resolvers: [ElementPlusResolver()],
        eslintrc: {
          enabled: true,
          filepath: './.auto-import.json',
          globalsPropValue: true
        }
      }),
      // 自动按需导入组件
      Components({
        dts: 'src/types/import/components.d.ts',
        resolvers: [ElementPlusResolver()]
      }),
      // 按需定制主题配置
      ElementPlus({
        useSource: true
      }),
      // 压缩
      viteCompression({
        verbose: false, // 是否在控制台输出压缩结果
        disable: false, // 是否禁用
        algorithm: 'gzip', // 压缩算法
        ext: '.gz', // 压缩后的文件名后缀
        threshold: 10240, // 只有大小大于该值的资源会被处理 10240B = 10KB
        deleteOriginFile: false // 压缩后是否删除原文件
      }),
      vueDevTools()
      // 打包分析
      // visualizer({
      //   open: true,
      //   gzipSize: true,
      //   brotliSize: true,
      //   filename: 'dist/stats.html' // 分析图生成的文件名及路径
      // }),
    ],
    // 依赖预构建：避免运行时重复请求与转换，提升首次加载速度
    optimizeDeps: {
      include: [
        'echarts/core',
        'echarts/charts',
        'echarts/components',
        'echarts/renderers',
        'xlsx',
        'xgplayer',
        'crypto-js',
        'file-saver',
        'vue-img-cutter',
        'element-plus/es',
        'element-plus/es/components/*/style/css',
        'element-plus/es/components/*/style/index'
      ]
    },
    css: {
      preprocessorOptions: {
        // sass variable and mixin
        scss: {
          additionalData: `
            @use "@styles/core/el-light.scss" as *; 
            @use "@styles/core/mixin.scss" as *;
          `
        }
      },
      postcss: {
        plugins: [
          {
            postcssPlugin: 'internal:charset-removal',
            AtRule: {
              charset: (atRule) => {
                if (atRule.name === 'charset') {
                  atRule.remove()
                }
              }
            }
          }
        ]
      }
    }
  })
}

function resolvePath(paths: string) {
  return path.resolve(configDir, paths)
}

function iconifySubsetPlugin(root: string) {
  return {
    name: 'kube-nova-iconify-subset',
    resolveId(id: string) {
      if (id === iconifyVirtualModuleId) return resolvedIconifyVirtualModuleId
      if (id.startsWith(iconifyVirtualModulePrefix)) return `\0${id}`
    },
    load(id: string) {
      if (id === resolvedIconifyVirtualModuleId) {
        const iconNames = [...scanUsedIconifyIcons(root)].sort()
        const prefixes = [...new Set(iconNames.map((icon) => icon.split(':')[0]).filter(Boolean))]
        const loaders = prefixes
          .map(
            (prefix) =>
              `${JSON.stringify(prefix)}: () => import(${JSON.stringify(
                `${iconifyVirtualModulePrefix}${prefix}`
              )})`
          )
          .join(',')

        return [
          `export const iconNames = ${JSON.stringify(iconNames)}`,
          `export const iconLoaders = {${loaders}}`
        ].join('\n')
      }

      if (!id.startsWith(resolvedIconifyVirtualModulePrefix)) return
      const prefix = id.slice(resolvedIconifyVirtualModulePrefix.length)
      return `export default ${JSON.stringify(resolveUsedIconifyIcons(root, prefix))}`
    }
  }
}

function resolveUsedIconifyIcons(root: string, onlyPrefix?: string) {
  const usedIcons = scanUsedIconifyIcons(root)
  const iconSets = new Map<string, any>()
  const result: Record<string, any> = {}

  for (const icon of usedIcons) {
    const [prefix, name] = icon.split(':')
    if (!prefix || !name) continue
    if (onlyPrefix && prefix !== onlyPrefix) continue

    const iconSet = loadIconifySet(root, prefix, iconSets)
    if (!iconSet) continue

    const iconData = resolveIconifyIconData(iconSet, name)
    if (iconData) result[icon] = iconData
  }

  return result
}

function scanUsedIconifyIcons(root: string) {
  const result = new Set<string>()
  const iconPattern =
    /(?<![\w-])([a-z0-9]+(?:-[a-z0-9]+)*):([a-z0-9]+(?:[-_.][a-z0-9]+)*)(?![\w-])/gi
  const scanTargets = [
    { path: path.join(root, 'src'), pattern: /\.(vue|ts|js|tsx|jsx)$/ },
    { path: path.join(root, '..', 'kube-nova', 'sql'), pattern: /\.sql$/ },
    {
      path: path.join(
        root,
        '..',
        'kube-nova',
        'application',
        'devops-manager-rpc',
        'internal',
        'model'
      ),
      pattern: /\.go$/
    },
    { path: path.join(root, '..', 'sql'), pattern: /\.sql$/ }
  ]

  const scanContent = (content: string) => {
    for (const match of content.matchAll(iconPattern)) {
      const prefix = match[1]
      if (iconifyPrefixes.includes(prefix)) result.add(`${prefix}:${match[2]}`)
    }
  }

  const scanDir = (dir: string, pattern: RegExp) => {
    if (!fs.existsSync(dir)) return
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const filePath = path.join(dir, entry.name)
      if (entry.isDirectory()) {
        scanDir(filePath, pattern)
        continue
      }
      if (!pattern.test(entry.name)) continue
      scanContent(fs.readFileSync(filePath, 'utf-8'))
    }
  }

  for (const target of scanTargets) {
    scanDir(target.path, target.pattern)
  }

  return result
}

function loadIconifySet(root: string, prefix: string, cache: Map<string, any>) {
  if (cache.has(prefix)) return cache.get(prefix)

  const iconSetPath = path.join(root, 'node_modules', '@iconify-json', prefix, 'icons.json')
  if (!fs.existsSync(iconSetPath)) {
    cache.set(prefix, undefined)
    return
  }

  const iconSet = JSON.parse(fs.readFileSync(iconSetPath, 'utf-8'))
  cache.set(prefix, iconSet)
  return iconSet
}

function resolveIconifyIconData(iconSet: any, name: string, stack: string[] = []): any {
  const icon = iconSet.icons?.[name]
  if (icon) return mergeIconifyData(iconSet, icon)

  const alias = iconSet.aliases?.[name]
  if (!alias || stack.includes(name)) return

  const parent = resolveIconifyIconData(iconSet, alias.parent, [...stack, name])
  if (!parent) return

  return mergeIconifyData(parent, alias)
}

function mergeIconifyData(base: any, extra: any) {
  const result = { ...base, ...extra }
  delete result.icons
  delete result.aliases
  delete result.chars
  delete result.prefix
  delete result.provider
  delete result.info
  delete result.metadata
  delete result.hidden
  delete result.parent
  return result
}

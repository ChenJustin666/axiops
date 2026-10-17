import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'
import * as vue from 'vue'

function setup() {
  const source = readFileSync(new URL('../src/store/modules/user.ts', import.meta.url), 'utf8')
  const code = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
  }).outputText
  let resets = 0,
    logoutDone
  const storage = { getItem: () => null, setItem() {}, removeItem() {} }
  const module = { exports: {} }
  const modules = {
    vue,
    pinia: { defineStore: (_, factory) => factory },
    '@/composables/axiops-access-state': { resetAxiOpsAccess: () => resets++ },
    '@/api/axiops/auth': {
      logoutAxiOpsApi: () =>
        new Promise((resolve) => {
          logoutDone = resolve
        })
    },
    '@/enums/appEnum': { LanguageEnum: { ZH: 'zh' } },
    '@/router': { router: { push() {} } },
    './worktab': { useWorktabStore: () => ({}) },
    './platform': { usePlatformStore: () => ({ clearPlatformState() {} }) },
    './menu': { useMenuStore: () => ({ setHomePath() {} }) },
    '@/store/modules/project.ts': { useProjectStore: () => ({ clearAllStoredData() {} }) },
    '@/router/guards/beforeEach': { resetRouterState() {} },
    '@/utils/storage/storage-config': {
      StorageConfig: { LAST_USER_ID_KEY: 'last', createKeyPattern: () => /^unmatched$/ }
    }
  }
  vm.runInNewContext(code, {
    module,
    exports: module.exports,
    require: (name) => modules[name] || {},
    localStorage: storage,
    sessionStorage: storage,
    console
  })
  return {
    state: module.exports.useUserStore(),
    resets: () => resets,
    finishLogout: () => logoutDone()
  }
}
test('new role snapshots replace revoked roles, explicit empty clears roles, partial profile updates preserve them', () => {
  const v = setup()
  v.state.setUserInfo({ userId: 1, roles: ['super_admin'] })
  v.state.setUserInfo({ roles: ['viewer', 'viewer'] })
  assert.deepEqual(Array.from(v.state.info.value.roles), ['viewer'])
  v.state.setUserInfo({ nickName: 'new name' })
  assert.deepEqual(Array.from(v.state.info.value.roles), ['viewer'])
  v.state.setUserInfo({ roles: [] })
  assert.equal(v.state.info.value.roles.length, 0)
})
test('changing account clears old role and profile data and resets policy state', () => {
  const v = setup()
  v.state.setUserInfo({ userId: 1, roles: ['super_admin'], email: 'old@example.invalid' })
  const before = v.resets()
  v.state.setUserInfo({ userId: 2, userName: 'reader' })
  assert.equal(v.state.info.value.roles, undefined)
  assert.equal(v.state.info.value.email, undefined)
  assert.equal(v.resets(), before + 1)
})
test('token replacement and logout invalidate policies before awaiting network', async () => {
  const v = setup()
  v.state.setToken('first')
  const before = v.resets()
  v.state.setToken('first')
  assert.equal(v.resets(), before)
  v.state.setToken('second')
  assert.equal(v.resets(), before + 1)
  const logout = v.state.logOut()
  assert.equal(v.resets(), before + 2)
  v.finishLogout()
  await logout
  assert.equal(v.state.accessToken.value, '')
})

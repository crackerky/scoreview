import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'

// CLI からのデプロイで実在の個人名を含むデータを送らない(セキュリティ点検 L-7)。

test('.vercelignore が取込データとビルド済みのバイナリと秘密を除く', () => {
  const lines = readFileSync(new URL('../../../.vercelignore', import.meta.url), 'utf8')
    .split('\n')
    .map((line) => line.trim())
  for (const entry of ['.local/', 'apps/api/bin/', '.env', '.env.local']) {
    assert.ok(lines.includes(entry), `${entry} が無い`)
  }
})

import { readFileSync } from 'node:fs'

// fixture の JSON は frame を "frame-1" のような名前で参照する(同じ frame を各ファイルに重複させないため)。
// 読み込み時に frame-*.json の中身へ差し替えて、api の応答と同じ形にする。

function readJson(name: string): unknown {
  const url = new URL(`./${name}.json`, import.meta.url)
  return JSON.parse(readFileSync(url, 'utf8'))
}

/** test/fixtures/{name}.json を読み、frame の参照を展開した api 応答相当の値を返す。 */
export function loadFixture(name: string): unknown {
  const body = readJson(name)
  if (typeof body === 'object' && body !== null && 'frame' in body) {
    const ref = body.frame
    if (typeof ref === 'string') {
      return { ...body, frame: readJson(ref) }
    }
  }
  return body
}

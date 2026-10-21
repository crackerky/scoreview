import type { ReactNode } from 'react'

// clone-spec A 7: 参照サイト(CodeIgniter)の素朴なエラー画面。外枠もサイトの CSS も使わない。
// 参照サイトは HTTP 500 だが、ここでは 404 で返す(requirements「意図的な差分」)。
// 文面は lang に関係なく日本語固定。

// 参照サイトのエラー画面の <style> の値。legacy.css の html / body / h1 の規則が漏れないよう打ち消す
const ciErrorCss = `
html { overflow: visible; }
::selection { background-color: #E13300; color: white; }
body {
  background-color: #fff;
  margin: 40px;
  font: 13px/20px normal Helvetica, Arial, sans-serif;
  color: #4F5155;
  letter-spacing: normal;
}
a { color: #003399; background-color: transparent; font-weight: normal; }
h1 {
  color: #444;
  background-color: transparent;
  border-bottom: 1px solid #D0D0D0;
  font-size: 19px;
  font-weight: normal;
  line-height: normal;
  margin: 0 0 14px 0;
  padding: 14px 15px 10px 15px;
}
#container { margin: 10px; border: 1px solid #D0D0D0; box-shadow: 0 0 8px #D0D0D0; }
p { margin: 12px 15px 12px 15px; }
`

function CiErrorPage({
  title,
  heading,
  children,
}: {
  title: string
  heading: string
  children: ReactNode
}) {
  return (
    <>
      <title>{title}</title>
      <style>{ciErrorCss}</style>
      <div id="container">
        <h1>{heading}</h1>
        <p>{children}</p>
      </div>
    </>
  )
}

// 参照サイトの戻り先は管理画面のログイン。推測: 本サイトの管理画面も同じパスに置く前提
const menuUrl = '/auth/login'

/**
 * 本サイト独自: 戻り先の無いエラー画面に付ける大会一覧へのリンク。参照サイトは戻り先を出さず、
 * URL を誤った観戦者が行き止まりになる(docs/requirements.md「意図的な差分」)
 */
function ListLink() {
  return (
    <>
      <br />
      <br />
      <a href="/sctnmt" data-scoreview-addition="">
        ←大会一覧へ
      </a>
    </>
  )
}

/**
 * ID が存在しないときの画面。message は api の not_found の message("TOURNAMENT_ID [999999]" など)。
 * withMenuLink: /scview/{id} だけは「メニュー画面より…」の文と戻るリンクが付く(clone-spec A 7)。
 */
export function NotFoundScreen({
  message,
  withMenuLink,
}: {
  message: string
  withMenuLink: boolean
}) {
  return (
    <CiErrorPage title="Error" heading="An Error Was Encountered">
      {message}での操作に失敗しました。
      <br />
      情報が既に削除されているか、システムエラーの可能性があります。
      <br />
      {withMenuLink && (
        <>
          メニュー画面より、再操作し確認して下さい。
          <br />
          <br />
          <a href={menuUrl}>←メニュー画面に戻る</a>
        </>
      )}
      {!withMenuLink && <ListLink />}
    </CiErrorPage>
  )
}

/** ルートに合わない URL(ID が数値でないなど)の画面(clone-spec A 7 の /scview/abc)。 */
export function RouteNotFoundScreen() {
  return (
    <CiErrorPage title="404 Page Not Found" heading="404 Page Not Found">
      The page you requested was not found.
      <ListLink />
    </CiErrorPage>
  )
}

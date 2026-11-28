import type { ReactNode } from 'react'
import { SiteShell } from '@/components/site/SiteShell'

// サイトの下の「個人情報保護方針」の行き先。このサイトが実際に扱う情報(公開する大会の記録、
// 管理画面のアカウント、アクセスの記録)だけを書く。正式公開のときに運営会社の
// 方針と照らして見直す
function InfoSection({ heading, children }: { heading: string; children: ReactNode }) {
  return (
    <section className="site-doc-section">
      <h2>{heading}</h2>
      {children}
    </section>
  )
}

export default function PrivacyPage() {
  return (
    <SiteShell>
      <title>個人情報保護方針 - SCORE VIEW</title>
      <div className="site-doc">
        <h1 className="site-title">個人情報保護方針</h1>
        <p className="site-doc-note">
          デモ版の方針です。正式公開のときに、運営会社の名前と窓口を加えて改めます。
        </p>
        <p>
          SCORE VIEW
          運営事務局(以下「運営者」)は、このサイトで扱う個人情報を、次の方針で取り扱います。
        </p>
        <InfoSection heading="1. 扱う情報">
          <dl>
            <dt>大会の記録</dt>
            <dd>
              チーム名、選手の氏名と背番号、試合の得点とアシストの記録です。大会の主催者が登録し、観戦アプリで公開します。
            </dd>
            <dt>管理画面のアカウント</dt>
            <dd>
              大会の主催者と記録係のメールアドレス、名前、パスワード(元に戻せない形に変換して保存します)です。
            </dd>
            <dt>大会への申込</dt>
            <dd>
              公式 LINE
              から申し込んだときの、チーム名、代表の名前と連絡先(電話番号、メールアドレス)、メンバーの名前と背番号、申込の質問への答えです。大会の主催者が求めた大会では、メンバーの性別と生年月日も記録します。性別と生年月日は主催者の管理画面にだけ出し、観戦アプリやほかのメンバー、代表には見せません。性別は、大会の記録の選手の性別にも使います。あわせて、公式
              LINE から見たあなたの LINE のユーザー ID と、公式 LINE
              を友だちに追加しているかを記録します。次の申込で入力を省けるよう、あなたが最後に入れた名前、連絡先、性別、生年月日と、前に申し込んだチームを覚えておき、あなたが開いた申込の画面にだけ出します。LINE
              の表示名やプロフィールの画像は記録しません。
            </dd>
            <dt>アクセスの記録</dt>
            <dd>
              ページを表示したときの IP
              アドレス、日時、ブラウザの種類です。不正なアクセスの調査に使います。
            </dd>
          </dl>
        </InfoSection>
        <InfoSection heading="2. 使う目的">
          <ul>
            <li>大会の日程、結果、順位を公開するため</li>
            <li>大会の主催者と記録係が管理画面にログインするため</li>
            <li>大会への申込を受け付け、申し込んだ人に公式 LINE で大会のお知らせを送るため</li>
            <li>
              次の申込で、前に入れた名前、連絡先、性別、生年月日、チームを最初から入れておくため
            </li>
            <li>
              大会の主催者が、申込で求めた項目(性別、生年月日、質問への答え)を大会の運営に使うため
            </li>
            <li>お問い合わせに回答するため</li>
            <li>サイトの不正な利用を防ぎ、障害を調べるため</li>
          </ul>
        </InfoSection>
        <InfoSection heading="3. 第三者への提供">
          <p>
            法令に基づく場合を除き、本人の同意なく第三者に提供しません。大会の記録は、主催者の判断で観戦アプリに公開されます。
          </p>
        </InfoSection>
        <InfoSection heading="4. 外部のサービス">
          <p>
            サイトの配信とデータの保存に、外部のクラウドサービスを使います。申込の受付とお知らせには、LINE
            ヤフー株式会社の LINE 公式アカウントを使います。広告の配信と、閲覧者を追跡するための
            Cookie は使いません。管理画面では、ログインの状態を保つための Cookie だけを使います。
          </p>
        </InfoSection>
        <InfoSection heading="5. 掲載内容の訂正と削除">
          <p>
            選手の氏名などの掲載内容の訂正や削除は、その大会の主催者へご連絡ください。主催者に連絡がつかない場合は、運営事務局へご連絡ください。
          </p>
        </InfoSection>
        <InfoSection heading="6. 方針の変更">
          <p>この方針を変えるときは、このページでお知らせします。</p>
        </InfoSection>
      </div>
    </SiteShell>
  )
}

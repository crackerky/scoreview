import { InfoPage, InfoSection } from '@/components/InfoPage'

// ヘッダーの「お問い合わせ」の行き先。デモ版では受け付けの窓口を持たず、訂正の連絡先と、
// 窓口を正式公開のときに載せることだけを案内する(参照サイトのヘッダーはメールのリンク)
export default function ContactPage() {
  return (
    <InfoPage title="お問い合わせ">
      <p className="sv_doc_note">
        デモ版のため、お問い合わせの窓口(メールアドレス)は正式公開のときにこのページに載せます。
      </p>
      <InfoSection heading="大会の日程、結果、選手の掲載内容について">
        <p>
          訂正や削除は、その大会の主催者へご連絡ください。主催者が管理画面で直すと、閲覧ページにすぐ反映されます。
        </p>
      </InfoSection>
      <InfoSection heading="掲載や導入のご相談">
        <p>
          大会の掲載やサイトの導入のご相談は、正式公開のときに載せる窓口で受け付けます。サイトでできることは
          <a href="/about#organizers">大会主催者の方へ</a>をご覧ください。
        </p>
      </InfoSection>
      <p>
        このサイトに載っている大会とチームは架空のものです。<a href="/sctnmt">大会一覧へ戻る</a>
      </p>
    </InfoPage>
  )
}

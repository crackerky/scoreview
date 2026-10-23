import { InfoPage, InfoSection } from '@/components/InfoPage'

// ヘッダーの「About」「Organizers」「運営者情報」の行き先
export default function AboutPage() {
  return (
    <InfoPage title="SCORE VIEW について">
      <p className="sv_doc_note">
        このサイトはデモ版です。載っている大会、チーム、選手はすべて架空のものです。
      </p>
      <InfoSection heading="観戦者と選手の方へ">
        <p>
          SCORE VIEW は、フライングディスク競技(アルティメット、ビーチアルティメット、
          ディスクゴルフなど)の大会のスコア速報サイトです。会場でも自宅でも、スマートフォンから次のことを確かめられます。
        </p>
        <ul>
          <li>試合の日程(日付、時刻、コート、対戦相手)と、試合中のスコア</li>
          <li>プールの順位表とトーナメント表、ディビジョンの最終順位</li>
          <li>チームごとの成績と次の試合、選手ごとの得点とアシスト</li>
          <li>試合の得点経過(誰が決めて誰がアシストしたか)</li>
        </ul>
        <p>
          ページの見方は<a href="/guide">見方ガイド</a>にまとめています。大会は
          <a href="/sctnmt">大会一覧</a>から選んでください。
        </p>
      </InfoSection>
      <InfoSection id="organizers" heading="大会主催者の方へ">
        <p>
          SCORE VIEW は、スコアを載せるだけでなく、大会運営の手間を減らすための道具です。
          本部が夜まで表計算に転記する作業を、なくすことを目指しています。
        </p>
        <h3>大会の前</h3>
        <ul>
          <li>
            チーム数とコート数を入れると、大会の形(プール、順位決定トーナメント)の案が出ます。
          </li>
          <li>前回の大会の形をコピーして、今回のコート数と時刻で作り直せます。</li>
          <li>組み合わせ表と時間割を自動で作り、印刷用のスコアシートを出せます。</li>
        </ul>
        <h3>大会の当日</h3>
        <ul>
          <li>
            記録係はスマートフォンで、背番号のボタンを押して得点を記録します。電波が切れても端末に残し、つながったときに送ります。
          </li>
          <li>
            試合の結果を入れると、順位表、トーナメントの勝ち上がり、閲覧ページが同時に変わります。
          </li>
          <li>
            プールの結果は本部が「確定」を押してから次の段に流れます。同率が規程で決まらないときは、止まって知らせます。
          </li>
          <li>遅れや雷雨による中断を、時間割と閲覧ページにまとめて反映できます。</li>
        </ul>
        <p>
          掲載や導入のご相談は<a href="/contact">お問い合わせ</a>からどうぞ。
        </p>
      </InfoSection>
      <InfoSection id="operator" heading="運営者情報">
        <dl>
          <dt>サイト名</dt>
          <dd>SCORE VIEW(スコアビュー)</dd>
          <dt>運営</dt>
          <dd>SCORE VIEW 運営事務局(デモ版のため、運営会社の情報は正式公開のときに載せます)</dd>
          <dt>お問い合わせ</dt>
          <dd>
            <a href="/contact">お問い合わせ</a>
          </dd>
          <dt>個人情報の扱い</dt>
          <dd>
            <a href="/privacy">個人情報保護方針</a>
          </dd>
        </dl>
      </InfoSection>
    </InfoPage>
  )
}

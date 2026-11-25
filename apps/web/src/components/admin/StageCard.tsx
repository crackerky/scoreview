import type { Stage } from '@/admin/tournamentHome.utils'
import { Chip } from './AdminParts'

// 大会ホームの段のカード 1 枚。左に段の名前、今の様子、次にやること、右にその段の入口を並べる。
// 今の段は枠を太くし「次はここ」と添える(主な操作は画面の上の 1 つだけにするので、カードの中の
// 次にやることは枠だけのボタンにする)。
export function StageCard({ stage, current }: { stage: Stage; current: boolean }) {
  const headingId = `stage-${stage.key}`
  return (
    <section className={`adm-stage${current ? ' current' : ''}`} aria-labelledby={headingId}>
      <div className="adm-stage-main">
        <div className="adm-stage-head">
          <h2 id={headingId} className="adm-stage-title">
            {stage.title}
          </h2>
          {stage.chip ? <Chip tone={stage.chip.tone}>{stage.chip.label}</Chip> : null}
          {current ? <span className="adm-stage-now">次はここ</span> : null}
        </div>
        <p className="adm-stage-status">{stage.status}</p>
        {stage.numbers ? (
          <dl className="adm-stage-numbers">
            <div>
              <dt>いま試合中</dt>
              <dd className={stage.numbers.live > 0 ? 'live' : undefined}>{stage.numbers.live}</dd>
            </div>
            <div>
              <dt>終わった試合</dt>
              <dd>
                {stage.numbers.finished} / {stage.numbers.total}
              </dd>
            </div>
            <div>
              <dt>本部の知らせ</dt>
              <dd className={stage.numbers.alerts > 0 ? 'warn' : undefined}>
                {stage.numbers.alerts}
              </dd>
            </div>
          </dl>
        ) : null}
        {stage.note ? <p className="adm-stage-note">{stage.note}</p> : null}
        {stage.next ? (
          <a className="adm-stage-next" href={stage.next.href}>
            {stage.next.label}
            <span aria-hidden="true">›</span>
          </a>
        ) : null}
      </div>
      {stage.links.length > 0 ? (
        <ul className="adm-stage-links">
          {stage.links.map((link) => (
            <li key={link.href} className="adm-stage-item">
              <a className="adm-stage-link" href={link.href}>
                <span className="adm-stage-link-text">
                  <span className="adm-stage-link-label">{link.label}</span>
                  {link.sub ? <span className="adm-stage-link-sub">{link.sub}</span> : null}
                </span>
                {link.chip ? <Chip tone={link.chip.tone}>{link.chip.label}</Chip> : null}
                <span className="adm-chevron" aria-hidden="true">
                  ›
                </span>
              </a>
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  )
}

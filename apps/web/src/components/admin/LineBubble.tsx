import type { NotificationButton } from '@/api/notify.types'
import { buttonDestination } from '@/notify/notify.utils'

type Button = Pick<NotificationButton, 'kind' | 'label' | 'url'>

// 公式 LINE で届く形の見本。ボタンが無ければ文だけの吹き出し、あれば文の下にボタンを並べたカード
// (api が送る Flex Message と同じ並び。先頭のボタンだけ色を付ける)。見本なので、ボタンは押せない。
// 行き先は見本の下に書く(運営が送る前と送った後に、どこへ行くボタンかを確かめられるように)。
export function LineBubble({ text, buttons }: { text: string; buttons: readonly Button[] }) {
  if (buttons.length === 0) {
    return <p className="adm-bubble">{text}</p>
  }
  return (
    <>
      <div className="adm-bubble adm-bubble-card">
        <p className="adm-bubble-text">{text}</p>
        <div className="adm-bubble-buttons">
          {buttons.map((button, index) => (
            <span
              key={`${button.kind}:${button.label}:${button.url}`}
              className={index === 0 ? 'adm-bubble-button primary' : 'adm-bubble-button'}
            >
              {button.label}
            </span>
          ))}
        </div>
      </div>
      <ul className="adm-bubble-dest">
        {buttons.map((button) => (
          <li key={`${button.kind}:${button.label}:${button.url}`}>
            {button.label}: {buttonDestination(button)}
          </li>
        ))}
      </ul>
    </>
  )
}

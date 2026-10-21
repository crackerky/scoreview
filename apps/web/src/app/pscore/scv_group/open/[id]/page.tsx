import { getGroup } from '@/api/public'
import { loadPageData } from '@/app/_lib/notFound'
import { type IdPageProps, readIdPage } from '@/app/_lib/pageProps'
import { GroupView } from '@/components/GroupView'
import { SiteFrame } from '@/components/SiteFrame'
import { tournamentPageTitle } from '@/components/title.utils'
import { pickText } from '@/i18n/lang'

// clone-spec C 2: グループ成績。title の接尾辞は "-{グループ名}-"
export default async function GroupPage(props: IdPageProps) {
  const { id, lang } = await readIdPage(props)
  const data = await loadPageData(id, getGroup)
  // 本文の上の「HH:MM 時点」(api から読み終えた時刻)
  const now = new Date()
  return (
    <>
      <title>{tournamentPageTitle(data.frame, lang, `-${data.group.name}-`)}</title>
      <SiteFrame
        heading={`${pickText(lang, data.group.division.name)} ${data.group.name}`}
        asOf={now}
        variant={{ kind: 'tournament', frame: data.frame, lang, scoreMenu: true }}
        topMenu={null}
      >
        <GroupView data={data} lang={lang} />
      </SiteFrame>
    </>
  )
}
